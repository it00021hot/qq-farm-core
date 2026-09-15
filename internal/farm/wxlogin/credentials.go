package wxlogin

import "github.com/it00021hot/qq-farm-core/internal/app/model"

// CredentialsFromAccount builds Yingyongbao credentials from a farm account row.
func CredentialsFromAccount(acc *model.FarmAccount) YybCredentials {
	if acc == nil {
		return YybCredentials{}
	}
	return YybCredentials{
		OpenID:                 acc.WxOpenID,
		AccessToken:            acc.WxAccessToken,
		RefreshToken:           acc.WxRefreshToken,
		LoginBuffer:            acc.WxLoginBuffer,
		ExpiresAt:              acc.WxTokenExpiresAt,
		RefreshTokenObservedAt: acc.WxRefreshTokenObservedAt,
		BufferConsumed:         acc.WxBufferConsumed,
	}
}

// CredentialPersistUpdates is the DB patch for a successful token/buffer refresh.
func CredentialPersistUpdates(creds YybCredentials, now uint) map[string]any {
	creds = creds.EnsureObservedAt(int64(now))
	return map[string]any{
		"wx_login_buffer":              creds.LoginBuffer,
		"wx_access_token":              creds.AccessToken,
		"wx_refresh_token":             creds.RefreshToken,
		"wx_token_expires_at":          creds.ExpiresAt,
		"wx_refresh_token_observed_at": creds.RefreshTokenObservedAt,
		// login_buffer 一次性消费标记随凭据一起落盘（rust worker.rs
		// persist_wx_gateway_credentials 的 wx_buffer_consumed）。
		"wx_buffer_consumed": creds.BufferConsumed,
		"updated_at":         now,
	}
}

// ClearWxAuthUpdates returns DB updates that clear reusable Yingyongbao tickets but keep openid.
func ClearWxAuthUpdates(now uint) map[string]any {
	return map[string]any{
		"code":                         "",
		"wx_login_buffer":              "",
		"wx_access_token":              "",
		"wx_refresh_token":             "",
		"wx_token_expires_at":          int64(0),
		"wx_refresh_token_observed_at": int64(0),
		// 清授权同时复位消费标记（对齐 rust accounts.rs clear_wx_auth）。
		"wx_buffer_consumed": false,
		"updated_at":         now,
	}
}
