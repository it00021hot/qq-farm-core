package runtime

import (
	"testing"
	"time"
)

func TestShouldAttemptWxReconnect(t *testing.T) {
	if !shouldAttemptWxReconnect(false, true, "") {
		t.Fatal("authorized disconnect should reconnect")
	}
	if shouldAttemptWxReconnect(true, true, "") {
		t.Fatal("manual stop must not reconnect")
	}
	if shouldAttemptWxReconnect(false, false, "") {
		t.Fatal("no wx auth must not reconnect")
	}
	// rust：wx_auth_failed / wx_mint_failed 均不重连。
	if shouldAttemptWxReconnect(false, true, "disconnect:wx_auth_failed") {
		t.Fatal("wx auth failure must not reconnect")
	}
	if shouldAttemptWxReconnect(false, true, "disconnect:wx_mint_failed") {
		t.Fatal("wx mint failure must not reconnect")
	}
	if !shouldAttemptWxReconnect(false, true, "disconnect:ws_closed") {
		t.Fatal("plain transport error should reconnect")
	}
}

func TestWxReconnectDelayTableMatchesRust(t *testing.T) {
	// rust timing.rs：第 1 次 15 分钟、第 2/3 次 10 分钟、被踢固定 3 分钟。
	if wxReconnectDelay(1) != 15*time.Minute {
		t.Fatalf("first delay=%v", wxReconnectDelay(1))
	}
	if wxReconnectDelay(2) != 10*time.Minute || wxReconnectDelay(3) != 10*time.Minute {
		t.Fatalf("retry delay=%v/%v", wxReconnectDelay(2), wxReconnectDelay(3))
	}
	if wxKickoutReconnectDelay() != 3*time.Minute {
		t.Fatalf("kickout delay=%v", wxKickoutReconnectDelay())
	}
	if WxReconnectMaxAttempts != 3 {
		t.Fatalf("max=%d", WxReconnectMaxAttempts)
	}
	if WxRestartGrace != 3*time.Second {
		t.Fatalf("restart grace=%v", WxRestartGrace)
	}
	if durationZh(3*time.Minute) != "3 分钟" || durationZh(15*time.Minute) != "15 分钟" || durationZh(45*time.Second) != "45 秒" {
		t.Fatalf("zh formatting unexpected")
	}
}

func TestPlanWxReconnectCapsAttemptsAndInflight(t *testing.T) {
	m := NewAccountManager(nil)
	d := m.planWxReconnect("a1")
	if d.plan != wxReconnectSpawn || d.attempt != 1 {
		t.Fatalf("first=%+v", d)
	}
	if m.planWxReconnect("a1").plan != wxReconnectSkip {
		t.Fatal("inflight should skip")
	}
	m.dropWxReconnectInflight("a1")
	d = m.planWxReconnect("a1")
	if d.plan != wxReconnectSpawn || d.attempt != 2 {
		t.Fatalf("second=%+v", d)
	}
	m.dropWxReconnectInflight("a1")
	d = m.planWxReconnect("a1")
	if d.plan != wxReconnectSpawn || d.attempt != 3 {
		t.Fatalf("third=%+v", d)
	}
	m.dropWxReconnectInflight("a1")
	if m.planWxReconnect("a1").plan != wxReconnectGiveUp {
		t.Fatal("fourth should give up")
	}
	m.clearWxReconnect("a1")
	d = m.planWxReconnect("a1")
	if d.plan != wxReconnectSpawn || d.attempt != 1 {
		t.Fatalf("after clear=%+v", d)
	}
}

func TestClearWxReconnectCancelsDelayedStart(t *testing.T) {
	m := NewAccountManager(nil)
	d := m.planWxReconnect("a1")
	gen := d.gen
	m.clearWxReconnect("a1")
	if m.wxReconnectGen("a1") == gen {
		t.Fatal("manual stop must bump generation so delayed start aborts")
	}
}

func TestClearWxReconnectAttemptsKeepsGeneration(t *testing.T) {
	m := NewAccountManager(nil)
	d := m.planWxReconnect("a1")
	gen := d.gen
	m.clearWxReconnectAttempts("a1")
	if m.wxReconnectGen("a1") != gen {
		t.Fatal("online success should not bump generation")
	}
	d = m.planWxReconnect("a1")
	if d.plan != wxReconnectSpawn || d.attempt != 1 {
		t.Fatalf("attempts should reset without skip: %+v", d)
	}
}

func TestAccountNoticeContentAlignsRust(t *testing.T) {
	cases := []struct {
		kind   AccountNoticeKind
		name   string
		id     string
		expect string
	}{
		{NoticeOffline, "大号", "42", "账号 大号 已下线"},
		{NoticeOnline, "大号", "42", "账号 大号 已上线"},
		{NoticeYybQr, "大号", "42", "账号 大号 应用宝授权失效，请扫描二维码重新登录"},
		{NoticeOnline, "", "42", "账号 42 已上线"},
		{NoticeYybQr, "", "", "账号 未知账号 应用宝授权失效，请扫描二维码重新登录"},
	}
	for _, c := range cases {
		if got := accountNoticeContent(c.kind, c.name, c.id); got != c.expect {
			t.Fatalf("kind=%q got %q want %q", c.kind, got, c.expect)
		}
	}
}

func TestAccountNoticeLabel(t *testing.T) {
	if accountNoticeLabel(NoticeOffline) != "下线" ||
		accountNoticeLabel(NoticeOnline) != "上线" ||
		accountNoticeLabel(NoticeYybQr) != "应用宝授权二维码" {
		t.Fatal("unexpected kind labels")
	}
}
