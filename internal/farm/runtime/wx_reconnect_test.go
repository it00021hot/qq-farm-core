package runtime

import "testing"

func TestShouldAttemptWxReconnect(t *testing.T) {
	if !shouldAttemptWxReconnect(false, true) {
		t.Fatal("authorized disconnect should reconnect")
	}
	if shouldAttemptWxReconnect(true, true) {
		t.Fatal("manual stop must not reconnect")
	}
	if shouldAttemptWxReconnect(false, false) {
		t.Fatal("no wx auth must not reconnect")
	}
}

func TestWxReconnectWaitsFiveMinutes(t *testing.T) {
	if WxReconnectDelay.Minutes() != 5 {
		t.Fatalf("delay=%v", WxReconnectDelay)
	}
	if wxReconnectDelayZh() != "5 分钟" {
		t.Fatalf("zh=%q", wxReconnectDelayZh())
	}
	if WxReconnectMaxAttempts != 3 {
		t.Fatalf("max=%d", WxReconnectMaxAttempts)
	}
}

func TestBootReconnectIsImmediate(t *testing.T) {
	if wxReconnectDelayZh() != "5 分钟" {
		t.Fatalf("disconnect reconnect delay changed: %q", wxReconnectDelayZh())
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
