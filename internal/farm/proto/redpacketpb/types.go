// Package redpacketpb provides hand-written red packet protobuf types.
// Source: qq-farm-bot/core/src/proto/redpacketpb.proto
package redpacketpb

import "github.com/MQEnergy/go-skeleton/internal/farm/proto/corepb"

// GetTodayClaimStatusRequest fetches today's claim status.
type GetTodayClaimStatusRequest struct{}

// RedPacketInfo is one red packet claim entry.
type RedPacketInfo struct {
	ID       int32
	CanClaim bool
}

// GetTodayClaimStatusReply lists red packet claim status.
type GetTodayClaimStatusReply struct {
	Infos []*RedPacketInfo
}

// ClaimRedPacketRequest claims a red packet.
type ClaimRedPacketRequest struct {
	ID int32
}

// ClaimRedPacketReply is the claim result.
type ClaimRedPacketReply struct {
	Status int32
	Item   *corepb.Item
}
