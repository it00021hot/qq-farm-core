package logic

import (
	"sort"
	"strings"
	"sync"
	"time"
)

// ActivityRegistryItem is one known activity (id, type, schedule).
// It is populated from season info and game push notifies so that
// activity-restricted logic (e.g. selling) can be gated correctly.
type ActivityRegistryItem struct {
	ActivityID string `json:"activityId"`
	Type       int64  `json:"type"`
	Name       string `json:"name"`
	BeginTime  int64  `json:"beginTime"`
	EndTime    int64  `json:"endTime"`
}

type activityRegistry struct {
	mu sync.RWMutex
	m  map[string]ActivityRegistryItem
}

var globalActivityRegistry = activityRegistry{
	m: make(map[string]ActivityRegistryItem),
}

// RegisterActivity upserts one activity into the registry.
func RegisterActivity(item ActivityRegistryItem) {
	if item.ActivityID == "" {
		return
	}
	globalActivityRegistry.mu.Lock()
	defer globalActivityRegistry.mu.Unlock()
	if prev, ok := globalActivityRegistry.m[item.ActivityID]; ok && prev.Name != "" && item.Name == "" {
		item.Name = prev.Name
	}
	globalActivityRegistry.m[item.ActivityID] = item
}

// RegisterActivities bulk-upserts activities (e.g. from GetSeasonInfo reply).
func RegisterActivities(items []ActivityRegistryItem) {
	for i := range items {
		RegisterActivity(items[i])
	}
}

// ResetActivityRegistry clears all entries (test helper / config reload).
func ResetActivityRegistry() {
	globalActivityRegistry.mu.Lock()
	defer globalActivityRegistry.mu.Unlock()
	clear(globalActivityRegistry.m)
}

// ActivityRegistrySnapshot returns a copy of all known activities.
func ActivityRegistrySnapshot() []ActivityRegistryItem {
	globalActivityRegistry.mu.RLock()
	defer globalActivityRegistry.mu.RUnlock()
	out := make([]ActivityRegistryItem, 0, len(globalActivityRegistry.m))
	for _, item := range globalActivityRegistry.m {
		out = append(out, item)
	}
	return out
}

// ActivityEndTime returns the known end time (Unix seconds) for an activity.
func ActivityEndTime(activityID string) (int64, bool) {
	if activityID == "" {
		return 0, false
	}
	globalActivityRegistry.mu.RLock()
	defer globalActivityRegistry.mu.RUnlock()
	item, ok := globalActivityRegistry.m[activityID]
	if !ok {
		return 0, false
	}
	return item.EndTime, true
}

// ActivityActive reports whether an activity is still ongoing at now (Unix
// seconds). Unknown activities default to active=true so restricted items are
// conservatively kept out of auto-sell batches.
func ActivityActive(activityID string, now int64) bool {
	end, ok := ActivityEndTime(activityID)
	if !ok || end <= 0 {
		return true
	}
	return now < end
}

// activityByID returns one registry entry by id.
func activityByID(activityID string) (ActivityRegistryItem, bool) {
	if activityID == "" {
		return ActivityRegistryItem{}, false
	}
	globalActivityRegistry.mu.RLock()
	defer globalActivityRegistry.mu.RUnlock()
	item, ok := globalActivityRegistry.m[activityID]
	return item, ok
}

// IsGreenPlumActivity reports whether an activity looks like the recurring
// 青梅 / 青酿换万金 activity by stable features (name keyword or type code)
// rather than by a hard-coded id that changes every run.
func IsGreenPlumActivity(activityType int64, name string) bool {
	if activityType == 12 {
		return true
	}
	return strings.Contains(name, "青梅") || strings.Contains(name, "青酿")
}

// FindGreenPlumActivity returns the registry entry matching the 青梅 activity
// by feature, preferring the currently ongoing one. Unknown activities keep it
// nil so callers can fall back to an explicit id passed by the user.
func FindGreenPlumActivity() *ActivityRegistryItem {
	now := time.Now().Unix()
	var fallback *ActivityRegistryItem
	for _, item := range ActivityRegistrySnapshot() {
		if !IsGreenPlumActivity(item.Type, item.Name) {
			continue
		}
		it := item
		if (it.BeginTime <= 0 || now >= it.BeginTime) && (it.EndTime <= 0 || now <= it.EndTime) {
			return &it
		}
		if fallback == nil {
			fallback = &it
		}
	}
	return fallback
}

// GreenPlumActivities returns every registry entry matching the 青梅 activity
// feature, sorted by activity id so the daily seed entry (…01) and the brew
// entry (…02) keep a stable order.
func GreenPlumActivities() []ActivityRegistryItem {
	out := make([]ActivityRegistryItem, 0)
	for _, item := range ActivityRegistrySnapshot() {
		if IsGreenPlumActivity(item.Type, item.Name) {
			out = append(out, item)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ActivityID < out[j].ActivityID })
	return out
}
