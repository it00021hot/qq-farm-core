package logic

import "strings"

// SellConditionContext is the runtime data needed to evaluate sell_cond.
type SellConditionContext struct {
	NowSec                int64
	ExpireTime            int64
	ActivityWindowsLoaded bool
}

type parsedSellCondition struct {
	Type  string
	Value string
}

func parseSellConditions(condition string) []parsedSellCondition {
	parts := strings.Split(condition, ";")
	out := make([]parsedSellCondition, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		typeName, value := part, ""
		if i := strings.Index(part, ":"); i >= 0 {
			typeName = strings.TrimSpace(part[:i])
			value = strings.TrimSpace(part[i+1:])
		}
		out = append(out, parsedSellCondition{Type: typeName, Value: value})
	}
	return out
}

func isActivityEnded(window ActivityWindow, ok bool, nowSec int64) bool {
	return !ok || window.EndTime <= nowSec
}

func isActivityActive(window ActivityWindow, ok bool, nowSec int64) bool {
	return ok && window.BeginTime <= nowSec && nowSec <= window.EndTime
}

func isSingleSellConditionSatisfied(cond parsedSellCondition, ctx SellConditionContext) bool {
	nowSec := ctx.NowSec
	if cond.Type == "道具过期后" {
		return ctx.ExpireTime > 0 && nowSec >= ctx.ExpireTime
	}
	if !ctx.ActivityWindowsLoaded || cond.Value == "" {
		return false
	}
	window, ok := ActivityWindowByID(cond.Value)
	switch cond.Type {
	case "活动结束后":
		return isActivityEnded(window, ok, nowSec)
	case "活动结束前":
		return !isActivityEnded(window, ok, nowSec)
	case "活动区间外":
		return !isActivityActive(window, ok, nowSec)
	default:
		return false
	}
}

// IsSellConditionSatisfied reports whether every `;`-joined sell_cond clause holds.
func IsSellConditionSatisfied(condition string, ctx SellConditionContext) bool {
	conditions := parseSellConditions(condition)
	if len(conditions) == 0 {
		return false
	}
	for _, entry := range conditions {
		if !isSingleSellConditionSatisfied(entry, ctx) {
			return false
		}
	}
	return true
}

// DefaultSellConditionContext uses cached activity windows and the given clock.
func DefaultSellConditionContext(nowSec int64) SellConditionContext {
	return SellConditionContext{
		NowSec:                nowSec,
		ActivityWindowsLoaded: ActivityWindowsLoaded(),
	}
}
