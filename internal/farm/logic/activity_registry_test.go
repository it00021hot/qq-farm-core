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

func TestParseActivitySellCond(t *testing.T) {
	greenPlum := "活动结束后:2026080102"
	if cond := logic.ParseActivitySellCond(&greenPlum); cond == nil || cond.ActivityID != "2026080102" {
		t.Fatalf("parse activity sell cond failed: %+v", cond)
	}
	empty := ""
	if cond := logic.ParseActivitySellCond(&empty); cond != nil {
		t.Fatalf("empty should be nil, got %+v", cond)
	}
	nilCond := (*string)(nil)
	if cond := logic.ParseActivitySellCond(nilCond); cond != nil {
		t.Fatalf("nil should be nil, got %+v", cond)
	}
	unrelated := "金币:100"
	if cond := logic.ParseActivitySellCond(&unrelated); cond != nil {
		t.Fatalf("unrelated should be nil, got %+v", cond)
	}
}

func TestIsActivityRestrictedForSaleGreenPlum(t *testing.T) {
	loadGameConfig(t)

	// 青梅 fruit (id 41221) has sell_cond 活动结束后:2026080102 (a stale id;
	// the running instance uses a fresh id like 2026081202).
	const greenPlumFruit = 41221
	now := time.Now().Unix()

	// Unknown activity must be treated as active (conservative skip).
	logic.ResetActivityRegistry()
	if !logic.IsActivityRestrictedForSale(greenPlumFruit, now) {
		t.Fatal("expected green plum fruit restricted when activity unknown")
	}

	// Referenced activity ended -> sale allowed.
	logic.ResetActivityRegistry()
	logic.RegisterActivity(logic.ActivityRegistryItem{
		ActivityID: "2026080102",
		Type:       12,
		EndTime:    now - 3600,
	})
	if logic.IsActivityRestrictedForSale(greenPlumFruit, now) {
		t.Fatal("expected green plum fruit sellable after activity end")
	}

	// Referenced activity ended, but a same-type instance is ongoing -> still
	// restricted (config keeps a stale id for recurring activities).
	logic.ResetActivityRegistry()
	logic.RegisterActivity(logic.ActivityRegistryItem{
		ActivityID: "2026080102",
		Type:       12,
		EndTime:    now - 3600,
	})
	logic.RegisterActivity(logic.ActivityRegistryItem{
		ActivityID: "2026081202",
		Type:       12,
		EndTime:    now + 86400,
	})
	if !logic.IsActivityRestrictedForSale(greenPlumFruit, now) {
		t.Fatal("expected green plum fruit restricted while a same-type activity runs")
	}

	// Activity ongoing -> sale blocked.
	logic.ResetActivityRegistry()
	logic.RegisterActivity(logic.ActivityRegistryItem{
		ActivityID: "2026080102",
		Type:       12,
		EndTime:    now + 86400,
	})
	if !logic.IsActivityRestrictedForSale(greenPlumFruit, now) {
		t.Fatal("expected green plum fruit restricted during activity")
	}
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
