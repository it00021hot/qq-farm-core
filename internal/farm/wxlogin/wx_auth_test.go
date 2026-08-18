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
	if c.TokenDueForRefresh(0) {
		t.Fatal("token with 30m left should not refresh when ahead=0")
	}
	if !c.TokenDueForRefresh(WxKeepaliveAheadSecs) {
		t.Fatal("token with 30m left should refresh under 45m keepalive window")
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

func TestRescanRecommended(t *testing.T) {
	now := time.Now().Unix()
	if RescanRecommended("rt", now-24*24*60*60, now) {
		t.Fatal("24d should not recommend")
	}
	if !RescanRecommended("rt", now-26*24*60*60, now) {
		t.Fatal("26d should recommend")
	}
	if RescanRecommended("", now-26*24*60*60, now) {
		t.Fatal("empty refresh should not recommend")
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
