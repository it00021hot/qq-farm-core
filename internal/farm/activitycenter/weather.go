package activitycenter

// 雨落成诗（对齐 bot services/weather-activity.ts 核心路径）。
//
// 操作：兑换采集瓶(1) / 好友雷雨采集(9) / 推进气象研究(40)，
// 另有雷雨召唤瓶/青蛙使坏瓶/乌云使坏瓶经 ItemService.Use 使用。

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/farm/game"
	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/activitypb"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/corepb"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/weatherpb"
)

const (
	WeatherGroupID          int64 = 2026070300
	WeatherShopActivityID   int64 = 2026070301
	WeatherMutationActivityID int64 = 2026070302
	WeatherBottleActivityID int64 = 2026070303
	WeatherResearchActivityID int64 = 2026070304
	WeatherTaskActivityID   int64 = 2026070305

	operateWeatherExchange int64 = 1
	operateWeatherCollect  int64 = 9
	operateWeatherResearch int64 = 40

	collectorBottleID    int64 = 5001
	summonBottleID       int64 = 5002
	frogMischiefBottleID int64 = 5005
	cloudMischiefBottleID int64 = 5006
	lightningBadgeID     int64 = 1027
	lightningMutantCfgID int64 = 12
	thunderstormType     int64 = 1
	collectCycleMarker   int64 = 4

	friendWeatherCacheTTL  = 600
	friendWeatherBatchMax  = 5
	friendWeatherScanGapMs = 300

	collectDailyLimit  = 10
	mischiefDailyLimit = 100
)

var weatherItemIDs = []int64{4002, 4003, 5001, 5002, 5003, 5004, 5005, 5006, 5007, 5008}

// weatherItemDTO builds one inventory/bottle row.
func weatherItemDTO(id, count int64) map[string]any {
	name := ""
	fallback := fmt.Sprintf("物品 %d", id)
	if id == lightningBadgeID {
		fallback = "雷电徽章"
	}
	if info := logic.GetItemByID(id); info != nil && info.Name != "" {
		name = info.Name
	} else {
		name = fallback
	}
	rarity := int64(0)
	if info := logic.GetItemByID(id); info != nil {
		rarity = info.Rarity
	}
	return map[string]any{
		"id":    strconv.FormatInt(id, 10),
		"count": strconv.FormatInt(count, 10),
		"name":  name,
		"image": logic.SeedImagePath(id),
		"rarity": rarity,
	}
}

// weatherStatusDTO mirrors bot weatherStatusDto.
func weatherStatusDTO(weather *weatherpb.WeatherStatus, hostGID int64) map[string]any {
	now := logic.GetServerTimeSec()
	active := weather.GetWeatherType() > 0 && weather.GetStatus() > 0 && (weather.GetEndTime() == 0 || weather.GetEndTime() > now)
	remaining := int64(0)
	if active && weather.GetEndTime() > 0 {
		remaining = weather.GetEndTime() - now
		if remaining < 0 {
			remaining = 0
		}
	}
	return map[string]any{
		"hostGid":            strconv.FormatInt(hostGID, 10),
		"type":               weather.GetWeatherType(),
		"status":             weather.GetStatus(),
		"beginTime":          weather.GetBeginTime(),
		"endTime":            weather.GetEndTime(),
		"source":             weather.GetSource(),
		"field8":             weather.GetField_8(),
		"friendMarker":       weather.GetField_9(),
		"active":             active,
		"isThunderstorm":     active && weather.GetWeatherType() == thunderstormType,
		"collectedThisCycle": weather.GetField_9() == collectCycleMarker,
		"remainingSec":       remaining,
	}
}

// weatherAvailability mirrors bot weatherAvailability.
func weatherAvailability(weather map[string]any) (state, reason string) {
	wType, _ := weather["type"].(int64)
	active, _ := weather["active"].(bool)
	isThunder, _ := weather["isThunderstorm"].(bool)
	collected, _ := weather["collectedThisCycle"].(bool)
	switch {
	case isThunder && collected:
		return "collected", "当前这轮雷雨已经采过，下轮雷雨可再次采集"
	case isThunder:
		return "available", ""
	case wType == thunderstormType && !active:
		return "expired", "这场雷雨已经结束"
	default:
		return "unavailable", "好友农场当前不是雷雨天气"
	}
}

func bagBalanceMap(items []corepb.Item) map[int64]int64 {
	out := make(map[int64]int64, 8)
	for _, item := range items {
		if item.Id > 0 {
			out[item.Id] += item.Count
		}
	}
	return out
}

func findWeatherChild(group *activitypb.ActivityData, activityID int64) *activitypb.ActivityData {
	if group == nil {
		return nil
	}
	for _, child := range group.Children {
		if child != nil && child.Activity != nil && child.Activity.ActivityId == activityID {
			return child
		}
	}
	return nil
}

// BuildWeather builds the 雨落成诗 snapshot (bot buildWeatherActivitySnapshot).
func BuildWeather(ctx context.Context, api *game.API, myGID int64) (map[string]any, error) {
	// QQ 网关对活动读取的并发很敏感，按官方客户端顺序串行请求。
	groupReply, err := api.GetActivityGroup(ctx, WeatherGroupID)
	if err != nil {
		return nil, err
	}
	group := groupReply.GetGroup()
	if group == nil || group.Activity == nil || group.Activity.ActivityId != WeatherGroupID {
		return nil, weatherErr("WEATHER_ACTIVITY_UNAVAILABLE", "服务端未发现雨落成诗活动")
	}
	bag, err := api.Bag(ctx)
	if err != nil {
		return nil, err
	}
	ownWeatherReply, err := api.GetWeatherStatus(ctx)
	if err != nil {
		return nil, err
	}

	balances := bagBalanceMap(game.GetBagItems(bag))
	activity := group.Activity
	active := weatherActivityActive(activity, logic.GetServerTimeSec())
	shopChild := findWeatherChild(group, WeatherShopActivityID)
	mutationChild := findWeatherChild(group, WeatherMutationActivityID)
	bottleChild := findWeatherChild(group, WeatherBottleActivityID)
	researchChild := findWeatherChild(group, WeatherResearchActivityID)
	taskChild := findWeatherChild(group, WeatherTaskActivityID)
	ownWeather := weatherStatusDTO(ownWeatherReply.GetWeather(), myGID)

	// shop (采集瓶兑换)
	var shop map[string]any
	if shopChild != nil && shopChild.Catalog != nil {
		var picked *activitypb.StarSandGoods
		for _, goods := range shopChild.Catalog.Goods {
			if goods == nil {
				continue
			}
			if goods.GoodsId == 200 {
				picked = goods
				break
			}
		}
		if picked == nil && len(shopChild.Catalog.Goods) > 0 {
			picked = shopChild.Catalog.Goods[0]
		}
		for _, goods := range []*activitypb.StarSandGoods{picked} {
			if goods == nil {
				continue
			}
			costID := goods.Cost.ItemId
			balance := balances[costID]
			available := active && !goods.Owned && goods.Status != 0 && balance >= goods.Cost.GetCount()
			reason := ""
			switch {
			case !active:
				reason = "活动尚未开放或已经结束"
			case goods.Owned:
				reason = "今日已经兑换过天气采集瓶"
			case balance < goods.Cost.GetCount():
				reason = "金豆豆不足"
			}
			shop = map[string]any{
				"activityId": strconv.FormatInt(WeatherShopActivityID, 10),
				"goodsId":    strconv.FormatInt(goods.GoodsId, 10),
				"item":       weatherItemDTO(goods.Item.ItemId, goods.Item.GetCount()),
				"cost":       weatherItemDTO(goods.Cost.ItemId, goods.Cost.GetCount()),
				"balance":    strconv.FormatInt(balance, 10),
				"owned":      goods.Owned,
				"statusCode": strconv.FormatInt(goods.Status, 10),
				"dailyLimit": 1,
				"available":  available,
				"reason":     reason,
			}
			break
		}
	}

	// collector bottle config
	var collector map[string]any
	if bottleChild != nil && bottleChild.WeatherBottle != nil {
		cfg := bottleChild.WeatherBottle
		rewards := make([]map[string]any, 0, len(cfg.Rewards))
		for _, reward := range cfg.Rewards {
			if reward == nil {
				continue
			}
			rewards = append(rewards, map[string]any{
				"id":          strconv.FormatInt(reward.RewardId, 10),
				"reward":      weatherItemDTO(reward.Reward.ItemId, reward.Reward.GetCount()),
				"statusCode":  strconv.FormatInt(reward.Status, 10),
				"probability": fmt.Sprint(reward.Probability),
			})
		}
		collector = map[string]any{
			"activityId":        strconv.FormatInt(WeatherBottleActivityID, 10),
			"collectorItemId":   strconv.FormatInt(cfg.CollectorItemId, 10),
			"collectorItemCount": strconv.FormatInt(cfg.CollectorItemCount, 10),
			"rewards":           rewards,
		}
	}

	// tasks
	tasks := []map[string]any{}
	if taskChild != nil && taskChild.WeatherTasks != nil {
		for _, task := range taskChild.WeatherTasks.Tasks {
			if task == nil {
				continue
			}
			tasks = append(tasks, map[string]any{
				"id":            strconv.FormatInt(task.TaskId, 10),
				"triggerItemId": strconv.FormatInt(task.TriggerItemId, 10),
				"title":         task.Title,
				"reward":        weatherItemDTO(task.Reward.ItemId, task.Reward.GetCount()),
				"dailyLimit":    strconv.FormatInt(task.DailyLimit, 10),
				"current":       strconv.FormatInt(task.Current, 10),
				"progressKnown": true,
			})
		}
	}

	// research
	var research map[string]any
	if researchChild != nil && researchChild.WeatherResearch != nil && researchChild.WeatherResearch.Track != nil {
		track := researchChild.WeatherResearch.Track
		badgeBalance := balances[lightningBadgeID]
		nodes := make([]map[string]any, 0, len(track.Nodes))
		var nextNode map[string]any
		for _, node := range track.Nodes {
			if node == nil {
				continue
			}
			costID := node.Cost.ItemId
			statusCode := node.Status
			availableByStatus := statusCode == 2
			completed := statusCode == 4 || node.Claimed
			affordable := costID == lightningBadgeID && badgeBalance >= node.Cost.GetCount()
			prereq := make([]string, 0, len(node.PrerequisiteNodeIds))
			for _, p := range node.PrerequisiteNodeIds {
				prereq = append(prereq, strconv.FormatInt(p, 10))
			}
			dto := map[string]any{
				"id":                  strconv.FormatInt(node.NodeId, 10),
				"prerequisiteNodeIds": prereq,
				"statusCode":          strconv.FormatInt(statusCode, 10),
				"cost":                weatherItemDTO(node.Cost.ItemId, node.Cost.GetCount()),
				"reward":              weatherItemDTO(node.Reward.ItemId, node.Reward.GetCount()),
				"availableByStatus":   availableByStatus,
				"completed":           completed,
				"locked":              !completed && !availableByStatus,
				"affordable":          affordable,
			}
			nodes = append(nodes, dto)
			if nextNode == nil && availableByStatus {
				nextNode = dto
			}
		}
		research = map[string]any{
			"activityId":      strconv.FormatInt(WeatherResearchActivityID, 10),
			"currentStage":    strconv.FormatInt(track.CurrentStage, 10),
			"badgeBalance":    strconv.FormatInt(badgeBalance, 10),
			"nodes":           nodes,
			"nextNode":        nextNode,
			"operateSupported": true,
		}
	}

	// inventory
	inventory := make([]map[string]any, 0, len(weatherItemIDs)+1)
	for _, id := range weatherItemIDs {
		inventory = append(inventory, weatherItemDTO(id, balances[id]))
	}
	inventory = append(inventory, weatherItemDTO(lightningBadgeID, balances[lightningBadgeID]))

	mutationActive := mutationChild != nil && mutationChild.Activity != nil && weatherActivityActive(mutationChild.Activity, logic.GetServerTimeSec())
	baseRate := int64(0)
	if shopChild != nil && shopChild.Activity != nil {
		baseRate = shopChild.Activity.Field_21
	}
	summonAvailable := active && balances[summonBottleID] > 0 && !ownWeather["active"].(bool)
	summonReason := ""
	if !active {
		summonReason = "活动尚未开放或已经结束"
	} else if ownWeather["active"].(bool) {
		if ownWeather["isThunderstorm"].(bool) {
			summonReason = "雷雨正在进行中"
		} else {
			summonReason = "当前已有其他特殊天气"
		}
	} else if balances[summonBottleID] <= 0 {
		summonReason = "背包中没有可用的雷雨召唤瓶"
	}
	advanceEnabled := false
	advanceNode := ""
	advanceReason := ""
	if !active {
		advanceReason = "活动尚未开放或已经结束"
	} else if research == nil || research["nextNode"] == nil {
		advanceReason = "气象研究已经全部完成"
	} else if nn, _ := research["nextNode"].(map[string]any); nn != nil {
		affordable, _ := nn["affordable"].(bool)
		if !affordable {
			advanceReason = "雷电徽章不足"
		} else {
			advanceEnabled = true
			advanceNode, _ = nn["id"].(string)
		}
	}

	return map[string]any{
		"groupId":    strconv.FormatInt(WeatherGroupID, 10),
		"activity":   map[string]any{"id": strconv.FormatInt(activity.ActivityId, 10), "name": activity.Name, "startTime": strconv.FormatInt(activity.BeginTime, 10), "endTime": strconv.FormatInt(activity.EndTime, 10)},
		"rules":      activityRulesDTO(activity.Extra),
		"active":     active,
		"serverTime": logic.GetServerTimeSec(),
		"mutation": map[string]any{
			"activityId":            strconv.FormatInt(WeatherMutationActivityID, 10),
			"active":                mutationActive,
			"mutantConfigId":        lightningMutantCfgID,
			"baseRatePercent":       baseRate,
			"sellMultiplier":        4,
			"excludedCropQualities": []int64{1, 2},
		},
		"ownWeather": ownWeather,
		"shop":       shop,
		"collector":  collector,
		"tasks":      tasks,
		"research":   research,
		"inventory":  inventory,
		"actions": map[string]any{
			"exchangeCollector": map[string]any{"enabled": shop != nil && shop["available"].(bool)},
			"collectWeather":    map[string]any{"enabled": active && balances[collectorBottleID] > 0, "dailyLimit": collectDailyLimit},
			"scanFriendWeather": map[string]any{"enabled": active, "batchSize": friendWeatherBatchMax},
			"frogMischief":      map[string]any{"enabled": active && balances[frogMischiefBottleID] > 0, "dailyLimit": mischiefDailyLimit},
			"cloudMischief":     map[string]any{"enabled": active && balances[cloudMischiefBottleID] > 0, "dailyLimit": mischiefDailyLimit},
			"summonThunderstorm": map[string]any{"enabled": summonAvailable, "reason": summonReason},
			"advanceResearch":   map[string]any{"enabled": advanceEnabled, "nodeId": advanceNode, "reason": advanceReason},
		},
	}, nil
}

func weatherErr(code, message string) error {
	return activityError{Code: code, Message: message}
}

// availableWeatherStack finds the soonest-expiring bag stack for an item.
func availableWeatherStack(items []corepb.Item, itemID int64) *corepb.Item {
	var best *corepb.Item
	for i := range items {
		item := &items[i]
		if item.Id != itemID || item.Count <= 0 {
			continue
		}
		if best == nil || item.ExpireTime < best.ExpireTime {
			best = item
		}
	}
	return best
}

// ExchangeWeatherCollector exchanges one collector bottle (operate 1).
func ExchangeWeatherCollector(ctx context.Context, api *game.API) (map[string]any, error) {
	snap, err := BuildWeather(ctx, api, 0)
	if err != nil {
		return nil, err
	}
	shop, _ := snap["shop"].(map[string]any)
	if shop == nil {
		return nil, weatherErr("WEATHER_SHOP_UNAVAILABLE", "天气采集瓶商店暂不可用")
	}
	if owned, _ := shop["owned"].(bool); owned {
		return nil, weatherErr("WEATHER_SHOP_ALREADY_EXCHANGED", "今日已经兑换过天气采集瓶")
	}
	if available, _ := shop["available"].(bool); !available {
		reason, _ := shop["reason"].(string)
		if reason == "" {
			reason = "天气采集瓶当前不可兑换"
		}
		return nil, weatherErr("WEATHER_SHOP_UNAVAILABLE", reason)
	}
	goodsID, _ := strconv.ParseInt(shop["goodsId"].(string), 10, 64)
	req := &activitypb.ExchangeShopRequest{
		ActivityId:  WeatherShopActivityID,
		OperateType: operateWeatherExchange,
		ExchangeShopOperate: &activitypb.ExchangeShopOperateParams{GoodsId: goodsID, Count: 1},
	}
	reply, err := api.OperateRaw(ctx, req, WeatherShopActivityID, operateWeatherExchange)
	if err != nil {
		return nil, err
	}
	rewards := make([]map[string]any, 0, len(reply.Rewards))
	for _, item := range reply.Rewards {
		if item != nil {
			rewards = append(rewards, weatherItemDTO(item.Id, item.Count))
		}
	}
	snapshot, _ := BuildWeather(ctx, api, 0)
	return map[string]any{
		"outcome":     "exchanged",
		"rewards":     rewards,
		"activityId":  strconv.FormatInt(reply.ActivityId, 10),
		"operateType": strconv.FormatInt(reply.OperateType, 10),
		"snapshot":    snapshot,
	}, nil
}

// CollectFriendWeather uses the collector bottle on a friend's thunderstorm (operate 9).
func CollectFriendWeather(ctx context.Context, api *game.API, friendGID, myGID int64) (map[string]any, error) {
	if friendGID <= 0 || friendGID == myGID {
		return nil, weatherErr("INVALID_WEATHER_FRIEND_GID", "天气采集瓶只能在好友农场使用")
	}
	bag, err := api.Bag(ctx)
	if err != nil {
		return nil, err
	}
	if availableWeatherStack(game.GetBagItems(bag), collectorBottleID) == nil {
		return nil, weatherErr("WEATHER_COLLECTOR_UNAVAILABLE", "背包中没有可用的天气采集瓶")
	}
	detail, err := api.VisitEnterDetailed(ctx, friendGID, 2)
	if err != nil {
		return nil, err
	}
	defer func() { _ = api.VisitLeave(ctx, friendGID) }()
	before := weatherStatusFromDetail(detail, friendGID)
	if !before["isThunderstorm"].(bool) {
		return nil, weatherErr("WEATHER_FRIEND_NOT_THUNDERSTORM", "该好友农场当前不是雷雨天气")
	}
	if before["collectedThisCycle"].(bool) {
		return nil, weatherErr("WEATHER_ALREADY_COLLECTED", "当前这轮雷雨已经采过，下轮雷雨可再次采集")
	}
	req := &activitypb.CollectWeatherRequest{
		ActivityId:  WeatherBottleActivityID,
		OperateType: operateWeatherCollect,
		WeatherCollectOperate: &activitypb.WeatherCollectOperateParams{HostGid: friendGID},
	}
	reply, err := api.OperateRaw(ctx, req, WeatherBottleActivityID, operateWeatherCollect)
	if err != nil {
		if strings.Contains(err.Error(), "code=1034040") {
			return nil, weatherErr("WEATHER_ALREADY_COLLECTED", "当前这轮雷雨已经采过，下轮雷雨可再次采集")
		}
		return nil, err
	}
	rewards := make([]map[string]any, 0, len(reply.Rewards))
	for _, item := range reply.Rewards {
		if item != nil {
			rewards = append(rewards, weatherItemDTO(item.Id, item.Count))
		}
	}
	// 采集成功后按官方客户端方式再次进入，记录服务端更新后的现场标记。
	after := InspectFriendWeather(ctx, api, friendGID, myGID, true)
	snapshot, _ := BuildWeather(ctx, api, myGID)
	return map[string]any{
		"outcome":       "collected",
		"friendGid":     strconv.FormatInt(friendGID, 10),
		"activityId":    strconv.FormatInt(reply.ActivityId, 10),
		"operateType":   strconv.FormatInt(reply.OperateType, 10),
		"rewards":       rewards,
		"weatherBefore": before,
		"weatherAfter":  after,
		"snapshot":      snapshot,
	}, nil
}

func weatherStatusFromDetail(detail *game.EnterReplyDetail, hostGID int64) map[string]any {
	if detail == nil {
		return weatherStatusDTO(nil, hostGID)
	}
	w := &weatherpb.WeatherStatus{WeatherType: detail.Weather, Status: detail.WeatherStatus}
	return weatherStatusDTO(w, hostGID)
}

// SummonThunderstorm uses the summon bottle on the own farm.
func SummonThunderstorm(ctx context.Context, api *game.API, myGID int64) (map[string]any, error) {
	before, err := api.GetWeatherStatus(ctx)
	if err != nil {
		return nil, err
	}
	beforeDTO := weatherStatusDTO(before.GetWeather(), myGID)
	if beforeDTO["active"].(bool) {
		return nil, weatherErr("WEATHER_ALREADY_ACTIVE", "自己的农场当前已有特殊天气，暂时无法召唤雷雨")
	}
	bag, err := api.Bag(ctx)
	if err != nil {
		return nil, err
	}
	stack := availableWeatherStack(game.GetBagItems(bag), summonBottleID)
	if stack == nil {
		return nil, weatherErr("WEATHER_SUMMON_UNAVAILABLE", "背包中没有可用的雷雨召唤瓶")
	}
	if _, err := api.UseTargeted(ctx, summonBottleID, stack.Uid, myGID, nil); err != nil {
		return nil, err
	}
	after, _ := api.GetWeatherStatus(ctx)
	snapshot, _ := BuildWeather(ctx, api, myGID)
	return map[string]any{
		"outcome":  "summoned",
		"weather":  weatherStatusDTO(after.GetWeather(), myGID),
		"snapshot": snapshot,
	}, nil
}

// UseWeatherFarmBottle uses the frog bottle (farm-level) on a friend farm.
func UseWeatherFarmBottle(ctx context.Context, api *game.API, friendGID, myGID, bottleID int64, landIDs []int64) (map[string]any, error) {
	if friendGID <= 0 || friendGID == myGID {
		return nil, weatherErr("INVALID_WEATHER_FRIEND_GID", "该道具只能在好友农场使用")
	}
	bag, err := api.Bag(ctx)
	if err != nil {
		return nil, err
	}
	stack := availableWeatherStack(game.GetBagItems(bag), bottleID)
	if stack == nil {
		return nil, weatherErr("WEATHER_BOTTLE_UNAVAILABLE", "背包中没有可用的道具")
	}
	detail, err := api.VisitEnterDetailed(ctx, friendGID, 2)
	if err != nil {
		return nil, err
	}
	defer func() { _ = api.VisitLeave(ctx, friendGID) }()
	// 乌云瓶：未指定地块时从可作用地块中取第一块；指定地块不在可作用集合内则拒绝。
	if bottleID == cloudMischiefBottleID {
		eligible := cloudEligibleLandIds(detail.Lands)
		if len(landIDs) == 0 {
			if len(eligible) == 0 {
				return nil, weatherErr("WEATHER_CLOUD_TARGET_UNAVAILABLE", "好友当前没有可使用乌云使坏瓶的作物")
			}
			landIDs = []int64{eligible[0]}
		} else {
			okLand := false
			for _, id := range eligible {
				if id == landIDs[0] {
					okLand = true
					break
				}
			}
			if !okLand {
				return nil, weatherErr("WEATHER_CLOUD_TARGET_UNAVAILABLE", "好友当前没有可使用乌云使坏瓶的作物")
			}
		}
	}
	reply, err := api.UseTargeted(ctx, bottleID, stack.Uid, friendGID, landIDs)
	if err != nil {
		return nil, err
	}
	outcome := "frog-used"
	if bottleID == cloudMischiefBottleID {
		outcome = "cloud-used"
	}
	rewards := make([]map[string]any, 0)
	for _, item := range append(append([]*corepb.Item{}, reply.GetItems()...), reply.GetLandReward().GetItems()...) {
		if item != nil {
			rewards = append(rewards, weatherItemDTO(item.Id, item.Count))
		}
	}
	snapshot, _ := BuildWeather(ctx, api, myGID)
	return map[string]any{
		"outcome":     outcome,
		"friendGid":   strconv.FormatInt(friendGID, 10),
		"rewards":     rewards,
		"snapshot":    snapshot,
	}, nil
}

// AdvanceWeatherResearch advances one research node (operate 40).
func AdvanceWeatherResearch(ctx context.Context, api *game.API, nodeID, myGID int64) (map[string]any, error) {
	if nodeID <= 0 {
		return nil, weatherErr("INVALID_WEATHER_RESEARCH_NODE", "nodeId 必须是正十进制整数")
	}
	snap, err := BuildWeather(ctx, api, myGID)
	if err != nil {
		return nil, err
	}
	if active, _ := snap["active"].(bool); !active {
		return nil, weatherErr("WEATHER_ACTIVITY_UNAVAILABLE", "雨落成诗活动尚未开放或已经结束")
	}
	research, _ := snap["research"].(map[string]any)
	if research == nil {
		return nil, weatherErr("WEATHER_RESEARCH_UNAVAILABLE", "服务端未返回气象研究数据")
	}
	nodes, _ := research["nodes"].([]map[string]any)
	var node map[string]any
	for _, entry := range nodes {
		if id, _ := entry["id"].(string); id == strconv.FormatInt(nodeID, 10) {
			node = entry
			break
		}
	}
	if node == nil {
		return nil, weatherErr("INVALID_WEATHER_RESEARCH_NODE", "气象研究节点不存在")
	}
	if completed, _ := node["completed"].(bool); completed {
		return nil, weatherErr("WEATHER_RESEARCH_ALREADY_COMPLETED", "该气象研究节点已经完成")
	}
	if available, _ := node["availableByStatus"].(bool); !available {
		return nil, weatherErr("WEATHER_RESEARCH_LOCKED", "请先完成前置气象研究节点")
	}
	if affordable, _ := node["affordable"].(bool); !affordable {
		return nil, weatherErr("INSUFFICIENT_LIGHTNING_BADGES", "雷电徽章不足")
	}
	req := &activitypb.AdvanceWeatherResearchRequest{
		ActivityId:  WeatherResearchActivityID,
		OperateType: operateWeatherResearch,
		WeatherResearchOperate: &activitypb.WeatherResearchOperateParams{NodeId: nodeID},
	}
	reply, err := api.OperateRaw(ctx, req, WeatherResearchActivityID, operateWeatherResearch)
	if err != nil {
		return nil, err
	}
	rewards := []map[string]any{}
	if reward, _ := node["reward"].(map[string]any); reward != nil {
		if id, _ := reward["id"].(string); id != "" && id != "0" {
			rewards = append(rewards, reward)
		}
	}
	snapshot, _ := BuildWeather(ctx, api, myGID)
	return map[string]any{
		"outcome":     "advanced",
		"nodeId":      strconv.FormatInt(nodeID, 10),
		"activityId":  strconv.FormatInt(reply.ActivityId, 10),
		"operateType": strconv.FormatInt(reply.OperateType, 10),
		"rewards":     rewards,
		"snapshot":    snapshot,
	}, nil
}

// --- 好友现场天气扫描（bot scanWeatherFriends，带 10 分钟缓存）---

type friendWeatherInspection struct {
	gid        int64
	weather    map[string]any
	inspectedAt int64
	scanError  string
}

var weatherMu sync.Mutex
var friendWeatherCache = map[int64]*friendWeatherInspection{}

// InspectFriendWeather enters one friend farm (low priority) and caches the
// on-site weather from the Enter reply (zero extra RPC).
func InspectFriendWeather(ctx context.Context, api *game.API, friendGID, myGID int64, force bool) map[string]any {
	weatherMu.Lock()
	if !force {
		if cached, ok := friendWeatherCache[friendGID]; ok && logic.GetServerTimeSec()-cached.inspectedAt <= friendWeatherCacheTTL {
			weatherMu.Unlock()
			return friendWeatherDTO(cached)
		}
	}
	weatherMu.Unlock()

	detail, err := api.VisitEnterDetailed(ctx, friendGID, 2)
	inspection := &friendWeatherInspection{gid: friendGID, inspectedAt: logic.GetServerTimeSec()}
	if err != nil {
		inspection.scanError = err.Error()
	} else {
		inspection.weather = weatherStatusFromDetail(detail, friendGID)
		_ = api.VisitLeave(ctx, friendGID)
	}
	weatherMu.Lock()
	friendWeatherCache[friendGID] = inspection
	weatherMu.Unlock()
	return friendWeatherDTO(inspection)
}

func friendWeatherDTO(inspection *friendWeatherInspection) map[string]any {
	if inspection == nil || inspection.weather == nil || inspection.scanError != "" {
		return map[string]any{
			"gid":        strconv.FormatInt(inspection.gid, 10),
			"inspected":  false,
			"scanError":  inspection.scanError,
			"availability": "unknown",
			"canCollect": false,
		}
	}
	state, reason := weatherAvailability(inspection.weather)
	return map[string]any{
		"gid":                strconv.FormatInt(inspection.gid, 10),
		"inspected":          true,
		"inspectedAt":        inspection.inspectedAt,
		"availability":       state,
		"availabilityReason": reason,
		"canCollect":         state == "available",
		"weather":            inspection.weather,
	}
}

// ClearFriendWeatherCache drops all cached friend inspections
// (bot: weatherChanged / activitiesChanged / disconnected 时清空)。
func ClearFriendWeatherCache() {
	weatherMu.Lock()
	friendWeatherCache = map[int64]*friendWeatherInspection{}
	weatherMu.Unlock()
}

// ScanFriendWeather inspects a batch of friends (≤5) with spacing.
// waitIdle（可为 nil）：进每位好友农场前等好友巡查空闲，等不到把剩下的交回前端稍后重试。
func ScanFriendWeather(ctx context.Context, api *game.API, gids []int64, myGID int64, waitIdle func(context.Context) bool) (map[string]any, error) {
	if len(gids) == 0 {
		return nil, weatherErr("INVALID_WEATHER_FRIEND_GID", "请先选择需要检查现场天气的好友")
	}
	if len(gids) > friendWeatherBatchMax {
		return nil, weatherErr("WEATHER_SCAN_BATCH_TOO_LARGE", fmt.Sprintf("单次最多检查 %d 位好友，请分批发起", friendWeatherBatchMax))
	}
	sort.Slice(gids, func(i, j int) bool { return gids[i] < gids[j] })
	friends := make([]map[string]any, 0, len(gids))
	deferred := []int64{}
	visited := 0
	for i, gid := range gids {
		if gid == myGID {
			continue
		}
		weatherMu.Lock()
		cached, hasFresh := friendWeatherCache[gid]
		fresh := hasFresh && logic.GetServerTimeSec()-cached.inspectedAt <= friendWeatherCacheTTL
		weatherMu.Unlock()
		if fresh {
			friends = append(friends, friendWeatherDTO(cached))
			continue
		}
		// 好友任务同样要进出好友农场，先给它让路；等不到空闲就交回前端稍后重试。
		if waitIdle != nil && !waitIdle(ctx) {
			deferred = append(deferred, gids[i:]...)
			break
		}
		// 逐位拉开进农场间隔，避免整批瞬时突发。
		if visited > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(friendWeatherScanGapMs * time.Millisecond):
			}
		}
		visited++
		friends = append(friends, InspectFriendWeather(ctx, api, gid, myGID, false))
	}
	return map[string]any{
		"outcome":     "scanned",
		"serverTime":  logic.GetServerTimeSec(),
		"friends":     friends,
		"deferredGids": deferred,
	}, nil
}

func weatherActivityActive(activity *activitypb.ActivityContent, serverTime int64) bool {
	if activity == nil {
		return false
	}
	if activity.BeginTime > 0 && serverTime < activity.BeginTime {
		return false
	}
	if activity.EndTime > 0 && serverTime > activity.EndTime {
		return false
	}
	return true
}

// GetWeatherFriends lists friend basics for the scan panel (no farm visits).
func GetWeatherFriends(ctx context.Context, s interface {
	FriendsList(ctx context.Context) (any, error)
}) (any, error) {
	return s.FriendsList(ctx)
}

// cloudEligibleLandIds mirrors bot cloudEligibleLandIds + isEligibleInteractionTarget(5006)：
// 未成熟（SEED..BLOOMING）且未作用过乌云瓶的地块，含占位地从属归并。
func cloudEligibleLandIds(lands []logic.LandInfo) []int64 {
	landsMap := logic.BuildLandMap(lands)
	result := make([]int64, 0, len(lands))
	seen := map[int64]struct{}{}
	for i := range lands {
		land := &lands[i]
		if !land.Unlocked {
			continue
		}
		// 多格占位：以源地块为准。
		source := land
		if land.MasterLandID > 0 {
			if master, ok := landsMap[land.MasterLandID]; ok {
				source = master
			}
		}
		if _, dup := seen[source.ID]; dup {
			continue
		}
		plant := source.Plant
		if plant == nil || plant.ID <= 0 || len(plant.Phases) == 0 {
			continue
		}
		current := logic.GetCurrentPhase(plant.Phases)
		if current == nil {
			continue
		}
		phase := current.Phase
		if phase < logic.PhaseSeed || phase >= logic.PhaseMature {
			continue
		}
		hasCloud := false
		for _, id := range plant.InteractionItemIDs {
			if id == cloudMischiefBottleID {
				hasCloud = true
				break
			}
		}
		if hasCloud {
			continue
		}
		seen[source.ID] = struct{}{}
		result = append(result, source.ID)
	}
	return result
}
