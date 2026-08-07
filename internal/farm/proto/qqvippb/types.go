// Package qqvippb provides hand-written QQ VIP service protobuf types.
// Source: qq-farm-bot/core/src/proto/qqvippb.proto
package qqvippb

import "github.com/MQEnergy/go-skeleton/internal/farm/proto/corepb"

// GetDailyGiftStatusRequest fetches VIP gift status.
type GetDailyGiftStatusRequest struct{}

// GetDailyGiftStatusReply reports VIP gift availability.
type GetDailyGiftStatusReply struct {
	CanClaim bool
	HasGift  bool
}

// ClaimDailyGiftRequest claims the VIP daily gift.
type ClaimDailyGiftRequest struct{}

// ClaimDailyGiftReply is the claim result.
type ClaimDailyGiftReply struct {
	Items []*corepb.Item
}

// VipInfoUpdatedNTF is a server push when VIP info updates.
type VipInfoUpdatedNTF struct {
	VipLevel int64
	CanClaim bool
	HasGift  bool
}
