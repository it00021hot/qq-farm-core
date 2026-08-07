// Package guidepb provides hand-written guide service protobuf types.
// Source: qq-farm-bot/core/src/proto/guidepb.proto
package guidepb

import "github.com/MQEnergy/go-skeleton/internal/farm/proto/corepb"

// GuideNode is one guide/tutorial node.
type GuideNode struct {
	NodeID        int64
	Name          string
	Completed     bool
	RewardClaimed bool
	Rewards       []*corepb.Item
}

// SetWeakGuideNodeCompleteRequest marks a guide node complete.
type SetWeakGuideNodeCompleteRequest struct {
	NodeID int64
}

// SetWeakGuideNodeCompleteReply is the complete result.
type SetWeakGuideNodeCompleteReply struct {
	Success bool
}

// ClaimWeakGuideRewardRequest claims a guide reward.
type ClaimWeakGuideRewardRequest struct {
	NodeID int64
}

// ClaimWeakGuideRewardReply is the claim result.
type ClaimWeakGuideRewardReply struct {
	Items   []*corepb.Item
	Success bool
}
