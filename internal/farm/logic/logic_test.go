package logic_test

import (
	"testing"

	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/plantpb"
)

func TestAnalyzeLandsEmptyAndMature(t *testing.T) {
	logic.SyncServerTime(1_700_000_000_000) // ms
	lands := []logic.LandInfo{
		{ID: 1, Unlocked: true},
		{
			ID: 2, Unlocked: true,
			Plant: &logic.PlantInfo{
				ID: 100, Name: "测试",
				Phases: []logic.PlantPhaseInfo{
					{Phase: logic.PhaseMature, BeginTime: 1_600_000_000},
				},
			},
		},
		{ID: 3, Unlocked: false, CouldUnlock: true},
	}
	result := logic.AnalyzeLands(lands)
	if len(result.Empty) != 1 || result.Empty[0] != 1 {
		t.Fatalf("empty=%v", result.Empty)
	}
	if len(result.Harvestable) != 1 || result.Harvestable[0] != 2 {
		t.Fatalf("harvestable=%v", result.Harvestable)
	}
	if len(result.Unlockable) != 1 || result.Unlockable[0] != 3 {
		t.Fatalf("unlockable=%v", result.Unlockable)
	}
}

func TestGetCurrentPhaseUsesLatestBegun(t *testing.T) {
	logic.SyncServerTime(1_700_000_000_000) // → ~1_700_000_000 sec
	now := logic.GetServerTimeSec()
	phases := []logic.PlantPhaseInfo{
		{Phase: logic.PhaseSeed, BeginTime: now - 300},
		{Phase: logic.PhaseGermination, BeginTime: now - 100},
		{Phase: logic.PhaseMature, BeginTime: now + 200},
	}
	cur := logic.GetCurrentPhase(phases)
	if cur == nil || cur.Phase != logic.PhaseGermination {
		t.Fatalf("current=%v", cur)
	}

	allFuture := []logic.PlantPhaseInfo{
		{Phase: logic.PhaseSeed, BeginTime: now + 10},
		{Phase: logic.PhaseMature, BeginTime: now + 100},
	}
	cur = logic.GetCurrentPhase(allFuture)
	if cur == nil || cur.Phase != logic.PhaseSeed {
		t.Fatalf("all-future should use first phase, got %v", cur)
	}
}

func TestGetFastMatureLandsThreshold(t *testing.T) {
	logic.SyncServerTime(1_700_000_000_000)
	now := logic.GetServerTimeSec()
	left := int64(2)
	zero := int64(0)
	lands := []logic.LandInfo{
		{ // within 300s
			ID: 1, Unlocked: true,
			Plant: &logic.PlantInfo{
				ID: 1, LeftInorcFertTimes: &left,
				Phases: []logic.PlantPhaseInfo{
					{Phase: logic.PhaseBlooming, BeginTime: now - 10},
					{Phase: logic.PhaseMature, BeginTime: now + 120},
				},
			},
		},
		{ // beyond threshold
			ID: 2, Unlocked: true,
			Plant: &logic.PlantInfo{
				ID: 2, LeftInorcFertTimes: &left,
				Phases: []logic.PlantPhaseInfo{
					{Phase: logic.PhaseBlooming, BeginTime: now - 10},
					{Phase: logic.PhaseMature, BeginTime: now + 900},
				},
			},
		},
		{ // already mature
			ID: 3, Unlocked: true,
			Plant: &logic.PlantInfo{
				ID: 3,
				Phases: []logic.PlantPhaseInfo{
					{Phase: logic.PhaseMature, BeginTime: now - 1},
				},
			},
		},
		{ // no fert times left
			ID: 4, Unlocked: true,
			Plant: &logic.PlantInfo{
				ID: 4, LeftInorcFertTimes: &zero,
				Phases: []logic.PlantPhaseInfo{
					{Phase: logic.PhaseBlooming, BeginTime: now - 10},
					{Phase: logic.PhaseMature, BeginTime: now + 60},
				},
			},
		},
	}
	got := logic.GetFastMatureLands(lands, 300)
	if len(got) != 1 || got[0] != 1 {
		t.Fatalf("fast mature=%v want [1]", got)
	}
}

func TestAnalyzeLandsNeedWaterWeedBug(t *testing.T) {
	logic.SyncServerTime(1_700_000_000_000)
	now := logic.GetServerTimeSec()
	lands := []logic.LandInfo{
		{
			ID: 1, Unlocked: true,
			Plant: &logic.PlantInfo{
				ID: 1, DryNum: 1,
				Phases: []logic.PlantPhaseInfo{
					{Phase: logic.PhaseSmallLeaves, BeginTime: now - 20, DryTime: 0, WeedsTime: 0, InsectTime: 0},
					{Phase: logic.PhaseMature, BeginTime: now + 400},
				},
			},
		},
		{
			ID: 2, Unlocked: true,
			Plant: &logic.PlantInfo{
				ID: 2, WeedOwners: []int64{1}, InsectOwners: []int64{2},
				Phases: []logic.PlantPhaseInfo{
					{Phase: logic.PhaseLargeLeaves, BeginTime: now - 20},
					{Phase: logic.PhaseMature, BeginTime: now + 400},
				},
			},
		},
		{
			ID: 3, Unlocked: true,
			Plant: &logic.PlantInfo{
				ID: 3,
				Phases: []logic.PlantPhaseInfo{
					{
						Phase: logic.PhaseBlooming, BeginTime: now - 20,
						DryTime: now - 5, WeedsTime: now - 3, InsectTime: now - 1,
					},
					{Phase: logic.PhaseMature, BeginTime: now + 400},
				},
			},
		},
	}
	result := logic.AnalyzeLands(lands)
	if len(result.NeedWater) != 2 {
		t.Fatalf("needWater=%v", result.NeedWater)
	}
	if len(result.NeedWeed) != 2 {
		t.Fatalf("needWeed=%v", result.NeedWeed)
	}
	if len(result.NeedBug) != 2 {
		t.Fatalf("needBug=%v", result.NeedBug)
	}
	if len(result.Growing) != 3 {
		t.Fatalf("growing=%v", result.Growing)
	}
}

func TestSortBagSeedsForPlanting(t *testing.T) {
	seeds := []logic.BagSeed{
		{SeedID: 3, RequiredLevel: 1, Count: 1, PlantSize: 1},
		{SeedID: 1, RequiredLevel: 5, Count: 1, PlantSize: 1},
		{SeedID: 2, RequiredLevel: 9, Count: 1, PlantSize: 1},
	}
	sorted := logic.SortBagSeedsForPlanting(seeds, []int64{2, 1})
	if sorted[0].SeedID != 2 || sorted[1].SeedID != 1 || sorted[2].SeedID != 3 {
		t.Fatalf("order=%v", []int64{sorted[0].SeedID, sorted[1].SeedID, sorted[2].SeedID})
	}
}

func TestDefaultAccountConfig(t *testing.T) {
	cfg := logic.DefaultAccountConfig()
	if cfg.Automation.Fertilizer != logic.FertilizerSmart {
		t.Fatalf("fertilizer=%s", cfg.Automation.Fertilizer)
	}
	if len(cfg.Automation.FertilizerLandTypes) != 5 {
		t.Fatalf("land types=%v", cfg.Automation.FertilizerLandTypes)
	}
}

func TestAnalyzeFriendHelpLandsIgnoresDueTimes(t *testing.T) {
	logic.SyncServerTime(1_700_000_000_000)
	now := logic.GetServerTimeSec()
	lands := []logic.LandInfo{
		{
			ID: 1, Unlocked: true,
			Plant: &logic.PlantInfo{
				ID: 1, DryNum: 1,
				Phases: []logic.PlantPhaseInfo{
					{Phase: logic.PhaseSmallLeaves, BeginTime: now - 20},
					{Phase: logic.PhaseMature, BeginTime: now + 400},
				},
			},
		},
		{
			ID: 2, Unlocked: true,
			Plant: &logic.PlantInfo{
				ID: 2, WeedOwners: []int64{9}, InsectOwners: []int64{8},
				Phases: []logic.PlantPhaseInfo{
					{Phase: logic.PhaseLargeLeaves, BeginTime: now - 20},
					{Phase: logic.PhaseMature, BeginTime: now + 400},
				},
			},
		},
		{
			// Due times only — own-farm AnalyzeLands would flag these; friend help must not.
			ID: 3, Unlocked: true,
			Plant: &logic.PlantInfo{
				ID: 3,
				Phases: []logic.PlantPhaseInfo{
					{
						Phase: logic.PhaseBlooming, BeginTime: now - 20,
						DryTime: now - 5, WeedsTime: now - 3, InsectTime: now - 1,
					},
					{Phase: logic.PhaseMature, BeginTime: now + 400},
				},
			},
		},
	}
	water, weed, bug := logic.AnalyzeFriendHelpLands(lands)
	if len(water) != 1 || water[0] != 1 {
		t.Fatalf("needWater=%v want [1]", water)
	}
	if len(weed) != 1 || weed[0] != 2 {
		t.Fatalf("needWeed=%v want [2]", weed)
	}
	if len(bug) != 1 || bug[0] != 2 {
		t.Fatalf("needBug=%v want [2]", bug)
	}
}

func TestLandsFromPlantPBLeftInorcAbsentWhenZero(t *testing.T) {
	lands := logic.LandsFromPlantPB([]*plantpb.LandInfo{
		{
			Id: 1, Unlocked: true,
			Plant: &plantpb.PlantInfo{
				Id:                 10,
				LeftInorcFertTimes: 0,
				Phases:             []*plantpb.PlantPhaseInfo{{Phase: int32(plantpb.PlantPhase_SEED), BeginTime: 1}},
			},
		},
		{
			Id: 2, Unlocked: true,
			Plant: &plantpb.PlantInfo{
				Id:                 11,
				LeftInorcFertTimes: 3,
				Phases:             []*plantpb.PlantPhaseInfo{{Phase: int32(plantpb.PlantPhase_SEED), BeginTime: 1}},
			},
		},
	})
	if len(lands) != 2 {
		t.Fatalf("lands=%d", len(lands))
	}
	if lands[0].Plant.LeftInorcFertTimes != nil {
		t.Fatalf("zero should map to nil, got %v", *lands[0].Plant.LeftInorcFertTimes)
	}
	if lands[1].Plant.LeftInorcFertTimes == nil || *lands[1].Plant.LeftInorcFertTimes != 3 {
		t.Fatalf("positive left_inorc=%v", lands[1].Plant.LeftInorcFertTimes)
	}
}
