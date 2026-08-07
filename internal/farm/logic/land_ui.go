package logic

import (
	"math"
	"strings"
)

// UILandRow is one land card for the personal farm panel.
type UILandRow struct {
	ID               int64   `json:"id"`
	Unlocked         bool    `json:"unlocked"`
	Status           string  `json:"status"`
	PlantName        string  `json:"plantName"`
	SeedID           int64   `json:"seedId,omitempty"`
	SeedImage        string  `json:"seedImage,omitempty"`
	PhaseName        string  `json:"phaseName,omitempty"`
	CurrentSeason    int64   `json:"currentSeason,omitempty"`
	TotalSeason      int64   `json:"totalSeason,omitempty"`
	MatureInSec      int64   `json:"matureInSec,omitempty"`
	TotalGrowTime    int64   `json:"totalGrowTime,omitempty"`
	NeedWater        bool    `json:"needWater,omitempty"`
	NeedWeed         bool    `json:"needWeed,omitempty"`
	NeedBug          bool    `json:"needBug,omitempty"`
	Stealable        bool    `json:"stealable,omitempty"`
	Level            int64   `json:"level"`
	MaxLevel         int64   `json:"maxLevel"`
	LandsLevel       int64   `json:"landsLevel"`
	LandSize         int64   `json:"landSize"`
	CouldUnlock      bool    `json:"couldUnlock,omitempty"`
	CouldUpgrade     bool    `json:"couldUpgrade,omitempty"`
	MasterLandID     int64   `json:"masterLandId,omitempty"`
	PlantSize        int64   `json:"plantSize,omitempty"`
	OccupiedByMaster bool    `json:"occupiedByMaster,omitempty"`
	OccupiedLandIDs  []int64 `json:"occupiedLandIds,omitempty"`
}

// LandsUIResponse is returned by GET /farm/lands.
type LandsUIResponse struct {
	Lands   []UILandRow `json:"lands"`
	Summary LandSummary `json:"summary"`
}

// ResolvePlantDisplayName picks the UI plant name / catalog id.
// Mutated crops often keep the base plant.id and even plant.name ("艾草") while
// mutant_config_ids / phase mutants / fruit_id point at 黄金·艾草 (1121135 / fruit 1041135).
func ResolvePlantDisplayName(plant *PlantInfo) (displayPlantID int64, name string) {
	if plant == nil {
		return 0, "未知"
	}
	displayPlantID = plant.ID

	// 1) Explicit mutant config ids (plant field 20 + phase mutants merged upstream).
	for _, mid := range plant.MutantConfigIDs {
		if mid <= 0 {
			continue
		}
		if p := GlobalGameConfig.GetPlantByID(mid); p != nil && strings.TrimSpace(p.Name) != "" {
			return mid, p.Name
		}
	}

	// 2) Fruit id → mutant plant / 变异果实 (type 17) name. Base 艾草 fruit=41135;
	// golden 黄金·艾草 fruit=1041135 while plant.id may still be 1021135.
	if plant.FruitID > 0 {
		if fp := GlobalGameConfig.GetPlantByFruitID(plant.FruitID); fp != nil && strings.TrimSpace(fp.Name) != "" {
			if fp.ID != plant.ID || looksLikeMutantName(fp.Name) {
				return fp.ID, fp.Name
			}
		}
		if item := GlobalGameConfig.GetItemByID(plant.FruitID); item != nil {
			n := strings.TrimSpace(item.Name)
			if n != "" && (item.Type == 17 || looksLikeMutantName(n)) {
				if fp := GlobalGameConfig.GetPlantByFruitID(plant.FruitID); fp != nil && strings.TrimSpace(fp.Name) != "" {
					return fp.ID, fp.Name
				}
				return plant.ID, n
			}
		}
	}

	// 3) Runtime name only when it is more specific than the base catalog name.
	runtimeName := strings.TrimSpace(plant.Name)
	baseName := ""
	if plant.ID > 0 {
		if p := GlobalGameConfig.GetPlantByID(plant.ID); p != nil {
			baseName = strings.TrimSpace(p.Name)
		}
	}
	if runtimeName != "" && (looksLikeMutantName(runtimeName) || (baseName != "" && runtimeName != baseName)) {
		return displayPlantID, runtimeName
	}
	if baseName != "" {
		return plant.ID, baseName
	}
	if runtimeName != "" {
		return displayPlantID, runtimeName
	}
	if plant.ID > 0 {
		return plant.ID, GetPlantName(plant.ID)
	}
	return 0, "未知"
}

func looksLikeMutantName(name string) bool {
	n := strings.TrimSpace(name)
	if n == "" {
		return false
	}
	return strings.Contains(n, "黄金") || strings.Contains(n, "·") || strings.Contains(n, "变异")
}

// ResolvePlantSeedImage returns an icon path for the plant (mutants fall back to base seed).
func ResolvePlantSeedImage(displayPlantID int64) string {
	if displayPlantID <= 0 {
		return ""
	}
	if cfg := GlobalGameConfig.GetPlantByID(displayPlantID); cfg != nil && cfg.SeedID != nil && *cfg.SeedID > 0 {
		return SeedImagePath(*cfg.SeedID)
	}
	// Golden mutants are typically 11xxxxx while base plants are 10xxxxx.
	if displayPlantID >= 1_100_000 && displayPlantID < 1_200_000 {
		baseID := displayPlantID - 100_000
		if cfg := GlobalGameConfig.GetPlantByID(baseID); cfg != nil && cfg.SeedID != nil && *cfg.SeedID > 0 {
			return SeedImagePath(*cfg.SeedID)
		}
	}
	return ""
}

func resolveUIPlantSize(land *LandInfo, plantCfg *PlantItem, occupiedLandIDs []int64) int64 {
	plantSize := int64(1)
	if plantCfg != nil && plantCfg.Size != nil && *plantCfg.Size > 0 {
		plantSize = *plantCfg.Size
	}
	if land != nil && land.LandSize > plantSize {
		plantSize = land.LandSize
	}
	n := len(occupiedLandIDs)
	if n > 1 {
		side := int64(math.Sqrt(float64(n)) + 0.5)
		if side > 1 && side*side == int64(n) && side > plantSize {
			plantSize = side
		}
	}
	if plantSize < 1 {
		plantSize = 1
	}
	return plantSize
}

// FormatLandsResponse maps raw session lands into UI rows plus summary counts.
// Aligns with bot getLandsDetail: plant display fields come from GetDisplayLandContext.SourceLand
// so multi-cell slave plots inherit the master's plant instead of showing as empty.
func FormatLandsResponse(lands []LandInfo) LandsUIResponse {
	nowSec := GetServerTimeSec()
	landsMap := BuildLandMap(lands)
	out := make([]UILandRow, 0, len(lands))

	for i := range lands {
		land := &lands[i]
		id := land.ID
		level := land.Level
		maxLevel := land.MaxLevel
		landsLevel := land.LandsLevel
		landSize := land.LandSize
		couldUnlock := land.CouldUnlock
		couldUpgrade := land.CouldUpgrade

		ctx := GetDisplayLandContext(land, landsMap)
		occupiedByMaster := ctx.OccupiedByMaster
		masterLandID := ctx.MasterLandID
		occupiedLandIDs := ctx.OccupiedLandIDs

		if !land.Unlocked {
			out = append(out, UILandRow{
				ID: id, Unlocked: false, Status: "locked", PlantName: "-", PhaseName: "未解锁",
				Level: level, MaxLevel: maxLevel, LandsLevel: landsLevel, LandSize: landSize,
				CouldUnlock: couldUnlock, CouldUpgrade: couldUpgrade,
				OccupiedByMaster: false, MasterLandID: 0, PlantSize: 1,
			})
			continue
		}

		var plant *PlantInfo
		if ctx.SourceLand != nil {
			plant = ctx.SourceLand.Plant
		}
		if plant == nil || len(plant.Phases) == 0 {
			out = append(out, UILandRow{
				ID: id, Unlocked: true, Status: "empty", PlantName: "", PhaseName: "空地",
				Level: level, MaxLevel: maxLevel, LandsLevel: landsLevel, LandSize: landSize,
				CouldUnlock: couldUnlock, CouldUpgrade: couldUpgrade,
				MasterLandID: masterLandID, OccupiedByMaster: occupiedByMaster,
				OccupiedLandIDs: occupiedLandIDs, PlantSize: 1,
			})
			continue
		}

		current := GetCurrentPhase(plant.Phases)
		if current == nil {
			out = append(out, UILandRow{
				ID: id, Unlocked: true, Status: "empty", PlantName: "", PhaseName: "",
				Level: level, MaxLevel: maxLevel, LandsLevel: landsLevel, LandSize: landSize,
				CouldUnlock: couldUnlock, CouldUpgrade: couldUpgrade,
				MasterLandID: masterLandID, OccupiedByMaster: occupiedByMaster,
				OccupiedLandIDs: occupiedLandIDs, PlantSize: 1,
			})
			continue
		}

		phaseVal := current.Phase
		displayPlantID, plantName := ResolvePlantDisplayName(plant)
		plantID := plant.ID
		if displayPlantID > 0 {
			plantID = displayPlantID
		}
		if plantName == "" {
			plantName = "未知"
		}

		plantCfg := GlobalGameConfig.GetPlantByID(plantID)
		if plantCfg == nil && plant.ID > 0 && plant.ID != plantID {
			plantCfg = GlobalGameConfig.GetPlantByID(plant.ID)
		}
		seedImage := ResolvePlantSeedImage(plantID)
		if seedImage == "" && plant.ID > 0 && plant.ID != plantID {
			seedImage = ResolvePlantSeedImage(plant.ID)
		}
		var seedID int64
		if plantCfg != nil && plantCfg.SeedID != nil {
			seedID = *plantCfg.SeedID
		}
		if seedID == 0 && plant.ID > 0 {
			if baseCfg := GlobalGameConfig.GetPlantByID(plant.ID); baseCfg != nil && baseCfg.SeedID != nil {
				seedID = *baseCfg.SeedID
			}
		}

		plantSize := resolveUIPlantSize(land, plantCfg, occupiedLandIDs)
		if plantCfg == nil || plantCfg.Size == nil {
			if baseCfg := GlobalGameConfig.GetPlantByID(plant.ID); baseCfg != nil {
				plantSize = resolveUIPlantSize(land, baseCfg, occupiedLandIDs)
			}
		}
		totalSeason := int64(1)
		if plantCfg != nil && plantCfg.Seasons > 0 {
			totalSeason = plantCfg.Seasons
		} else if baseCfg := GlobalGameConfig.GetPlantByID(plant.ID); baseCfg != nil && baseCfg.Seasons > 0 {
			totalSeason = baseCfg.Seasons
		}

		currentSeason := plant.Season
		if currentSeason <= 0 {
			currentSeason = 1
		}
		if currentSeason > totalSeason {
			currentSeason = totalSeason
		}

		phaseName := ""
		if phaseVal >= 0 && phaseVal < len(PhaseNames) {
			phaseName = PhaseNames[phaseVal]
		}

		var matureBegin int64
		for _, ph := range plant.Phases {
			if ph.Phase == PhaseMature {
				matureBegin = ToTimeSec(ph.BeginTime)
				break
			}
		}
		// Bot getLandsDetail: matureInSec = matureBegin > nowSec ? matureBegin - nowSec : 0
		matureInSec := MatureInSec(matureBegin, nowSec)
		totalGrowTime := GetPlantGrowTime(plantID)

		// Status matches bot getLandsDetail (stealable is a flag, not a status override).
		landStatus := "growing"
		switch phaseVal {
		case PhaseMature:
			landStatus = "harvestable"
		case PhaseDead:
			landStatus = "dead"
		case PhaseUnknown:
			landStatus = "empty"
		}

		needWater := plant.DryNum > 0 || (ToTimeSec(current.DryTime) > 0 && ToTimeSec(current.DryTime) <= nowSec)
		needWeed := len(plant.WeedOwners) > 0 || (ToTimeSec(current.WeedsTime) > 0 && ToTimeSec(current.WeedsTime) <= nowSec)
		needBug := len(plant.InsectOwners) > 0 || (ToTimeSec(current.InsectTime) > 0 && ToTimeSec(current.InsectTime) <= nowSec)

		out = append(out, UILandRow{
			ID: id, Unlocked: true, Status: landStatus, PlantName: plantName,
			SeedID: seedID, SeedImage: seedImage, PhaseName: phaseName,
			CurrentSeason: currentSeason, TotalSeason: totalSeason,
			MatureInSec: matureInSec, TotalGrowTime: totalGrowTime,
			NeedWater: needWater, NeedWeed: needWeed, NeedBug: needBug,
			Stealable: plant.Stealable, Level: level, MaxLevel: maxLevel,
			LandsLevel: landsLevel, LandSize: landSize,
			CouldUnlock: couldUnlock, CouldUpgrade: couldUpgrade,
			MasterLandID: masterLandID, PlantSize: plantSize,
			OccupiedByMaster: occupiedByMaster, OccupiedLandIDs: occupiedLandIDs,
		})
	}

	summaryRows := make([]LandInfo, len(out))
	for i, row := range out {
		summaryRows[i] = LandInfo{
			ID: row.ID, Unlocked: row.Unlocked, Level: row.Level,
			Status: row.Status, NeedWater: row.NeedWater, NeedWeed: row.NeedWeed, NeedBug: row.NeedBug,
		}
	}

	return LandsUIResponse{Lands: out, Summary: SummarizeLandDetails(summaryRows)}
}

// FormatFriendLandsResponse maps lands for a friend farm panel.
// Aligns with bot getFriendLandsDetail: status is "stealable" only when phase is MATURE
// and plant.stealable; immature plots never advertise 可偷 (proto stealable alone is insufficient).
func FormatFriendLandsResponse(lands []LandInfo) LandsUIResponse {
	res := FormatLandsResponse(lands)
	for i := range res.Lands {
		row := &res.Lands[i]
		switch row.Status {
		case "harvestable":
			if row.Stealable {
				row.Status = "stealable"
			} else {
				row.Status = "harvested"
				row.Stealable = false
			}
		case "growing", "dead", "empty", "locked":
			// Bot LandCard only shows 可偷 when status==='stealable'; clear premature flag.
			row.Stealable = false
		}
	}
	summaryRows := make([]LandInfo, len(res.Lands))
	for i, row := range res.Lands {
		summaryRows[i] = LandInfo{
			ID: row.ID, Unlocked: row.Unlocked, Level: row.Level,
			Status: row.Status, NeedWater: row.NeedWater, NeedWeed: row.NeedWeed, NeedBug: row.NeedBug,
		}
	}
	res.Summary = SummarizeLandDetails(summaryRows)
	return res
}

// DecrementMatureInSec lowers countdown timers by one second (client-side tick helper).
func DecrementMatureInSec(rows []UILandRow) {
	for i := range rows {
		if rows[i].MatureInSec > 0 {
			rows[i].MatureInSec = int64(math.Max(0, float64(rows[i].MatureInSec-1)))
		}
	}
}
