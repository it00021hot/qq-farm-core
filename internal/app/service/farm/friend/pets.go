package friend

// 宠物（狗）面板服务 + 图鉴快照服务（挂在 friend 服务下复用 liveSession）。

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/it00021hot/qq-farm-core/internal/farm/activitycenter"
	"github.com/it00021hot/qq-farm-core/internal/farm/game"
	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/corepb"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/dogpb"
	"github.com/it00021hot/qq-farm-core/internal/types/farm"
)

// petSnapshotWithBag 拉取宠物信息 + 背包并构建页面快照（activate/deploy 的
// 前置与事后校验共用）。
func petSnapshotWithBag(ctx context.Context, api *game.API) (map[string]any, error) {
	info, err := api.GetDogInfo(ctx)
	if err != nil {
		return nil, err
	}
	bag, err := api.Bag(ctx)
	if err != nil {
		return nil, err
	}
	return petSnapshot(info, game.GetBagItems(bag)), nil
}

// petSnapshotMap 在快照里按 id 找一只狗的条目。
func petSnapshotMap(snapshot map[string]any, dogID int64) map[string]any {
	dogs, _ := snapshot["dogs"].([]map[string]any)
	for _, dog := range dogs {
		if id, _ := dog["id"].(int64); id == dogID {
			return dog
		}
	}
	return nil
}

// petDogActivatable 对齐 rust services/pets.rs（bot 9907ffd）：未拥有，
// 且 field_6=1（背包有同 ID 宠物卡）或背包里真有未锁定的同 ID 卡片；
// 激活消耗卡片后 field_6 消失。
func petDogActivatable(owned bool, raw *dogpb.DogInfo, bag []corepb.Item, id int64) bool {
	if owned {
		return false
	}
	hasBagCard := int64(0)
	for i := range bag {
		it := &bag[i]
		if it.GetId() == id && !it.GetLocked() && it.GetCount() > 0 {
			hasBagCard += it.GetCount()
		}
	}
	return (raw != nil && raw.GetField_6() == 1) || hasBagCard > 0
}

// DogInfo returns the pet page payload, 1:1 with rust PetService.get_pet_info
// （狗列表含技能文案与用量、狗粮读背包、护主剩余时间、待领礼包数）。
func (s *Service) DogInfo(ctx fiber.Ctx, req farm.AccountIDReq) (map[string]any, error) {
	session, err := s.liveSession(req.ID)
	if err != nil {
		return nil, err
	}
	api := session.GameAPI()
	callCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	return petSnapshotWithBag(callCtx, api)
}

// petPetSnapshot 的常量与文案表对齐 rust services/pets.rs。
var petIDs = []int64{90001, 90002, 90003, 90011, 90021, 90031}

const petMaxProtectDurationSeconds int64 = 30 * 24 * 60 * 60

func petFoodDuration(id int64) (int64, bool) {
	switch id {
	case 90004:
		return 24 * 60 * 60, true
	case 90005:
		return 3 * 24 * 60 * 60, true
	case 90006:
		return 5 * 24 * 60 * 60, true
	}
	return 0, false
}

func petRarityLabel(rarity int64) string {
	switch rarity {
	case 1:
		return "普通"
	case 2:
		return "稀有"
	case 3:
		return "珍品"
	case 4:
		return "天工"
	}
	return "未知"
}

func petObtainCondition(id int64) string {
	switch id {
	case 90001:
		return "参与分享任务可获得"
	case 90002:
		return "商店购买：100 点券"
	case 90003:
		return "商店购买：200 点券"
	case 90011:
		return "商店购买：200 点券"
	case 90021:
		return "限时活动获得"
	case 90031:
		return "萌宠成长日记：将比熊幼崽培育至成年后永久获得"
	}
	return "游戏内活动或购买获得"
}

// petSkillDefinitions is the client-static skill copy table (TS PET_SKILLS).
func petSkillDefinitions(petID int64) []map[string]any {
	loyalty := func(rate int64) map[string]any {
		return map[string]any{
			"name":        "忠心护主",
			"description": fmt.Sprintf("作物被偷时，有%d%%概率触发看护，成功后扣除偷窃者一定金币。", rate),
			"triggerRate": rate,
			"source":      "game-config",
		}
	}
	switch petID {
	case 90001:
		return []map[string]any{loyalty(10)}
	case 90002:
		return []map[string]any{loyalty(30)}
	case 90003, 90011:
		return []map[string]any{loyalty(50)}
	case 90021:
		return []map[string]any{
			loyalty(50),
			{
				"skillId":     2001,
				"name":        "同气连枝",
				"description": "好友前来农场互助（浇水/除草/除虫）时，有概率掉落同气连枝礼包（每日限30次），主人与好友均可获得奖励。",
				"dailyLimit":  30,
				"source":      "client-static",
			},
		}
	case 90031:
		return []map[string]any{
			loyalty(50),
			{
				"skillId":     3001,
				"name":        "比熊润田",
				"description": "看护状态下，作物有概率触发比熊变异（售价 ×4），可叠加冰冻、爱心、暗化、湿润等变异效果。",
				"source":      "game-config",
			},
		}
	}
	return []map[string]any{}
}

func petSnapshot(info *dogpb.GetDogInfoReply, bag []corepb.Item) map[string]any {
	currentDogID := info.GetCurrentDogId()
	skillUsages := info.GetSkillUsages()

	dogIDs := append([]int64(nil), petIDs...)
	for _, raw := range info.GetDogs() {
		if raw.GetId() > 0 && !slices.Contains(dogIDs, raw.GetId()) {
			dogIDs = append(dogIDs, raw.GetId())
		}
	}

	dogs := []map[string]any{}
	for _, id := range dogIDs {
		var raw *dogpb.DogInfo
		for _, d := range info.GetDogs() {
			if d.GetId() == id {
				raw = d
				break
			}
		}
		name := fmt.Sprintf("宠物#%d", id)
		rarity := int64(0)
		if item := logic.GetItemByID(id); item != nil {
			if strings.TrimSpace(item.Name) != "" {
				name = item.Name
			}
			rarity = item.Rarity
		}
		if raw != nil && strings.TrimSpace(raw.GetName()) != "" {
			name = raw.GetName()
		}
		owned := raw != nil && raw.GetOwned() == 1
		if id == currentDogID {
			owned = true
		}
		activatable := petDogActivatable(owned, raw, bag, id)

		skills := []map[string]any{}
		for _, def := range petSkillDefinitions(id) {
			skillID, _ := def["skillId"].(int64)
			if skillID > 0 {
				for _, usage := range skillUsages {
					if usage.GetSkillId() == skillID && usage.GetDogId() == id {
						dailyLimit := usage.GetDailyLimit()
						if dailyLimit <= 0 {
							if v, ok := def["dailyLimit"].(int64); ok {
								dailyLimit = v
							}
						}
						def["dailyLimit"] = dailyLimit
						def["usedCount"] = usage.GetUsedCount()
						def["remainingCount"] = maxInt64(dailyLimit-usage.GetUsedCount(), 0)
					}
				}
			}
			skills = append(skills, def)
		}
		skillDescription := "暂无技能说明"
		if len(skills) > 0 {
			if desc, ok := skills[0]["description"].(string); ok && desc != "" {
				skillDescription = desc
			}
		}

		price, level, status := int64(0), int64(0), int64(0)
		if raw != nil {
			price, level, status = raw.GetPrice(), raw.GetLevel(), raw.GetStatus()
		}
		dogs = append(dogs, map[string]any{
			"id": id, "name": name, "image": "", "rarity": rarity,
			"rarityLabel": petRarityLabel(rarity), "skills": skills,
			"skillDescription": skillDescription, "obtainCondition": petObtainCondition(id),
			"price": price, "level": level, "status": status,
			"owned": owned, "activatable": activatable, "active": id == currentDogID,
		})
	}

	// 狗粮库存：背包是唯一依据。
	bagCounts := map[int64]int64{}
	for _, item := range bag {
		if _, isFood := petFoodDuration(item.Id); isFood && item.Count > 0 {
			bagCounts[item.Id] += item.Count
		}
	}
	foods := []map[string]any{}
	for _, id := range []int64{90004, 90005, 90006} {
		fallback, _ := petFoodDuration(id)
		duration := fallback
		for _, f := range info.GetItems() {
			if f.GetId() == id && f.GetDuration() > 0 {
				duration = f.GetDuration()
				break
			}
		}
		name := fmt.Sprintf("狗粮#%d", id)
		if item := logic.GetItemByID(id); item != nil && strings.TrimSpace(item.Name) != "" {
			name = item.Name
		}
		foods = append(foods, map[string]any{
			"id": id, "name": name, "image": "", "duration": duration,
			"count": bagCounts[id],
		})
	}

	protectDuration := maxInt64(info.GetProtectTime(), 0)
	maxProtect := info.GetMaxProtectTime()
	if maxProtect <= 0 {
		maxProtect = petMaxProtectDurationSeconds
	}
	if maxProtect < protectDuration {
		maxProtect = protectDuration
	}
	return map[string]any{
		"dogs":                     dogs,
		"foods":                    foods,
		"protectDuration":          protectDuration,
		"maxProtectDuration":       maxProtect,
		"remainingDuration":        protectDuration,
		"pendingGiftCount":         maxInt64(info.GetPendingGiftCount(), 0),
		"activeDogId":              currentDogID,
		"activeControlSupported":   true,
		"guardianRecordsSupported": true,
		"skillCatalog":             map[string]any{"source": "client-static", "requestVerified": true},
	}
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

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
		// 前置状态校验（对齐 rust deploy_dog）：未获得时报错按 field_6 区分
		// 「尚未激活」与「未获得」，提示用户先去宠物页激活。
		before, err := petSnapshotWithBag(callCtx, api)
		if err != nil {
			return nil, err
		}
		beforeDog := petSnapshotMap(before, req.DogID)
		if beforeDog == nil {
			return nil, errors.New("该宠物不在图鉴中")
		}
		owned, _ := beforeDog["owned"].(bool)
		if !owned {
			if activatable, _ := beforeDog["activatable"].(bool); activatable {
				return nil, errors.New("该宠物尚未激活，请先激活")
			}
			return nil, errors.New("未获得该宠物，无法上场")
		}
		reply, err := api.DeployDog(callCtx, req.DogID)
		if err != nil {
			return nil, err
		}
		return map[string]any{"dogId": reply.GetDogId(), "message": "狗狗已上场"}, nil
	case "activate":
		// 宠物激活（对齐 bot 9907ffd / rust activate_dog）：消耗背包中的
		// 宠物卡，把图鉴项变成可上场的已获得宠物。
		if req.DogID <= 0 {
			return nil, errors.New("dogId 必填")
		}
		before, err := petSnapshotWithBag(callCtx, api)
		if err != nil {
			return nil, err
		}
		beforeDog := petSnapshotMap(before, req.DogID)
		if beforeDog == nil {
			return nil, errors.New("该宠物不在图鉴中")
		}
		if owned, _ := beforeDog["owned"].(bool); owned {
			return nil, errors.New("该宠物已获得，无需重复激活")
		}
		if activatable, _ := beforeDog["activatable"].(bool); !activatable {
			return nil, errors.New("背包中没有该宠物的卡片，无法激活")
		}
		if _, err := api.ActivateDog(callCtx, req.DogID); err != nil {
			return nil, err
		}
		after, err := petSnapshotWithBag(callCtx, api)
		if err != nil {
			return nil, err
		}
		afterDog := petSnapshotMap(after, req.DogID)
		if afterDog == nil {
			return nil, errors.New("宠物激活状态未更新，请稍后重试")
		}
		if owned, _ := afterDog["owned"].(bool); !owned {
			return nil, errors.New("宠物激活状态未更新，请稍后重试")
		}
		return map[string]any{"dogId": req.DogID, "message": "宠物已激活"}, nil
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
			"id":            0,
			"timestamp":     entry.GetTimestamp(),
			"friendGid":     entry.GetFriendGid(),
			"friendName":    entry.GetFriendName(),
			"friendAvatar":  entry.GetFriendAvatar(),
			"stolenCount":   entry.GetStolenCount(),
			"protectedGold": entry.GetProtectedGold(),
			"dogId":         entry.GetDogId(),
			"dogName":       entry.GetDogName(),
		})
	}
	return map[string]any{
		"logs":   logs,
		"total":  reply.GetTotal(),
		"limit":  len(logs),
		"offset": 0,
	}, nil
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
			"owned":    dog.GetOwned() == 1,
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
		"dogs":             dogs,
		"currentDogId":     info.GetCurrentDogId(),
		"protectTime":      info.GetProtectTime(),
		"maxProtectTime":   info.GetMaxProtectTime(),
		"items":            items,
		"pendingGiftCount": info.GetPendingGiftCount(),
		"skillUsages":      skills,
	}
}
