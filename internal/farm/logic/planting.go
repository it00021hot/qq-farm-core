package logic

import (
	"math"
	"sort"
)

// PlantingStrategyLabels mirrors planting.ts PLANTING_STRATEGY_LABELS.
var PlantingStrategyLabels = map[string]string{
	StrategyPreferred:     "优先种植种子",
	StrategyLevel:         "最高等级作物",
	StrategyMaxExp:        "最大经验时",
	StrategyMaxFertExp:    "最大普通肥经验/时",
	StrategyMaxProfit:     "最大净利润/时",
	StrategyMaxFertProfit: "最大普通肥净利润/时",
	StrategyBagPriority:   "背包种子优先",
}

// GetPlantingStrategyLabel returns the Chinese label for a strategy key.
func GetPlantingStrategyLabel(strategy string) string {
	if label, ok := PlantingStrategyLabels[strategy]; ok {
		return label
	}
	return strategy
}

// GetPlantSizeBySeedID returns plant size (min 1) from game config.
func GetPlantSizeBySeedID(seedID int64) int64 {
	cfg := GetPlantBySeedID(seedID)
	if cfg == nil || cfg.Size == nil || *cfg.Size < 1 {
		return 1
	}
	return *cfg.Size
}

// ConfirmsPlantedFootprint checks that occupied set covers expected IDs and master has plant.
func ConfirmsPlantedFootprint(expectedLandIDs []int64, masterLandID int64, occupiedLandIDs []int64, lands []LandInfo) bool {
	occ := map[int64]struct{}{}
	for _, id := range occupiedLandIDs {
		occ[id] = struct{}{}
	}
	for _, id := range expectedLandIDs {
		if _, ok := occ[id]; !ok {
			return false
		}
	}
	landsMap := BuildLandMap(lands)
	master := landsMap[masterLandID]
	return master != nil && master.Plant != nil
}

// SortBagSeedsForPlanting sorts bag seeds by priority list, then level desc, then seedId.
func SortBagSeedsForPlanting(bagSeeds []BagSeed, priorityList []int64) []BagSeed {
	indexMap := map[int64]int{}
	for i, seedID := range priorityList {
		if seedID > 0 {
			indexMap[seedID] = i
		}
	}
	out := append([]BagSeed(nil), bagSeeds...)
	sort.SliceStable(out, func(i, j int) bool {
		ai, aok := indexMap[out[i].SeedID]
		bi, bok := indexMap[out[j].SeedID]
		if !aok {
			ai = math.MaxInt32
		}
		if !bok {
			bi = math.MaxInt32
		}
		if ai != bi {
			return ai < bi
		}
		if out[i].RequiredLevel != out[j].RequiredLevel {
			return out[i].RequiredLevel > out[j].RequiredLevel
		}
		return out[i].SeedID < out[j].SeedID
	})
	return out
}

// FilterUsableBagSeeds drops zero-count / invalid-size seeds (level-lock still kept).
func FilterUsableBagSeeds(seeds []BagSeed) (usable []BagSeed, skipped []BagSeed, levelLocked []BagSeed, stateLevel int64) {
	// stateLevel is filled by caller via seed annotation; kept for API parity.
	for _, seed := range seeds {
		if seed.Count <= 0 {
			skipped = append(skipped, seed)
			continue
		}
		if seed.PlantSize < 1 {
			skipped = append(skipped, seed)
			continue
		}
		usable = append(usable, seed)
	}
	return usable, skipped, levelLocked, stateLevel
}

// RankingRow is a plant analytics ranking row (seedId + optional level gate).
type RankingRow struct {
	SeedID int64
	Level  float64 // NaN / non-finite → ignore level gate
}

// SortSeedCandidatesByStrategy sorts shop candidates (pure; no network).
// analyticsRankings is used for max_exp / max_fert_* strategies; may be nil.
func SortSeedCandidatesByStrategy(
	available []SeedCandidate,
	strategy string,
	preferredSeedID int64,
	playerLevel int64,
	analyticsRankings []RankingRow,
) []SeedCandidate {
	out := append([]SeedCandidate(nil), available...)
	byLevelAndID := func(a, b SeedCandidate) bool {
		if a.RequiredLevel != b.RequiredLevel {
			return a.RequiredLevel > b.RequiredLevel
		}
		return a.SeedID < b.SeedID
	}

	analyticsSortBy := map[string]bool{
		StrategyMaxExp:        true,
		StrategyMaxFertExp:    true,
		StrategyMaxProfit:     true,
		StrategyMaxFertProfit: true,
	}
	if analyticsSortBy[strategy] && len(analyticsRankings) > 0 {
		rankBySeed := map[int64]int{}
		for i, row := range analyticsRankings {
			if row.SeedID <= 0 {
				continue
			}
			if !math.IsNaN(row.Level) && !math.IsInf(row.Level, 0) && row.Level > float64(playerLevel) {
				continue
			}
			if _, exists := rankBySeed[row.SeedID]; !exists {
				rankBySeed[row.SeedID] = i
			}
		}
		sort.SliceStable(out, func(i, j int) bool {
			ai, aok := rankBySeed[out[i].SeedID]
			bi, bok := rankBySeed[out[j].SeedID]
			if !aok {
				ai = math.MaxInt32
			}
			if !bok {
				bi = math.MaxInt32
			}
			if ai != bi {
				return ai < bi
			}
			return byLevelAndID(out[i], out[j])
		})
	} else {
		sort.SliceStable(out, func(i, j int) bool {
			return byLevelAndID(out[i], out[j])
		})
	}

	if strategy == StrategyPreferred && preferredSeedID > 0 {
		for i, c := range out {
			if c.SeedID == preferredSeedID {
				item := out[i]
				out = append(out[:i], out[i+1:]...)
				out = append([]SeedCandidate{item}, out...)
				break
			}
		}
	}
	return out
}

// PlanBagPlantingLayouts selects non-overlapping layouts for one bag seed against remaining lands.
func PlanBagPlantingLayouts(remainingLandIDs []int64, plantSize, seedCount int64) (all, selected []PlantingLayout) {
	all = BuildPlantingLayouts(remainingLandIDs, plantSize)
	selected = SelectNonOverlappingLayouts(all, seedCount)
	return all, selected
}

// ComputeShopPurchaseUnits mirrors plantFromShop purchase math (pure).
func ComputeShopPurchaseLayouts(
	candidate SeedCandidate,
	remainingLandIDs []int64,
	gold int64,
) (layouts []PlantingLayout, purchaseUnits, needCount int64, plantSize int64) {
	plantSize = GetPlantSizeBySeedID(candidate.SeedID)
	allLayouts := BuildPlantingLayouts(remainingLandIDs, plantSize)
	layouts = SelectNonOverlappingLayouts(allLayouts, int64(len(allLayouts)))
	if len(layouts) == 0 {
		return nil, 0, 0, plantSize
	}
	unitItemCount := candidate.UnitItemCount
	if unitItemCount < 1 {
		unitItemCount = 1
	}
	requiredSeedCount := int64(len(layouts))
	requiredPurchaseUnits := (requiredSeedCount + unitItemCount - 1) / unitItemCount
	maxPurchaseUnits := requiredPurchaseUnits
	if !math.IsInf(candidate.MaxPurchaseCount, 1) {
		maxPurchaseUnits = int64(candidate.MaxPurchaseCount)
	}
	affordable := int64(0)
	if candidate.Price > 0 {
		affordable = gold / candidate.Price
	}
	purchaseUnits = requiredPurchaseUnits
	if maxPurchaseUnits < purchaseUnits {
		purchaseUnits = maxPurchaseUnits
	}
	if affordable < purchaseUnits {
		purchaseUnits = affordable
	}
	if purchaseUnits <= 0 {
		return nil, 0, 0, plantSize
	}
	needCount = requiredSeedCount
	if capped := purchaseUnits * unitItemCount; capped < needCount {
		needCount = capped
	}
	if int64(len(layouts)) > needCount {
		layouts = layouts[:needCount]
	}
	return layouts, purchaseUnits, needCount, plantSize
}

// RemovableHarvestResult is the pure classify result (network refresh is caller's job).
type RemovableHarvestResult struct {
	Removable       []int64
	Growing         []int64
	FallbackRemoved int
}

// ResolveRemovableHarvestedLandsPure classifies from harvest reply lands;
// unknown IDs are treated as removable (legacy compatibility) when no refresh map is provided.
func ResolveRemovableHarvestedLandsPure(harvestedLandIDs []int64, harvestReplyLands []LandInfo, refreshedLands []LandInfo) RemovableHarvestResult {
	ids := uniquePositive(harvestedLandIDs)
	if len(ids) == 0 {
		return RemovableHarvestResult{}
	}
	replyMap := BuildLandMap(harvestReplyLands)
	first := ClassifyHarvestedLandsByMap(ids, replyMap)
	removable := append([]int64(nil), first.Removable...)
	growing := append([]int64(nil), first.Growing...)
	unknown := append([]int64(nil), first.Unknown...)
	fallback := 0

	if len(unknown) > 0 && len(refreshedLands) > 0 {
		latestMap := BuildLandMap(refreshedLands)
		second := ClassifyHarvestedLandsByMap(unknown, latestMap)
		removable = append(removable, second.Removable...)
		growing = append(growing, second.Growing...)
		unknown = second.Unknown
	}
	if len(unknown) > 0 {
		removable = append(removable, unknown...)
		fallback = len(unknown)
	}
	return RemovableHarvestResult{
		Removable:       uniquePositive(removable),
		Growing:         uniquePositive(growing),
		FallbackRemoved: fallback,
	}
}
