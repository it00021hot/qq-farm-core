package logic_test

import (
	"reflect"
	"testing"

	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
)

func TestResolveSeedLandTypes(t *testing.T) {
	restrictions := map[string][]string{
		"20108": {"red", "black"},
		"20129": {},
		"21380": {"purple-gold", "gold", "black", "red", "normal"},
	}
	got, ok := logic.ResolveSeedLandTypes(restrictions, 20108)
	if !ok || !reflect.DeepEqual(got, []string{"red", "black"}) {
		t.Fatalf("restricted seed resolve = %v %v", got, ok)
	}
	// 空数组 → 不限制。
	if _, ok := logic.ResolveSeedLandTypes(restrictions, 20129); ok {
		t.Fatal("empty list must mean no restriction")
	}
	// 勾满全部五类 → 不限制。
	if _, ok := logic.ResolveSeedLandTypes(restrictions, 21380); ok {
		t.Fatal("all five types must mean no restriction")
	}
	// 缺 key → 不限制。
	if _, ok := logic.ResolveSeedLandTypes(restrictions, 99999); ok {
		t.Fatal("missing key must mean no restriction")
	}
	// 限制表为空 → 不限制。
	if _, ok := logic.ResolveSeedLandTypes(nil, 20108); ok {
		t.Fatal("nil table must mean no restriction")
	}
	if !logic.SeedHasLandRestriction(restrictions, 20108) {
		t.Fatal("20108 should be restricted")
	}
	if logic.SeedHasLandRestriction(restrictions, 20129) {
		t.Fatal("20129 should not be restricted")
	}
	if logic.SeedHasLandRestriction(restrictions, 99999) {
		t.Fatal("unknown seed should not be restricted")
	}
}

func TestSortSeedsRestrictedFirst(t *testing.T) {
	seeds := []logic.BagSeed{
		{SeedID: 1, Count: 9},
		{SeedID: 2, Count: 9},
		{SeedID: 3, Count: 9},
		{SeedID: 4, Count: 9},
	}
	ids := func(in []logic.BagSeed) []int64 {
		out := make([]int64, 0, len(in))
		for _, s := range in {
			out = append(out, s.SeedID)
		}
		return out
	}
	// 单个受限种子排最前，其余保持原相对顺序。
	ordered := logic.SortSeedsRestrictedFirst(seeds, map[string][]string{"2": {"red"}})
	if want := []int64{2, 1, 3, 4}; !reflect.DeepEqual(ids(ordered), want) {
		t.Fatalf("ordered=%v want %v", ids(ordered), want)
	}
	// 多个受限种子之间保持原相对顺序（稳定分区）。
	ordered2 := logic.SortSeedsRestrictedFirst(seeds, map[string][]string{
		"4": {"gold"}, "1": {"red"},
	})
	if want := []int64{1, 4, 2, 3}; !reflect.DeepEqual(ids(ordered2), want) {
		t.Fatalf("ordered2=%v want %v", ids(ordered2), want)
	}
	// 空限制表：全部保留、顺序不变。
	ordered3 := logic.SortSeedsRestrictedFirst(seeds, nil)
	if !reflect.DeepEqual(ids(ordered3), ids(seeds)) {
		t.Fatalf("nil restrictions must keep order, got %v", ids(ordered3))
	}
}
