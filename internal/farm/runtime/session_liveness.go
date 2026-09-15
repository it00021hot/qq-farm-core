// 会话存活持久化：记录每个账号「最后确认在线」的时刻。
// 移植 rust crates/qq-farm-core/src/infra/session_liveness.rs。
//
// 用途：进程重启后的自动重连必须避开服务端旧 session 释放窗口
// （重启前会话还活着、杀进程后 TCP 未优雅登出，立刻重登会被判
// 「已在其他终端登录」连环踢——同踢线重登等 3 分钟的教训）。
// rust 侧 2026-09-11 实测：重启后 0~2 分钟内自动登录的三个账号全部
// 登录后 ~1 秒入站冻结；15 分钟后的重连全部健康。
package runtime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/vars"
	"github.com/it00021hot/qq-farm-core/pkg/config"
)

// sessionLivenessWindow 服务端旧 session 释放窗口：最后确认在线之后的
// 这段时间内，进程重启后的自动登录需要等待剩余时间（rust boot_delay_ms
// 使用 WX_KICKOUT_RECONNECT_DELAY_MS = 3 分钟作窗口）。
const sessionLivenessWindow = WxKickoutReconnectDelay

// livenessFile 落盘结构：account_id -> 最后确认在线的 epoch 毫秒。
// 文件名与 rust 一致（<data>/session-liveness.json）。
type livenessFile struct {
	LastSeen map[string]int64 `json:"last_seen"`
}

// sessionLiveness 单例状态（rust：static STATE + parking_lot::Mutex）。
type sessionLiveness struct {
	mu          sync.Mutex
	path        string
	data        livenessFile
	lastPersist map[string]time.Time
	loaded      bool
}

// globalLiveness 进程级单例；路径可用 setLivenessPathForTest 注入（仅测试用）。
var globalLiveness = &sessionLiveness{lastPersist: make(map[string]time.Time)}

// defaultLivenessPath 数据目录下的 session-liveness.json（rust liveness_path：
// get_data_dir().join("session-liveness.json")；数据目录对齐 boots/farm.go
// 的 farm.tsdkDataDir 约定，默认 runtime/data）。
// vars.Config 未初始化（如单元测试环境）时直接用默认目录，避免空指针。
func defaultLivenessPath() string {
	root := ""
	if vars.Config != (config.Config{}) {
		root = vars.Config.GetString("farm.tsdkDataDir")
	}
	if root == "" {
		root = "runtime/data"
	}
	return filepath.Join(root, "session-liveness.json")
}

// setLivenessPathForTest 替换落盘路径（仅单元测试使用）。
func setLivenessPathForTest(path string) {
	globalLiveness.mu.Lock()
	defer globalLiveness.mu.Unlock()
	globalLiveness.path = path
	globalLiveness.data = livenessFile{}
	globalLiveness.lastPersist = make(map[string]time.Time)
	globalLiveness.loaded = false
}

// ensureLoadedLocked 首次访问时从磁盘加载（rust with_state 的 lazy load）。
func (l *sessionLiveness) ensureLoadedLocked() {
	if l.loaded {
		return
	}
	if l.path == "" {
		l.path = defaultLivenessPath()
	}
	l.data = loadLivenessFromDisk(l.path)
	l.loaded = true
}

// loadLivenessFromDisk 读失败/损坏时按空数据处理（rust load_from_disk 语义）。
func loadLivenessFromDisk(path string) livenessFile {
	raw, err := os.ReadFile(path)
	if err != nil {
		return livenessFile{}
	}
	var f livenessFile
	if json.Unmarshal(raw, &f) != nil || f.LastSeen == nil {
		return livenessFile{}
	}
	return f
}

// persistLocked 同步写盘（极小文件 + 30s 节流，rust persist 直接 fs::write）。
func (l *sessionLiveness) persistLocked() {
	if l.path == "" {
		l.path = defaultLivenessPath()
	}
	raw, err := json.Marshal(l.data)
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return
	}
	// 与 rust 一致：直接同步写；再补一个同目录临时文件替换，规避写一半损坏。
	tmp := l.path + ".tmp"
	if os.WriteFile(tmp, raw, 0o644) == nil && os.Rename(tmp, l.path) == nil {
		return
	}
	_ = os.WriteFile(l.path, raw, 0o644)
}

// NoteSessionOnline 账号确认在线（登录成功与后续心跳确认时调用，内部节流落盘；
// rust note_online）。
func NoteSessionOnline(accountID string) {
	if accountID == "" {
		return
	}
	now := time.Now()
	l := globalLiveness
	l.mu.Lock()
	defer l.mu.Unlock()
	l.ensureLoadedLocked()
	if l.data.LastSeen == nil {
		l.data.LastSeen = make(map[string]int64)
	}
	l.data.LastSeen[accountID] = now.UnixMilli()
	// 落盘节流（rust PERSIST_THROTTLE_MS）：期间只更新内存。
	if last, ok := l.lastPersist[accountID]; !ok || now.Sub(last) >= SessionLivenessPersistThrottle {
		l.lastPersist[accountID] = now
		l.persistLocked()
	}
}

// NoteSessionOffline 账号确认离线（worker 正常停止时调用，立即落盘清除打点；
// rust note_offline）。清除后重启不再需要等释放窗口。
// 注意：进程被直接杀掉时本函数不会执行，磁盘上的最后在线时间得以保留，
// 这正是重启后延迟重登的依据。
func NoteSessionOffline(accountID string) {
	if accountID == "" {
		return
	}
	l := globalLiveness
	l.mu.Lock()
	defer l.mu.Unlock()
	l.ensureLoadedLocked()
	delete(l.data.LastSeen, accountID)
	delete(l.lastPersist, accountID)
	l.persistLocked()
}

// SessionBootDelay 进程启动后自动登录前应额外等待的时长（0 = 可立即登录；
// rust boot_delay_ms）。上个进程的会话若在释放窗口内（sessionLivenessWindow），
// 等待剩余时间再登，避免撞上服务端旧 session 释放。
func SessionBootDelay(accountID string) time.Duration {
	return globalLiveness.bootDelay(accountID, time.Now())
}

func (l *sessionLiveness) bootDelay(accountID string, now time.Time) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.ensureLoadedLocked()
	seenMs, ok := l.data.LastSeen[accountID]
	if !ok {
		return 0
	}
	delay := time.UnixMilli(seenMs).Add(sessionLivenessWindow).Sub(now)
	if delay <= 0 {
		return 0
	}
	return delay
}
