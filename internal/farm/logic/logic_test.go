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
		{ // 服务端从不显式下发 left_inorc_fert_times=0（额度 0 时省略），
			// 不过滤即与 bot hasOwn 行为 wire 等价（rust f347147）。
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
	if len(got) != 2 || got[0] != 1 || got[1] != 4 {
		t.Fatalf("fast mature=%v want [1 4]", got)
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

// rust f347147: 有机肥目标不按 left_inorc_fert_times 过滤——官方向量证实
// 服务端额度=0 时省略字段、从不显式发 0，不过滤与 bot hasOwn 行为 wire 等价。
func TestOrganicTargetsIgnoreLeftInorcQuotaField(t *testing.T) {
	logic.SyncServerTime(1_700_000_000_000)
	now := logic.GetServerTimeSec()
	zero := int64(0)
	lands := []logic.LandInfo{
		{ // 额度字段带 0（真实报文不会出现）也要当可施目标
			ID: 1, Unlocked: true,
			Plant: &logic.PlantInfo{
				ID: 1, LeftInorcFertTimes: &zero,
				Phases: []logic.PlantPhaseInfo{
					{Phase: logic.PhaseBlooming, BeginTime: now - 10},
				},
			},
		},
		{ // 成熟地不排除（rust/bot 只排除枯死）
			ID: 2, Unlocked: true,
			Plant: &logic.PlantInfo{
				ID: 2,
				Phases: []logic.PlantPhaseInfo{
					{Phase: logic.PhaseMature, BeginTime: now - 1},
				},
			},
		},
		{ // 枯死地不可施
			ID: 3, Unlocked: true,
			Plant: &logic.PlantInfo{
				ID: 3,
				Phases: []logic.PlantPhaseInfo{
					{Phase: logic.PhaseDead, BeginTime: now - 1},
				},
			},
		},
	}
	got := logic.GetOrganicFertilizerTargetsFromLands(lands)
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("organic targets=%v want [1 2]", got)
	}
}

// rust f347147: 普通肥目标 = 未成熟 且 本季任一阶段 ferts_used 不含 1011。
func TestNormalFertilizerTargetsUseFertsUsed(t *testing.T) {
	logic.SyncServerTime(1_700_000_000_000)
	now := logic.GetServerTimeSec()
	lands := []logic.LandInfo{
		{ // 已施过普通肥
			ID: 1, Unlocked: true,
			Plant: &logic.PlantInfo{
				ID: 1,
				Phases: []logic.PlantPhaseInfo{
					{Phase: logic.PhaseBlooming, BeginTime: now - 10, FertsUsed: map[int64]int64{logic.NormalContainerID: 1}},
				},
			},
		},
		{ // 未施过
			ID: 2, Unlocked: true,
			Plant: &logic.PlantInfo{
				ID: 2,
				Phases: []logic.PlantPhaseInfo{
					{Phase: logic.PhaseBlooming, BeginTime: now - 10},
				},
			},
		},
		{ // 已成熟
			ID: 3, Unlocked: true,
			Plant: &logic.PlantInfo{
				ID: 3,
				Phases: []logic.PlantPhaseInfo{
					{Phase: logic.PhaseMature, BeginTime: now - 1},
				},
			},
		},
	}
	got := logic.GetNormalFertilizerTargetsFromLands(lands)
	if len(got) != 1 || got[0] != 2 {
		t.Fatalf("normal targets=%v want [2]", got)
	}
}

// rust 4fe322f: 新账号默认对齐账号 1。
func TestDefaultAccountConfigAlignedWithRust(t *testing.T) {
	cfg := logic.DefaultAccountConfig()
	if cfg.PlantingStrategy != logic.StrategyBagPriority {
		t.Fatalf("planting strategy=%s want bag_priority", cfg.PlantingStrategy)
	}
	if cfg.Intervals.StealMin != 60 || cfg.Intervals.StealMax != 90 {
		t.Fatalf("steal interval=%d-%d want 60-90", cfg.Intervals.StealMin, cfg.Intervals.StealMax)
	}
	if !cfg.FriendQuietHours.Enabled || cfg.FriendQuietHours.End != "08:30" {
		t.Fatalf("quiet hours=%+v want enabled 01:00-08:30", cfg.FriendQuietHours)
	}
	wantPriority := []int64{29003, 20129, 21380, 20108, 26032}
	if len(cfg.BagSeedPriority) != len(wantPriority) {
		t.Fatalf("bag seed priority=%v want %v", cfg.BagSeedPriority, wantPriority)
	}
	for i, id := range wantPriority {
		if cfg.BagSeedPriority[i] != id {
			t.Fatalf("bag seed priority=%v want %v", cfg.BagSeedPriority, wantPriority)
		}
	}
	if cfg.BagSeedFallbackStrategy != logic.StrategyPreferred {
		t.Fatalf("bag fallback=%s want preferred", cfg.BagSeedFallbackStrategy)
	}
	if cfg.Automation.FertilizerSmartSeconds != 360 {
		t.Fatalf("smart seconds=%d want 360", cfg.Automation.FertilizerSmartSeconds)
	}
}

// rust ee8ac0e: 分析页排除白萝卜种子 29999。
func TestPlantRankingsExcludeRadishSeed(t *testing.T) {
	rankings := logic.GetPlantRankings("exp")
	for _, row := range rankings {
		if row.SeedID == 29999 {
			t.Fatalf("rankings should exclude radish seed 29999")
		}
	}
}

func TestHasOwnerCleanableInteraction(t *testing.T) {
	if logic.HasOwnerCleanableInteraction(nil) {
		t.Fatal("nil plant should not need cleanup")
	}
	for _, itemID := range logic.OwnerCleanableInteractionItemIDs {
		plant := &logic.PlantInfo{InteractionItemIDs: []int64{itemID}}
		if !logic.HasOwnerCleanableInteraction(plant) {
			t.Fatalf("item %d should be owner-cleanable", itemID)
		}
	}
	if logic.HasOwnerCleanableInteraction(&logic.PlantInfo{InteractionItemIDs: []int64{301103}}) {
		t.Fatal("301103 (七夕灵露) is not owner-cleanable")
	}
	if logic.HasOwnerCleanableInteraction(&logic.PlantInfo{}) {
		t.Fatal("plant without interactions should not need cleanup")
	}
}

func TestAnalyzeLandsNeedInteraction(t *testing.T) {
	logic.SyncServerTime(1_700_000_000_000)
	now := logic.GetServerTimeSec()
	lands := []logic.LandInfo{
		{
			// growing land with a golden worm → cleanup target
			ID: 1, Unlocked: true,
			Plant: &logic.PlantInfo{
				ID:                 1,
				InteractionItemIDs: []int64{301101},
				Phases:             []logic.PlantPhaseInfo{{Phase: logic.PhaseSmallLeaves, BeginTime: now - 20}},
			},
		},
		{
			// dead land with a cloud (5006) → still counted (rust pushes before phase checks)
			ID: 2, Unlocked: true,
			Plant: &logic.PlantInfo{
				ID:                 2,
				InteractionItemIDs: []int64{5006},
				Phases:             []logic.PlantPhaseInfo{{Phase: logic.PhaseDead, BeginTime: now - 100}},
			},
		},
		{
			// growing land with a non-cleanable item (301103) → not a target
			ID: 3, Unlocked: true,
			Plant: &logic.PlantInfo{
				ID:                 3,
				InteractionItemIDs: []int64{301103},
				Phases:             []logic.PlantPhaseInfo{{Phase: logic.PhaseSmallLeaves, BeginTime: now - 20}},
			},
		},
		{
			// occupied slave land is skipped entirely
			ID: 4, Unlocked: true, MasterLandID: 5,
			Plant: &logic.PlantInfo{
				ID:                 4,
				InteractionItemIDs: []int64{301102},
				Phases:             []logic.PlantPhaseInfo{{Phase: logic.PhaseSmallLeaves, BeginTime: now - 20}},
			},
		},
		{
			ID: 5, Unlocked: true, SlaveLandIDs: []int64{4},
			Plant: &logic.PlantInfo{
				ID:     5,
				Phases: []logic.PlantPhaseInfo{{Phase: logic.PhaseSmallLeaves, BeginTime: now - 20}},
			},
		},
	}
	result := logic.AnalyzeLands(lands)
	if len(result.NeedInteraction) != 2 || result.NeedInteraction[0] != 1 || result.NeedInteraction[1] != 2 {
		t.Fatalf("needInteraction=%v", result.NeedInteraction)
	}
	if len(result.Dead) != 1 || result.Dead[0] != 2 {
		t.Fatalf("dead=%v", result.Dead)
	}
}
