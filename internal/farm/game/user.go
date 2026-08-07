package game

import (
	"context"

	"github.com/it00021hot/qq-farm-core/internal/farm/proto/userpb"
)

func (a *API) sendUser(ctx context.Context, method string, body []byte) ([]byte, error) {
	if err := a.requireSender(); err != nil {
		return nil, err
	}
	raw, _, err := a.Sender.Send(ctx, userService, method, nonNilBody(body))
	return raw, err
}

// GetUserSettings fetches user settings.
func (a *API) GetUserSettings(ctx context.Context) (*userpb.GetUserSettingsReply, error) {
	raw, err := a.sendUser(ctx, "GetUserSettings", marshalMessage(&userpb.GetUserSettingsRequest{}))
	if err != nil {
		return nil, err
	}
	reply := &userpb.GetUserSettingsReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// ReportArkClick reports a share-link click (WX invite path).
func (a *API) ReportArkClick(ctx context.Context, req *userpb.ReportArkClickRequest) (*userpb.ReportArkClickReply, error) {
	if req == nil {
		req = &userpb.ReportArkClickRequest{}
	}
	raw, err := a.sendUser(ctx, "ReportArkClick", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &userpb.ReportArkClickReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// BatchGetBasicInfo fetches basic info for the given GIDs.
func (a *API) BatchGetBasicInfo(ctx context.Context, gids []int64) (*userpb.BatchGetBasicInfoReply, error) {
	req := &userpb.BatchGetBasicInfoRequest{Gids: gids}
	raw, err := a.sendUser(ctx, "BatchGetBasicInfo", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &userpb.BatchGetBasicInfoReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// SetDisplayInfo updates display name/signature/gender/avatar.
func (a *API) SetDisplayInfo(ctx context.Context, req *userpb.SetDisplayInfoRequest) (*userpb.SetDisplayInfoReply, error) {
	if req == nil {
		req = &userpb.SetDisplayInfoRequest{}
	}
	raw, err := a.sendUser(ctx, "SetDisplayInfo", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &userpb.SetDisplayInfoReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// SetQQFriendRecommendAuthorized sets QQ friend recommend authorization.
func (a *API) SetQQFriendRecommendAuthorized(ctx context.Context, authorized int64) (*userpb.SetQQFriendRecommendAuthorizedReply, error) {
	req := &userpb.SetQQFriendRecommendAuthorizedRequest{Authorized: authorized}
	raw, err := a.sendUser(ctx, "SetQQFriendRecommendAuthorized", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &userpb.SetQQFriendRecommendAuthorizedReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// BatchClientReportFlow reports client flow events.
func (a *API) BatchClientReportFlow(ctx context.Context, items []*userpb.ClientFlowItem) (*userpb.BatchClientReportFlowReply, error) {
	req := &userpb.BatchClientReportFlowRequest{Items: items}
	raw, err := a.sendUser(ctx, "BatchClientReportFlow", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &userpb.BatchClientReportFlowReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}
