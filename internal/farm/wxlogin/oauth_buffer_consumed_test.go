package wxlogin

import (
	"context"
	"errors"
	"testing"
)

// stubMintCode 替换原生换码入口（生产为 getNativeWxLoginCode，会走真实网络），
// 返回还原函数。
func stubMintCode(t *testing.T, fn func(ctx context.Context, loginBuffer, appID string) (string, error)) func() {
	t.Helper()
	prev := mintCodeFn
	mintCodeFn = fn
	return func() { mintCodeFn = prev }
}

// TestMintGatewayCodeConsumesFreshBufferFirst：未消费的 buffer 直接换码，
// 不触发刷新（对齐 rust service.rs：未消费过的 buffer 直接换）。
func TestMintGatewayCodeConsumesFreshBufferFirst(t *testing.T) {
	calls := 0
	defer stubMintCode(t, func(ctx context.Context, loginBuffer, appID string) (string, error) {
		calls++
		if loginBuffer != "fresh-buffer" {
			t.Fatalf("expected fresh buffer to be used, got %q", loginBuffer)
		}
		return "code-1", nil
	})()

	creds := YybCredentials{
		OpenID: "oid", AccessToken: "tok", RefreshToken: "rt",
		LoginBuffer: "fresh-buffer", BufferConsumed: false,
	}
	code, updated, err := NewWxLoginService().MintGatewayCode(context.Background(), creds, TargetMiniProgramID)
	if err != nil {
		t.Fatal(err)
	}
	if code != "code-1" || calls != 1 {
		t.Fatalf("code=%q calls=%d", code, calls)
	}
	// 成功换码后消费标记置位（rust service.rs:428 buffer_consumed = true）。
	if !updated.BufferConsumed {
		t.Fatal("BufferConsumed should be set after a successful mint")
	}
}

// TestMintGatewayCodeSkipsConsumedBuffer：已消费的 buffer 不得先撞原生换码；
// 无任何可刷新凭据时应直接报错，而不是复用旧 buffer（对齐 rust service.rs:408-410
// 先 refresh_credentials_and_buffer；这里用无凭据场景断言「必然失败的请求」被省掉）。
func TestMintGatewayCodeSkipsConsumedBuffer(t *testing.T) {
	calls := 0
	defer stubMintCode(t, func(ctx context.Context, loginBuffer, appID string) (string, error) {
		calls++
		return "", errors.New("ManualAuth rejected")
	})()

	creds := YybCredentials{LoginBuffer: "stale-buffer", BufferConsumed: true}
	_, _, err := NewWxLoginService().MintGatewayCode(context.Background(), creds, TargetMiniProgramID)
	if err == nil {
		t.Fatal("expected error when consumed buffer cannot be re-issued")
	}
	if calls != 0 {
		t.Fatalf("consumed buffer must not hit native mint directly, got %d calls", calls)
	}
}

// TestRefreshCredentialsAndBufferResetsConsumedFlag：无法直连网络，改为校验
// 重签出口的复位逻辑语义——由 MintGatewayCode 的已消费短路 + CredentialPersist
// 落盘组合覆盖；此处校验 parseRefreshTokenJSON 透传消费标记（对齐 rust 透传）。
func TestParseRefreshTokenJSONCarriesConsumedFlag(t *testing.T) {
	base := YybCredentials{OpenID: "oid", RefreshToken: "rt", LoginBuffer: "buf", BufferConsumed: true}
	updated, err := parseRefreshTokenJSON([]byte(`{"code":0,"user_info":{"access_token":"new","expires_in":600}}`), base)
	if err != nil {
		t.Fatal(err)
	}
	if !updated.BufferConsumed {
		t.Fatal("consumed flag should carry through refresh-token parse")
	}
}

// TestCredentialPersistUpdatesBufferConsumed：消费标记随凭据落盘 / 清授权复位。
func TestCredentialPersistUpdatesBufferConsumed(t *testing.T) {
	updates := CredentialPersistUpdates(YybCredentials{LoginBuffer: "b", BufferConsumed: true}, 100)
	if v, ok := updates["wx_buffer_consumed"].(bool); !ok || !v {
		t.Fatalf("wx_buffer_consumed should persist as true, got %v", updates["wx_buffer_consumed"])
	}
	updates = CredentialPersistUpdates(YybCredentials{LoginBuffer: "b"}, 100)
	if v, ok := updates["wx_buffer_consumed"].(bool); !ok || v {
		t.Fatalf("wx_buffer_consumed should persist as false, got %v", updates["wx_buffer_consumed"])
	}

	clears := ClearWxAuthUpdates(100)
	if v, ok := clears["wx_buffer_consumed"].(bool); !ok || v {
		t.Fatalf("clear should reset wx_buffer_consumed, got %v", clears["wx_buffer_consumed"])
	}
}
