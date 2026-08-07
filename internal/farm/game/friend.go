package game

import (
	"context"

	"github.com/MQEnergy/go-skeleton/internal/farm/logic"
	"github.com/MQEnergy/go-skeleton/internal/farm/proto/friendpb"
	"github.com/MQEnergy/go-skeleton/internal/farm/proto/plantpb"
	"github.com/MQEnergy/go-skeleton/internal/farm/proto/visitpb"
)

func (a *API) sendFriend(ctx context.Context, method string, body []byte) ([]byte, error) {
	if err := a.requireSender(); err != nil {
		return nil, err
	}
	raw, _, err := a.Sender.Send(ctx, friendService, method, nonNilBody(body))
	return raw, err
}

func (a *API) sendVisit(ctx context.Context, method string, body []byte) ([]byte, error) {
	if err := a.requireSender(); err != nil {
		return nil, err
	}
	raw, _, err := a.Sender.Send(ctx, visitService, method, nonNilBody(body))
	return raw, err
}

// GetAllFriends fetches the full friend list.
func (a *API) GetAllFriends(ctx context.Context) (*friendpb.GetAllReply, error) {
	req := &friendpb.GetAllRequest{}
	raw, err := a.sendFriend(ctx, "GetAll", req.Marshal())
	if err != nil {
		return nil, err
	}
	reply := &friendpb.GetAllReply{}
	if err := reply.Unmarshal(raw); err != nil {
		return nil, err
	}
	return reply, nil
}

// GetGameFriends fetches friend entries for the given GIDs.
func (a *API) GetGameFriends(ctx context.Context, gids []int64) (*friendpb.GetGameFriendsReply, error) {
	req := &friendpb.GetGameFriendsRequest{GIDs: gids}
	raw, err := a.sendFriend(ctx, "GetGameFriends", req.Marshal())
	if err != nil {
		return nil, err
	}
	reply := &friendpb.GetGameFriendsReply{}
	if err := reply.Unmarshal(raw); err != nil {
		return nil, err
	}
	return reply, nil
}

// GetApplications fetches pending friend applications.
func (a *API) GetApplications(ctx context.Context) (*friendpb.GetApplicationsReply, error) {
	req := &friendpb.GetApplicationsRequest{}
	raw, err := a.sendFriend(ctx, "GetApplications", req.Marshal())
	if err != nil {
		return nil, err
	}
	reply := &friendpb.GetApplicationsReply{}
	if err := reply.Unmarshal(raw); err != nil {
		return nil, err
	}
	return reply, nil
}

// AcceptFriends accepts friend applications for the given GIDs.
func (a *API) AcceptFriends(ctx context.Context, friendGIDs []int64) (*friendpb.AcceptFriendsReply, error) {
	req := &friendpb.AcceptFriendsRequest{FriendGIDs: friendGIDs}
	raw, err := a.sendFriend(ctx, "AcceptFriends", req.Marshal())
	if err != nil {
		return nil, err
	}
	reply := &friendpb.AcceptFriendsReply{}
	if err := reply.Unmarshal(raw); err != nil {
		return nil, err
	}
	return reply, nil
}

// SyncAll syncs friends by open IDs (QQ legacy friend list path).
func (a *API) SyncAll(ctx context.Context, openIDs []string) (*friendpb.SyncAllReply, error) {
	req := &friendpb.SyncAllRequest{OpenIDs: openIDs}
	raw, err := a.sendFriend(ctx, "SyncAll", req.Marshal())
	if err != nil {
		return nil, err
	}
	reply := &friendpb.SyncAllReply{}
	if err := reply.Unmarshal(raw); err != nil {
		return nil, err
	}
	return reply, nil
}

// RejectFriends rejects friend applications for the given GIDs.
func (a *API) RejectFriends(ctx context.Context, friendGIDs []int64) (*friendpb.RejectFriendsReply, error) {
	req := &friendpb.RejectFriendsRequest{FriendGIDs: friendGIDs}
	raw, err := a.sendFriend(ctx, "RejectFriends", req.Marshal())
	if err != nil {
		return nil, err
	}
	reply := &friendpb.RejectFriendsReply{}
	if err := reply.Unmarshal(raw); err != nil {
		return nil, err
	}
	return reply, nil
}

// SetBlockApplications toggles blocking of friend applications.
func (a *API) SetBlockApplications(ctx context.Context, block bool) (*friendpb.SetBlockApplicationsReply, error) {
	req := &friendpb.SetBlockApplicationsRequest{Block: block}
	raw, err := a.sendFriend(ctx, "SetBlockApplications", req.Marshal())
	if err != nil {
		return nil, err
	}
	reply := &friendpb.SetBlockApplicationsReply{}
	if err := reply.Unmarshal(raw); err != nil {
		return nil, err
	}
	return reply, nil
}

// GetShareKey fetches a friend-share key for the given share config ID.
func (a *API) GetShareKey(ctx context.Context, shareCfgID int64) (*friendpb.GetShareKeyReply, error) {
	req := &friendpb.GetShareKeyRequest{ShareCfgID: shareCfgID}
	raw, err := a.sendFriend(ctx, "GetShareKey", req.Marshal())
	if err != nil {
		return nil, err
	}
	reply := &friendpb.GetShareKeyReply{}
	if err := reply.Unmarshal(raw); err != nil {
		return nil, err
	}
	return reply, nil
}

// VisitEnter enters a friend's farm and returns mapped land info.
func (a *API) VisitEnter(ctx context.Context, hostGID int64, reason int32) ([]logic.LandInfo, error) {
	req := &visitpb.EnterRequest{HostGID: hostGID, Reason: reason}
	raw, err := a.sendVisit(ctx, "Enter", req.Marshal())
	if err != nil {
		return nil, err
	}
	reply := &visitpb.EnterReply{}
	if err := reply.Unmarshal(raw); err != nil {
		return nil, err
	}
	return plantpb.LandsToLogic(reply.Lands), nil
}

// VisitLeave leaves a friend's farm.
func (a *API) VisitLeave(ctx context.Context, hostGID int64) error {
	req := &visitpb.LeaveRequest{HostGID: hostGID}
	_, err := a.sendVisit(ctx, "Leave", req.Marshal())
	return err
}

// FriendHarvest steals harvest from a friend's lands and returns the decoded reply
// (items include activity score / fruit rewards — needed for steal logs).
func (a *API) FriendHarvest(ctx context.Context, hostGID int64, landIDs []int64) (*plantpb.HarvestReply, error) {
	req := &plantpb.HarvestRequest{
		LandIDs: landIDs,
		HostGID: hostGID,
		IsAll:   true,
	}
	raw, err := a.sendPlant(ctx, "Harvest", req.Marshal())
	if err != nil {
		return nil, err
	}
	reply := &plantpb.HarvestReply{}
	if err := reply.Unmarshal(raw); err != nil {
		return nil, err
	}
	return reply, nil
}

// FriendFarming weeds/bugs/(and often water) on a friend's lands; returns decoded reply for limits/exp.
func (a *API) FriendFarming(ctx context.Context, hostGID int64, landIDs []int64) (*plantpb.FarmingReply, error) {
	req := &plantpb.FarmingRequest{
		LandIDs: landIDs,
		HostGID: hostGID,
		Field3:  0,
		Field4:  2,
	}
	raw, err := a.sendPlant(ctx, "Farming", req.Marshal())
	if err != nil {
		return nil, err
	}
	reply := &plantpb.FarmingReply{}
	if err := reply.Unmarshal(raw); err != nil {
		return nil, err
	}
	return reply, nil
}

// FriendWater waters a friend's lands; returns decoded reply for operation limits.
func (a *API) FriendWater(ctx context.Context, hostGID int64, landIDs []int64) (*plantpb.WaterLandReply, error) {
	req := &plantpb.WaterLandRequest{LandIDs: landIDs, HostGID: hostGID}
	raw, err := a.sendPlant(ctx, "WaterLand", req.Marshal())
	if err != nil {
		return nil, err
	}
	reply := &plantpb.WaterLandReply{}
	if err := reply.Unmarshal(raw); err != nil {
		return nil, err
	}
	return reply, nil
}
