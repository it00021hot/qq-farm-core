package activitycenter

// 萌宠日记操作（bot operatePetDiary 全前置校验 + 记录/好友信息查询）。

import (
	"context"
	"strconv"

	"github.com/it00021hot/qq-farm-core/internal/farm/game"
	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/activitypb"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/corepb"
)

// GetPetDiaryRecords returns interact (31) or plundered (44) logs.
func GetPetDiaryRecords(ctx context.Context, api *game.API, kind string) ([]map[string]any, error) {
	if kind != "interact" && kind != "plunder" {
		return nil, petDiaryErr("未知记录类型")
	}
	operateType := operatePetQueryLog
	req := &activitypb.PetDiaryOperateRequest{PetTreasureHuntGetLog: &activitypb.PetTreasureHuntGetLogReq{}}
	if kind == "plunder" {
		operateType = operatePetPlunderedLog
		req = &activitypb.PetDiaryOperateRequest{PetTreasureHuntGetPlunderedLog: &activitypb.PetTreasureHuntGetPlunderedLogReq{}}
	}
	reply, err := api.PetDiaryOperate(ctx, req, PetDiaryPetID, operateType)
	if err != nil {
		return nil, err
	}
	logs := []map[string]any{}
	if kind == "interact" {
		for _, entry := range reply.GetPetTreasureHuntGetLog().GetLogs() {
			if entry == nil {
				continue
			}
			skins := []string{}
			for _, id := range entry.DogSkinIds {
				skins = append(skins, strconv.FormatInt(id, 10))
			}
			logs = append(logs, map[string]any{
				"time": entry.Ts * 1000, "type": entry.Type,
				"costs": petItemRows(entry.Costs), "rewards": petItemRows(entry.Rewards),
				"dogId": strconv.FormatInt(entry.DogId, 10), "skins": skins,
			})
		}
		return logs, nil
	}
	for _, entry := range reply.GetPetTreasureHuntGetPlunderedLog().GetLogs() {
		if entry == nil {
			continue
		}
		attackerCharms, defenderCharms := []int64{}, []int64{}
		for _, id := range entry.AttackerCharm {
			attackerCharms = append(attackerCharms, int64(id))
		}
		for _, id := range entry.DefenderCharm {
			defenderCharms = append(defenderCharms, int64(id))
		}
		logs = append(logs, map[string]any{
			"time": entry.Ts * 1000, "attackerGid": strconv.FormatInt(entry.AttackerGid, 10),
			"name": entry.AttackerName, "won": entry.AttackerWon, "treasureId": entry.TreasureId,
			"challenge": petItemRow(entry.ChallengeItemId, 1), "level": entry.AttackerLevel,
			"attackerCharms": attackerCharms, "defenderCharms": defenderCharms,
			"lost": petItemRows(entry.LostItems), "injected": petItemRows(entry.InjectedItems),
			"fake": entry.IsFake,
		})
	}
	return logs, nil
}

// GetPetDiaryFriendInfo returns one friend's treasures + defender charms (operate 47).
func GetPetDiaryFriendInfo(ctx context.Context, api *game.API, friendGID int64) (map[string]any, error) {
	if friendGID <= 0 {
		return nil, petDiaryErr("好友 GID 必须是正十进制整数")
	}
	req := &activitypb.PetDiaryOperateRequest{
		PetTreasureHuntGetFriendActivityInfo: &activitypb.PetTreasureHuntGetFriendActivityInfoReq{FriendGid: friendGID},
	}
	reply, err := api.PetDiaryOperate(ctx, req, PetDiaryPetID, operatePetFriendInfo)
	if err != nil {
		return nil, err
	}
	result := reply.GetPetTreasureHuntGetFriendActivityInfo()
	if result.GetGid() != friendGID {
		return nil, petDiaryErr("好友响应不匹配")
	}
	treasures := []map[string]any{}
	for _, t := range result.GetInfo().GetTreasures() {
		if t != nil {
			treasures = append(treasures, petTreasureDTO(t))
		}
	}
	charms := []int64{}
	for _, id := range result.GetInfo().GetDefenderCharmIds() {
		charms = append(charms, int64(id))
	}
	return map[string]any{"gid": friendGID, "treasures": treasures, "charms": charms}, nil
}

// OperatePetDiary runs one action with the full bot preconditions.
func OperatePetDiary(ctx context.Context, api *game.API, action string, opts map[string]any) (map[string]any, error) {
	operateType, ok := petDiaryCommands[action]
	if !ok {
		return nil, petDiaryErr("未知萌宠操作")
	}
	pet, seeds, _, err := readPetDiaryGroup(ctx, api)
	if err != nil {
		return nil, err
	}
	if !petDiaryHeadActive(pet.GetHead()) {
		return nil, petDiaryErr("萌宠成长日记当前不在活动时间内")
	}
	state := pet.PetTreasureHunt
	nurture := state.GetNurture()
	battle := state.GetBattle()
	catalog, _, err := loadPetDiaryCatalog()
	if err != nil {
		return nil, err
	}
	bag, err := petBalances(ctx, api)
	if err != nil {
		return nil, err
	}

	activityID := PetDiaryPetID
	req := &activitypb.PetDiaryOperateRequest{}
	switch action {
	case "feed":
		if nurture.GetStage() != 1 || int64(state.GetFeed().GetFeedCount()) >= catalog.ActivityPetTreasureHuntBase[0].DailyFeedLimit {
			return nil, petDiaryErr("当前不可投喂")
		}
		if !petCostsAvailable(petFeedCosts(catalog), bag, 1) {
			return nil, petDiaryErr("萌宠元气糕不足，请先种植活动作物")
		}
		req.PetTreasureHuntFeed = &activitypb.PetTreasureHuntFeedReq{}
	case "draw":
		if nurture.GetStage() != 2 || int64(state.GetHunt().GetTreasureCount()) >= catalog.ActivityPetTreasureHuntBase[0].DailyTreasureLimit {
			return nil, petDiaryErr("当前不可寻宝")
		}
		if !petCostsAvailable(state.GetHunt().TreasureCost, bag, 1) {
			return nil, petDiaryErr("寻宝消耗不足")
		}
		req.PetTreasureHuntDraw = &activitypb.PetTreasureHuntDrawReq{}
	case "initialize":
		if nurture.GetCgPlayed() {
			return nil, petDiaryErr("已领养比熊，请刷新状态")
		}
		req.PetTreasureHuntFinishCg = &activitypb.PetTreasureHuntFinishCgReq{}
	case "claimDog":
		if nurture.GetStage() != 2 || nurture.GetDogGranted() {
			return nil, petDiaryErr("比熊尚未成年或已经领取")
		}
		req.PetTreasureHuntClaimDog = &activitypb.PetTreasureHuntClaimDogReq{}
	case "story":
		order, _ := opts["order"].(int64)
		if order <= 0 {
			return nil, petDiaryErr("手记编号必须是正十进制整数")
		}
		found := false
		for _, s := range state.GetStory().GetStories() {
			if s != nil && int64(s.Order) == order && s.Unlocked && !s.Claimed {
				found = true
				break
			}
		}
		if !found {
			return nil, petDiaryErr("手记尚未解锁或已领取")
		}
		req.PetTreasureHuntClaimStory = &activitypb.PetTreasureHuntClaimStoryReq{Order: int32(order)}
	case "seeds":
		claimable := false
		if seeds != nil && petDiaryHeadActive(seeds.GetHead()) {
			for _, reward := range seeds.GetMegaEvent().GetRewards() {
				if reward != nil && reward.Claimable && !reward.Claimed {
					claimable = true
					break
				}
			}
		}
		if !claimable {
			return nil, petDiaryErr("当前没有可领取的种子礼包")
		}
		activityID = PetDiarySeedsID
		req.MegaEventClaimAll = &activitypb.PetTreasureHuntFinishCgReq{}
	case "refreshCharm":
		if nurture.GetStage() != 2 {
			return nil, petDiaryErr("比熊成年后才可刷新锦囊")
		}
		// 先选择/保留当前锦囊才能刷新。
		if len(battle.GetCharmEquipped()) > 0 && !battle.GetCharmPickUsed() {
			return nil, petDiaryErr("请先替换或保留当前锦囊，再刷新")
		}
		free := int64(battle.GetCharmFreeRefreshCount()) < catalog.ActivityPetTreasureCharmRefresh[0].FreeRefreshDailyLimit
		// 付费/计数是面板前置条件，不在游戏请求里；过期免费点击绝不能变成付费刷新，
		// 重复付费点击也不允许扣两次。
		payment, _ := opts["payment"].(string)
		if payment == "diamonds" || (payment != "" && payment != "free" && payment != "tickets") {
			return nil, petDiaryErr("锦囊刷新不支持使用钻石")
		}
		if free {
			if payment != "" && payment != "free" {
				return nil, petDiaryErr("刷新次数已变化，请刷新状态后重试")
			}
		} else {
			if payment != "tickets" {
				return nil, petDiaryErr("免费刷新已用完，请确认点券费用后再操作")
			}
			paidCount := battle.GetCharmPaidRefreshCount()
			if paidCount >= catalog.ActivityPetTreasureCharmRefresh[0].ManualRefreshDailyLimit {
				return nil, petDiaryErr("今日付费刷新次数已用完")
			}
			expected, hasExpected := opts["expectedPaidRefreshCount"].(int64)
			if !hasExpected || expected != paidCount {
				return nil, petDiaryErr("刷新次数已变化，请刷新状态后重试")
			}
			// 实机命令 41 扣 1002 x30；发送前重读点券余额，官方客户端余额不足会回退钻石。
			refreshCost := []*corepb.Item{{
				Id:    catalog.ActivityPetTreasureCharmRefresh[0].ManualRefreshCostID,
				Count: catalog.ActivityPetTreasureCharmRefresh[0].ManualRefreshCostCount,
			}}
			if !petCostsAvailable(refreshCost, bag, 1) {
				return nil, petDiaryErr("点券不足，已停止刷新，不使用钻石")
			}
		}
		req.PetTreasureHuntRefreshCharmPool = &activitypb.PetTreasureHuntRefreshCharmPoolReq{}
	case "equipCharm":
		charmID, _ := opts["charmId"].(int64)
		if charmID <= 0 {
			return nil, petDiaryErr("锦囊编号必须是正十进制整数")
		}
		valid := false
		for _, id := range append(append([]int32{}, battle.CharmDailyPool...), battle.CharmEquipped...) {
			if int64(id) == charmID {
				valid = true
				break
			}
		}
		if nurture.GetStage() != 2 || battle.GetCharmPickUsed() || !valid {
			return nil, petDiaryErr("该锦囊不可选择或本轮已经选择")
		}
		req.PetTreasureHuntEquipCharms = &activitypb.PetTreasureHuntEquipCharmsReq{CharmIds: []int32{int32(charmID)}}
	case "openTreasure":
		// 必须有已完成护送的宝藏（status=3，或 status=2 且到期）。
		now := logic.GetServerTimeSec()
		eligible := false
		for _, t := range state.GetPool().GetTreasures() {
			if t == nil {
				continue
			}
			if t.Status == 3 || (t.Status == 2 && t.EndAt > 0 && t.EndAt <= now) {
				eligible = true
				break
			}
		}
		if !eligible {
			return nil, petDiaryErr("还没有完成护送的宝藏")
		}
		req.PetTreasureHuntOpenTreasure = &activitypb.PetTreasureHuntOpenTreasureReq{}
	case "compensation":
		if state.GetPlunder().GetPlunderCompensationCount() <= 0 {
			return nil, petDiaryErr("当前没有可领取的夺宝补偿")
		}
		req.PetTreasureHuntClaimPlunderCompensation = &activitypb.PetTreasureHuntClaimPlunderCompensationReq{}
	case "battle":
		if !state.GetHunt().GetCanPlayPlunder() || int64(battle.GetBattleCount()) >= catalog.ActivityPetTreasureHuntFight[0].DailyBattleLimit {
			return nil, petDiaryErr("当前不可夺宝")
		}
		gid, _ := opts["gid"].(int64)
		challengeID, _ := opts["challengeId"].(int64)
		treasureID, _ := opts["treasureId"].(int64)
		if gid <= 0 {
			return nil, petDiaryErr("好友 GID 必须是正十进制整数")
		}
		if challengeID <= 0 {
			return nil, petDiaryErr("挑战书编号必须是正十进制整数")
		}
		if !(challengeID == 80101 || challengeID == 80102 || challengeID == 80103) {
			return nil, petDiaryErr("挑战书类型无效")
		}
		// 先取好友活动信息验证宝藏状态，状态变化时拒绝开打。
		friend, err := GetPetDiaryFriendInfo(ctx, api, gid)
		if err != nil {
			return nil, err
		}
		var treasure map[string]any
		for _, t := range friend["treasures"].([]map[string]any) {
			if id, _ := t["id"].(string); id == strconv.FormatInt(treasureID, 10) {
				treasure = t
				break
			}
		}
		if treasure == nil {
			return nil, petDiaryErr("好友宝藏状态已变化，请重新查看")
		}
		if status, _ := treasure["status"].(int32); status != 2 {
			return nil, petDiaryErr("好友宝藏状态已变化，请重新查看")
		}
		canStart := false
		for _, p := range treasure["previews"].([]map[string]any) {
			if cid, _ := p["challengeId"].(string); cid == strconv.FormatInt(challengeID, 10) {
				if startable, _ := p["canStart"].(bool); startable {
					canStart = true
					break
				}
			}
		}
		if !canStart {
			return nil, petDiaryErr("好友宝藏状态已变化，请重新查看")
		}
		if !petCostsAvailable([]*corepb.Item{{Id: challengeID, Count: 1}}, bag, 1) {
			return nil, petDiaryErr("对应挑战书不足")
		}
		req.PetTreasureHuntStartBattle = &activitypb.PetTreasureHuntStartBattleReq{
			DefenderGid: gid, TreasureId: strconv.FormatInt(treasureID, 10), ChallengeItemId: challengeID,
		}
	case "skipBattle":
		skip, hasSkip := opts["skip"].(bool)
		if !hasSkip {
			return nil, petDiaryErr("跳过动画设置无效")
		}
		req.PetTreasureHuntSetSkipBattleCg = &activitypb.PetTreasureHuntSetSkipBattleCgReq{Skip: skip}
	case "markStories":
		ordersRaw, _ := opts["orders"].([]int64)
		if len(ordersRaw) == 0 {
			return nil, petDiaryErr("手记编号无效")
		}
		for _, order := range ordersRaw {
			found := false
			for _, s := range state.GetStory().GetStories() {
				if s != nil && int64(s.Order) == order && s.Unlocked {
					found = true
					break
				}
			}
			if !found {
				return nil, petDiaryErr("手记编号无效")
			}
		}
		orders32 := make([]int32, 0, len(ordersRaw))
		for _, o := range ordersRaw {
			orders32 = append(orders32, int32(o))
		}
		req.PetTreasureHuntMarkStoryAnimated = &activitypb.PetTreasureHuntMarkStoryAnimatedReq{Orders: orders32}
	case "exchange":
		activityID = PetDiaryShopID
		goodsID, _ := opts["goodsId"].(int64)
		count, _ := opts["count"].(int64)
		if goodsID <= 0 {
			return nil, petDiaryErr("商品编号必须是正十进制整数")
		}
		if count <= 0 {
			count = 1
		}
		shopData, shopErr := api.PetDiaryQueryShop(ctx)
		if shopErr != nil || shopData == nil {
			return nil, petDiaryErr("拾物小铺当前不可兑换")
		}
		if !petDiaryHeadActive(shopData.GetHead()) {
			return nil, petDiaryErr("拾物小铺当前不可兑换")
		}
		var goods *activitypb.PetDiaryShopGoodsInfo
		for _, g := range shopData.GetShop().GetGoods() {
			if g != nil && g.Id == goodsID {
				goods = g
				break
			}
		}
		if goods == nil {
			return nil, petDiaryErr("服务端目录未发现该商品")
		}
		if goods.DiamondCostCount > 0 {
			return nil, petDiaryErr("该商品可能消耗钻石，已阻止兑换")
		}
		for _, c := range goods.Cost {
			if c.GetId() == petDiamondID {
				return nil, petDiaryErr("该商品可能消耗钻石，已阻止兑换")
			}
		}
		if goods.PurchaseLimit > 0 && goods.PurchasedCount+count > goods.PurchaseLimit {
			return nil, petDiaryErr("兑换数量超过剩余限购次数")
		}
		if !petCostsAvailable(petShopCosts(goods.Cost), bag, count) {
			return nil, petDiaryErr("兑换余额不足")
		}
		req.ShopBuy = &activitypb.PetDiaryShopBuyReq{GoodsId: goodsID, Count: count}
	}

	reply, err := api.PetDiaryOperate(ctx, req, activityID, operateType)
	if err != nil {
		return nil, err
	}
	result := petDiaryResult(action, reply)
	snapshot, snapErr := BuildPetDiary(ctx, api)
	refreshError := ""
	if snapErr != nil {
		refreshError = "操作已成功，刷新失败：" + snapErr.Error()
		snapshot = nil
	}
	message := "操作成功"
	if action == "battle" {
		if reply.GetPetTreasureHuntStartBattle().GetWon() {
			message = "夺宝成功"
		} else {
			message = "本次夺宝未获胜，已按规则结算"
		}
	}
	return map[string]any{
		"action":       action,
		"rewards":      petDiaryRewards(action, reply),
		"costs":        petItemRows(petDiaryCosts(action, reply)),
		"result":       result,
		"snapshot":     snapshot,
		"refreshError": refreshError,
		"message":      message,
	}, nil
}

// petDiaryResult extracts the per-action reply payload (bot result passthrough).
func petDiaryResult(action string, reply *activitypb.PetDiaryOperateReply) map[string]any {
	switch action {
	case "feed":
		r := reply.GetPetTreasureHuntFeed()
		return map[string]any{"growth": r.GetGrowth(), "stage": r.GetStage(), "becameAdult": r.GetBecameAdult(), "feedCount": r.GetFeedCount(), "times": r.GetTimes()}
	case "draw":
		r := reply.GetPetTreasureHuntDraw()
		return map[string]any{"treasureTotal": r.GetTreasureTotal(), "treasureCount": r.GetTreasureCount(), "times": r.GetTimes()}
	case "refreshCharm":
		r := reply.GetPetTreasureHuntRefreshCharmPool()
		pool := []int64{}
		for _, id := range r.GetCharmDailyPool() {
			pool = append(pool, int64(id))
		}
		return map[string]any{"charmDailyPool": pool, "refreshTs": r.GetRefreshTs(), "freeRefresh": r.GetFreeRefresh(), "freeRefreshCount": r.GetFreeRefreshCount(), "paidRefreshCount": r.GetPaidRefreshCount()}
	case "equipCharm":
		equipped := []int64{}
		for _, id := range reply.GetPetTreasureHuntEquipCharms().GetCharmEquipped() {
			equipped = append(equipped, int64(id))
		}
		return map[string]any{"charmEquipped": equipped}
	case "battle":
		r := reply.GetPetTreasureHuntStartBattle()
		return map[string]any{"won": r.GetWon(), "plundered": r.GetPlundered(), "defenderName": r.GetDefenderName(), "streakTriggered": r.GetStreakTriggered(), "attackerDice": r.GetAttackerDice(), "defenderDice": r.GetDefenderDice()}
	case "story":
		return map[string]any{"desc": reply.GetPetTreasureHuntClaimStory().GetDesc()}
	case "claimDog":
		return map[string]any{"dogId": reply.GetPetTreasureHuntClaimDog().GetDogId()}
	case "skipBattle":
		return map[string]any{"skip": reply.GetPetTreasureHuntSetSkipBattleCg().GetIsSkipBattleCg()}
	case "seeds":
		days := []int64{}
		for _, d := range reply.GetMegaEventClaimAll().GetNewlyClaimedDays() {
			days = append(days, d)
		}
		return map[string]any{"newlyClaimedDays": days}
	default:
		return map[string]any{}
	}
}

func petDiaryRewards(action string, reply *activitypb.PetDiaryOperateReply) []map[string]any {
	switch action {
	case "feed":
		return petItemRows(reply.GetPetTreasureHuntFeed().GetRewards())
	case "draw":
		return petItemRows(reply.GetPetTreasureHuntDraw().GetRewards())
	case "battle":
		out := petItemRows(reply.GetPetTreasureHuntStartBattle().GetRewards())
		if r := reply.GetPetTreasureHuntStartBattle().GetStreakReward(); r != nil {
			out = append(out, petItemRow(r.Id, r.Count))
		}
		return out
	case "openTreasure":
		return petItemRows(reply.GetPetTreasureHuntOpenTreasure().GetRewards())
	case "compensation":
		return petItemRows(reply.GetPetTreasureHuntClaimPlunderCompensation().GetRewards())
	case "story":
		return petItemRows(reply.GetPetTreasureHuntClaimStory().GetAwards())
	case "seeds":
		return petItemRows(reply.GetMegaEventClaimAll().GetAwards())
	case "exchange":
		return petItemRows(reply.GetShopBuy().GetAwards())
	default:
		return []map[string]any{}
	}
}

func petDiaryCosts(action string, reply *activitypb.PetDiaryOperateReply) []*corepb.Item {
	switch action {
	case "feed":
		return reply.GetPetTreasureHuntFeed().GetCosts()
	case "draw":
		return reply.GetPetTreasureHuntDraw().GetCosts()
	case "refreshCharm":
		return reply.GetPetTreasureHuntRefreshCharmPool().GetCosts()
	case "exchange":
		return reply.GetShopBuy().GetCosts()
	default:
		return nil
	}
}
