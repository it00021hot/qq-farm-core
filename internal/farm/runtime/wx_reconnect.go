package runtime

import (
	"fmt"
	"log/slog"
	"math/rand/v2"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/app/model"
	"github.com/it00021hot/qq-farm-core/internal/farm/hub"
	"github.com/it00021hot/qq-farm-core/internal/vars"
)

// 重连节奏对齐 rust constants/timing.rs：
//   - 第 1 次掉线重连等 15 分钟（WX_RECONNECT_FIRST_DELAY_MS）：心跳超时类“半死”
//     会话服务端释放很慢，高频重登本身是风控信号；
//   - 第 2～3 次等 10 分钟（WX_RECONNECT_RETRY_DELAY_MS）；
//   - 被踢（“已在其他终端登录”）固定等 3 分钟（WX_KICKOUT_RECONNECT_DELAY_MS）：
//     服务端旧 session 释放需要时间，重登过快会连环被踢；
//   - 主动重启 worker 时旧连接关闭到新登录之间留 3s 宽限（WX_RESTART_GRACE_MS）。
const (
	WxReconnectMaxAttempts  uint32 = 3
	WxReconnectFirstDelay          = 15 * time.Minute
	WxReconnectRetryDelay          = 10 * time.Minute
	WxKickoutReconnectDelay        = 3 * time.Minute
	WxRestartGrace                 = 3 * time.Second
	// wxStopWaitCap 主动重启时等待旧会话退出的上限（rust WORKER_STOP_WAIT_MS）。
	wxStopWaitCap = 10 * time.Second

	// 进程启动后已授权微信账号首次自动重连的等待时间（rust WX_STARTUP_RECONNECT_DELAY_MS）。
	WxStartupReconnectDelay = 60 * time.Second
	// 启动重连时相邻账号之间的随机间隔范围：所有账号同一时刻集中登录本身是风控信号
	// （rust WX_STARTUP_RECONNECT_STAGGER_MIN_MS / WX_STARTUP_RECONNECT_STAGGER_MAX_MS）。
	WxStartupReconnectStaggerMin = 15 * time.Second
	WxStartupReconnectStaggerMax = 45 * time.Second
	// 会话存活打点落盘节流：期间只更新内存（rust session_liveness.rs PERSIST_THROTTLE_MS）。
	SessionLivenessPersistThrottle = 30 * time.Second

	logEventLogin     = "登录"
	logEventReconnect = "重连"
)

type wxReconnectPlan int

const (
	wxReconnectSkip wxReconnectPlan = iota
	wxReconnectSpawn
	wxReconnectGiveUp
)

type wxReconnectDecision struct {
	plan    wxReconnectPlan
	attempt uint32
	gen     uint64
}

type wxReconnectState struct {
	mu       sync.Mutex
	attempts map[string]uint32
	inflight map[string]struct{}
	gen      map[string]uint64
}

// wxReconnectDelay mirrors rust wx_reconnect_delay_ms: 第 1 次 15 分钟，之后 10 分钟。
func wxReconnectDelay(attempt uint32) time.Duration {
	if attempt <= 1 {
		return WxReconnectFirstDelay
	}
	return WxReconnectRetryDelay
}

// wxKickoutReconnectDelay 固定 3 分钟，不随次数缩短。
func wxKickoutReconnectDelay() time.Duration { return WxKickoutReconnectDelay }

// wxStartupReconnectStagger 启动重连相邻账号之间的随机间隔（rust wx_startup_reconnect_stagger_ms：
// 闭区间 [15s, 45s] 内均匀取值，首个账号不等待）。
func wxStartupReconnectStagger() time.Duration {
	span := int64(WxStartupReconnectStaggerMax - WxStartupReconnectStaggerMin)
	return WxStartupReconnectStaggerMin + time.Duration(rand.Int64N(span+1))
}

// durationZh mirrors rust duration_ms_zh（“3 分钟” / “45 秒”）。
func durationZh(d time.Duration) string {
	secs := int(d / time.Second)
	if secs >= 60 && secs%60 == 0 {
		return strconv.Itoa(secs/60) + " 分钟"
	}
	return strconv.Itoa(secs) + " 秒"
}

// shouldAttemptWxReconnect mirrors rust engine.rs should_attempt_wx_reconnect:
// 主动停止、无应用宝授权、授权失效 / 换码失败均不重连。
func shouldAttemptWxReconnect(userStop, hasWxAuth bool, reason string) bool {
	return !userStop && hasWxAuth &&
		!strings.Contains(reason, "wx_auth_failed") &&
		!strings.Contains(reason, "wx_mint_failed")
}

func (m *AccountManager) clearWxReconnect(accountID string) {
	if m == nil {
		return
	}
	g := &m.wxReconnect
	g.mu.Lock()
	defer g.mu.Unlock()
	g.attempts[accountID] = 0
	delete(g.attempts, accountID)
	delete(g.inflight, accountID)
	g.gen[accountID] = g.gen[accountID] + 1
}

func (m *AccountManager) clearWxReconnectAttempts(accountID string) {
	if m == nil {
		return
	}
	g := &m.wxReconnect
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.attempts, accountID)
	delete(g.inflight, accountID)
}

func (m *AccountManager) planWxReconnect(accountID string) wxReconnectDecision {
	g := &m.wxReconnect
	g.mu.Lock()
	defer g.mu.Unlock()
	if _, ok := g.inflight[accountID]; ok {
		return wxReconnectDecision{plan: wxReconnectSkip}
	}
	n := g.attempts[accountID] + 1
	g.attempts[accountID] = n
	if n > WxReconnectMaxAttempts {
		delete(g.inflight, accountID)
		return wxReconnectDecision{plan: wxReconnectGiveUp, attempt: n}
	}
	g.inflight[accountID] = struct{}{}
	return wxReconnectDecision{plan: wxReconnectSpawn, attempt: n, gen: g.gen[accountID]}
}

func (m *AccountManager) hasWorker(accountID string) bool {
	switch m.Status(accountID) {
	case StatusStarting, StatusRunning, StatusStopping:
		return true
	default:
		return false
	}
}

func (m *AccountManager) dropWxReconnectInflight(accountID string) {
	g := &m.wxReconnect
	g.mu.Lock()
	delete(g.inflight, accountID)
	g.mu.Unlock()
}

func (m *AccountManager) wxReconnectGen(accountID string) uint64 {
	g := &m.wxReconnect
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.gen[accountID]
}

func accountCanWxReconnect(accountID uint64) bool {
	if accountID == 0 || vars.DB == nil {
		return false
	}
	var acc model.FarmAccount
	if err := vars.DB.Where("id = ?", accountID).First(&acc).Error; err != nil {
		return false
	}
	return acc.CanWxReconnect()
}

func loadFarmAccount(accountID uint64) (model.FarmAccount, bool) {
	var acc model.FarmAccount
	if vars.DB == nil || accountID == 0 {
		return acc, false
	}
	if err := vars.DB.Where("id = ?", accountID).First(&acc).Error; err != nil {
		return acc, false
	}
	return acc, true
}

func appendRuntimeLog(accountID uint64, event, msg string, isWarn bool) {
	if accountID == 0 {
		slog.Info(msg, "event", event)
		return
	}
	tag := "系统"
	if isWarn {
		tag = "错误"
	}
	hub.Default.PublishJSON("runtime_log", accountID, map[string]any{
		"tag":       tag,
		"event":     event,
		"message":   msg,
		"isWarn":    isWarn,
		"accountId": accountID,
	})
}

func (m *AccountManager) scheduleWxReconnect(accountID string, kicked bool, reason string) {
	if m == nil || accountID == "" {
		return
	}
	acc, ok := loadFarmAccount(parseAccountID(accountID))
	hasWx := ok && acc.CanWxReconnect()
	if !shouldAttemptWxReconnect(false, hasWx, reason) {
		return
	}
	display := acc.Name
	if strings.TrimSpace(display) == "" {
		display = accountID
	}
	decision := m.planWxReconnect(accountID)
	switch decision.plan {
	case wxReconnectSkip:
		return
	case wxReconnectGiveUp:
		// rust GiveUp 文案：被踢提示确认其他设备，普通掉线提示已停止。
		msg := "账号 " + display + " 自动重连已达 " + strconv.Itoa(int(WxReconnectMaxAttempts)) + " 次上限，已停止运行"
		if kicked {
			msg = "账号 " + display + " 被踢下线，自动重登已达 " + strconv.Itoa(int(WxReconnectMaxAttempts)) + " 次上限，已停止（如确认无其他设备登录，可手动重新上号）"
		}
		appendRuntimeLog(acc.ID, logEventReconnect, msg, true)
		notifyWxOffline(acc, msg)
		return
	case wxReconnectSpawn:
		// 被踢固定 3 分钟后重登：服务端旧 session 释放需要时间，
		// 过快重登会连续触发“已在其他终端登录”踢循环。
		wait := wxReconnectDelay(decision.attempt)
		if kicked {
			wait = wxKickoutReconnectDelay()
		}
		waitZh := durationZh(wait)
		msg := "账号 " + display + " 连接已断开，将在 " + waitZh + "后用应用宝授权重连（第 " + strconv.Itoa(int(decision.attempt)) + "/" + strconv.Itoa(int(WxReconnectMaxAttempts)) + " 次）"
		if kicked {
			msg = "账号 " + display + " 被踢下线，将在 " + waitZh + "后用应用宝授权重连（第 " + strconv.Itoa(int(decision.attempt)) + "/" + strconv.Itoa(int(WxReconnectMaxAttempts)) + " 次）"
		}
		appendRuntimeLog(acc.ID, logEventReconnect, msg, false)
		gen := decision.gen
		attempt := decision.attempt
		go func() {
			time.Sleep(wait)
			if m.wxReconnectGen(accountID) != gen {
				return
			}
			m.dropWxReconnectInflight(accountID)
			m.startWxAuthorizedAccount(accountID, attempt)
		}()
	}
}

// startWxAuthorizedAccount 用应用宝授权拉起一个账号（定时重连 / 启动重连共用）。
// 对齐 rust engine.rs start_wx_authorized_account（:1266-1299）+ 触发点持生命周期锁
// （:823-826）：hasWorker 检查与后续启动在同一把账号生命周期锁内原子完成，
// 杜绝定时器触发与手动启动竞态导致同账号并发拉起两个会话互踢。
func (m *AccountManager) startWxAuthorizedAccount(accountID string, attempt uint32) {
	acc, ok := loadFarmAccount(parseAccountID(accountID))
	if !ok || !acc.CanWxReconnect() {
		return
	}
	if acc.Status != vars.StatusNormal {
		return
	}
	// 账号生命周期锁（rust engine.rs lifecycle_locks）：与手动 StartAccount 串行化。
	lock := m.lifecycleLock(accountID)
	lock.Lock()
	defer lock.Unlock()
	// 持锁后重查：锁等待期间账号可能已被手动启动。
	if m.hasWorker(accountID) {
		return
	}
	display := acc.Name
	if strings.TrimSpace(display) == "" {
		display = accountID
	}
	msg := "账号 " + display + " 开始用应用宝授权自动重连"
	if attempt > 0 {
		msg = "账号 " + display + " 开始重连（第 " + strconv.Itoa(int(attempt)) + " 次）"
	}
	appendRuntimeLog(acc.ID, logEventReconnect, msg, false)
	// 调用方已持该账号生命周期锁：走不重复加锁的启动路径。
	if err := Default.startUnderLifecycleLock(acc.ID); err != nil {
		appendRuntimeLog(acc.ID, logEventReconnect, "账号 "+display+" 重连启动失败: "+err.Error(), true)
	}
}

// ScheduleWxAuthorizedStart 进程启动后重连持有应用宝授权的微信账号。
// 对齐 rust engine.rs schedule_wx_authorized_start（:1088-1158）：
//   - 先等 WxStartupReconnectDelay（60s）再开始；
//   - 多账号之间随机错峰 15~45s（WxStartupReconnectStaggerMin/Max，首个账号不等待）；
//   - 每个账号启动前检查 session liveness（session_liveness.go）：上个会话结束不足
//     3 分钟则等过服务端旧 session 释放窗口再登；
//   - 每个账号启动时持账号生命周期锁。
func ScheduleWxAuthorizedStart() {
	m := getManager()
	if m == nil || vars.DB == nil {
		return
	}
	var accounts []model.FarmAccount
	if err := vars.DB.Where("platform = ? AND wx_login_buffer <> '' AND status = ?", "wx", vars.StatusNormal).Find(&accounts).Error; err != nil {
		slog.Warn("list wx-authorized accounts failed", "err", err)
		return
	}
	if len(accounts) == 0 {
		return
	}
	// rust：启动总览日志只发一条（不带账号 ID），多账号时附加错峰说明。
	n := len(accounts)
	suffix := ""
	if n > 1 {
		suffix = "（各账号间随机错峰启动）"
	}
	appendRuntimeLog(0, logEventReconnect,
		fmt.Sprintf("发现 %d 个已授权微信账号，将在 %s后自动重连%s", n, durationZh(WxStartupReconnectDelay), suffix), false)
	go func() {
		time.Sleep(WxStartupReconnectDelay)
		first := true
		for i := range accounts {
			acc := accounts[i]
			id := strconv.FormatUint(acc.ID, 10)
			// rust：逐个账号重新读取最新账号数据，授权被清掉则跳过。
			latest, ok := loadFarmAccount(acc.ID)
			if !ok || !latest.CanWxReconnect() {
				continue
			}
			display := wxAccountDisplay(latest, id)
			if !first {
				stagger := wxStartupReconnectStagger()
				appendRuntimeLog(latest.ID, logEventReconnect,
					fmt.Sprintf("账号 %s 将在 %d 秒后自动重连（错峰）", display, int(stagger/time.Second)), false)
				time.Sleep(stagger)
			}
			first = false
			// 上个进程的会话若刚被杀（TCP 未优雅登出），服务端释放旧 session
			// 需要时间，立刻重登会被判“已在其他终端登录”连环踢（rust :1134-1151）。
			if extra := SessionBootDelay(id); extra > 0 {
				appendRuntimeLog(latest.ID, logEventReconnect,
					fmt.Sprintf("账号 %s 上个会话结束不足 3 分钟，服务端旧会话释放中，延迟 %d 秒后自动重连", display, int(extra/time.Second)), false)
				time.Sleep(extra)
			}
			// startWxAuthorizedAccount 内部持账号生命周期锁。
			m.startWxAuthorizedAccount(id, 0)
		}
	}()
}

// wxAccountDisplay 取账号展示名（rust：name 为空时回退账号 ID）。
func wxAccountDisplay(acc model.FarmAccount, id string) string {
	if name := strings.TrimSpace(acc.Name); name != "" {
		return name
	}
	return id
}

func notifyWxOffline(acc model.FarmAccount, msg string) {
	go SendOfflineReminder(strings.TrimSpace(acc.Name), msg)
}
