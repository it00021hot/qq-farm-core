package runtime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// newTestLiveness 独立的 liveness 状态（不落全局单例），路径指向临时目录。
func newTestLiveness(t *testing.T) *sessionLiveness {
	t.Helper()
	return &sessionLiveness{
		path:        filepath.Join(t.TempDir(), "session-liveness.json"),
		lastPersist: make(map[string]time.Time),
		loaded:      true,
	}
}

// TestSessionBootDelayRespectsWindow 对齐 rust session_liveness.rs boot_delay_ms：
// 上次确认在线落在释放窗口内 → 等剩余时间；超出窗口或无记录 → 0。
func TestSessionBootDelayRespectsWindow(t *testing.T) {
	if sessionLivenessWindow != 3*time.Minute {
		t.Fatalf("window=%v, rust 使用 WX_KICKOUT_RECONNECT_DELAY_MS(3 分钟)", sessionLivenessWindow)
	}
	l := newTestLiveness(t)
	now := time.Now()

	// 无记录：可立即登录。
	if d := l.bootDelay("a1", now); d != 0 {
		t.Fatalf("missing record delay=%v, want 0", d)
	}

	// 上次在线 1 秒前：应等剩余窗口（> 0 且 <= 窗口）。
	l.data.LastSeen = map[string]int64{"a1": now.Add(-time.Second).UnixMilli()}
	d := l.bootDelay("a1", now)
	if d <= 0 || d > sessionLivenessWindow {
		t.Fatalf("recent session delay=%v, want (0, %v]", d, sessionLivenessWindow)
	}
	if want := sessionLivenessWindow - time.Second; d > want || d < want-time.Second {
		t.Fatalf("delay=%v, want ≈%v", d, want)
	}

	// 超过窗口（15 分钟前）：可立即登录。
	l.data.LastSeen["a1"] = now.Add(-15 * time.Minute).UnixMilli()
	if d := l.bootDelay("a1", now); d != 0 {
		t.Fatalf("stale record delay=%v, want 0", d)
	}

	// 恰好到达窗口边界：可立即登录。
	l.data.LastSeen["a1"] = now.Add(-sessionLivenessWindow).UnixMilli()
	if d := l.bootDelay("a1", now); d != 0 {
		t.Fatalf("boundary record delay=%v, want 0", d)
	}
}

// TestSessionLivenessPersistAndOffline 落盘/清除往返：note_offline 清除记录后
// 重启不再等待；文件格式与 rust session-liveness.json 一致。
func TestSessionLivenessPersistAndOffline(t *testing.T) {
	l := newTestLiveness(t)
	now := time.Now()
	l.data.LastSeen = map[string]int64{"a1": now.UnixMilli(), "a2": now.UnixMilli()}
	l.persistLocked()

	raw, err := os.ReadFile(l.path)
	if err != nil {
		t.Fatalf("read liveness file: %v", err)
	}
	var f livenessFile
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("liveness file not valid json: %v", err)
	}
	if _, ok := f.LastSeen["a1"]; !ok {
		t.Fatal("a1 must be persisted")
	}

	// 损坏文件按空数据处理（rust load_from_disk 的 unwrap_or_default）。
	if err := os.WriteFile(l.path, []byte("not-json"), 0o644); err != nil {
		t.Fatalf("write garbage: %v", err)
	}
	l2 := &sessionLiveness{path: l.path, lastPersist: make(map[string]time.Time)}
	l2.ensureLoadedLocked()
	if len(l2.data.LastSeen) != 0 {
		t.Fatalf("corrupt file should load empty, got %v", l2.data.LastSeen)
	}
}

// TestSessionLivenessNoteOnlineThrottle 落盘节流（rust PERSIST_THROTTLE_MS=30s）：
// 节流窗口内的 note_online 只更新内存、不重写文件；note_offline 立即落盘。
func TestSessionLivenessNoteOnlineThrottle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "session-liveness.json")
	setLivenessPathForTest(path)
	t.Cleanup(func() { setLivenessPathForTest("") })

	NoteSessionOnline("42")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("first note must persist: %v", err)
	}
	// 删掉磁盘文件模拟“未写盘”，节流窗口内的再次 note 不应触发落盘。
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove: %v", err)
	}
	NoteSessionOnline("42")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("throttled note_online must not persist within 30s window")
	}
	// offline 立即落盘并清除记录。
	NoteSessionOffline("42")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("note_offline must persist: %v", err)
	}
	var f livenessFile
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("offline file not valid json: %v", err)
	}
	if _, ok := f.LastSeen["42"]; ok {
		t.Fatal("note_offline must remove the record")
	}
}

// TestWxStartupReconnectConstantsAlignRust 启动重连节奏常量对齐 rust timing.rs:66-70。
func TestWxStartupReconnectConstantsAlignRust(t *testing.T) {
	if WxStartupReconnectDelay != 60*time.Second {
		t.Fatalf("startup delay=%v, want 60s (WX_STARTUP_RECONNECT_DELAY_MS)", WxStartupReconnectDelay)
	}
	if WxStartupReconnectStaggerMin != 15*time.Second || WxStartupReconnectStaggerMax != 45*time.Second {
		t.Fatalf("stagger range=%v~%v, want 15s~45s", WxStartupReconnectStaggerMin, WxStartupReconnectStaggerMax)
	}
	if SessionLivenessPersistThrottle != 30*time.Second {
		t.Fatalf("persist throttle=%v, want 30s (PERSIST_THROTTLE_MS)", SessionLivenessPersistThrottle)
	}
	// rust wx_startup_reconnect_stagger_ms：闭区间 [min, max] 均匀随机。
	for i := 0; i < 200; i++ {
		s := wxStartupReconnectStagger()
		if s < WxStartupReconnectStaggerMin || s > WxStartupReconnectStaggerMax {
			t.Fatalf("stagger %v out of [%v, %v]", s, WxStartupReconnectStaggerMin, WxStartupReconnectStaggerMax)
		}
	}
}
