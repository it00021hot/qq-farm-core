package game

import (
	"context"

	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/friendpb"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/plantpb"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/userpb"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/visitpb"
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
	raw, err := a.sendFriend(ctx, "GetAll", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &friendpb.GetAllReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// GetGameFriends fetches friend entries for the given GIDs.
func (a *API) GetGameFriends(ctx context.Context, gids []int64) (*friendpb.GetGameFriendsReply, error) {
	req := &friendpb.GetGameFriendsRequest{Gids: gids}
	raw, err := a.sendFriend(ctx, "GetGameFriends", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &friendpb.GetGameFriendsReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// GetApplications fetches pending friend applications.
func (a *API) GetApplications(ctx context.Context) (*friendpb.GetApplicationsReply, error) {
	req := &friendpb.GetApplicationsRequest{}
	raw, err := a.sendFriend(ctx, "GetApplications", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &friendpb.GetApplicationsReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// AcceptFriends accepts friend applications for the given GIDs.
func (a *API) AcceptFriends(ctx context.Context, friendGIDs []int64) (*friendpb.AcceptFriendsReply, error) {
	req := &friendpb.AcceptFriendsRequest{FriendGids: friendGIDs}
	raw, err := a.sendFriend(ctx, "AcceptFriends", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &friendpb.AcceptFriendsReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// SyncAll syncs friends by open IDs (QQ legacy friend list path).
func (a *API) SyncAll(ctx context.Context, openIDs []string) (*friendpb.SyncAllReply, error) {
	req := &friendpb.SyncAllRequest{OpenIds: openIDs}
	raw, err := a.sendFriend(ctx, "SyncAll", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &friendpb.SyncAllReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// RejectFriends rejects friend applications for the given GIDs.
func (a *API) RejectFriends(ctx context.Context, friendGIDs []int64) (*friendpb.RejectFriendsReply, error) {
	req := &friendpb.RejectFriendsRequest{FriendGids: friendGIDs}
	raw, err := a.sendFriend(ctx, "RejectFriends", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &friendpb.RejectFriendsReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// SetBlockApplications toggles blocking of friend applications.
func (a *API) SetBlockApplications(ctx context.Context, block bool) (*friendpb.SetBlockApplicationsReply, error) {
	req := &friendpb.SetBlockApplicationsRequest{Block: block}
	raw, err := a.sendFriend(ctx, "SetBlockApplications", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &friendpb.SetBlockApplicationsReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// GetShareKey fetches a friend-share key for the given share config ID.
func (a *API) GetShareKey(ctx context.Context, shareCfgID int64) (*friendpb.GetShareKeyReply, error) {
	req := &friendpb.GetShareKeyRequest{ShareCfgId: shareCfgID}
	raw, err := a.sendFriend(ctx, "GetShareKey", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &friendpb.GetShareKeyReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// VisitEnter enters a friend's farm and returns mapped land info.
// EnterReplyDetail carries the parts of VisitService.Enter that callers beyond
// land analysis need (bot enterFriendFarm returns the whole reply).
type EnterReplyDetail struct {
	Lands         []logic.LandInfo
	Basic         *userpb.BasicInfo
	DogID         int64 // brief_dog_info.dog_id; 0 = no dog deployed
	Weather       int64 // weather.weather_type (0 = none)
	WeatherStatus int64
}

// VisitEnterDetailed enters a friend farm and returns the full reply detail.
func (a *API) VisitEnterDetailed(ctx context.Context, hostGID int64, reason int32) (*EnterReplyDetail, error) {
	req := &visitpb.EnterRequest{HostGid: hostGID, Reason: reason}
	raw, err := a.sendVisit(ctx, "Enter", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &visitpb.EnterReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	out := &EnterReplyDetail{
		Lands: logic.LandsFromPlantPB(reply.Lands),
		Basic: reply.Basic,
	}
	if reply.BriefDogInfo != nil {
		out.DogID = reply.BriefDogInfo.DogId
	}
	if reply.Weather != nil {
		out.Weather = reply.Weather.WeatherType
		out.WeatherStatus = reply.Weather.Status
	}
	return out, nil
}

// VisitEnter enters a friend's farm and returns their lands.
func (a *API) VisitEnter(ctx context.Context, hostGID int64, reason int32) ([]logic.LandInfo, error) {
	detail, err := a.VisitEnterDetailed(ctx, hostGID, reason)
	if err != nil {
		return nil, err
	}
	return detail.Lands, nil
}

// VisitLeave leaves a friend's farm.
func (a *API) VisitLeave(ctx context.Context, hostGID int64) error {
	req := &visitpb.LeaveRequest{HostGid: hostGID}
	_, err := a.sendVisit(ctx, "Leave", marshalMessage(req))
	return err
}

// FriendHarvest steals harvest from a friend's lands and returns the decoded reply
// (items include activity score / fruit rewards — needed for steal logs).
func (a *API) FriendHarvest(ctx context.Context, hostGID int64, landIDs []int64, isAll bool) (*plantpb.HarvestReply, error) {
	req := &plantpb.HarvestRequest{
		LandIds: landIDs,
		HostGid: hostGID,
		IsAll:   isAll,
	}
	raw, err := a.sendPlant(ctx, "Harvest", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &plantpb.HarvestReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// FriendFarming weeds/bugs/(and often water) on a friend's lands; returns decoded reply for limits/exp.
func (a *API) FriendFarming(ctx context.Context, hostGID int64, landIDs []int64) (*plantpb.FarmingReply, error) {
	req := &plantpb.FarmingRequest{
		LandIds: landIDs,
		HostGid: hostGID,
		Field_3: 0,
		Field_4: 2,
	}
	raw, err := a.sendPlant(ctx, "Farming", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &plantpb.FarmingReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// FriendWater waters a friend's lands; returns decoded reply for operation limits.
func (a *API) FriendWater(ctx context.Context, hostGID int64, landIDs []int64) (*plantpb.WaterLandReply, error) {
	req := &plantpb.WaterLandRequest{LandIds: landIDs, HostGid: hostGID}
	raw, err := a.sendPlant(ctx, "WaterLand", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &plantpb.WaterLandReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// DelFriend removes a friend by GID.
func (a *API) DelFriend(ctx context.Context, friendGID int64) error {
	req := &friendpb.DelFriendRequest{FriendGid: friendGID}
	_, err := a.sendFriend(ctx, "DelFriend", marshalMessage(req))
	return err
}
