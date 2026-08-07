package logic

import (
	"math"
	"sort"
	"strconv"
	"strings"
)

// PlantRanking is one seed efficiency row (offline analytics).
type PlantRanking struct {
	ID                            int64   `json:"id"`
	SeedID                        int64   `json:"seedId"`
	Name                          string  `json:"name"`
	Seasons                       int64   `json:"seasons"`
	Level                         *int64  `json:"level"`
	GrowTime                      int64   `json:"growTime"`
	GrowTimeStr                   string  `json:"growTimeStr"`
	ReduceSec                     int64   `json:"reduceSec"`
	ReduceSecApplied              int64   `json:"reduceSecApplied"`
	ExpPerHour                    float64 `json:"expPerHour"`
	NormalFertilizerExpPerHour    float64 `json:"normalFertilizerExpPerHour"`
	GoldPerHour                   float64 `json:"goldPerHour"`
	ProfitPerHour                 float64 `json:"profitPerHour"`
	NormalFertilizerProfitPerHour float64 `json:"normalFertilizerProfitPerHour"`
	Income                        int64   `json:"income"`
	NetProfit                     int64   `json:"netProfit"`
	FruitID                       int64   `json:"fruitId"`
	FruitCount                    int64   `json:"fruitCount"`
	FruitPrice                    int64   `json:"fruitPrice"`
	SeedPrice                     int64   `json:"seedPrice"`
	Image                         string  `json:"image"`
}

// GetPlantRankings returns sorted seed efficiency rankings from loaded game config.
// sortBy: exp, fert_exp (alias fert), profit, fert_profit, gold, level.
func GetPlantRankings(sortBy string) []PlantRanking {
	sortBy = strings.TrimSpace(sortBy)
	if sortBy == "" {
		sortBy = "exp"
	}

	plants := GlobalGameConfig.allPlantsForRanking()
	results := make([]PlantRanking, 0, len(plants))
	for _, plant := range plants {
		if plant.SeedID == nil || *plant.SeedID <= 0 || plant.GrowPhases == "" {
			continue
		}
		baseGrowTime := growTimeFromPhases(plant.GrowPhases)
		if baseGrowTime <= 0 {
			continue
		}
		seasons := plant.Seasons
		if seasons <= 0 {
			seasons = 1
		}
		isTwoSeason := seasons == 2
		growTime := baseGrowTime
		if isTwoSeason {
			growTime = int64(float64(baseGrowTime) * 1.5)
		}

		harvestExpBase := plant.Exp
		harvestExp := harvestExpBase
		if isTwoSeason {
			harvestExp = harvestExpBase * 2
		}
		expPerHour := (float64(harvestExp) / float64(growTime)) * 3600

		reduceSecBase := parseNormalFertilizerReduceSec(plant.GrowPhases)
		reduceSecApplied := reduceSecBase
		if isTwoSeason {
			reduceSecApplied = reduceSecBase * 2
		}
		fertilizedGrowTime := growTime - reduceSecApplied
		safeFertilizedTime := fertilizedGrowTime
		if safeFertilizedTime <= 0 {
			safeFertilizedTime = 1
		}
		normalFertilizerExpPerHour := (float64(harvestExp) / float64(safeFertilizedTime)) * 3600

		fruitID := plant.Fruit.ID
		fruitCount := plant.Fruit.Count
		fruitPrice := GlobalGameConfig.GetFruitPrice(fruitID)
		seedPrice := GlobalGameConfig.GetSeedPrice(*plant.SeedID)

		seasonMul := int64(1)
		if isTwoSeason {
			seasonMul = 2
		}
		income := fruitCount * fruitPrice * seasonMul
		netProfit := income - seedPrice
		goldPerHour := (float64(income) / float64(growTime)) * 3600
		profitPerHour := (float64(netProfit) / float64(growTime)) * 3600
		normalFertilizerProfitPerHour := (float64(netProfit) / float64(safeFertilizedTime)) * 3600

		var level *int64
		if seedItem := GlobalGameConfig.GetItemByID(*plant.SeedID); seedItem != nil && seedItem.Level != nil && *seedItem.Level > 0 {
			lv := *seedItem.Level
			level = &lv
		} else if plant.LandLevelNeed > 0 {
			lv := plant.LandLevelNeed
			level = &lv
		}

		results = append(results, PlantRanking{
			ID:                            plant.ID,
			SeedID:                        *plant.SeedID,
			Name:                          plant.Name,
			Seasons:                       seasons,
			Level:                         level,
			GrowTime:                      growTime,
			GrowTimeStr:                   FormatGrowTime(growTime),
			ReduceSec:                     reduceSecBase,
			ReduceSecApplied:              reduceSecApplied,
			ExpPerHour:                    round2(expPerHour),
			NormalFertilizerExpPerHour:    round2(normalFertilizerExpPerHour),
			GoldPerHour:                   round2(goldPerHour),
			ProfitPerHour:                 round2(profitPerHour),
			NormalFertilizerProfitPerHour: round2(normalFertilizerProfitPerHour),
			Income:                        income,
			NetProfit:                     netProfit,
			FruitID:                       fruitID,
			FruitCount:                    fruitCount,
			FruitPrice:                    fruitPrice,
			SeedPrice:                     seedPrice,
			Image:                         SeedImagePath(*plant.SeedID),
		})
	}

	sortPlantRankings(results, sortBy)
	return results
}

func (g *GameConfig) allPlantsForRanking() []*PlantItem {
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := make([]*PlantItem, 0, len(g.plantByID))
	for _, p := range g.plantByID {
		out = append(out, p)
	}
	return out
}

func parseNormalFertilizerReduceSec(growPhases string) int64 {
	parts := strings.Split(growPhases, ";")
	var first string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			first = part
			break
		}
	}
	if first == "" {
		return 0
	}
	m := growPhaseRe.FindStringSubmatch(first)
	if len(m) != 2 {
		return 0
	}
	n, _ := strconv.ParseInt(m[1], 10, 64)
	return n
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func sortPlantRankings(results []PlantRanking, sortBy string) {
	switch sortBy {
	case "fert", "fert_exp":
		sort.Slice(results, func(i, j int) bool {
			return results[i].NormalFertilizerExpPerHour > results[j].NormalFertilizerExpPerHour
		})
	case "gold":
		sort.Slice(results, func(i, j int) bool {
			return results[i].GoldPerHour > results[j].GoldPerHour
		})
	case "profit":
		sort.Slice(results, func(i, j int) bool {
			return results[i].ProfitPerHour > results[j].ProfitPerHour
		})
	case "fert_profit":
		sort.Slice(results, func(i, j int) bool {
			return results[i].NormalFertilizerProfitPerHour > results[j].NormalFertilizerProfitPerHour
		})
	case "level":
		sort.Slice(results, func(i, j int) bool {
			return levelValue(results[i].Level) > levelValue(results[j].Level)
		})
	default:
		sort.Slice(results, func(i, j int) bool {
			return results[i].ExpPerHour > results[j].ExpPerHour
		})
	}
}

func levelValue(level *int64) int64 {
	if level == nil {
		return -1
	}
	return *level
}
