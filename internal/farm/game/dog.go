package game

import (
	"context"

	"github.com/it00021hot/qq-farm-core/internal/farm/proto/dogpb"
)

// DogSkillGiftItemID 是「同气连枝」礼包物品 id（宠物技能掉落，
// 帮忙务农回包 FarmingReply.results[].reward.id 识别用；rust game_ids.rs）。
const DogSkillGiftItemID int64 = 101351

func (a *API) sendDog(ctx context.Context, method string, body []byte) ([]byte, error) {
	if err := a.requireSender(); err != nil {
		return nil, err
	}
	raw, _, err := a.Sender.Send(ctx, dogService, method, nonNilBody(body))
	return raw, err
}

// GetDogInfo fetches dog info and protect items.
func (a *API) GetDogInfo(ctx context.Context) (*dogpb.GetDogInfoReply, error) {
	raw, err := a.sendDog(ctx, "GetDogInfo", marshalMessage(&dogpb.GetDogInfoRequest{}))
	if err != nil {
		return nil, err
	}
	reply := &dogpb.GetDogInfoReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// ActivateDog activates an unowned illustrated pet by consuming a bag pet card
// (bot 9907ffd：宠物页“激活”）。
func (a *API) ActivateDog(ctx context.Context, dogID int64) (*dogpb.ActivateDogReply, error) {
	raw, err := a.sendDog(ctx, "ActivateDog", marshalMessage(&dogpb.ActivateDogRequest{DogId: dogID}))
	if err != nil {
		return nil, err
	}
	reply := &dogpb.ActivateDogReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// DeployDog deploys the given dog (dog page "上场").
func (a *API) DeployDog(ctx context.Context, dogID int64) (*dogpb.DeployDogReply, error) {
	raw, err := a.sendDog(ctx, "DeployDog", marshalMessage(&dogpb.DeployDogRequest{DogId: dogID}))
	if err != nil {
		return nil, err
	}
	reply := &dogpb.DeployDogReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// WithdrawDog withdraws the deployed dog.
func (a *API) WithdrawDog(ctx context.Context) (*dogpb.WithdrawDogReply, error) {
	raw, err := a.sendDog(ctx, "WithdrawDog", marshalMessage(&dogpb.WithdrawDogRequest{}))
	if err != nil {
		return nil, err
	}
	reply := &dogpb.WithdrawDogReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// AddFood feeds the dog bowl (itemID = dog food item, count = servings).
func (a *API) AddFood(ctx context.Context, itemID, count int64) (*dogpb.AddFoodReply, error) {
	raw, err := a.sendDog(ctx, "AddFood", marshalMessage(&dogpb.AddFoodRequest{ItemId: itemID, Count: count}))
	if err != nil {
		return nil, err
	}
	reply := &dogpb.AddFoodReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// ClaimSkillGifts claims all pending "同气连枝" skill gifts at once.
func (a *API) ClaimSkillGifts(ctx context.Context) (*dogpb.ClaimSkillGiftsReply, error) {
	raw, err := a.sendDog(ctx, "ClaimSkillGifts", marshalMessage(&dogpb.ClaimSkillGiftsRequest{}))
	if err != nil {
		return nil, err
	}
	reply := &dogpb.ClaimSkillGiftsReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// GetProtectLogs fetches the dog guard history page.
func (a *API) GetProtectLogs(ctx context.Context) (*dogpb.GetProtectLogsReply, error) {
	// 请求抓包固定为 { field_1: 0, count: 100, field_3: 0 }。
	raw, err := a.sendDog(ctx, "GetProtectLogs", marshalMessage(&dogpb.GetProtectLogsRequest{
		Field_1: 0, Count: 100, Field_3: 0,
	}))
	if err != nil {
		return nil, err
	}
	reply := &dogpb.GetProtectLogsReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}
