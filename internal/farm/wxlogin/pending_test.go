package wxlogin

import (
	"testing"
	"time"
)

func TestPendingAuthTakenOnce(t *testing.T) {
	resetPendingAuthForTest()
	StorePendingAuth("wx-code", WxAuth{OpenID: "oid", LoginBuffer: "buf", AccessToken: "tok"})
	auth, ok := TakePendingAuth("wx-code")
	if !ok || auth.OpenID != "oid" || auth.LoginBuffer != "buf" || auth.AccessToken != "tok" {
		t.Fatalf("got %+v ok=%v", auth, ok)
	}
	if _, ok := TakePendingAuth("wx-code"); ok {
		t.Fatal("pending auth must be one-shot")
	}
}

func TestStorePendingAuthSkipsEmpty(t *testing.T) {
	resetPendingAuthForTest()
	StorePendingAuth("", WxAuth{LoginBuffer: "buf"})
	StorePendingAuth("code", WxAuth{})
	StorePendingAuth("   ", WxAuth{LoginBuffer: "buf"})
	if _, ok := TakePendingAuth("code"); ok {
		t.Fatal("empty buffer must not be stored")
	}
}

func TestPendingAuthTTL(t *testing.T) {
	resetPendingAuthForTest()
	pendingMu.Lock()
	pendingAuth["old"] = pendingWxAuth{
		auth:      WxAuth{LoginBuffer: "buf"},
		createdAt: time.Now().Add(-PendingAuthTTL - time.Second),
	}
	pendingMu.Unlock()
	if _, ok := TakePendingAuth("old"); ok {
		t.Fatal("expired pending auth must be pruned")
	}
}
