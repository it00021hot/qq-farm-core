package logic_test

import (
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
)

func loadGameConfig(t *testing.T) {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	if err := logic.LoadGameConfig(filepath.Join(root, "resource", "farm", "gameConfig")); err != nil {
		t.Fatal(err)
	}
}

// TestSellConditionTypes mirrors bot sell-conditions.ts semantics:
// 道具过期后 / 活动结束后 / 活动结束前 / 活动区间外.
func TestSellConditionTypes(t *testing.T) {
	now := time.Now().Unix()

	// 道具过期后 requires a positive expire time already reached.
	if logic.IsSellConditionSatisfied("道具过期后", logic.SellConditionContext{NowSec: now, ExpireTime: 0, ActivityWindowsLoaded: true}) {
		t.Fatal("道具过期后 without expire time should not be satisfied")
	}
	if !logic.IsSellConditionSatisfied("道具过期后", logic.SellConditionContext{NowSec: now, ExpireTime: now - 60, ActivityWindowsLoaded: true}) {
		t.Fatal("道具过期后 after expiry should be satisfied")
	}

	// Activity clauses need loaded windows.
	if logic.IsSellConditionSatisfied("活动结束后:2026081202", logic.SellConditionContext{NowSec: now}) {
		t.Fatal("activity condition without loaded windows should not be satisfied")
	}

	logic.ResetActivityWindows()
	logic.SetActivityWindows([]logic.ActivityWindow{{ID: "2026081202", Name: "青梅", BeginTime: now - 3600, EndTime: now + 86400}})
	loaded := logic.SellConditionContext{NowSec: now, ActivityWindowsLoaded: true}

	if logic.IsSellConditionSatisfied("活动结束后:2026081202", loaded) {
		t.Fatal("活动结束后 should not be satisfied while activity runs")
	}
	if !logic.IsSellConditionSatisfied("活动结束前:2026081202", loaded) {
		t.Fatal("活动结束前 should be satisfied while activity runs")
	}
	if logic.IsSellConditionSatisfied("活动区间外:2026081202", loaded) {
		t.Fatal("活动区间外 should not be satisfied inside the window")
	}

	// Unknown (stale) activity id: missing window counts as ended (bot semantics).
	if !logic.IsSellConditionSatisfied("活动结束后:2026080102", loaded) {
		t.Fatal("活动结束后 with stale id should be satisfied (missing window = ended)")
	}
	if !logic.IsSellConditionSatisfied("活动区间外:2026080102", loaded) {
		t.Fatal("活动区间外 with stale id should be satisfied (missing window = inactive)")
	}

	// After the window ends.
	logic.ResetActivityWindows()
	logic.SetActivityWindows([]logic.ActivityWindow{{ID: "2026081202", Name: "青梅", BeginTime: now - 86400, EndTime: now - 3600}})
	if !logic.IsSellConditionSatisfied("活动结束后:2026081202", loaded) {
		t.Fatal("活动结束后 should be satisfied after end")
	}
	if !logic.IsSellConditionSatisfied("活动区间外:2026081202", loaded) {
		t.Fatal("活动区间外 should be satisfied after end")
	}
	if logic.IsSellConditionSatisfied("活动结束前:2026081202", loaded) {
		t.Fatal("活动结束前 should not be satisfied after end")
	}
	logic.ResetActivityWindows()
}

// TestGreenPlumSellEligibility uses the real gameConfig: 青梅 (41221) carries
// cond "活动区间外:2026081202" so it is only sellable outside the activity window.
func TestGreenPlumSellEligibility(t *testing.T) {
	loadGameConfig(t)
	now := time.Now().Unix()

	logic.ResetActivityWindows()
	// Windows not loaded → condition unsatisfied → not sellable via cond_sells.
	if logic.GetEffectiveSellInfo(logic.GetItemByID(41221)).Sellable {
		t.Fatal("green plum should not be sellable when windows are not loaded")
	}

	// Same activity running → 区间外 false → not sellable.
	logic.SetActivityWindows([]logic.ActivityWindow{{ID: "2026081202", Name: "青梅", BeginTime: now - 60, EndTime: now + 86400}})
	if logic.GetEffectiveSellInfoAt(logic.GetItemByID(41221), logic.SellConditionContext{NowSec: now, ActivityWindowsLoaded: true}, 0).Sellable {
		t.Fatal("green plum should be restricted while the activity runs")
	}

	// Activity ended → 区间外 true → sellable at cond price.
	logic.ResetActivityWindows()
	logic.SetActivityWindows([]logic.ActivityWindow{{ID: "2026081202", Name: "青梅", BeginTime: now - 86400, EndTime: now - 60}})
	if !logic.GetEffectiveSellInfoAt(logic.GetItemByID(41221), logic.SellConditionContext{NowSec: now, ActivityWindowsLoaded: true}, 0).Sellable {
		t.Fatal("green plum should be sellable after the activity ends")
	}
	logic.ResetActivityWindows()
}

func TestActivityRegistryRegisterAndSnapshot(t *testing.T) {
	logic.ResetActivityRegistry()
	logic.RegisterActivity(logic.ActivityRegistryItem{ActivityID: "2026080102", Type: 99, EndTime: 123})
	items := logic.ActivityRegistrySnapshot()
	if len(items) != 1 || items[0].ActivityID != "2026080102" || items[0].EndTime != 123 {
		t.Fatalf("unexpected registry snapshot: %+v", items)
	}
	end, ok := logic.ActivityEndTime("2026080102")
	if !ok || end != 123 {
		t.Fatalf("end time lookup failed: %d %v", end, ok)
	}
	if _, ok := logic.ActivityEndTime("missing"); ok {
		t.Fatal("missing activity should not be found")
	}
}
