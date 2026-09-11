// Package protocol — request priority classes, ported 1:1 from
// qq-farm-bot/core/src/utils/request-priority.ts and low-priority-gate.ts.
//
// The gateway is a single multiplexed WebSocket shared by every caller, so the
// concurrency budget and class priorities live here (not at call sites).
//
// 班次（从高到低）:
//  1. critical   —— 心跳 / ACE AntiData，掉了就直接下线，各有独立保留槽位
//  2. foreground —— 面板上的前台操作，人在等结果
//  3. farm       —— 自己农场的后台定时任务
//  4. friend     —— 好友农场的后台定时任务
//  5. background —— 宠物同步等「补数据」任务，只在网关完全空闲时才发
package protocol

import (
	"context"
	"time"
)

// RequestClass is the scheduling class of a gateway request.
type RequestClass string

const (
	ClassCritical   RequestClass = "critical"
	ClassForeground RequestClass = "foreground"
	ClassFarm       RequestClass = "farm"
	ClassFriend     RequestClass = "friend"
	ClassBackground RequestClass = "background"
)

// CriticalLane reserves one dedicated slot each for heartbeat / ace traffic.
type CriticalLane string

const (
	LaneHeartbeat CriticalLane = "heartbeat"
	LaneAce       CriticalLane = "ace"
)

var requestClassOrder = []RequestClass{ClassCritical, ClassForeground, ClassFarm, ClassFriend, ClassBackground}
var criticalLanes = []CriticalLane{LaneHeartbeat, LaneAce}
var businessClasses = []RequestClass{ClassForeground, ClassFarm, ClassFriend}

const (
	// maxBusinessInFlight is the total in-flight budget for foreground+farm+friend.
	maxBusinessInFlight = 3
	// maxNonForegroundBusiness caps background automation so the panel keeps slots.
	maxNonForegroundBusiness = 1
)

// maxInFlightByClass is the per-class in-flight cap.
var maxInFlightByClass = map[RequestClass]int{
	ClassCritical:   2,
	ClassForeground: 3,
	ClassFarm:       1,
	ClassFriend:     1,
	ClassBackground: 1,
}

// maxQueuedByClass is the per-class queue cap; background is deliberately tiny.
var maxQueuedByClass = map[RequestClass]int{
	ClassCritical:   8,
	ClassForeground: 60,
	ClassFarm:       40,
	ClassFriend:     30,
	ClassBackground: 10,
}

// classStarvationMs promotes a low-class request that waited too long.
const classStarvationMs = 4000

// Low-priority gate constants (bot low-priority-gate.ts).
const (
	// lowPriorityQueueWaitMs: how long a background request may wait in queue
	// before yielding with GatewayBusyError.
	lowPriorityQueueWaitMs = 8000
	// gatewayStallPendingMs: an in-flight request older than this means the
	// gateway is stalling; background traffic must stop immediately.
	gatewayStallPendingMs = 5000
	// businessBackoff bounds for farm/friend health backoff.
	businessBackoffMinMs = 30000
	businessBackoffMaxMs = 60000
	// requestPressureLogIntervalMs throttles queue-pressure warnings.
	requestPressureLogIntervalMs = 5000
)

// defaultRequestTimeout matches bot sendMsgAsync default (20s).
const defaultRequestTimeout = 20 * time.Second

type classCandidate interface {
	requestClass() RequestClass
	lane() CriticalLane
	enqueuedAt() time.Time
}

type classInFlight interface {
	requestClass() RequestClass
	lane() CriticalLane
}

func normalizeRequestClass(v RequestClass) (RequestClass, bool) {
	for _, c := range requestClassOrder {
		if v == c {
			return v, true
		}
	}
	return "", false
}

func classOfRequest(c classCandidate) RequestClass {
	if cls, ok := normalizeRequestClass(c.requestClass()); ok {
		return cls
	}
	return ClassForeground
}

func isBusinessClass(c RequestClass) bool {
	for _, b := range businessClasses {
		if b == c {
			return true
		}
	}
	return false
}

// resolveRequestClass mirrors bot resolveRequestClass: explicit class wins,
// then legacy priority mapping, then the ambient scheduler class, then
// foreground. Heartbeat/AntiData map to critical lanes.
func resolveRequestClass(method string, explicit RequestClass, priority string, ambient RequestClass) (RequestClass, CriticalLane) {
	switch {
	case stringsEqualFold(method, "Heartbeat"):
		return ClassCritical, LaneHeartbeat
	case stringsEqualFold(method, "AntiData"):
		return ClassCritical, LaneAce
	}
	if priority == "high" {
		return ClassCritical, ""
	}
	if cls, ok := normalizeRequestClass(explicit); ok {
		return cls, ""
	}
	if priority == "low" {
		return ClassBackground, ""
	}
	if cls, ok := normalizeRequestClass(ambient); ok {
		return cls, ""
	}
	return ClassForeground, ""
}

// selectDispatchIndex picks the next queue entry allowed to fly. Returns -1
// when nothing is dispatchable. Pure function over snapshots — the caller owns
// the lock and mutation.
func selectDispatchIndex(queue []classCandidate, inFlight []classInFlight, now time.Time) int {
	if len(queue) == 0 {
		return -1
	}

	classOfInFlight := func(r classInFlight) RequestClass {
		if cls, ok := normalizeRequestClass(r.requestClass()); ok {
			return cls
		}
		return ClassForeground
	}
	countInFlight := func(pred func(cls RequestClass, lane CriticalLane) bool) int {
		n := 0
		for _, r := range inFlight {
			if pred(classOfInFlight(r), r.lane()) {
				n++
			}
		}
		return n
	}
	perClassInFlight := func() map[RequestClass]int {
		m := make(map[RequestClass]int, len(requestClassOrder))
		for _, r := range inFlight {
			m[classOfInFlight(r)]++
		}
		return m
	}

	// 1) 心跳 / ACE 各占一个独立保留槽位，谁也挤不掉谁。
	for _, lane := range criticalLanes {
		if countInFlight(func(cls RequestClass, l CriticalLane) bool { return cls == ClassCritical && l == lane }) >= 1 {
			continue
		}
		for i, r := range queue {
			if classOfRequest(r) == ClassCritical && r.lane() == lane {
				return i
			}
		}
	}

	// 2) 没有标记通道的 critical 请求只吃 critical 的普通预算。
	if countInFlight(func(cls RequestClass, _ CriticalLane) bool { return cls == ClassCritical }) < maxInFlightByClass[ClassCritical] {
		for i, r := range queue {
			if classOfRequest(r) == ClassCritical && r.lane() == "" {
				return i
			}
		}
	}

	// 3) 业务班次：总预算 + 每班次上限 + 前台保留槽位三重约束。
	if countInFlight(func(cls RequestClass, _ CriticalLane) bool { return isBusinessClass(cls) }) < maxBusinessInFlight {
		hasQueuedForeground := false
		for _, r := range queue {
			if classOfRequest(r) == ClassForeground {
				hasQueuedForeground = true
				break
			}
		}
		nonForeground := countInFlight(func(cls RequestClass, _ CriticalLane) bool {
			return isBusinessClass(cls) && cls != ClassForeground
		})
		perClass := perClassInFlight()

		eligible := make([]int, 0, len(queue))
		for i, r := range queue {
			cls := classOfRequest(r)
			if !isBusinessClass(cls) {
				continue
			}
			if perClass[cls] >= maxInFlightByClass[cls] {
				continue
			}
			if cls != ClassForeground && (hasQueuedForeground || nonForeground >= maxNonForegroundBusiness) {
				continue
			}
			eligible = append(eligible, i)
		}

		if len(eligible) > 0 {
			// 先救被插队太久的：等待最长且已超阈值的优先发送。
			starvedIndex := -1
			starvedWaitMs := int64(classStarvationMs)
			for _, i := range eligible {
				waited := now.Sub(queue[i].enqueuedAt()).Milliseconds()
				if waited >= starvedWaitMs {
					starvedWaitMs = waited
					starvedIndex = i
				}
			}
			if starvedIndex >= 0 {
				return starvedIndex
			}
			// 否则按班次优先级、同班次内 FIFO。
			for _, cls := range requestClassOrder {
				if !isBusinessClass(cls) {
					continue
				}
				for _, i := range eligible {
					if classOfRequest(queue[i]) == cls {
						return i
					}
				}
			}
		}
	}

	// 4) background 是「补数据」：只在连接彻底空闲、且队列里没有别的班次时才发。
	if len(inFlight) > 0 {
		return -1
	}
	for _, r := range queue {
		if classOfRequest(r) != ClassBackground {
			return -1
		}
	}
	for i, r := range queue {
		if classOfRequest(r) == ClassBackground {
			return i
		}
	}
	return -1
}

// gatewayLoad is the load snapshot used by the low-priority gate.
type gatewayLoad struct {
	pending           int
	queued            int
	blockingQueued    int
	criticalPending   int
	businessPending   int
	foregroundPending int
	backgroundPending int
	heartbeatMisses   int
	oldestPendingAgeMs int64
}

// isGatewayIdleForLowPriority mirrors bot: background may only fly when no
// business traffic is waiting or in flight and the connection looks healthy.
func isGatewayIdleForLowPriority(load gatewayLoad) bool {
	if load.blockingQueued > 0 || load.businessPending > 0 || load.backgroundPending > 0 {
		return false
	}
	if load.heartbeatMisses > 0 {
		return false
	}
	if load.oldestPendingAgeMs >= gatewayStallPendingMs {
		return false
	}
	return true
}

// isGatewayHealthyForBusiness is the farm/friend health gate: only requires
// the connection to still be answering.
func isGatewayHealthyForBusiness(load gatewayLoad) bool {
	if load.heartbeatMisses > 0 {
		return false
	}
	if load.oldestPendingAgeMs >= gatewayStallPendingMs {
		return false
	}
	return true
}

// nextBusinessBackoffMs: first 30s, then double capped at 60s.
func nextBusinessBackoffMs(previous int64) int64 {
	if previous <= 0 {
		return businessBackoffMinMs
	}
	if next := previous * 2; next < businessBackoffMaxMs {
		return next
	}
	return businessBackoffMaxMs
}

// --- ambient request class injection (bot request-context.ts AsyncLocalStorage) ---

type ambientClassKey struct{}

// WithRequestClass tags ctx with the ambient scheduling class; all sends on
// that ctx inherit it unless explicitly overridden.
func WithRequestClass(ctx context.Context, class RequestClass) context.Context {
	return context.WithValue(ctx, ambientClassKey{}, class)
}

// AmbientRequestClass returns the injected class, if any.
func AmbientRequestClass(ctx context.Context) RequestClass {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(ambientClassKey{}).(RequestClass); ok {
		return v
	}
	return ""
}

func stringsEqualFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if 'A' <= ca && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if 'A' <= cb && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
