package model

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestHasWxAuth(t *testing.T) {
	var empty *FarmAccount
	if empty.HasWxAuth() || empty.CanWxReconnect() {
		t.Fatal("nil account must not be authorized")
	}
	acc := FarmAccount{Platform: "wx", WxLoginBuffer: "buf"}
	if !acc.HasWxAuth() || !acc.CanWxReconnect() {
		t.Fatal("wx + buffer should reconnect")
	}
	acc.Platform = "qq"
	if !acc.HasWxAuth() {
		t.Fatal("HasWxAuth ignores platform")
	}
	if acc.CanWxReconnect() {
		t.Fatal("qq platform should not reconnect")
	}
	acc.Platform = "wx"
	acc.WxLoginBuffer = "  "
	if acc.HasWxAuth() {
		t.Fatal("whitespace buffer is not auth")
	}
}

func TestFarmAccountJSONRedactsWxSecrets(t *testing.T) {
	acc := FarmAccount{
		ID:            1,
		Name:          "n",
		Code:          "one-shot",
		Platform:      "wx",
		WxOpenID:      "oid",
		WxLoginBuffer: "secret-buf",
		WxAccessToken: "secret-tok",
	}
	acc.FillWxAuthorized()
	raw, err := json.Marshal(acc)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if strings.Contains(s, "secret-buf") || strings.Contains(s, "secret-tok") {
		t.Fatalf("secrets leaked: %s", s)
	}
	if strings.Contains(s, "wx_login_buffer") || strings.Contains(s, "wxLoginBuffer") {
		t.Fatalf("buffer field present: %s", s)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["wxOpenid"] != "oid" {
		t.Fatalf("wxOpenid=%v", m["wxOpenid"])
	}
	if m["wxAuthorized"] != true {
		t.Fatalf("wxAuthorized=%v", m["wxAuthorized"])
	}
}
