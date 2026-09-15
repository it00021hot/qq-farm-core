package runtime

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/app/model"
	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/userpb"
	"github.com/it00021hot/qq-farm-core/internal/farm/wxlogin"
	"github.com/it00021hot/qq-farm-core/internal/vars"
)

// DB run_status
const (
	RunStopped uint8 = 0
	RunRunning uint8 = 1
	RunError   uint8 = 2
)

var (
	mgrMu sync.RWMutex
	mgr   *AccountManager
)

// Default is the HTTP-facing runtime facade (never nil).
var Default = &Facade{}

// SetManager injects AccountManager from bootstrap (optional).
func SetManager(m *AccountManager) {
	mgrMu.Lock()
	mgr = m
	mgrMu.Unlock()
}

func getManager() *AccountManager {
	mgrMu.RLock()
	defer mgrMu.RUnlock()
	return mgr
}

// Facade bridges farm CRUD services and AccountManager.
type Facade struct{}

func (f *Facade) Start(accountID uint64) error {
	return f.startAccount(accountID, true)
}

// startUnderLifecycleLock 在调用方已持有该账号生命周期锁时启动账号
// （定时重连 / 启动重连路径：hasWorker 检查与启动在同一把锁内原子完成，
// 对齐 rust start_wx_authorized_account 的持锁调用）。
func (f *Facade) startUnderLifecycleLock(accountID uint64) error {
	return f.startAccount(accountID, false)
}

// startAccount 换码 + 组装配置后启动账号；lockLifecycle=true 时由
// StartAccount 获取账号生命周期锁，false 表示调用方已持锁。
func (f *Facade) startAccount(accountID uint64, lockLifecycle bool) error {
	m := getManager()
	if m == nil {
		// 阶段2：无引擎时仅更新库表状态由 service 处理；此处 no-op 成功
		return nil
	}

	db := vars.DB
	var acc model.FarmAccount
	if err := db.Where("id = ?", accountID).First(&acc).Error; err != nil {
		return errors.New("账号不存在")
	}
	code := strings.TrimSpace(acc.Code)
	platform := acc.Platform
	if platform == "" {
		platform = "qq"
	}
	hasWx := acc.CanWxReconnect()
	if hasWx {
		appendRuntimeLog(accountID, logEventLogin, "正在用应用宝授权换取新的登录码", false)
		minted, creds, err := wxlogin.NewWxLoginService().MintGatewayCode(
			context.Background(),
			wxlogin.CredentialsFromAccount(&acc),
			wxlogin.TargetMiniProgramID,
		)
		if err != nil {
			msg := userFacingWxAuthError(err)
			appendRuntimeLog(accountID, logEventLogin, msg, true)
			persistRunStatus(accountID, RunError, false)
			if authErr, ok := err.(wxlogin.WxAuthError); ok && authErr.Kind == wxlogin.WxAuthErrorCredentialsDead {
				handleWxAuthDead(accountID, acc.Name, msg, m)
				return errors.New(msg)
			}
			// 换码失败（rust wx_mint_failed）：不再自动重连，等手动重扫/重试。
			// 外部推送 YybQr 通知（对齐 rust engine.rs wx_mint_failed 分支的
			// spawn_account_notice(AccountNoticeKind::YybQr)）。注意与授权失效路径
			// notifyWxAuthCleared 区分：此处授权仍存活，仅换码暂时失败。
			go SendAccountNotice(NoticeYybQr, accountID, acc.Name)
			m.scheduleWxReconnect(strconv.FormatUint(accountID, 10), false, "wx_mint_failed")
			return errors.New(msg)
		}
		if strings.TrimSpace(minted) == "" {
			msg := "应用宝换码失败，请重新扫码: empty code"
			appendRuntimeLog(accountID, logEventLogin, msg, true)
			persistRunStatus(accountID, RunError, false)
			// 空码同属「授权存活但换码失败」（rust wx_mint_failed 分支），同样外部推送。
			go SendAccountNotice(NoticeYybQr, accountID, acc.Name)
			m.scheduleWxReconnect(strconv.FormatUint(accountID, 10), false, "wx_mint_failed")
			return errors.New(msg)
		}
		persistWxGatewayCredentials(accountID, minted, creds)
		code = minted
		appendRuntimeLog(accountID, logEventLogin, "换码成功，正在连接网关", false)
	} else {
		if code == "" {
			return errors.New("连接缺少一次性 Code")
		}
		appendRuntimeLog(accountID, logEventLogin, "正在用已保存的登录码连接网关", false)
	}

	cfg := SessionConfig{
		AccountID:     strconv.FormatUint(accountID, 10),
		Code:          code,
		Platform:      platform,
		OSName:        strings.TrimSpace(acc.LoginOS),
		ClientVersion: strings.TrimSpace(acc.ClientVer),
		GatewayURL:    vars.Config.GetString("farm.gatewayUrl"),
		WASMPath:      vars.Config.GetString("farm.wasmPath"),
		DataRoot:      vars.Config.GetString("farm.tsdkDataDir"),
		ShareFile:     vars.Config.GetString("farm.shareFile"),
		PushWebhook:   vars.Config.GetString("farm.pushWebhook"),
		HasWxAuth:     hasWx,
	}
	if cfg.ClientVersion == "" {
		cfg.ClientVersion = vars.Config.GetString("farm.clientVersion")
	}
	if cfg.OSName == "" {
		cfg.OSName = vars.Config.GetString("farm.os")
	}

	var cfgRow model.FarmAccountConfig
	if err := db.Where("account_id = ?", accountID).First(&cfgRow).Error; err == nil {
		cfg.AccountConfig = logic.ParseAccountConfigJSON(cfgRow.ConfigJSON)
	} else {
		cfg.AccountConfig = logic.DefaultAccountConfig()
	}

	if lockLifecycle {
		return m.StartAccount(context.Background(), cfg)
	}
	return m.startAccountLocked(context.Background(), cfg)
}

func (f *Facade) Stop(accountID uint64) error {
	// 手动停止 / 删除账号时中止在途的 YYB 扫码重登录 watcher（go 侧取消语义，
	// rust 原实现轮询至超时自然结束；见 relogin_watcher.go 文件头）。
	CancelReloginWatchersFor(accountID)
	m := getManager()
	if m == nil {
		return nil
	}
	id := strconv.FormatUint(accountID, 10)
	m.clearWxReconnect(id)
	if m.Status(id) == StatusStopped {
		return nil
	}
	return m.StopAccount(id)
}

// StopAll stops every in-process farm session (desktop / process shutdown).
func (f *Facade) StopAll() {
	// 进程级停止同时中止全部在途重登录 watcher（relogin_watcher.go）。
	CancelAllReloginWatchers()
	m := getManager()
	if m == nil {
		return
	}
	m.StopAll()
}

func (f *Facade) GetStatus(accountID uint64) (uint8, string) {
	m := getManager()
	if m == nil {
		return RunStopped, ""
	}
	switch m.Status(strconv.FormatUint(accountID, 10)) {
	case StatusRunning, StatusStarting:
		return RunRunning, ""
	case StatusError:
		return RunError, "runtime error"
	case StatusStopping:
		return RunRunning, "stopping"
	default:
		return RunStopped, ""
	}
}

func (f *Facade) ApplyConfig(accountID uint64, cfg logic.AccountConfig) {
	m := getManager()
	if m == nil {
		return
	}
	_ = m.ApplyConfig(strconv.FormatUint(accountID, 10), cfg)
}

func (f *Facade) IsRunning(accountID uint64) bool {
	st, _ := f.GetStatus(accountID)
	return st == RunRunning
}

// Session returns the live session for accountID, if connected.
func (f *Facade) Session(accountID uint64) (*Session, bool) {
	m := getManager()
	if m == nil {
		return nil, false
	}
	return m.Session(strconv.FormatUint(accountID, 10))
}

// ResetPersistedRunStatus clears stale DB run_status after process boot
// (no in-memory sessions exist yet).
func ResetPersistedRunStatus() {
	db := vars.DB
	now := uint(time.Now().Unix())
	res := db.Model(&model.FarmAccount{}).
		Where("run_status <> ?", RunStopped).
		Updates(map[string]any{
			"run_status": RunStopped,
			"updated_at": now,
		})
	if res.Error != nil {
		return
	}
	if res.RowsAffected > 0 {
		slog.Info("reset stale farm account run_status", "rows", res.RowsAffected)
	}
}

func persistRunStatus(accountID uint64, status uint8, online bool) {
	if accountID == 0 {
		return
	}
	db := vars.DB
	now := uint(time.Now().Unix())
	updates := map[string]any{
		"run_status": status,
		"updated_at": now,
	}
	if online {
		updates["last_online_at"] = now
	}
	_ = db.Model(&model.FarmAccount{}).Where("id = ?", accountID).Updates(updates).Error
}

func persistAccountProfile(accountID uint64, basic *userpb.BasicInfo) {
	if accountID == 0 || basic == nil {
		return
	}
	db := vars.DB
	var acc model.FarmAccount
	if err := db.Where("id = ?", accountID).First(&acc).Error; err != nil {
		return
	}
	updates := map[string]any{
		"updated_at": uint(time.Now().Unix()),
	}
	if basic.OpenId != "" {
		updates["uin"] = basic.OpenId
	}
	// Only auto-fill display name when still the placeholder 账号{id}
	placeholder := "账号" + strconv.FormatUint(accountID, 10)
	if basic.Name != "" && (acc.Name == "" || acc.Name == placeholder) {
		updates["name"] = basic.Name
	}
	_ = db.Model(&acc).Updates(updates).Error
}
