// Package sharepb provides hand-written share service protobuf types.
// Source: internal/farm/proto/sharepb.proto
package sharepb

import "github.com/MQEnergy/go-skeleton/internal/farm/proto/corepb"

// CheckCanShareRequest checks share availability.
type CheckCanShareRequest struct{}

// CheckCanShareReply reports whether sharing is available.
type CheckCanShareReply struct {
	CanShare bool
}

// ReportShareRequest reports a share action (wire fields from capture).
type ReportShareRequest struct {
	Field1 bool
	Field4 int32
}

// ReportShareReply is the report result.
type ReportShareReply struct {
	Result []byte
}

// ClaimShareRewardRequest requests share reward (request shape pending full capture).
type ClaimShareRewardRequest struct {
	Claimed bool
}

// ClaimShareRewardReply is the share reward result.
type ClaimShareRewardReply struct {
	Items []*corepb.Item
}

// InviteUser is one invited/inviter user entry.
type InviteUser struct {
	GID    int64
	Field2 int64
	Name   string
	Avatar string
}

// InviteRewardStage is one invite reward stage.
type InviteRewardStage struct {
	Index       int64
	RewardIndex int64
	Item        *corepb.Item
	Status      int32
}

// InviteInfo is one invite relationship entry.
type InviteInfo struct {
	UserA        *InviteUser
	Status       int32
	UserB        *InviteUser
	RewardStages []*InviteRewardStage
}

// GetInviteInfoRequest fetches invite info.
type GetInviteInfoRequest struct{}

// GetInviteInfoReply is the invite info list.
type GetInviteInfoReply struct {
	Infos []*InviteInfo
}

// GetInviteAwardRequest claims an invite award.
type GetInviteAwardRequest struct {
	ShareCfgID int64
}

// GetInviteAwardReply is the invite award result.
type GetInviteAwardReply struct {
	Success bool
	Items   []*corepb.Item
}
