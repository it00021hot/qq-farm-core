package runtime

import (
	"context"
	"log/slog"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/app/model"
	"github.com/it00021hot/qq-farm-core/internal/farm/wxlogin"
	"github.com/it00021hot/qq-farm-core/internal/vars"
)

// StartWxKeepalive periodically renews Yingyongbao tokens and login_buffer.
func StartWxKeepalive() {
	go func() {
		ticker := time.NewTicker(wxlogin.WxKeepaliveInterval)
		defer ticker.Stop()
		runWxKeepaliveTick()
		for range ticker.C {
			runWxKeepaliveTick()
		}
	}()
}

func runWxKeepaliveTick() {
	if vars.DB == nil {
		return
	}
	var accounts []model.FarmAccount
	if err := vars.DB.Where("wx_login_buffer <> '' AND wx_refresh_token <> '' AND status = ?", vars.StatusNormal).Find(&accounts).Error; err != nil {
		slog.Warn("wx keepalive list accounts failed", "err", err)
		return
	}
	svc := wxlogin.NewWxLoginService()
	ahead := wxlogin.WxKeepaliveAheadSecs
	for i := range accounts {
		acc := accounts[i]
		if !acc.CanRefreshWxToken() {
			continue
		}
		creds := wxlogin.CredentialsFromAccount(&acc)
		if !creds.TokenDueForRefresh(ahead) {
			continue
		}
		updated, err := svc.RefreshCredentialsAndBuffer(context.Background(), creds)
		if err != nil {
			if authErr, ok := err.(wxlogin.WxAuthError); ok && authErr.Kind == wxlogin.WxAuthErrorCredentialsDead {
				slog.Warn("wx keepalive credentials dead", "accountId", acc.ID, "err", err)
				msg := userFacingWxAuthError(err)
				appendRuntimeLog(acc.ID, logEventLogin, msg, true)
				handleWxAuthDead(acc.ID, acc.Name, msg, getManager())
				continue
			}
			slog.Warn("wx keepalive temporary failure", "accountId", acc.ID, "err", err)
			continue
		}
		updates := wxlogin.CredentialPersistUpdates(updated, uint(time.Now().Unix()))
		if err := vars.DB.Model(&model.FarmAccount{}).Where("id = ?", acc.ID).Updates(updates).Error; err != nil {
			slog.Warn("wx keepalive persist failed", "accountId", acc.ID, "err", err)
			continue
		}
		slog.Info("wx keepalive renewed credentials", "accountId", acc.ID, "expiresAt", updated.ExpiresAt)
	}
}
