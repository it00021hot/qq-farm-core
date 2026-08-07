// Package avatarframepb provides hand-written avatar frame protobuf types.
// Source: qq-farm-bot/core/src/proto/avatarframepb.proto
package avatarframepb

// AvatarFrame is one avatar frame entry.
type AvatarFrame struct {
	FrameID    int64
	Status     int64
	Equipped   int64
	ExpireTime int64
}

// AvatarFramesOwnedRequest fetches owned avatar frames.
type AvatarFramesOwnedRequest struct{}

// AvatarFramesOwnedReply lists owned avatar frames.
type AvatarFramesOwnedReply struct {
	Frames []*AvatarFrame
}

// AvatarFrameRedDotNotify is an empty red-dot notify.
type AvatarFrameRedDotNotify struct{}
