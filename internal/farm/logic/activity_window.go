package logic

import (
	"sync"
	"time"
)

const (
	activityWindowsCacheTTL = 5 * time.Minute
)

// ActivityWindow is one activity schedule row from ActivityService.List.
type ActivityWindow struct {
	ID        string
	Name      string
	BeginTime int64
	EndTime   int64
}

type activityWindowsState struct {
	mu       sync.RWMutex
	windows  map[string]ActivityWindow
	loaded   bool
	loadedAt time.Time
}

var globalActivityWindows = activityWindowsState{
	windows: make(map[string]ActivityWindow),
}

// SetActivityWindows replaces the cached activity windows.
func SetActivityWindows(windows []ActivityWindow) {
	globalActivityWindows.mu.Lock()
	defer globalActivityWindows.mu.Unlock()
	next := make(map[string]ActivityWindow, len(windows))
	for _, w := range windows {
		if w.ID == "" {
			continue
		}
		next[w.ID] = w
	}
	globalActivityWindows.windows = next
	globalActivityWindows.loaded = len(next) > 0
	globalActivityWindows.loadedAt = time.Now()
}

// ActivityWindowsSnapshot returns a copy of cached windows.
func ActivityWindowsSnapshot() []ActivityWindow {
	globalActivityWindows.mu.RLock()
	defer globalActivityWindows.mu.RUnlock()
	out := make([]ActivityWindow, 0, len(globalActivityWindows.windows))
	for _, w := range globalActivityWindows.windows {
		out = append(out, w)
	}
	return out
}

// ActivityWindowsFresh reports whether the cache is still within TTL.
func ActivityWindowsFresh() bool {
	globalActivityWindows.mu.RLock()
	defer globalActivityWindows.mu.RUnlock()
	return globalActivityWindows.loaded && time.Since(globalActivityWindows.loadedAt) < activityWindowsCacheTTL
}

// ActivityWindowsLoaded reports whether any window cache has been populated.
func ActivityWindowsLoaded() bool {
	globalActivityWindows.mu.RLock()
	defer globalActivityWindows.mu.RUnlock()
	return globalActivityWindows.loaded
}

// InvalidateActivityWindows 立即失效活动窗口缓存（对齐 rust
// config/activity_windows.rs invalidate_activity_windows：ActiviesChangeNotify
// 推送到达时 loaded_at=None，下次访问因不在 TTL 内而重新拉取；缓存值与 loaded
// 标记保留，供失效窗口期的兜底读取）。
func InvalidateActivityWindows() {
	globalActivityWindows.mu.Lock()
	defer globalActivityWindows.mu.Unlock()
	globalActivityWindows.loadedAt = time.Time{}
}

// ActivityWindowByID returns one cached window.
func ActivityWindowByID(id string) (ActivityWindow, bool) {
	if id == "" {
		return ActivityWindow{}, false
	}
	globalActivityWindows.mu.RLock()
	defer globalActivityWindows.mu.RUnlock()
	w, ok := globalActivityWindows.windows[id]
	return w, ok
}

// ResetActivityWindows clears the window cache (tests).
func ResetActivityWindows() {
	globalActivityWindows.mu.Lock()
	defer globalActivityWindows.mu.Unlock()
	globalActivityWindows.windows = make(map[string]ActivityWindow)
	globalActivityWindows.loaded = false
	globalActivityWindows.loadedAt = time.Time{}
}
