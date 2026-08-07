// Package friendpb provides hand-written friend service protobuf types (protoc unavailable).
// Source: internal/farm/proto/friendpb.proto
package friendpb

// Plant is farm summary info on a friend entry.
type Plant struct {
	DryTimeSec    int64
	WeedTimeSec   int64
	InsectTimeSec int64
	RipeTimeSec   int64
	RipeFruitID   int64
	StealPlantNum int64
	DryNum        int64
	WeedNum       int64
	InsectNum     int64
}

// Tags are friend list tags.
type Tags struct {
	IsNew    bool
	IsFollow bool
}

// GameFriend is one game friend entry.
type GameFriend struct {
	GID               int64
	OpenID            string
	Name              string
	AvatarURL         string
	Remark            string
	Level             int64
	Gold              int64
	Tags              *Tags
	Plant             *Plant
	AuthorizedStatus  int32
}

// GetAllRequest is gamepb.friendpb.GetAllRequest.
type GetAllRequest struct{}

// GetAllReply is gamepb.friendpb.GetAllReply.
type GetAllReply struct {
	GameFriends      []GameFriend
	ApplicationCount int64
}

// GetGameFriendsRequest is gamepb.friendpb.GetGameFriendsRequest.
type GetGameFriendsRequest struct {
	GIDs []int64
}

// GetGameFriendsReply is gamepb.friendpb.GetGameFriendsReply.
type GetGameFriendsReply struct {
	GameFriends      []GameFriend
	ApplicationCount int64
}

// Application is one friend application.
type Application struct {
	GID       int64
	TimeAt    int64
	OpenID    string
	Name      string
	AvatarURL string
	Level     int64
}

// GetApplicationsRequest is gamepb.friendpb.GetApplicationsRequest.
type GetApplicationsRequest struct{}

// GetApplicationsReply is gamepb.friendpb.GetApplicationsReply.
type GetApplicationsReply struct {
	Applications      []Application
	BlockApplications bool
}

// AcceptFriendsRequest is gamepb.friendpb.AcceptFriendsRequest.
type AcceptFriendsRequest struct {
	FriendGIDs []int64
}

// AcceptFriendsReply is gamepb.friendpb.AcceptFriendsReply.
type AcceptFriendsReply struct {
	Friends []GameFriend
}

// SyncAllRequest is gamepb.friendpb.SyncAllRequest.
type SyncAllRequest struct {
	OpenIDs []string
}

// SyncAllReply is gamepb.friendpb.SyncAllReply.
type SyncAllReply struct {
	GameFriends      []GameFriend
	ApplicationCount int64
}

// RejectFriendsRequest is gamepb.friendpb.RejectFriendsRequest.
type RejectFriendsRequest struct {
	FriendGIDs []int64
}

// RejectFriendsReply is gamepb.friendpb.RejectFriendsReply.
type RejectFriendsReply struct{}

// SetBlockApplicationsRequest is gamepb.friendpb.SetBlockApplicationsRequest.
type SetBlockApplicationsRequest struct {
	Block bool
}

// SetBlockApplicationsReply is gamepb.friendpb.SetBlockApplicationsReply.
type SetBlockApplicationsReply struct {
	Block bool
}

// GetShareKeyRequest is gamepb.friendpb.GetShareKeyRequest.
type GetShareKeyRequest struct {
	ShareCfgID int64
}

// GetShareKeyReply is gamepb.friendpb.GetShareKeyReply.
type GetShareKeyReply struct {
	ShareKey   string
	ShareURL   string
	ShareCfgID int64
}

// FriendApplicationReceivedNotify is gamepb.friendpb.FriendApplicationReceivedNotify.
type FriendApplicationReceivedNotify struct {
	Applications []Application
}

// FriendAddedNotify is gamepb.friendpb.FriendAddedNotify.
type FriendAddedNotify struct {
	Friends []GameFriend
}
