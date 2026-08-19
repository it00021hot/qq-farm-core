package activitycenter

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/it00021hot/qq-farm-core/internal/farm/game"
	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/activitypb"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/corepb"
)

type activityError struct {
	Code    string
	Message string
}

func (e activityError) Error() string {
	if e.Code == "" {
		return e.Message
	}
	return e.Code + ": " + e.Message
}

func qixiErr(code, message string) error {
	return activityError{Code: code, Message: message}
}

func findQixiChild(group *activitypb.ActivityData, activityID int64) *activitypb.ActivityData {
	if group == nil {
		return nil
	}
	for _, child := range group.Children {
		if child == nil || child.Activity == nil {
			continue
		}
		if child.Activity.ActivityId == activityID {
			return child
		}
	}
	return nil
}

func qixiActivityActive(activity *activitypb.ActivityContent, serverTime int64) bool {
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

func bagBalances(ctx context.Context, api *game.API, ids []int64) map[int64]int64 {
	out := make(map[int64]int64, len(ids))
	for _, id := range ids {
		out[id] = 0
	}
	if api == nil {
		return out
	}
	bag, err := api.Bag(ctx)
	if err != nil {
		return nil
	}
	wanted := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		wanted[id] = struct{}{}
	}
	for _, item := range game.GetBagItems(bag) {
		if _, ok := wanted[item.Id]; !ok {
			continue
		}
		if item.Count > 0 {
			out[item.Id] += item.Count
		}
	}
	return out
}

func qixiDTO(groupReply *activitypb.GetGroupReply, balances map[int64]int64) (map[string]any, error) {
	if groupReply == nil || groupReply.Group == nil {
		return nil, qixiErr("QIXI_UNAVAILABLE", "服务端未发现鹊桥寄情活动")
	}
	bridgeChild := findQixiChild(groupReply.Group, game.QixiBridgeActivityID)
	giftChild := findQixiChild(groupReply.Group, game.QixiGiftActivityID)
	if giftChild == nil {
		giftChild = bridgeChild
	}
	if bridgeChild == nil || bridgeChild.Activity == nil || giftChild == nil || giftChild.Activity == nil {
		return nil, qixiErr("QIXI_UNAVAILABLE", "服务端未发现鹊桥寄情活动")
	}
	bridgeActivity := bridgeChild.Activity
	serverTime := logic.GetServerTimeSec()
	if serverTime <= 0 {
		serverTime = 0
	}
	config := bridgeChild.QixiBridge
	if config == nil {
		config = &activitypb.QixiBridgeConfig{}
	}
	gift := giftChild.QixiGift
	if gift == nil {
		gift = &activitypb.QixiGiftProgress{}
	}
	currentStage := config.CurrentStage
	bridgeClaimable := bridgeActivity.Field_23 != 0
	stages := make([]map[string]any, 0, len(config.Stages))
	for _, stage := range config.Stages {
		if stage == nil {
			continue
		}
		statusCode := strconv.FormatInt(stage.Status, 10)
		completed := statusCode == "2" || (currentStage > 0 && stage.Stage > 0 && stage.Stage <= currentStage)
		claimable := bridgeClaimable && stage.Stage == currentStage
		rewards := make([]map[string]any, 0, len(stage.Rewards))
		for _, reward := range stage.Rewards {
			if reward == nil {
				continue
			}
			rewards = append(rewards, activityItemDTO(reward.ItemId, reward.Count))
		}
		var cost map[string]any
		if stage.Cost != nil {
			cost = activityItemDTO(stage.Cost.ItemId, stage.Cost.Count)
		} else {
			cost = activityItemDTO(0, 0)
		}
		stages = append(stages, map[string]any{
			"id":        strconv.FormatInt(stage.Stage, 10),
			"stage":     stage.Stage,
			"statusCode": statusCode,
			"completed": completed,
			"claimed":   completed && !claimable,
			"claimable": claimable,
			"current":   stage.Stage == currentStage,
			"cost":      cost,
			"rewards":   rewards,
		})
	}
	readBalance := func(id int64) *string {
		if balances == nil {
			return nil
		}
		v := strconv.FormatInt(balances[id], 10)
		return &v
	}
	featherBalance := readBalance(game.QixiFeatherItemID)
	sachetBalance := readBalance(game.QixiSachetItemID)
	receivedBalance := readBalance(game.QixiReceivedSachetItemID)
	active := qixiActivityActive(bridgeActivity, serverTime)
	sachetCount := int64(0)
	if sachetBalance != nil {
		sachetCount, _ = strconv.ParseInt(*sachetBalance, 10, 64)
	}
	giftEnabled := active && (balances == nil || sachetCount > 0)
	displayItems := make([]map[string]any, 0, len(config.DisplayItems))
	for _, item := range config.DisplayItems {
		if item == nil {
			continue
		}
		displayItems = append(displayItems, activityItemDTO(item.ItemId, item.Count))
	}
	exchange := map[string]any{
		"sentItem":     activityItemDTO(0, 0),
		"receivedItem": activityItemDTO(0, 0),
		"field3":       false,
		"enabled":      false,
	}
	if gift.Exchange != nil {
		if gift.Exchange.SentItem != nil {
			exchange["sentItem"] = activityItemDTO(gift.Exchange.SentItem.ItemId, gift.Exchange.SentItem.Count)
		}
		if gift.Exchange.ReceivedItem != nil {
			exchange["receivedItem"] = activityItemDTO(gift.Exchange.ReceivedItem.ItemId, gift.Exchange.ReceivedItem.Count)
		}
		exchange["field3"] = gift.Exchange.Field_3
		exchange["enabled"] = gift.Exchange.Enabled
	}
	zero := "0"
	featherCount := zero
	if featherBalance != nil {
		featherCount = *featherBalance
	}
	sachetCountStr := zero
	if sachetBalance != nil {
		sachetCountStr = *sachetBalance
	}
	receivedCountStr := zero
	if receivedBalance != nil {
		receivedCountStr = *receivedBalance
	}
	return map[string]any{
		"groupId":          strconv.FormatInt(game.QixiGroupID, 10),
		"bridgeActivityId": strconv.FormatInt(game.QixiBridgeActivityID, 10),
		"giftActivityId":   strconv.FormatInt(game.QixiGiftActivityID, 10),
		"activityId":       strconv.FormatInt(game.QixiBridgeActivityID, 10),
		"name":             stringOr(bridgeActivity.Name, "鹊桥寄情"),
		"title":            stringOr(bridgeActivity.Name, "鹊桥寄情"),
		"startTime":        strconv.FormatInt(bridgeActivity.BeginTime, 10),
		"endTime":          strconv.FormatInt(bridgeActivity.EndTime, 10),
		"serverTime":       strconv.FormatInt(serverTime, 10),
		"active":           active,
		"rules":            activityRulesDTO(bridgeActivity.Extra),
		"feather":          activityItemDTO(game.QixiFeatherItemID, parseCount(featherCount)),
		"sachet":           activityItemDTO(game.QixiSachetItemID, parseCount(sachetCountStr)),
		"receivedSachet":   activityItemDTO(game.QixiReceivedSachetItemID, parseCount(receivedCountStr)),
		"balances": map[string]any{
			"feather":        featherBalance,
			"sachet":         sachetBalance,
			"receivedSachet": receivedBalance,
			"known":          balances != nil,
		},
		"bridge": map[string]any{
			"currentStage": currentStage,
			"stages":       stages,
			"claimable":    bridgeClaimable,
			"rewardRedDot": bridgeClaimable,
			"displayItems": displayItems,
		},
		"gift": map[string]any{
			"sentCount":  strconv.FormatInt(gift.SentCount, 10),
			"field2Code": strconv.FormatInt(gift.Field_2, 10),
			"field3Code": strconv.FormatInt(gift.Field_3, 10),
			"exchange":   exchange,
		},
		"actions": map[string]any{
			"bridge": map[string]any{"enabled": active && bridgeClaimable, "available": active && bridgeClaimable, "availabilityKnown": true},
			"gift":   map[string]any{"enabled": giftEnabled, "available": giftEnabled, "availabilityKnown": balances != nil},
		},
	}, nil
}

func stringOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func activityRulesDTO(extra []byte) map[string]any {
	trimmed := strings.TrimSpace(string(extra))
	empty := map[string]any{"title": "", "paragraphs": []string{}}
	if trimmed == "" {
		return empty
	}
	var parsed any
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		return map[string]any{"title": "", "paragraphs": []string{trimmed}}
	}
	obj, _ := parsed.(map[string]any)
	if obj == nil {
		return map[string]any{"title": "", "paragraphs": []string{trimmed}}
	}
	if tips, ok := obj["tips"].(map[string]any); ok {
		title, _ := tips["title"].(string)
		paragraphs := stringSlice(tips["txt"])
		if len(paragraphs) > 0 {
			return map[string]any{"title": strings.TrimSpace(title), "paragraphs": paragraphs}
		}
	}
	if paragraphs := stringSlice(obj["paragraphs"]); len(paragraphs) > 0 {
		title, _ := obj["title"].(string)
		return map[string]any{"title": strings.TrimSpace(title), "paragraphs": paragraphs}
	}
	return empty
}

func parseCount(v string) int64 {
	n, _ := strconv.ParseInt(v, 10, 64)
	return n
}

// BuildQixi loads the 鹊桥寄情 DTO. Missing activity is not an error for snapshot.
func BuildQixi(ctx context.Context, api *game.API) (map[string]any, error) {
	if api == nil {
		return nil, qixiErr("QIXI_UNAVAILABLE", "游戏连接未就绪")
	}
	reply, err := api.GetActivityGroup(ctx, game.QixiGroupID)
	if err != nil {
		return nil, err
	}
	balances := bagBalances(ctx, api, []int64{
		game.QixiFeatherItemID,
		game.QixiSachetItemID,
		game.QixiReceivedSachetItemID,
	})
	return qixiDTO(reply, balances)
}

// ClaimQixiBridge claims the current stage reward and returns a snapshot extra payload.
func ClaimQixiBridge(ctx context.Context, api *game.API) (map[string]any, error) {
	activity, err := BuildQixi(ctx, api)
	if err != nil {
		return nil, err
	}
	actions, _ := activity["actions"].(map[string]any)
	bridge, _ := actions["bridge"].(map[string]any)
	if enabled, _ := bridge["enabled"].(bool); !enabled {
		return nil, qixiErr("QIXI_BRIDGE_UNAVAILABLE", "当前没有可领取的鹊桥奖励")
	}
	reply, err := api.ClaimQixiBridgeRewards(ctx)
	if err != nil {
		return nil, err
	}
	claimed := make([]string, 0)
	rewards := make([]map[string]any, 0)
	if reply.QixiBridgeResult != nil {
		for _, stage := range reply.QixiBridgeResult.ClaimedStages {
			claimed = append(claimed, strconv.FormatInt(stage, 10))
		}
		rewards = append(rewards, coreItemsToDTOs(reply.QixiBridgeResult.Rewards)...)
	}
	if len(rewards) == 0 {
		rewards = append(rewards, coreItemsToDTOs(reply.Rewards)...)
	}
	message := "鹊桥奖励领取成功"
	if len(claimed) > 0 {
		message = fmt.Sprintf("已完成第 %s 阶段鹊桥并领取奖励", strings.Join(claimed, "、"))
	}
	return map[string]any{
		"claimedStages": claimed,
		"rewards":       rewards,
		"message":       message,
	}, nil
}

// GiftQixiSachet gifts sachets to a friend.
func GiftQixiSachet(ctx context.Context, api *game.API, friendGID, count int64) (map[string]any, error) {
	if friendGID <= 0 {
		return nil, qixiErr("INVALID_QIXI_FRIEND_GID", "好友 GID 必须是正十进制整数")
	}
	if count <= 0 {
		return nil, qixiErr("INVALID_QIXI_SACHET_COUNT", "赠送数量必须是正十进制整数")
	}
	activity, err := BuildQixi(ctx, api)
	if err != nil {
		return nil, err
	}
	actions, _ := activity["actions"].(map[string]any)
	gift, _ := actions["gift"].(map[string]any)
	if enabled, _ := gift["enabled"].(bool); !enabled {
		return nil, qixiErr("QIXI_GIFT_UNAVAILABLE", "当前无法赠送鹊羽香囊")
	}
	balances, _ := activity["balances"].(map[string]any)
	if known, _ := balances["known"].(bool); known {
		sachet := "0"
		if v, ok := balances["sachet"].(*string); ok && v != nil {
			sachet = *v
		} else if v, ok := balances["sachet"].(string); ok {
			sachet = v
		}
		have, _ := strconv.ParseInt(sachet, 10, 64)
		if have < count {
			return nil, qixiErr("INSUFFICIENT_QIXI_SACHET", "鹊羽香囊数量不足")
		}
	}
	if _, err := api.GiftQixiSachet(ctx, friendGID, count); err != nil {
		return nil, err
	}
	return map[string]any{
		"friendGid": strconv.FormatInt(friendGID, 10),
		"count":     strconv.FormatInt(count, 10),
		"message":   fmt.Sprintf("已赠送 %d 个鹊羽香囊", count),
	}, nil
}

func coreItemsToDTOs(items []*corepb.Item) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, activityItemDTO(item.Id, item.Count))
	}
	return out
}

type gameplayBinding struct {
	GameplayKey  string
	DetailTarget string
	Priority     int
}

func buildActivityDirectory(windows []logic.ActivityWindow, season, shop, solarTerms, constellation, qixi map[string]any) []map[string]any {
	bindings := map[string][]gameplayBinding{}
	add := func(ids []string, key, target string, priority int) {
		for _, id := range ids {
			id = strings.TrimSpace(id)
			if id == "" || id == "0" {
				continue
			}
			list := bindings[id]
			dup := false
			for _, existing := range list {
				if existing.GameplayKey == key && existing.DetailTarget == target {
					dup = true
					break
				}
			}
			if dup {
				continue
			}
			list = append(list, gameplayBinding{GameplayKey: key, DetailTarget: target, Priority: priority})
			bindings[id] = list
		}
	}
	seasonPass, _ := season["pass"].(map[string]any)
	add([]string{fmt.Sprint(seasonPass["activityId"])}, "stellar", "travel", 10)
	add([]string{fmt.Sprint(constellation["activityId"])}, "stellar", "constellation", 20)
	add([]string{fmt.Sprint(shop["activityId"])}, "stellar", "shop", 30)
	solarIDs := []string{fmt.Sprint(nested(solarTerms, "currentConfig", "activityId"))}
	if configs, ok := solarTerms["configs"].([]map[string]any); ok {
		for _, cfg := range configs {
			solarIDs = append(solarIDs, fmt.Sprint(cfg["activityId"]))
		}
	}
	add(solarIDs, "stellar", "solar", 40)
	add([]string{
		fmt.Sprint(qixi["groupId"]),
		fmt.Sprint(qixi["bridgeActivityId"]),
		fmt.Sprint(qixi["giftActivityId"]),
	}, "qixi", "qixi", 50)

	type group struct {
		id, name     string
		start, end   int64
		activityIDs  []string
	}
	groups := make([]group, 0)
	for _, window := range windows {
		id := strings.TrimSpace(window.ID)
		if id == "" {
			continue
		}
		name := strings.TrimSpace(window.Name)
		if name == "" {
			name = "活动 " + id
		}
		matched := -1
		for i := range groups {
			g := groups[i]
			if g.name != name {
				continue
			}
			if (g.end <= 0 || window.BeginTime <= 0 || g.end >= window.BeginTime) &&
				(window.EndTime <= 0 || g.start <= 0 || window.EndTime >= g.start) {
				matched = i
				break
			}
		}
		if matched >= 0 {
			g := groups[matched]
			g.activityIDs = append(g.activityIDs, id)
			if g.start > 0 && window.BeginTime > 0 {
				if window.BeginTime < g.start {
					g.start = window.BeginTime
				}
			} else if window.BeginTime > g.start {
				g.start = window.BeginTime
			}
			if window.EndTime > g.end {
				g.end = window.EndTime
			}
			if !strings.HasSuffix(g.id, "00") && strings.HasSuffix(id, "00") {
				g.id = id
			}
			groups[matched] = g
			continue
		}
		groups = append(groups, group{id: id, name: name, start: window.BeginTime, end: window.EndTime, activityIDs: []string{id}})
	}
	out := make([]map[string]any, 0, len(groups))
	for _, g := range groups {
		matches := make([]gameplayBinding, 0)
		seen := map[string]struct{}{}
		for _, id := range g.activityIDs {
			for _, binding := range bindings[id] {
				key := binding.GameplayKey + ":" + binding.DetailTarget
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				matches = append(matches, binding)
			}
		}
		for i := 0; i < len(matches); i++ {
			for j := i + 1; j < len(matches); j++ {
				if matches[j].Priority < matches[i].Priority {
					matches[i], matches[j] = matches[j], matches[i]
				}
			}
		}
		gameplayKeys := make([]string, 0)
		targets := make([]string, 0, len(matches))
		for _, m := range matches {
			dup := false
			for _, existing := range gameplayKeys {
				if existing == m.GameplayKey {
					dup = true
					break
				}
			}
			if !dup {
				gameplayKeys = append(gameplayKeys, m.GameplayKey)
			}
			targets = append(targets, m.DetailTarget)
		}
		var gameplayKey, detailTarget any
		if len(gameplayKeys) > 0 {
			gameplayKey = gameplayKeys[0]
		}
		if len(targets) > 0 {
			detailTarget = targets[0]
		}
		out = append(out, map[string]any{
			"id":              g.id,
			"name":            g.name,
			"startTime":       g.start,
			"endTime":         g.end,
			"activityIds":     g.activityIDs,
			"gameplayKey":     gameplayKey,
			"gameplayKeys":    gameplayKeys,
			"detailTarget":    detailTarget,
			"gameplayTargets": targets,
		})
	}
	return out
}

func nested(root map[string]any, keys ...string) any {
	cur := any(root)
	for _, key := range keys {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = m[key]
	}
	return cur
}
