// Package illustratedpb provides hand-written illustrated service protobuf types.
// Source: qq-farm-bot/core/src/proto/illustratedpb.proto
package illustratedpb

import "github.com/MQEnergy/go-skeleton/internal/farm/proto/corepb"

// GetIllustratedListV2Request fetches the illustrated handbook list.
type GetIllustratedListV2Request struct {
	Refresh bool
	Full    bool
}

// IllustratedItem is one handbook entry.
type IllustratedItem struct {
	SeedID       int64
	Unlocked     bool
	Planted      bool
	PlantedCount int32
	HarvestCount int32
	RewardDetail []byte
	HasReward    bool
}

// GetIllustratedListV2Reply lists handbook items.
type GetIllustratedListV2Reply struct {
	Items []IllustratedItem
}

// ClaimAllRewardsV2Request claims all illustrated rewards.
type ClaimAllRewardsV2Request struct {
	OnlyClaimable bool
}

// ClaimAllRewardsV2Reply is the claim result.
type ClaimAllRewardsV2Reply struct {
	Items      []*corepb.Item
	BonusItems []*corepb.Item
}

// IllustratedRewardRedDotNotifyV2 is an empty reward red-dot push.
type IllustratedRewardRedDotNotifyV2 struct{}

// ClearNewUnlockedFruitsV2Request clears the new-unlock marker.
type ClearNewUnlockedFruitsV2Request struct {
	SeedID int64
}

// ClearNewUnlockedFruitsV2Reply is the clear result.
type ClearNewUnlockedFruitsV2Reply struct{}
