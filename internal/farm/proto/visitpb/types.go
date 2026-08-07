// Package visitpb provides hand-written visit service protobuf types (protoc unavailable).
// Source: internal/farm/proto/visitpb.proto
package visitpb

import (
	"github.com/MQEnergy/go-skeleton/internal/farm/proto/plantpb"
	"github.com/MQEnergy/go-skeleton/internal/farm/proto/userpb"
)

// Enter reason values (gamepb.visitpb.EnterReason).
const (
	EnterReasonUnknown  int32 = 0
	EnterReasonBubble   int32 = 1
	EnterReasonFriend   int32 = 2
	EnterReasonInteract int32 = 3
)

// EnterRequest is gamepb.visitpb.EnterRequest.
type EnterRequest struct {
	HostGID int64
	Reason  int32
}

// EnterReply is gamepb.visitpb.EnterReply.
type EnterReply struct {
	Basic *userpb.BasicInfo
	Lands []*plantpb.LandInfo
}

// LeaveRequest is gamepb.visitpb.LeaveRequest.
type LeaveRequest struct {
	HostGID int64
}

// LeaveReply is gamepb.visitpb.LeaveReply.
type LeaveReply struct{}
