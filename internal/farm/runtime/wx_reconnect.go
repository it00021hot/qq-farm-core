package runtime

import (
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/app/model"
	"github.com/it00021hot/qq-farm-core/internal/farm/hub"
	"github.com/it00021hot/qq-farm-core/internal/farm/push"
	"github.com/it00021hot/qq-farm-core/internal/vars"
)

const (
	WxReconnectMaxAttempts uint32 = 3
	WxReconnectDelay              = 5 * time.Minute

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

func wxReconnectDelayZh() string {
	secs := int(WxReconnectDelay / time.Second)
	if secs >= 60 && secs%60 == 0 {
		return strconv.Itoa(secs/60) + " 分钟"
	}
	return strconv.Itoa(secs) + " 秒"
}

func shouldAttemptWxReconnect(userStop, hasWxAuth bool) bool {
	return !userStop && hasWxAuth
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

func (m *AccountManager) scheduleWxReconnect(accountID string, kicked bool) {
	if m == nil || accountID == "" {
		return
	}
	acc, ok := loadFarmAccount(parseAccountID(accountID))
	hasWx := ok && acc.CanWxReconnect()
	if !shouldAttemptWxReconnect(false, hasWx) {
		return
	}
	display := acc.Name
	if strings.TrimSpace(display) == "" {
		display = accountID
	}
	decision := m.planWxReconnect(accountID)
	wait := wxReconnectDelayZh()
	switch decision.plan {
	case wxReconnectSkip:
		return
	case wxReconnectGiveUp:
		msg := "账号 " + display + " 应用宝授权失效，请重新扫码"
		appendRuntimeLog(acc.ID, logEventReconnect, msg, true)
		notifyWxOffline(acc, msg)
		return
	case wxReconnectSpawn:
		msg := "账号 " + display + " 连接已断开，将在 " + wait + "后用应用宝授权重连（第 " + strconv.Itoa(int(decision.attempt)) + "/" + strconv.Itoa(int(WxReconnectMaxAttempts)) + " 次）"
		if kicked {
			msg = "账号 " + display + " 被踢下线，将在 " + wait + "后用应用宝授权重连（第 " + strconv.Itoa(int(decision.attempt)) + "/" + strconv.Itoa(int(WxReconnectMaxAttempts)) + " 次）"
		}
		appendRuntimeLog(acc.ID, logEventReconnect, msg, false)
		gen := decision.gen
		attempt := decision.attempt
		go func() {
			time.Sleep(WxReconnectDelay)
			if m.wxReconnectGen(accountID) != gen {
				return
			}
			m.dropWxReconnectInflight(accountID)
			m.startWxAuthorizedAccount(accountID, attempt)
		}()
	}
}

func (m *AccountManager) startWxAuthorizedAccount(accountID string, attempt uint32) {
	acc, ok := loadFarmAccount(parseAccountID(accountID))
	if !ok || !acc.CanWxReconnect() {
		return
	}
	if acc.Status != vars.StatusNormal {
		return
	}
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
	if err := Default.Start(acc.ID); err != nil {
		appendRuntimeLog(acc.ID, logEventReconnect, "账号 "+display+" 重连启动失败: "+err.Error(), true)
	}
}

// ScheduleWxAuthorizedStart reconnects WeChat accounts that have Yingyongbao tickets after a delay.
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
	wait := wxReconnectDelayZh()
	slog.Info("wx-authorized accounts will reconnect after delay", "count", len(accounts), "wait", wait)
	for i := range accounts {
		acc := accounts[i]
		appendRuntimeLog(acc.ID, logEventReconnect, "发现已授权微信账号，将在 "+wait+"后自动重连", false)
	}
	go func() {
		time.Sleep(WxReconnectDelay)
		for i := range accounts {
			acc := accounts[i]
			id := strconv.FormatUint(acc.ID, 10)
			m.startWxAuthorizedAccount(id, 0)
		}
	}()
}

func notifyWxOffline(acc model.FarmAccount, msg string) {
	webhook := vars.Config.GetString("farm.pushWebhook")
	if webhook == "" {
		return
	}
	title := "农场账号 " + acc.Name + " 离线"
	if strings.TrimSpace(acc.Name) == "" {
		title = "农场账号离线"
	}
	go func() { _ = push.Notify(webhook, title, msg) }()
}
