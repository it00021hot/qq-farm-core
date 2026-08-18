package runtime

import (
	"testing"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/farm/wxlogin"
)

func TestWxKeepaliveConstants(t *testing.T) {
	if wxlogin.WxKeepaliveInterval != 30*time.Minute {
		t.Fatalf("interval=%v", wxlogin.WxKeepaliveInterval)
	}
	if wxlogin.WxKeepaliveAheadSecs != 45*60 {
		t.Fatalf("ahead=%d", wxlogin.WxKeepaliveAheadSecs)
	}
	if wxlogin.WxRefreshTokenRescanSecs != 25*24*60*60 {
		t.Fatalf("rescan=%d", wxlogin.WxRefreshTokenRescanSecs)
	}
}

func TestClearWxAuthUpdatesShape(t *testing.T) {
	updates := wxlogin.ClearWxAuthUpdates(1)
	if updates["wx_login_buffer"] != "" || updates["wx_refresh_token"] != "" || updates["code"] != "" {
		t.Fatalf("updates=%v", updates)
	}
	if updates["wx_token_expires_at"] != int64(0) || updates["wx_refresh_token_observed_at"] != int64(0) {
		t.Fatalf("expires=%v observed=%v", updates["wx_token_expires_at"], updates["wx_refresh_token_observed_at"])
	}
}
