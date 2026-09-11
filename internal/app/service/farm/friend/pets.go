package friend

// 宠物（狗）面板服务 + 图鉴快照服务（挂在 friend 服务下复用 liveSession）。

import (
	"context"
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/it00021hot/qq-farm-core/internal/farm/activitycenter"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/dogpb"
	"github.com/it00021hot/qq-farm-core/internal/types/farm"
)

// DogInfo returns the dog page payload (deployed dog, food items, skill usages, pending gifts).
func (s *Service) DogInfo(ctx fiber.Ctx, req farm.AccountIDReq) (map[string]any, error) {
	session, err := s.liveSession(req.ID)
	if err != nil {
		return nil, err
	}
	api := session.GameAPI()
	callCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	info, err := api.GetDogInfo(callCtx)
	if err != nil {
		return nil, err
	}
	return dogInfoDTO(info), nil
}

// DogOperate handles deploy/withdraw/addFood/claimSkillGifts.
func (s *Service) DogOperate(ctx fiber.Ctx, req farm.DogOpReq) (map[string]any, error) {
	session, err := s.liveSession(req.AccountID)
	if err != nil {
		return nil, err
	}
	api := session.GameAPI()
	callCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	switch req.Op {
	case "deploy":
		if req.DogID <= 0 {
			return nil, errors.New("dogId 必填")
		}
		reply, err := api.DeployDog(callCtx, req.DogID)
		if err != nil {
			return nil, err
		}
		return map[string]any{"dogId": reply.GetDogId(), "message": "狗狗已上场"}, nil
	case "withdraw":
		reply, err := api.WithdrawDog(callCtx)
		if err != nil {
			return nil, err
		}
		return map[string]any{"dogId": reply.GetDogId(), "message": "狗狗已下场"}, nil
	case "addFood":
		if req.ItemID <= 0 {
			return nil, errors.New("itemId 必填（狗粮道具）")
		}
		count := req.Count
		if count <= 0 {
			count = 1
		}
		reply, err := api.AddFood(callCtx, req.ItemID, count)
		if err != nil {
			return nil, err
		}
		return map[string]any{"protectTime": reply.GetProtectTime(), "message": "狗粮已添加"}, nil
	case "claimSkillGifts":
		reply, err := api.ClaimSkillGifts(callCtx)
		if err != nil {
			return nil, err
		}
		claimed := reply.GetClaimedCount()
		if item := reply.GetItem(); item != nil && item.Count > claimed {
			claimed = item.Count
		}
		return map[string]any{"claimedCount": claimed, "message": "礼包已拾取"}, nil
	default:
		return nil, errors.New("未知宠物操作")
	}
}

// DogProtectLogs returns the guard history page.
func (s *Service) DogProtectLogs(ctx fiber.Ctx, req farm.AccountIDReq) (map[string]any, error) {
	session, err := s.liveSession(req.ID)
	if err != nil {
		return nil, err
	}
	api := session.GameAPI()
	callCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	reply, err := api.GetProtectLogs(callCtx)
	if err != nil {
		return nil, err
	}
	logs := []map[string]any{}
	for _, entry := range reply.GetLogs() {
		if entry == nil {
			continue
		}
		logs = append(logs, map[string]any{
			"time":          entry.GetTimestamp() * 1000,
			"friendGid":     entry.GetFriendGid(),
			"friendName":    entry.GetFriendName(),
			"friendAvatar":  entry.GetFriendAvatar(),
			"stolenCount":   entry.GetStolenCount(),
			"protectedGold": entry.GetProtectedGold(),
			"dogId":         entry.GetDogId(),
			"dogName":       entry.GetDogName(),
		})
	}
	return map[string]any{"logs": logs, "count": len(logs)}, nil
}

// IllustratedSnapshot returns the crop + mutant books.
func (s *Service) IllustratedSnapshot(ctx fiber.Ctx, req farm.AccountIDReq) (map[string]any, error) {
	session, err := s.liveSession(req.ID)
	if err != nil {
		return nil, err
	}
	api := session.GameAPI()
	callCtx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	return activitycenter.BuildIllustratedSnapshot(callCtx, api)
}

// dogInfoDTO builds the dog page payload from GetDogInfoReply.
func dogInfoDTO(info *dogpb.GetDogInfoReply) map[string]any {
	dogs := []map[string]any{}
	for _, dog := range info.GetDogs() {
		if dog == nil {
			continue
		}
		dogs = append(dogs, map[string]any{
			"id": dog.GetId(), "name": dog.GetName(), "price": dog.GetPrice(),
			"level": dog.GetLevel(),
			// 服务端对所有图鉴项均返回 1，不能据此判断是否已获得；
			// 真实列表与游戏锁定状态逐项比对：1=已获得；缺失=未获得。
			"owned": dog.GetOwned() == 1,
			"deployed": dog.GetId() == info.GetCurrentDogId(),
		})
	}
	items := []map[string]any{}
	for _, item := range info.GetItems() {
		if item == nil {
			continue
		}
		// 库存必须读取背包；这里的 status 只是状态位。
		items = append(items, map[string]any{
			"id": item.GetId(), "duration": item.GetDuration(), "status": item.GetStatus(),
		})
	}
	skills := []map[string]any{}
	for _, usage := range info.GetSkillUsages() {
		if usage == nil {
			continue
		}
		skills = append(skills, map[string]any{
			"skillId": usage.GetSkillId(), "usedCount": usage.GetUsedCount(),
			"dailyLimit": usage.GetDailyLimit(), "dogId": usage.GetDogId(),
		})
	}
	return map[string]any{
		"dogs":              dogs,
		"currentDogId":      info.GetCurrentDogId(),
		"protectTime":       info.GetProtectTime(),
		"maxProtectTime":    info.GetMaxProtectTime(),
		"items":             items,
		"pendingGiftCount":  info.GetPendingGiftCount(),
		"skillUsages":       skills,
	}
}
