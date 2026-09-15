package logic

import (
	"testing"
	"time"
)

// TestInvalidateActivityWindows 对齐 rust config/activity_windows.rs
// invalidate_activity_windows：推送到达后立即失效新鲜度（loaded_at=None），
// 缓存值与 loaded 标记保留供失效窗口期兜底读取。
func TestInvalidateActivityWindows(t *testing.T) {
	ResetActivityWindows()
	t.Cleanup(ResetActivityWindows)

	SetActivityWindows([]ActivityWindow{{ID: "2026081202", Name: "青梅", BeginTime: 1, EndTime: 2}})
	if !ActivityWindowsFresh() {
		t.Fatal("cache should be fresh right after SetActivityWindows")
	}

	InvalidateActivityWindows()

	if ActivityWindowsFresh() {
		t.Fatal("cache should be stale right after InvalidateActivityWindows")
	}
	// 缓存值保留（对照 rust：loaded_at=None 但 windows/loaded 不清空）。
	if !ActivityWindowsLoaded() {
		t.Fatal("loaded flag should survive invalidation")
	}
	if snap := ActivityWindowsSnapshot(); len(snap) != 1 || snap[0].ID != "2026081202" {
		t.Fatalf("cached windows should survive invalidation, got %+v", snap)
	}
	if _, ok := ActivityWindowByID("2026081202"); !ok {
		t.Fatal("ActivityWindowByID should still hit after invalidation")
	}
}

// TestActivityWindowsFreshTTL：TTL 内新鲜、过期后失效（go 侧 5min TTL 兜底语义）。
func TestActivityWindowsFreshTTL(t *testing.T) {
	ResetActivityWindows()
	t.Cleanup(ResetActivityWindows)

	SetActivityWindows([]ActivityWindow{{ID: "1"}})
	globalActivityWindows.mu.Lock()
	globalActivityWindows.loadedAt = time.Now().Add(-activityWindowsCacheTTL - time.Second)
	globalActivityWindows.mu.Unlock()

	if ActivityWindowsFresh() {
		t.Fatal("cache older than TTL should be stale")
	}
}
