// Package logic ports pure farm analysis and planting strategy from the Node core.
package logic

import "sort"

// PlantPhase matches plantpb / config.PlantPhase numeric values.
const (
	PhaseUnknown     = 0
	PhaseSeed        = 1
	PhaseGermination = 2
	PhaseSmallLeaves = 3
	PhaseLargeLeaves = 4
	PhaseBlooming    = 5
	PhaseMature      = 6
	PhaseDead        = 7
)

// PhaseNames is the Chinese display name table (index = phase).
var PhaseNames = []string{"未知", "种子", "发芽", "小叶", "大叶", "开花", "成熟", "枯死"}

// PlantPhaseInfo is one growth phase on a plant.
type PlantPhaseInfo struct {
	Phase      int   `json:"phase"`
	BeginTime  int64 `json:"begin_time"`
	DryTime    int64 `json:"dry_time"`
	WeedsTime  int64 `json:"weeds_time"`
	InsectTime int64 `json:"insect_time"`
}

// PlantActivityInfo is activity-score metadata from PlantInfo.field_36.
type PlantActivityInfo struct {
	ActivityID int64 `json:"activity_id"`
	Param1     int64 `json:"param1"`
	Param2     int64 `json:"param2"`
	Date       int64 `json:"date"`
}

// PlantInfo is the plant state on a land (fields used by analysis).
type PlantInfo struct {
	ID                 int64              `json:"id"`
	Name               string             `json:"name"`
	Phases             []PlantPhaseInfo   `json:"phases"`
	Season             int64              `json:"season"`
	DryNum             int64              `json:"dry_num"`
	FruitID            int64              `json:"fruit_id"`
	FruitNum           int64              `json:"fruit_num"`
	WeedOwners         []int64            `json:"weed_owners"`
	InsectOwners       []int64            `json:"insect_owners"`
	Stealers           []byte             `json:"stealers,omitempty"`
	Stealable          bool               `json:"stealable"`
	LeftInorcFertTimes *int64             `json:"left_inorc_fert_times,omitempty"`
	LeftFruitNum       int64              `json:"left_fruit_num"`
	MutantConfigIDs    []int64            `json:"mutant_config_ids,omitempty"`
	Activity           *PlantActivityInfo `json:"activity,omitempty"`
}

// LandInfo is one farmland plot.
type LandInfo struct {
	ID           int64      `json:"id"`
	Unlocked     bool       `json:"unlocked"`
	Level        int64      `json:"level"`
	MaxLevel     int64      `json:"max_level"`
	CouldUnlock  bool       `json:"could_unlock"`
	CouldUpgrade bool       `json:"could_upgrade"`
	Plant        *PlantInfo `json:"plant"`
	MasterLandID int64      `json:"master_land_id"`
	SlaveLandIDs []int64    `json:"slave_land_ids"`
	LandSize     int64      `json:"land_size"`
	LandsLevel   int64      `json:"lands_level"`
	Status       string     `json:"status,omitempty"` // UI summary fields
	NeedWater    bool       `json:"needWater,omitempty"`
	NeedWeed     bool       `json:"needWeed,omitempty"`
	NeedBug      bool       `json:"needBug,omitempty"`
}

// HarvestableInfo is attached to AnalyzeLands result.
type HarvestableInfo struct {
	LandID  int64  `json:"landId"`
	PlantID int64  `json:"plantId"`
	Name    string `json:"name"`
	Exp     int64  `json:"exp"`
}

// LandAnalysis is the result of AnalyzeLands.
type LandAnalysis struct {
	Harvestable     []int64           `json:"harvestable"`
	NeedWater       []int64           `json:"needWater"`
	NeedWeed        []int64           `json:"needWeed"`
	NeedBug         []int64           `json:"needBug"`
	Growing         []int64           `json:"growing"`
	Empty           []int64           `json:"empty"`
	Dead            []int64           `json:"dead"`
	Unlockable      []int64           `json:"unlockable"`
	Upgradable      []int64           `json:"upgradable"`
	HarvestableInfo []HarvestableInfo `json:"harvestableInfo"`
}

// LandSummary counts for UI.
type LandSummary struct {
	Harvestable int `json:"harvestable"`
	Growing     int `json:"growing"`
	Empty       int `json:"empty"`
	Dead        int `json:"dead"`
	NeedWater   int `json:"needWater"`
	NeedWeed    int `json:"needWeed"`
	NeedBug     int `json:"needBug"`
}

// PlantingLayout is a multi-cell plant footprint.
type PlantingLayout struct {
	AnchorLandID int64   `json:"anchorLandId"`
	LandIDs      []int64 `json:"landIds"`
}

// BagSeed is a seed entry used by bag-priority strategy (pure sorting).
type BagSeed struct {
	SeedID        int64  `json:"seedId"`
	Name          string `json:"name"`
	Count         int64  `json:"count"`
	RequiredLevel int64  `json:"requiredLevel"`
	PlantSize     int64  `json:"plantSize"`
}

// SeedCandidate is a shop seed candidate after filtering (pure ranking helpers).
type SeedCandidate struct {
	GoodsID          int64
	SeedID           int64
	Price            int64
	RequiredLevel    int64
	UnitItemCount    int64
	MaxPurchaseCount float64 // +Inf when unlimited
}

// AvailableShopSeed is the account-scoped shop seed row for settings preview (bot /api/seeds).
type AvailableShopSeed struct {
	SeedID        int64  `json:"seedId"`
	GoodsID       int64  `json:"goodsId"`
	Name          string `json:"name"`
	Price         *int64 `json:"price"`
	RequiredLevel *int64 `json:"requiredLevel"`
	Size          int64  `json:"size"`
	Locked        bool   `json:"locked"`
	SoldOut       bool   `json:"soldOut"`
	UnknownMeta   bool   `json:"unknownMeta,omitempty"`
}

// CatalogAvailableSeeds builds settings seed options from local game config (offline-safe).
func CatalogAvailableSeeds(playerLevel int64) []AvailableShopSeed {
	seeds := GetAllSeeds()
	list := make([]AvailableShopSeed, 0, len(seeds))
	for _, seed := range seeds {
		reqLv := seed.RequiredLevel
		price := seed.Price
		list = append(list, AvailableShopSeed{
			SeedID:        seed.SeedID,
			GoodsID:       0,
			Name:          seed.Name,
			Price:         &price,
			RequiredLevel: &reqLv,
			Size:          seed.Size,
			Locked:        playerLevel > 0 && seed.RequiredLevel > playerLevel,
			SoldOut:       false,
		})
	}
	sort.SliceStable(list, func(i, j int) bool {
		av, bv := int64(9999), int64(9999)
		if list[i].RequiredLevel != nil {
			av = *list[i].RequiredLevel
		}
		if list[j].RequiredLevel != nil {
			bv = *list[j].RequiredLevel
		}
		if av != bv {
			return av < bv
		}
		return list[i].SeedID < list[j].SeedID
	})
	return list
}
