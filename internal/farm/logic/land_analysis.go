package logic

import (
	"fmt"
	"sort"
	"strings"
)

// GetCurrentPhase returns the latest phase whose begin_time <= now (or first if all future).
func GetCurrentPhase(phases []PlantPhaseInfo) *PlantPhaseInfo {
	if len(phases) == 0 {
		return nil
	}
	nowSec := GetServerTimeSec()
	for i := len(phases) - 1; i >= 0; i-- {
		bt := ToTimeSec(phases[i].BeginTime)
		if bt > 0 && bt <= nowSec {
			return &phases[i]
		}
	}
	return &phases[0]
}

// GetOrganicFertilizerTargetsFromLands returns land IDs eligible for organic fertilizer.
func GetOrganicFertilizerTargetsFromLands(lands []LandInfo) []int64 {
	var targets []int64
	for i := range lands {
		land := &lands[i]
		if land == nil || !land.Unlocked {
			continue
		}
		landID := land.ID
		if landID == 0 {
			continue
		}
		plant := land.Plant
		if plant == nil || len(plant.Phases) == 0 {
			continue
		}
		current := GetCurrentPhase(plant.Phases)
		if current == nil || current.Phase == PhaseDead {
			continue
		}
		if plant.LeftInorcFertTimes != nil && *plant.LeftInorcFertTimes <= 0 {
			continue
		}
		targets = append(targets, landID)
	}
	return targets
}

// GetFastMatureLands returns growing lands that mature within thresholdSec.
func GetFastMatureLands(lands []LandInfo, thresholdSec int64) []int64 {
	if thresholdSec <= 0 {
		thresholdSec = 300
	}
	nowSec := GetServerTimeSec()
	var targets []int64
	for i := range lands {
		land := &lands[i]
		if !land.Unlocked || land.ID == 0 {
			continue
		}
		plant := land.Plant
		if plant == nil || len(plant.Phases) == 0 {
			continue
		}
		current := GetCurrentPhase(plant.Phases)
		if current == nil || current.Phase == PhaseDead || current.Phase == PhaseMature {
			continue
		}
		var mature *PlantPhaseInfo
		for j := range plant.Phases {
			if plant.Phases[j].Phase == PhaseMature {
				mature = &plant.Phases[j]
				break
			}
		}
		if mature == nil {
			continue
		}
		matureBegin := ToTimeSec(mature.BeginTime)
		if matureBegin <= 0 {
			continue
		}
		timeToMature := matureBegin - nowSec
		if timeToMature > thresholdSec || timeToMature < 0 {
			continue
		}
		if plant.LeftInorcFertTimes != nil && *plant.LeftInorcFertTimes <= 0 {
			continue
		}
		targets = append(targets, land.ID)
	}
	return targets
}

// GetSlaveLandIDs returns unique slave land ids.
func GetSlaveLandIDs(land *LandInfo) []int64 {
	if land == nil {
		return nil
	}
	seen := map[int64]struct{}{}
	var out []int64
	for _, id := range land.SlaveLandIDs {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// HasPlantData reports whether land has phase data.
func HasPlantData(land *LandInfo) bool {
	return land != nil && land.Plant != nil && len(land.Plant.Phases) > 0
}

// BuildLandMap indexes lands by id.
func BuildLandMap(lands []LandInfo) map[int64]*LandInfo {
	m := make(map[int64]*LandInfo, len(lands))
	for i := range lands {
		id := lands[i].ID
		if id > 0 {
			m[id] = &lands[i]
		}
	}
	return m
}

// GetLinkedMasterLand returns the master land when land is an occupied slave.
func GetLinkedMasterLand(land *LandInfo, landsMap map[int64]*LandInfo) *LandInfo {
	if land == nil {
		return nil
	}
	landID := land.ID
	masterLandID := land.MasterLandID
	if masterLandID == 0 || masterLandID == landID {
		return nil
	}
	master := landsMap[masterLandID]
	if master == nil {
		return nil
	}
	slaves := GetSlaveLandIDs(master)
	if len(slaves) > 0 {
		found := false
		for _, id := range slaves {
			if id == landID {
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}
	return master
}

// DisplayLandContext is the display source for multi-cell plants.
type DisplayLandContext struct {
	SourceLand       *LandInfo
	OccupiedByMaster bool
	MasterLandID     int64
	OccupiedLandIDs  []int64
}

// GetDisplayLandContext resolves display plant source for a land.
func GetDisplayLandContext(land *LandInfo, landsMap map[int64]*LandInfo) DisplayLandContext {
	master := GetLinkedMasterLand(land, landsMap)
	if master != nil && HasPlantData(master) {
		occupied := append([]int64{master.ID}, GetSlaveLandIDs(master)...)
		occupied = filterPositiveUnique(occupied)
		if len(occupied) == 0 {
			occupied = filterPositiveUnique([]int64{master.ID})
		}
		return DisplayLandContext{
			SourceLand:       master,
			OccupiedByMaster: true,
			MasterLandID:     master.ID,
			OccupiedLandIDs:  occupied,
		}
	}
	selfID := int64(0)
	if land != nil {
		selfID = land.ID
	}
	selfOccupied := filterPositiveUnique(append([]int64{selfID}, GetSlaveLandIDs(land)...))
	return DisplayLandContext{
		SourceLand:       land,
		OccupiedByMaster: false,
		MasterLandID:     selfID,
		OccupiedLandIDs:  uniqueInt64(selfOccupied),
	}
}

// IsOccupiedSlaveLand reports whether land is covered by a master plant.
func IsOccupiedSlaveLand(land *LandInfo, landsMap map[int64]*LandInfo) bool {
	return GetDisplayLandContext(land, landsMap).OccupiedByMaster
}

// BuildSlaveToMasterMap maps slave id → master id from slave_land_ids declarations.
func BuildSlaveToMasterMap(lands []LandInfo) map[int64]int64 {
	m := make(map[int64]int64)
	for i := range lands {
		slaves := GetSlaveLandIDs(&lands[i])
		masterID := lands[i].ID
		if len(slaves) == 0 || masterID <= 0 {
			continue
		}
		for _, slaveID := range slaves {
			if slaveID > 0 && slaveID != masterID {
				m[slaveID] = masterID
			}
		}
	}
	return m
}

// IsOccupiedSlaveLandWithMap uses a prebuilt slave→master map.
func IsOccupiedSlaveLandWithMap(land *LandInfo, _ map[int64]*LandInfo, slaveToMaster map[int64]int64) bool {
	if land == nil || land.ID == 0 {
		return false
	}
	_, ok := slaveToMaster[land.ID]
	return ok
}

// SummarizeLandDetails counts status flags on UI land rows.
func SummarizeLandDetails(lands []LandInfo) LandSummary {
	var s LandSummary
	for i := range lands {
		land := &lands[i]
		if !land.Unlocked {
			continue
		}
		switch land.Status {
		case "harvestable":
			s.Harvestable++
		case "dead":
			s.Dead++
		case "empty":
			s.Empty++
		case "growing", "stealable", "harvested":
			s.Growing++
		}
		if land.NeedWater {
			s.NeedWater++
		}
		if land.NeedWeed {
			s.NeedWeed++
		}
		if land.NeedBug {
			s.NeedBug++
		}
	}
	return s
}

// GetLandTypeByLevel maps land level to fertilizer land type key.
func GetLandTypeByLevel(level int64) string {
	switch {
	case level >= 5:
		return LandTypePurpleGold
	case level == 4:
		return LandTypeGold
	case level == 3:
		return LandTypeBlack
	case level == 2:
		return LandTypeRed
	default:
		return LandTypeNormal
	}
}

// NormalizeFertilizerLandTypes dedupes and validates land type keys.
func NormalizeFertilizerLandTypes(input []string) []string {
	source := input
	if source == nil {
		source = AllFertilizerLandTypes
	}
	allowed := map[string]struct{}{}
	for _, t := range AllFertilizerLandTypes {
		allowed[t] = struct{}{}
	}
	var result []string
	seen := map[string]struct{}{}
	for _, item := range source {
		value := strings.ToLower(strings.TrimSpace(item))
		if _, ok := allowed[value]; !ok {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

// FilterLandIDsByTypes keeps land IDs whose type is in selectedTypes.
func FilterLandIDsByTypes(landIDs []int64, landTypeByID map[int64]string, selectedTypes []string) []int64 {
	selected := map[string]struct{}{}
	for _, t := range NormalizeFertilizerLandTypes(selectedTypes) {
		selected[t] = struct{}{}
	}
	if len(selected) == 0 {
		return nil
	}
	if len(selected) == len(AllFertilizerLandTypes) {
		return append([]int64(nil), landIDs...)
	}
	var filtered []int64
	for _, id := range landIDs {
		t := landTypeByID[id]
		if t == "" {
			continue
		}
		if _, ok := selected[t]; ok {
			filtered = append(filtered, id)
		}
	}
	return filtered
}

// FormatFertilizerLandTypes returns Chinese labels.
func FormatFertilizerLandTypes(types []string) []string {
	norm := NormalizeFertilizerLandTypes(types)
	out := make([]string, 0, len(norm))
	for _, t := range norm {
		if label, ok := FertilizerLandTypeLabels[t]; ok {
			out = append(out, label)
		} else {
			out = append(out, t)
		}
	}
	return out
}

// AnalyzeLands classifies unlocked lands (ported from land-analysis.ts).
func AnalyzeLands(lands []LandInfo) LandAnalysis {
	result := LandAnalysis{}
	nowSec := GetServerTimeSec()
	landsMap := BuildLandMap(lands)

	for i := range lands {
		land := &lands[i]
		id := land.ID
		if !land.Unlocked {
			if land.CouldUnlock {
				result.Unlockable = append(result.Unlockable, id)
			}
			continue
		}
		if land.CouldUpgrade {
			result.Upgradable = append(result.Upgradable, id)
		}
		if IsOccupiedSlaveLand(land, landsMap) {
			continue
		}
		plant := land.Plant
		if plant == nil || len(plant.Phases) == 0 {
			result.Empty = append(result.Empty, id)
			continue
		}
		current := GetCurrentPhase(plant.Phases)
		if current == nil {
			result.Empty = append(result.Empty, id)
			continue
		}
		phaseVal := current.Phase
		if phaseVal == PhaseDead {
			result.Dead = append(result.Dead, id)
			continue
		}
		if phaseVal == PhaseMature {
			result.Harvestable = append(result.Harvestable, id)
			plantID := plant.ID
			name := GetPlantName(plantID)
			if name == "" || strings.HasPrefix(name, "植物") {
				if plant.Name != "" {
					name = plant.Name
				}
			}
			result.HarvestableInfo = append(result.HarvestableInfo, HarvestableInfo{
				LandID:  id,
				PlantID: plantID,
				Name:    name,
				Exp:     GetPlantExp(plantID),
			})
			continue
		}

		dryNum := plant.DryNum
		dryTime := ToTimeSec(current.DryTime)
		if dryNum > 0 || (dryTime > 0 && dryTime <= nowSec) {
			result.NeedWater = append(result.NeedWater, id)
		}
		weedsTime := ToTimeSec(current.WeedsTime)
		hasWeeds := len(plant.WeedOwners) > 0 || (weedsTime > 0 && weedsTime <= nowSec)
		if hasWeeds {
			result.NeedWeed = append(result.NeedWeed, id)
		}
		insectTime := ToTimeSec(current.InsectTime)
		hasBugs := len(plant.InsectOwners) > 0 || (insectTime > 0 && insectTime <= nowSec)
		if hasBugs {
			result.NeedBug = append(result.NeedBug, id)
		}
		result.Growing = append(result.Growing, id)
	}
	return result
}

// AnalyzeFriendHelpLands mirrors bot visit-strategy help filters: dry_num / weed_owners /
// insect_owners only (no dry_time / weeds_time / insect_time due checks).
func AnalyzeFriendHelpLands(lands []LandInfo) (needWater, needWeed, needBug []int64) {
	landsMap := BuildLandMap(lands)
	for i := range lands {
		land := &lands[i]
		if IsOccupiedSlaveLand(land, landsMap) {
			continue
		}
		plant := land.Plant
		if plant == nil || len(plant.Phases) == 0 {
			continue
		}
		current := GetCurrentPhase(plant.Phases)
		if current == nil {
			continue
		}
		phaseVal := current.Phase
		if phaseVal == PhaseMature || phaseVal == PhaseDead {
			continue
		}
		if plant.DryNum > 0 {
			needWater = append(needWater, land.ID)
		}
		if len(plant.WeedOwners) > 0 {
			needWeed = append(needWeed, land.ID)
		}
		if len(plant.InsectOwners) > 0 {
			needBug = append(needBug, land.ID)
		}
	}
	return needWater, needWeed, needBug
}

// SummarizeLandsPush builds a short LandsNotify summary string.
func SummarizeLandsPush(lands []LandInfo) string {
	if len(lands) == 0 {
		return "无土地变化"
	}
	nowSec := GetServerTimeSec()
	var parts []string
	for i := range lands {
		land := &lands[i]
		id := land.ID
		if id <= 0 {
			continue
		}
		plant := land.Plant
		if plant == nil || len(plant.Phases) == 0 {
			continue
		}
		current := GetCurrentPhase(plant.Phases)
		if current == nil {
			continue
		}
		phaseVal := current.Phase
		name := GetPlantName(plant.ID)
		if name == "" || strings.HasPrefix(name, "植物") {
			name = plant.Name
		}
		nameHint := ""
		if name != "" {
			nameHint = "(" + name + ")"
		}
		if phaseVal == PhaseMature {
			if plant.Stealable {
				parts = append(parts, fmt.Sprintf("可偷#%d%s", id, nameHint))
			} else {
				parts = append(parts, fmt.Sprintf("成熟可收#%d%s", id, nameHint))
			}
			continue
		}
		if phaseVal == PhaseDead {
			continue
		}
		var tags []string
		if plant.DryNum > 0 || (ToTimeSec(current.DryTime) > 0 && ToTimeSec(current.DryTime) <= nowSec) {
			tags = append(tags, "干旱")
		}
		weedsTime := ToTimeSec(current.WeedsTime)
		if len(plant.WeedOwners) > 0 || (weedsTime > 0 && weedsTime <= nowSec) {
			tags = append(tags, "有草")
		}
		insectTime := ToTimeSec(current.InsectTime)
		if len(plant.InsectOwners) > 0 || (insectTime > 0 && insectTime <= nowSec) {
			tags = append(tags, "有虫")
		}
		if len(tags) > 0 {
			parts = append(parts, fmt.Sprintf("%s#%d%s", strings.Join(tags, ""), id, nameHint))
		}
	}
	if len(parts) == 0 {
		return fmt.Sprintf("%d块状态更新", len(lands))
	}
	return fmt.Sprintf("%d块: %s", len(lands), strings.Join(parts, "; "))
}

// BuildPlantingLayouts enumerates NxN footprints on available lands.
func BuildPlantingLayouts(availableLandIDs []int64, plantSize int64) []PlantingLayout {
	size := plantSize
	if size < 1 {
		size = 1
	}
	ordered := uniquePositive(availableLandIDs)
	if size == 1 {
		out := make([]PlantingLayout, 0, len(ordered))
		for _, id := range ordered {
			out = append(out, PlantingLayout{AnchorLandID: id, LandIDs: []int64{id}})
		}
		return out
	}
	available := map[int64]struct{}{}
	for _, id := range ordered {
		available[id] = struct{}{}
	}
	var layouts []PlantingLayout
	seen := map[string]struct{}{}
	for _, anchorID := range ordered {
		anchor := GetLandConfigByID(anchorID)
		if anchor == nil {
			continue
		}
		footprint := make([]int64, 0, size*size)
		complete := true
		for y := int64(0); y < size && complete; y++ {
			for x := int64(0); x < size; x++ {
				land := GetLandConfigByCoordinate(anchor.GridX+int(x), anchor.GridY+int(y))
				if land == nil || land.ID == 0 {
					complete = false
					break
				}
				if _, ok := available[land.ID]; !ok {
					complete = false
					break
				}
				footprint = append(footprint, land.ID)
			}
		}
		if !complete {
			continue
		}
		keyIDs := append([]int64(nil), footprint...)
		sort.Slice(keyIDs, func(i, j int) bool { return keyIDs[i] < keyIDs[j] })
		key := joinIDs(keyIDs)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		layouts = append(layouts, PlantingLayout{AnchorLandID: anchorID, LandIDs: footprint})
	}
	return layouts
}

// SelectNonOverlappingLayouts picks a maximal non-overlapping subset (DFS, mirrors TS).
func SelectNonOverlappingLayouts(layouts []PlantingLayout, maxCount int64) []PlantingLayout {
	if maxCount <= 0 || len(layouts) == 0 {
		return nil
	}
	var best []PlantingLayout
	var visit func(index int, selected []PlantingLayout, occupied map[int64]struct{})
	visit = func(index int, selected []PlantingLayout, occupied map[int64]struct{}) {
		if len(selected) > len(best) {
			best = append([]PlantingLayout(nil), selected...)
		}
		if int64(len(selected)) >= maxCount || index >= len(layouts) {
			return
		}
		if len(selected)+len(layouts)-index <= len(best) {
			return
		}
		layout := layouts[index]
		ok := true
		for _, id := range layout.LandIDs {
			if _, hit := occupied[id]; hit {
				ok = false
				break
			}
		}
		if ok {
			nextOcc := cloneSet(occupied)
			for _, id := range layout.LandIDs {
				nextOcc[id] = struct{}{}
			}
			visit(index+1, append(selected, layout), nextOcc)
		}
		visit(index+1, selected, occupied)
	}
	visit(0, nil, map[int64]struct{}{})
	if int64(len(best)) > maxCount {
		best = best[:maxCount]
	}
	return best
}

// ResolveOccupiedLandIDs resolves master + occupied set after planting.
func ResolveOccupiedLandIDs(anchorLandID int64, lands []LandInfo) (masterLandID int64, occupiedLandIDs []int64) {
	landsMap := BuildLandMap(lands)
	slaveToMaster := BuildSlaveToMasterMap(lands)
	anchor := landsMap[anchorLandID]
	var declaredMaster int64
	if anchor != nil {
		declaredMaster = anchor.MasterLandID
	}
	masterLandID = declaredMaster
	if masterLandID == 0 {
		if m, ok := slaveToMaster[anchorLandID]; ok {
			masterLandID = m
		} else {
			masterLandID = anchorLandID
		}
	}
	master := landsMap[masterLandID]
	if master == nil {
		master = anchor
	}
	occupied := map[int64]struct{}{}
	if masterLandID != 0 {
		occupied[masterLandID] = struct{}{}
	}
	for _, id := range GetSlaveLandIDs(master) {
		occupied[id] = struct{}{}
	}
	for i := range lands {
		landID := lands[i].ID
		if landID != 0 && lands[i].MasterLandID == masterLandID {
			occupied[landID] = struct{}{}
		}
		if landID == masterLandID {
			for _, id := range GetSlaveLandIDs(&lands[i]) {
				occupied[id] = struct{}{}
			}
		}
	}
	if len(occupied) == 0 && anchorLandID != 0 {
		occupied[anchorLandID] = struct{}{}
	}
	if masterLandID == 0 {
		masterLandID = anchorLandID
	}
	return masterLandID, setToSlice(occupied)
}

// GetLandLifecycleState returns empty/dead/growing/unknown.
func GetLandLifecycleState(land *LandInfo) string {
	if land == nil {
		return "unknown"
	}
	plant := land.Plant
	if plant == nil || len(plant.Phases) == 0 {
		return "empty"
	}
	current := GetCurrentPhase(plant.Phases)
	if current == nil {
		return "empty"
	}
	phaseVal := current.Phase
	switch {
	case phaseVal == PhaseDead:
		return "dead"
	case phaseVal == PhaseUnknown:
		return "empty"
	case phaseVal >= PhaseSeed && phaseVal <= PhaseMature:
		return "growing"
	default:
		return "unknown"
	}
}

// HarvestClassify is the result of ClassifyHarvestedLandsByMap.
type HarvestClassify struct {
	Removable []int64
	Growing   []int64
	Unknown   []int64
}

// ClassifyHarvestedLandsByMap classifies harvested land IDs by lifecycle.
func ClassifyHarvestedLandsByMap(landIDs []int64, landsMap map[int64]*LandInfo) HarvestClassify {
	var out HarvestClassify
	for _, id := range landIDs {
		land := landsMap[id]
		if land == nil {
			out.Unknown = append(out.Unknown, id)
			continue
		}
		switch GetLandLifecycleState(land) {
		case "dead", "empty":
			out.Removable = append(out.Removable, id)
		case "growing":
			out.Growing = append(out.Growing, id)
		default:
			out.Unknown = append(out.Unknown, id)
		}
	}
	return out
}

// helpers

func filterPositiveUnique(ids []int64) []int64 {
	return uniquePositive(ids)
}

func uniquePositive(ids []int64) []int64 {
	seen := map[int64]struct{}{}
	var out []int64
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func uniqueInt64(ids []int64) []int64 {
	seen := map[int64]struct{}{}
	var out []int64
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func joinIDs(ids []int64) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = fmt.Sprintf("%d", id)
	}
	return strings.Join(parts, ",")
}

func cloneSet(in map[int64]struct{}) map[int64]struct{} {
	out := make(map[int64]struct{}, len(in))
	for k := range in {
		out[k] = struct{}{}
	}
	return out
}

func setToSlice(in map[int64]struct{}) []int64 {
	out := make([]int64, 0, len(in))
	for k := range in {
		out = append(out, k)
	}
	return out
}
