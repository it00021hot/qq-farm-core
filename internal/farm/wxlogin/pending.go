package wxlogin

import (
	"strings"
	"sync"
	"time"
)

const PendingAuthTTL = 10 * time.Minute

// WxAuth is Yingyongbao authorization that can mint one-shot gateway codes.
// It is never returned over HTTP/IPC.
type WxAuth struct {
	OpenID      string
	LoginBuffer string
	AccessToken string
}

type pendingWxAuth struct {
	auth      WxAuth
	createdAt time.Time
}

var pendingMu sync.Mutex
var pendingAuth = make(map[string]pendingWxAuth)

func prunePendingAuthLocked(now time.Time) {
	for code, item := range pendingAuth {
		if now.Sub(item.createdAt) > PendingAuthTTL {
			delete(pendingAuth, code)
		}
	}
}

// StorePendingAuth keeps Yingyongbao tickets keyed by the one-shot gateway code (10 min TTL).
func StorePendingAuth(code string, auth WxAuth) {
	code = strings.TrimSpace(code)
	if code == "" || strings.TrimSpace(auth.LoginBuffer) == "" {
		return
	}
	now := time.Now()
	pendingMu.Lock()
	defer pendingMu.Unlock()
	prunePendingAuthLocked(now)
	pendingAuth[code] = pendingWxAuth{auth: auth, createdAt: now}
}

// TakePendingAuth claims the scan-time Yingyongbao tickets exactly once.
func TakePendingAuth(code string) (WxAuth, bool) {
	code = strings.TrimSpace(code)
	if code == "" {
		return WxAuth{}, false
	}
	now := time.Now()
	pendingMu.Lock()
	defer pendingMu.Unlock()
	prunePendingAuthLocked(now)
	item, ok := pendingAuth[code]
	if !ok {
		return WxAuth{}, false
	}
	delete(pendingAuth, code)
	return item.auth, true
}

func resetPendingAuthForTest() {
	pendingMu.Lock()
	pendingAuth = make(map[string]pendingWxAuth)
	pendingMu.Unlock()
}
