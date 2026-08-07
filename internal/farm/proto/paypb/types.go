// Package paypb provides hand-written pay service protobuf types.
// Source: qq-farm-bot/core/src/proto/paypb.proto
package paypb

// GetRechargeInfoRequest fetches recharge info.
type GetRechargeInfoRequest struct {
	Platform int64
	Version  int64
}

// GetRechargeInfoReply is an empty recharge info reply.
type GetRechargeInfoReply struct{}

// RechargeInfoNotify notifies recharge status.
type RechargeInfoNotify struct {
	Status int64
}
