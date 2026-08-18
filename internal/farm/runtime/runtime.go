package runtime

import (
	"context"
	"errors"
	"fmt"
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
		minted, buf, err := wxlogin.NewWxLoginService().MintGatewayCode(
			context.Background(),
			acc.WxLoginBuffer,
			acc.WxOpenID,
			acc.WxAccessToken,
			wxlogin.TargetMiniProgramID,
		)
		if err != nil {
			msg := fmt.Sprintf("应用宝换码失败，请重新扫码: %v", err)
			appendRuntimeLog(accountID, logEventLogin, msg, true)
			persistRunStatus(accountID, RunError, false)
			m.scheduleWxReconnect(strconv.FormatUint(accountID, 10), false)
			return errors.New(msg)
		}
		if strings.TrimSpace(minted) == "" {
			msg := "应用宝换码失败，请重新扫码: empty code"
			appendRuntimeLog(accountID, logEventLogin, msg, true)
			persistRunStatus(accountID, RunError, false)
			m.scheduleWxReconnect(strconv.FormatUint(accountID, 10), false)
			return errors.New(msg)
		}
		persistWxGatewayCredentials(accountID, minted, buf)
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

	return m.StartAccount(context.Background(), cfg)
}

func (f *Facade) Stop(accountID uint64) error {
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

func persistWxGatewayCredentials(accountID uint64, code, loginBuffer string) {
	if accountID == 0 || strings.TrimSpace(code) == "" {
		return
	}
	updates := map[string]any{
		"code":       code,
		"updated_at": uint(time.Now().Unix()),
	}
	if strings.TrimSpace(loginBuffer) != "" {
		updates["wx_login_buffer"] = loginBuffer
	}
	_ = vars.DB.Model(&model.FarmAccount{}).Where("id = ?", accountID).Updates(updates).Error
}
