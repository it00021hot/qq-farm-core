package runtime

import (
	"strconv"
	"strings"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/app/model"
	"github.com/it00021hot/qq-farm-core/internal/farm/hub"
	"github.com/it00021hot/qq-farm-core/internal/farm/wxlogin"
	"github.com/it00021hot/qq-farm-core/internal/vars"
)

func handleWxAuthDead(accountID uint64, accountName, msg string, m *AccountManager) {
	clearWxAuth(accountID)
	id := strconv.FormatUint(accountID, 10)
	if m != nil {
		m.clearWxReconnect(id)
		if m.hasWorker(id) {
			_ = m.StopAccount(id)
		}
	}
	notifyWxAuthCleared(accountID, accountName, msg)
}

func clearWxAuth(accountID uint64) {
	if accountID == 0 || vars.DB == nil {
		return
	}
	now := uint(time.Now().Unix())
	_ = vars.DB.Model(&model.FarmAccount{}).Where("id = ?", accountID).Updates(wxlogin.ClearWxAuthUpdates(now)).Error
}

func userFacingWxAuthError(err error) string {
	if err == nil {
		return "应用宝授权暂时不可用"
	}
	authErr, ok := err.(wxlogin.WxAuthError)
	if !ok {
		return "应用宝授权暂时不可用"
	}
	if authErr.Kind == wxlogin.WxAuthErrorCredentialsDead {
		return "应用宝授权已失效，请重新扫码"
	}
	return "应用宝换码失败，已保留授权，稍后自动重试"
}

func notifyWxAuthCleared(accountID uint64, accountName, msg string) {
	display := strings.TrimSpace(accountName)
	if display == "" {
		display = strconv.FormatUint(accountID, 10)
	}
	hub.Default.PublishJSON("account_status", accountID, map[string]any{
		"status": string(StatusError),
		"detail": "应用宝授权已失效，请重新扫码",
	})
	notifyWxOffline(model.FarmAccount{ID: accountID, Name: display}, msg)
}

func persistWxGatewayCredentials(accountID uint64, code string, creds wxlogin.YybCredentials) {
	if accountID == 0 || strings.TrimSpace(code) == "" {
		return
	}
	now := uint(time.Now().Unix())
	updates := wxlogin.CredentialPersistUpdates(creds, now)
	updates["code"] = code
	_ = vars.DB.Model(&model.FarmAccount{}).Where("id = ?", accountID).Updates(updates).Error
}
