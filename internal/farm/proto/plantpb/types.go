// Package plantpb provides hand-written plant service protobuf types (protoc unavailable).
// Source: internal/farm/proto/plantpb.proto
package plantpb

import "github.com/MQEnergy/go-skeleton/internal/farm/proto/corepb"

// PlantPhaseInfo is one growth phase on a plant.
type PlantPhaseInfo struct {
	Phase      int32
	BeginTime  int64
	PhaseID    int64
	DryTime    int64
	WeedsTime  int64
	InsectTime int64
	Mutants    []MutantInfo
}

// MutantInfo is plantpb.MutantInfo on a growth phase.
type MutantInfo struct {
	MutantTime     int64
	MutantConfigID int64
	WeatherID      int64
}

// PlantActivityInfo is plant activity score metadata (proto field 36).
type PlantActivityInfo struct {
	ActivityID int64
	Param1     int64
	Param2     int64
	Date       int64
}

// PlantInfo is plant state on a land plot.
type PlantInfo struct {
	ID                 int64
	Name               string
	Phases             []PlantPhaseInfo
	Season             int64
	DryNum             int64
	StoleNum           int64
	FruitID            int64
	FruitNum           int64
	WeedOwners         []int64
	InsectOwners       []int64
	Stealers           []byte
	GrowSec            int64
	Stealable          bool
	LeftInorcFertTimes *int64
	LeftFruitNum       int64
	StealIntimacyLevel int64
	MutantConfigIDs    []int64
	IsNudged           bool
	StealPlayer        int64
	StealNum           []byte
	Field24            int64
	Field25            int64
	Field26            int64
	Field27            int64
	Field32            []byte
	Limit              *PlantLimitInfo
	Activity           *PlantActivityInfo
	Field37            int64
}

// PlantLimitInfo is plantpb.PlantLimitInfo.
type PlantLimitInfo struct {
	ConfigID int64
	Limit    int64
	Value    int64
}

// StealPlayer is plantpb.StealPlayer (legacy message form).
type StealPlayer struct {
	GID int64
	Num int64
}

// ItemChange is plantpb.ItemChange for PutSocialItem rewards/consumed.
type ItemChange struct {
	LandID int64
	ItemID int64
	Count  int64
}

// PutSocialItemRequest is gamepb.plantpb.PutSocialItemRequest.
type PutSocialItemRequest struct {
	HostGID int64
	LandID  int64
	ItemID  int64
}

// PutSocialItemReply is gamepb.plantpb.PutSocialItemReply.
type PutSocialItemReply struct {
	Land           *LandInfo
	OperationLimit *OperationLimit
	Rewards        []*ItemChange
	Consumed       []*ItemChange
}

// LandInfo is one farmland plot.
type LandInfo struct {
	ID           int64
	Unlocked     bool
	Level        int64
	MaxLevel     int64
	CouldUnlock  bool
	CouldUpgrade bool
	Plant        *PlantInfo
	MasterLandID int64
	SlaveLandIDs []int64
	LandSize     int64
	LandsLevel   int64
}

// OperationLimit is daily operation quota info.
type OperationLimit struct {
	ID            int64
	DayTimes      int64
	DayTimesLt    int64
	DayShareID    int64
	DayExpTimes   int64
	DayExpTimesLt int64
	DayExpShareID int64
}

// PlantItem is one seed-to-lands planting group.
type PlantItem struct {
	SeedID    int64
	LandIDs   []int64
	AutoSlave bool
}

// FarmingResult is one land reward from FarmingReply.
type FarmingResult struct {
	LandID int64
	Reward *corepb.Item
}

// AllLandsRequest is gamepb.plantpb.AllLandsRequest.
type AllLandsRequest struct {
	HostGID int64
}

// HarvestRequest is gamepb.plantpb.HarvestRequest.
type HarvestRequest struct {
	LandIDs []int64
	HostGID int64
	IsAll   bool
}

// WaterLandRequest is gamepb.plantpb.WaterLandRequest.
type WaterLandRequest struct {
	LandIDs []int64
	HostGID int64
}

// FarmingRequest is gamepb.plantpb.FarmingRequest.
type FarmingRequest struct {
	LandIDs []int64
	HostGID int64
	Field3  int32
	Field4  int32
}

// FertilizeRequest is gamepb.plantpb.FertilizeRequest.
type FertilizeRequest struct {
	LandIDs      []int64
	FertilizerID int64
}

// PlantRequest is gamepb.plantpb.PlantRequest.
type PlantRequest struct {
	Items []PlantItem
}

// RemovePlantRequest is gamepb.plantpb.RemovePlantRequest.
type RemovePlantRequest struct {
	LandIDs []int64
}

// UnlockLandRequest is gamepb.plantpb.UnlockLandRequest.
type UnlockLandRequest struct {
	LandID   int64
	DoShared bool
}

// UpgradeLandRequest is gamepb.plantpb.UpgradeLandRequest.
type UpgradeLandRequest struct {
	LandID int64
}

// AllLandsReply is gamepb.plantpb.AllLandsReply.
type AllLandsReply struct {
	Lands           []*LandInfo
	OperationLimits []*OperationLimit
}

// HarvestReply is gamepb.plantpb.HarvestReply.
type HarvestReply struct {
	Land            []*LandInfo
	Items           []*corepb.Item
	LostItems       []*corepb.Item
	OperationLimits []*OperationLimit
}

// WaterLandReply is gamepb.plantpb.WaterLandReply.
type WaterLandReply struct {
	Land            []*LandInfo
	OperationLimits []*OperationLimit
}

// FarmingReply is gamepb.plantpb.FarmingReply.
type FarmingReply struct {
	Land            []*LandInfo
	OperationLimits []*OperationLimit
	Results         []*FarmingResult
}

// FertilizeReply is gamepb.plantpb.FertilizeReply.
type FertilizeReply struct {
	Land            []*LandInfo
	OperationLimits []*OperationLimit
	Fertilizer      int64
}

// PlantReply is gamepb.plantpb.PlantReply.
type PlantReply struct {
	Land            []*LandInfo
	OperationLimits []*OperationLimit
}

// RemovePlantReply is gamepb.plantpb.RemovePlantReply.
type RemovePlantReply struct {
	Land            []*LandInfo
	OperationLimits []*OperationLimit
}

// UnlockLandReply is gamepb.plantpb.UnlockLandReply.
type UnlockLandReply struct {
	Land *LandInfo
}

// UpgradeLandReply is gamepb.plantpb.UpgradeLandReply.
type UpgradeLandReply struct {
	Land *LandInfo
}

// LandsNotify is gamepb.plantpb.LandsNotify.
type LandsNotify struct {
	Lands   []*LandInfo
	HostGID int64
}

// PutInsectsRequest is gamepb.plantpb.PutInsectsRequest.
type PutInsectsRequest struct {
	HostGID int64
	LandIDs []int64
}

// PutInsectsReply is gamepb.plantpb.PutInsectsReply.
type PutInsectsReply struct {
	Land            []*LandInfo
	OperationLimits []*OperationLimit
}

// PutWeedsRequest is gamepb.plantpb.PutWeedsRequest.
type PutWeedsRequest struct {
	HostGID int64
	LandIDs []int64
}

// PutWeedsReply is gamepb.plantpb.PutWeedsReply.
type PutWeedsReply struct {
	Land            []*LandInfo
	OperationLimits []*OperationLimit
}

// CheckCanOperateRequest is gamepb.plantpb.CheckCanOperateRequest.
type CheckCanOperateRequest struct {
	HostGID     int64
	OperationID int64
}

// CheckCanOperateReply is gamepb.plantpb.CheckCanOperateReply.
type CheckCanOperateReply struct {
	CanOperate  bool
	CanStealNum int64
}
