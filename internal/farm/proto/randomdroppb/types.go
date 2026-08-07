// Package randomdroppb provides hand-written random drop protobuf types.
// Source: qq-farm-bot/core/src/proto/randomdroppb.proto
package randomdroppb

// DropReward is one drop reward entry.
type DropReward struct {
	ItemID      int64
	Count       int64
	Probability int32
	Claimed     bool
}

// DropActivityInfo describes a drop activity.
type DropActivityInfo struct {
	ActivityID   int64
	Name         string
	Status       int32
	BeginTime    int64
	EndTime      int64
	Rewards      []*DropReward
	DropCount    int32
	MaxDropCount int32
}

// GetActivityInfoRequest fetches drop activities.
type GetActivityInfoRequest struct{}

// GetActivityInfoReply lists drop activities.
type GetActivityInfoReply struct {
	Activities []*DropActivityInfo
}
