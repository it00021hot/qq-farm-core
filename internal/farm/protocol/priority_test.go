package protocol

import (
	"testing"
	"time"
)

type testCandidate struct {
	class      RequestClass
	laneV      CriticalLane
	enqueuedMs int64 // age in ms
}

func (t testCandidate) requestClass() RequestClass { return t.class }
func (t testCandidate) lane() CriticalLane         { return t.laneV }
func (t testCandidate) enqueuedAt() time.Time {
	return time.Now().Add(-time.Duration(t.enqueuedMs) * time.Millisecond)
}

func TestSelectDispatchCriticalLanesReserved(t *testing.T) {
	now := time.Now()
	// heartbeat lane busy → queued heartbeat must wait, but ace lane still dispatches.
	inFlight := []classInFlight{testCandidate{class: ClassCritical, laneV: LaneHeartbeat}}
	queue := []classCandidate{
		testCandidate{class: ClassCritical, laneV: LaneHeartbeat},
		testCandidate{class: ClassCritical, laneV: LaneAce},
	}
	idx := selectDispatchIndex(queue, inFlight, now)
	if idx != 1 {
		t.Fatalf("expected ace lane to dispatch, got index %d", idx)
	}
}

func TestSelectDispatchForegroundReservedSlots(t *testing.T) {
	now := time.Now()
	// One farm request in flight + a queued foreground request → a queued farm
	// request must yield (per-class cap 1 already blocks it, foreground wins).
	inFlight := []classInFlight{testCandidate{class: ClassFarm}}
	queue := []classCandidate{
		testCandidate{class: ClassFarm},
		testCandidate{class: ClassForeground},
	}
	idx := selectDispatchIndex(queue, inFlight, now)
	if idx != 1 {
		t.Fatalf("expected foreground to dispatch ahead of queued farm, got %d", idx)
	}

	// Without queued foreground and idle gateway, farm may fly.
	queue = []classCandidate{testCandidate{class: ClassFarm}}
	if idx := selectDispatchIndex(queue, nil, now); idx != 0 {
		t.Fatalf("expected farm to dispatch when idle, got %d", idx)
	}
}

func TestSelectDispatchBusinessTotalBudget(t *testing.T) {
	now := time.Now()
	// foreground+farm+friend = 3 in flight → no more business may start.
	inFlight := []classInFlight{
		testCandidate{class: ClassForeground},
		testCandidate{class: ClassFarm},
		testCandidate{class: ClassFriend},
	}
	queue := []classCandidate{testCandidate{class: ClassForeground}}
	if idx := selectDispatchIndex(queue, inFlight, now); idx != -1 {
		t.Fatalf("expected no dispatch when business budget exhausted, got %d", idx)
	}
}

func TestSelectDispatchBackgroundOnlyWhenIdle(t *testing.T) {
	now := time.Now()
	// Any non-background queued → background must not dispatch.
	queue := []classCandidate{
		testCandidate{class: ClassBackground},
		testCandidate{class: ClassForeground},
	}
	if idx := selectDispatchIndex(queue, nil, now); idx != 1 {
		t.Fatalf("expected foreground to win over background, got %d", idx)
	}
	// Fully idle + only background queued → dispatch.
	queue = []classCandidate{testCandidate{class: ClassBackground}}
	if idx := selectDispatchIndex(queue, nil, now); idx != 0 {
		t.Fatalf("expected background to dispatch when idle, got %d", idx)
	}
	// Any pending in flight → background waits.
	inFlight := []classInFlight{testCandidate{class: ClassCritical, laneV: LaneAce}}
	if idx := selectDispatchIndex(queue, inFlight, now); idx != -1 {
		t.Fatalf("expected background to wait while critical pending, got %d", idx)
	}
}

func TestSelectDispatchStarvationPromotion(t *testing.T) {
	now := time.Now()
	// A farm request queued ≥4s is promoted over a younger friend request
	// (same-tier classes; foreground queued would suppress the promotion).
	queue := []classCandidate{
		testCandidate{class: ClassFarm, enqueuedMs: 5000},
		testCandidate{class: ClassFriend, enqueuedMs: 100},
	}
	if idx := selectDispatchIndex(queue, nil, now); idx != 0 {
		t.Fatalf("expected starved farm to be promoted, got %d", idx)
	}
	// Below the starvation threshold the class order wins (farm before friend).
	queue = []classCandidate{
		testCandidate{class: ClassFriend, enqueuedMs: 100},
		testCandidate{class: ClassFarm, enqueuedMs: 100},
	}
	if idx := selectDispatchIndex(queue, nil, now); idx != 1 {
		t.Fatalf("expected farm to win by class order, got %d", idx)
	}
}

func TestResolveRequestClass(t *testing.T) {
	if cls, lane := resolveRequestClass("Heartbeat", "", "", ""); cls != ClassCritical || lane != LaneHeartbeat {
		t.Fatalf("heartbeat should map to critical/heartbeat lane: %s/%s", cls, lane)
	}
	if cls, lane := resolveRequestClass("AntiData", "", "", ""); cls != ClassCritical || lane != LaneAce {
		t.Fatalf("antidata should map to critical/ace lane: %s/%s", cls, lane)
	}
	if cls, _ := resolveRequestClass("AllLands", "", "", ClassFriend); cls != ClassFriend {
		t.Fatalf("ambient class should win by default: %s", cls)
	}
	if cls, _ := resolveRequestClass("AllLands", ClassFarm, "", ClassFriend); cls != ClassFarm {
		t.Fatalf("explicit class should beat ambient: %s", cls)
	}
	if cls, _ := resolveRequestClass("AllLands", "", "high", ClassFriend); cls != ClassCritical {
		t.Fatalf("priority high should map to critical: %s", cls)
	}
	if cls, _ := resolveRequestClass("AllLands", "", "low", ClassFriend); cls != ClassBackground {
		t.Fatalf("priority low should map to background: %s", cls)
	}
	if cls, _ := resolveRequestClass("AllLands", "", "", ""); cls != ClassForeground {
		t.Fatalf("no marker should default to foreground: %s", cls)
	}
}

func TestLowPriorityGate(t *testing.T) {
	if isGatewayIdleForLowPriority(gatewayLoad{}) != true {
		t.Fatal("idle load should allow background")
	}
	if isGatewayIdleForLowPriority(gatewayLoad{businessPending: 1}) {
		t.Fatal("business pending should block background")
	}
	if isGatewayIdleForLowPriority(gatewayLoad{blockingQueued: 1}) {
		t.Fatal("blocking queued should block background")
	}
	if isGatewayIdleForLowPriority(gatewayLoad{heartbeatMisses: 1}) {
		t.Fatal("heartbeat miss should block background")
	}
	if isGatewayIdleForLowPriority(gatewayLoad{oldestPendingAgeMs: gatewayStallPendingMs}) {
		t.Fatal("stalled pending should block background")
	}
	if !isGatewayHealthyForBusiness(gatewayLoad{}) {
		t.Fatal("healthy load should allow business")
	}
	if isGatewayHealthyForBusiness(gatewayLoad{heartbeatMisses: 1}) {
		t.Fatal("heartbeat miss should block business")
	}
	if got := nextBusinessBackoffMs(0); got != businessBackoffMinMs {
		t.Fatalf("first backoff should be %d, got %d", businessBackoffMinMs, got)
	}
	if got := nextBusinessBackoffMs(40000); got != businessBackoffMaxMs {
		t.Fatalf("backoff should cap at %d, got %d", businessBackoffMaxMs, got)
	}
}

func TestIsGatewayYieldError(t *testing.T) {
	if !IsGatewayYieldError(&GatewayBusyError{Message: "网关繁忙，后台请求已让路"}) {
		t.Fatal("GatewayBusyError should yield")
	}
	if !IsGatewayYieldError(errString("请求等待队列已满: X")) {
		t.Fatal("queue full should yield")
	}
	if !IsGatewayYieldError(errString("连接未打开: X")) {
		t.Fatal("closed connection should yield")
	}
	if IsGatewayYieldError(errString("some business error")) {
		t.Fatal("ordinary errors must not yield")
	}
}

type errString string

func (e errString) Error() string { return string(e) }
