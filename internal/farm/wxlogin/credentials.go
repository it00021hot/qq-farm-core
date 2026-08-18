package wxlogin

import "github.com/it00021hot/qq-farm-core/internal/app/model"

// CredentialsFromAccount builds Yingyongbao credentials from a farm account row.
func CredentialsFromAccount(acc *model.FarmAccount) YybCredentials {
	if acc == nil {
		return YybCredentials{}
	}
	return YybCredentials{
		OpenID:       acc.WxOpenID,
		AccessToken:  acc.WxAccessToken,
		RefreshToken: acc.WxRefreshToken,
		LoginBuffer:  acc.WxLoginBuffer,
		ExpiresAt:    acc.WxTokenExpiresAt,
	}
}

// ClearWxAuthUpdates returns DB updates that clear reusable Yingyongbao tickets but keep openid.
func ClearWxAuthUpdates(now uint) map[string]any {
	return map[string]any{
		"code":                 "",
		"wx_login_buffer":      "",
		"wx_access_token":      "",
		"wx_refresh_token":     "",
		"wx_token_expires_at":  int64(0),
		"updated_at":           now,
	}
}
