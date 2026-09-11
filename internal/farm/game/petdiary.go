package game

// 萌宠成长日记的 game 层 RPC（对齐 bot rpc/operate 封装）。

import (
	"context"

	"github.com/it00021hot/qq-farm-core/internal/farm/proto/activitypb"
)

// GetPetDiaryGroup fetches the pet-diary activity group (dedicated reply shape).
func (a *API) GetPetDiaryGroup(ctx context.Context, groupID int64) (*activitypb.PetDiaryGetGroupReply, error) {
	req := &activitypb.GetGroupRequest{GroupId: groupID}
	raw, err := a.sendActivity(ctx, "GetGroup", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &activitypb.PetDiaryGetGroupReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// PetDiaryQueryShop loads the 拾物小铺 catalog (operate 7, no selector).
func (a *API) PetDiaryQueryShop(ctx context.Context) (*activitypb.PetDiaryActivityData, error) {
	req := &activitypb.PetDiaryOperateRequest{
		ActivityId:  PetDiaryShopActivityID,
		OperateType: 7,
	}
	raw, err := a.sendActivity(ctx, "Operate", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &activitypb.PetDiaryOperateReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	if reply.ActivityId != PetDiaryShopActivityID || reply.OperateType != 7 {
		return nil, errPetDiaryMismatch
	}
	if reply.Data == nil || reply.Data.Shop == nil {
		return nil, errPetDiaryMismatch
	}
	return reply.Data, nil
}

var errPetDiaryMismatch = &petDiaryError{message: "活动响应不匹配"}

type petDiaryError struct{ message string }

func (e *petDiaryError) Error() string { return e.message }

// PetDiaryShopActivityID mirrors bot SHOP_ID.
const PetDiaryShopActivityID int64 = 2026090103

// PetDiaryOperate sends one pet-diary operate (validate echo only).
func (a *API) PetDiaryOperate(ctx context.Context, req *activitypb.PetDiaryOperateRequest, activityID, operateType int64) (*activitypb.PetDiaryOperateReply, error) {
	req.ActivityId = activityID
	req.OperateType = operateType
	raw, err := a.sendActivity(ctx, "Operate", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &activitypb.PetDiaryOperateReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	if reply.ActivityId != activityID || reply.OperateType != operateType {
		return nil, errPetDiaryMismatch
	}
	return reply, nil
}
