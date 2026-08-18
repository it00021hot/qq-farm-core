package wxlogin

import (
	"testing"
	"time"
)

func TestRefreshTokenPayloadAndParse(t *testing.T) {
	base := YybCredentials{
		OpenID:       "oid",
		AccessToken:  "old",
		RefreshToken: "rt",
		LoginBuffer:  "buf",
		ExpiresIn:    7200,
	}
	payload, err := refreshTokenPayload(base)
	if err != nil {
		t.Fatal(err)
	}
	if payload == "" {
		t.Fatal("empty payload")
	}
	body := []byte(`{"code":0,"user_info":{"access_token":"new","refresh_token":"rt2","expires_in":3600}}`)
	updated, err := parseRefreshTokenJSON(body, base)
	if err != nil {
		t.Fatal(err)
	}
	if updated.AccessToken != "new" || updated.RefreshToken != "rt2" || updated.ExpiresIn != 3600 {
		t.Fatalf("updated=%+v", updated)
	}
	if updated.RefreshTokenObservedAt < time.Now().Unix()-2 {
		t.Fatalf("rotated observed_at=%d", updated.RefreshTokenObservedAt)
	}
	if updated.ExpiresAt <= time.Now().Unix() {
		t.Fatal("expires_at should be in the future")
	}
}

func TestParseRefreshTokenJSONKeepsObservedAt(t *testing.T) {
	base := YybCredentials{OpenID: "oid", RefreshToken: "rt", ExpiresIn: 7200, RefreshTokenObservedAt: 111}
	body := []byte(`{"code":0,"user_info":{"access_token":"new","refresh_token":"rt","expires_in":1800}}`)
	updated, err := parseRefreshTokenJSON(body, base)
	if err != nil {
		t.Fatal(err)
	}
	if updated.RefreshToken != "rt" || updated.RefreshTokenObservedAt != 111 {
		t.Fatalf("updated=%+v", updated)
	}
}

func TestParseRefreshTokenJSONMissingRefreshKeepsOld(t *testing.T) {
	base := YybCredentials{RefreshToken: "rt", RefreshTokenObservedAt: 222, ExpiresIn: 7200}
	updated, err := parseRefreshTokenJSON([]byte(`{"code":0,"user_info":{"access_token":"new"}}`), base)
	if err != nil {
		t.Fatal(err)
	}
	if updated.RefreshToken != "rt" || updated.RefreshTokenObservedAt != 222 {
		t.Fatalf("updated=%+v", updated)
	}
}

func TestParseRefreshTokenJSONCamelCase(t *testing.T) {
	base := YybCredentials{OpenID: "oid", RefreshToken: "rt", ExpiresIn: 7200}
	body := []byte(`{"code":0,"userInfo":{"accessToken":"newtok","refreshToken":"rt","expiresIn":1800}}`)
	updated, err := parseRefreshTokenJSON(body, base)
	if err != nil || updated.AccessToken != "newtok" || updated.ExpiresIn != 1800 {
		t.Fatalf("updated=%+v err=%v", updated, err)
	}
}

func TestParseRefreshTokenJSONRejectsFailure(t *testing.T) {
	_, err := parseRefreshTokenJSON([]byte(`{"code":1,"msg":"bad"}`), YybCredentials{})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.(WxAuthError).Kind != WxAuthErrorCredentialsDead {
		t.Fatalf("kind=%v", err)
	}
}

func TestParseRefreshTokenJSONMissingAccessToken(t *testing.T) {
	_, err := parseRefreshTokenJSON([]byte(`{"code":0,"user_info":{}}`), YybCredentials{})
	if err == nil {
		t.Fatal("expected error")
	}
}
