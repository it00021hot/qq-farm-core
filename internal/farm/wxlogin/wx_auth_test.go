package wxlogin

import (
	"testing"
	"time"
)

func TestParseQuickRedirectURLAcceptsValid(t *testing.T) {
	url := "https://yybadaccess.3g.qq.com/pc_yyb/pcyyb_oauth?login_type=WX&state=web&code=abc123"
	got, err := ParseQuickRedirectURL(url)
	if err != nil || got != "abc123" {
		t.Fatalf("got=%q err=%v", got, err)
	}
}

func TestParseQuickRedirectURLRejectsBadHost(t *testing.T) {
	_, err := ParseQuickRedirectURL("https://evil.com/pc_yyb/pcyyb_oauth?login_type=WX&state=web&code=x")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseQuickRedirectURLRejectsBadState(t *testing.T) {
	_, err := ParseQuickRedirectURL("https://yybadaccess.3g.qq.com/pc_yyb/pcyyb_oauth?login_type=WX&state=bad&code=x")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseQuickRedirectURLRejectsEmptyCode(t *testing.T) {
	_, err := ParseQuickRedirectURL("https://yybadaccess.3g.qq.com/pc_yyb/pcyyb_oauth?login_type=WX&state=web&code=")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTokenDueForRefresh(t *testing.T) {
	c := YybCredentials{ExpiresAt: time.Now().Unix() + 1800}
	if !c.TokenDueForRefresh(WxKeepaliveAheadSecs) {
		t.Fatal("token within ahead window should refresh")
	}
	c.ExpiresAt = time.Now().Unix() + 7200
	if c.TokenDueForRefresh(WxKeepaliveAheadSecs) {
		t.Fatal("token outside ahead window should not refresh")
	}
	c.ExpiresAt = 0
	if !c.TokenDueForRefresh(0) {
		t.Fatal("unknown expiry should refresh")
	}
}

func TestClassifyYybMessage(t *testing.T) {
	if classifyYybMessage("WeChat login buffer response is invalid") != WxAuthErrorCredentialsDead {
		t.Fatal("invalid buffer should be dead")
	}
	if classifyYybMessage("Unable to obtain WeChat login buffer (HTTP 502)") != WxAuthErrorTransient {
		t.Fatal("http error should be transient")
	}
}
