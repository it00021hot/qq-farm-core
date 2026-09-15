package activitycenter

// buildPetDiaryOnce — 萌宠日记快照构建（bot normalize 全字段）与操作（bot operatePetDiary 全前置）。

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/it00021hot/qq-farm-core/internal/farm/game"
	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/corepb"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/solartermspb"
)

type petStoryDescInfo struct {
	Photo string `json:"photo"`
	Say   any    `json:"say"`
}

func buildPetDiaryOnce(ctx context.Context, api *game.API) (map[string]any, error) {
	catalog, assets, err := loadPetDiaryCatalog()
	if err != nil {
		return nil, err
	}
	pet, seeds, shopChild, err := readPetDiaryGroup(ctx, api)
	if err != nil {
		return nil, err
	}
	shopData, shopErr := api.PetDiaryQueryShop(ctx)
	if shopErr == nil && shopData != nil && shopData.Shop != nil {
		shopChild = shopData
	}
	bag, bagErr := petBalances(ctx, api)
	warnings := []string{}
	if shopErr != nil {
		warnings = append(warnings, "拾物小铺："+shopErr.Error())
		shopChild = nil
	}
	if bagErr != nil {
		warnings = append(warnings, "背包读取失败，消耗资源的操作已暂停")
	}
	var solar *solartermspb.GetSolarTermsReply
	if solarReply, solarErr := api.GetSolarTerms(ctx); solarErr == nil {
		solar = solarReply
	} else {
		warnings = append(warnings, "节令小礼读取失败，请稍后刷新")
	}

	state := pet.PetTreasureHunt
	nurture := state.GetNurture()
	hunt := state.GetHunt()
	battle := state.GetBattle()
	head := pet.GetHead()
	active := petDiaryHeadActive(head)
	var base struct{ growthAdult, feedLimit, treasureLimit int64 }
	if len(catalog.ActivityPetTreasureHuntBase) > 0 {
		base.growthAdult = catalog.ActivityPetTreasureHuntBase[0].GrowthAdultThreshold
		base.feedLimit = catalog.ActivityPetTreasureHuntBase[0].DailyFeedLimit
		base.treasureLimit = catalog.ActivityPetTreasureHuntBase[0].DailyTreasureLimit
	}
	var fightLimit int64
	if len(catalog.ActivityPetTreasureHuntFight) > 0 {
		fightLimit = catalog.ActivityPetTreasureHuntFight[0].DailyBattleLimit
	}
	var refreshCfg struct{ freeLimit, paidLimit, costID, costCount int64 }
	if len(catalog.ActivityPetTreasureCharmRefresh) > 0 {
		refreshCfg.freeLimit = catalog.ActivityPetTreasureCharmRefresh[0].FreeRefreshDailyLimit
		refreshCfg.paidLimit = catalog.ActivityPetTreasureCharmRefresh[0].ManualRefreshDailyLimit
		refreshCfg.costID = catalog.ActivityPetTreasureCharmRefresh[0].ManualRefreshCostID
		refreshCfg.costCount = catalog.ActivityPetTreasureCharmRefresh[0].ManualRefreshCostCount
	}
	adult := nurture.GetStage() == 2
	feedCount := state.GetFeed().GetFeedCount()

	// seeds
	seedDays := []map[string]any{}
	canClaimSeeds := false
	if seeds != nil {
		for _, reward := range seeds.GetMegaEvent().GetRewards() {
			if reward == nil {
				continue
			}
			if reward.Claimable && !reward.Claimed {
				canClaimSeeds = true
			}
			seedDays = append(seedDays, map[string]any{
				"day": reward.UnlockDay, "claimed": reward.Claimed,
				"claimable": reward.Claimable, "rewards": petItemRows(reward.Reward),
			})
		}
	}

	// shop goods
	goods := []map[string]any{}
	if shopChild != nil && shopChild.GetShop() != nil {
		for _, g := range shopChild.GetShop().GetGoods() {
			if g == nil {
				continue
			}
			safeCosts := true
			for _, c := range g.Cost {
				if c.GetId() == petDiamondID {
					safeCosts = false
					break
				}
			}
			remaining := int64(-1)
			if g.PurchaseLimit > 0 {
				remaining = g.PurchaseLimit - g.PurchasedCount
				if remaining < 0 {
					remaining = 0
				}
			}
			desc := map[string]any{}
			_ = json.Unmarshal([]byte(g.Desc), &desc)
			resPath, _ := desc["res"].(string)
			image := petDiaryImage(assets, resPath)
			if image == "" && len(g.Item) > 0 && g.Item[0] != nil {
				image = logic.SeedImagePath(g.Item[0].Id)
			}
			category := g.CategoryTag
			if category == "" {
				category = "游记好礼"
			}
			goods = append(goods, map[string]any{
				"id": strconv.FormatInt(g.Id, 10), "name": g.Name, "image": image,
				"rewards": petShopItemRows(g.Item), "costs": petShopItemRows(g.Cost),
				"limit":     strconv.FormatInt(g.PurchaseLimit, 10),
				"purchased": strconv.FormatInt(g.PurchasedCount, 10),
				"remaining": remaining,
				"exchangeable": active && petDiaryHeadActive(shopChild.GetHead()) && safeCosts &&
					g.DiamondCostCount == 0 && remaining != 0 && petCostsAvailable(petShopCosts(g.Cost), bag, 1),
				"safeCosts": safeCosts, "order": g.Order, "category": category,
			})
		}
		sort.SliceStable(goods, func(i, j int) bool {
			return goods[i]["order"].(int64) < goods[j]["order"].(int64)
		})
	}

	// stories（含照片/配图映射）
	stories := []map[string]any{}
	for _, s := range state.GetStory().GetStories() {
		if s == nil {
			continue
		}
		var desc petStoryDescInfo
		_ = json.Unmarshal([]byte(s.SelectedDesc), &desc)
		// 素材收敛（bot 20260911 / rust ee8ac0e）：故事仅保留照片映射，
		// captionImage/caption 字段移除。
		stories = append(stories, map[string]any{
			"order": s.Order, "unlocked": s.Unlocked, "claimed": s.Claimed, "animated": s.Animated,
			"photo": petDiaryImage(assets, desc.Photo),
		})
	}

	// treasures
	treasures := []map[string]any{}
	for _, t := range state.GetPool().GetTreasures() {
		if t == nil {
			continue
		}
		treasures = append(treasures, petTreasureDTO(t))
	}

	// charms
	poolIDs := battle.CharmDailyPool
	equippedIDs := battle.CharmEquipped
	pool := make([]map[string]any, 0, len(poolIDs))
	for _, id := range poolIDs {
		pool = append(pool, petCharmDTO(catalog, assets, battle, int64(id)))
	}
	equipped := make([]map[string]any, 0, len(equippedIDs))
	for _, id := range equippedIDs {
		equipped = append(equipped, petCharmDTO(catalog, assets, battle, int64(id)))
	}
	allCharms := make([]map[string]any, 0, len(catalog.ActivityPetTreasureHuntCharm))
	for i := range catalog.ActivityPetTreasureHuntCharm {
		allCharms = append(allCharms, petCharmDTO(catalog, assets, battle, catalog.ActivityPetTreasureHuntCharm[i].CharmID))
	}
	freeRefreshRemaining := refreshCfg.freeLimit - battle.GetCharmFreeRefreshCount()
	if freeRefreshRemaining < 0 {
		freeRefreshRemaining = 0
	}
	paidRefreshCount := battle.GetCharmPaidRefreshCount()
	paidRefreshRemaining := refreshCfg.paidLimit - paidRefreshCount
	if paidRefreshRemaining < 0 {
		paidRefreshRemaining = 0
	}
	charmNeedsChoice := len(equippedIDs) > 0 && !battle.GetCharmPickUsed()
	refreshCost := []*corepb.Item{{Id: refreshCfg.costID, Count: refreshCfg.costCount}}
	canChooseCharm := active && adult && !battle.GetCharmPickUsed() && len(poolIDs) > 0
	canRefresh := active && adult && !charmNeedsChoice &&
		(freeRefreshRemaining > 0 || (paidRefreshRemaining > 0 && petCostsAvailable(refreshCost, bag, 1)))

	// solar terms（限定活动窗口内）
	var solarDTO map[string]any
	if solar != nil {
		terms := []map[string]any{}
		for _, term := range solar.Terms {
			if term == nil {
				continue
			}
			if term.EndTime >= head.GetStartTime() && term.BeginTime <= head.GetEndTime() {
				terms = append(terms, map[string]any{
					// Name 是 proto bytes（UTF-8 文本），必须解码否则前端显示 base64 乱码。
					"id": term.TermId, "name": bytesText(term.Name),
					"startTime": term.BeginTime, "endTime": term.EndTime,
					"status": term.Status, "canClaim": term.Status == 2,
				})
			}
		}
		solarDTO = map[string]any{"terms": terms}
	}

	headDesc := map[string]any{}
	_ = json.Unmarshal([]byte(head.GetDesc()), &headDesc)

	plants := []int64{20516, 29004, 25995, 21625, 20154, 21072}
	plantRows := make([]map[string]any, 0, len(plants))
	for _, id := range plants {
		count := int64(0)
		if bag != nil {
			count = bag[id]
		}
		plantRows = append(plantRows, petItemRow(id, count))
	}

	return map[string]any{
		"activityId": strconv.FormatInt(PetDiaryPetID, 10),
		"groupId":    strconv.FormatInt(PetDiaryGroupID, 10),
		"title":      "萌宠成长日记",
		"active":     active,
		"startTime":  head.GetStartTime() * 1000,
		"endTime":    head.GetEndTime() * 1000,
		"serverTime": logic.GetServerTimeSec() * 1000,
		"rules":        activityRulesDTO([]byte(head.GetDesc())),
		"treasureRules": treasureRulesFromDesc(headDesc),
		"warnings":   warnings,
		"balances":   petBalanceRows(bag),
		"nurture": map[string]any{
			"initialized": nurture.GetCgPlayed(), "adult": adult,
			"growth": nurture.GetGrowth(), "adultGrowth": base.growthAdult,
			"dogGranted": nurture.GetDogGranted(), "feedCount": feedCount, "feedLimit": base.feedLimit,
			"feedCosts": petItemRows(petFeedCosts(catalog)),
			"canFeed":   active && !adult && nurture.GetStage() == 1 && int64(feedCount) < base.feedLimit && petCostsAvailable(petFeedCosts(catalog), bag, 1),
		},
		"hunt": map[string]any{
			"count": hunt.GetTreasureCount(), "limit": base.treasureLimit,
			"total":          strconv.FormatInt(hunt.GetTreasureTotal(), 10),
			"luckyStarTotal": strconv.FormatInt(hunt.GetLuckyStarGainedTotal(), 10),
			"costs":          petItemRows(hunt.TreasureCost),
			"canDraw":        active && adult && int64(hunt.GetTreasureCount()) < base.treasureLimit && petCostsAvailable(hunt.TreasureCost, bag, 1),
			"canPlunder":     active && hunt.GetCanPlayPlunder() && int64(battle.GetBattleCount()) < fightLimit,
		},
		"seeds": map[string]any{
			"canClaim": active && seeds != nil && petDiaryHeadActive(seeds.GetHead()) && canClaimSeeds,
			"days":     seedDays,
		},
		"stories": stories,
		"charms": map[string]any{
			"pool": pool, "equipped": equipped, "all": allCharms,
			"picked": battle.GetCharmPickUsed(), "canChoose": canChooseCharm,
			"freeRefreshRemaining": freeRefreshRemaining, "freeRefreshLimit": refreshCfg.freeLimit,
			"paidRefreshCount": paidRefreshCount, "paidRefreshRemaining": paidRefreshRemaining, "paidRefreshLimit": refreshCfg.paidLimit,
			"refreshCost":    petItemRow(refreshCfg.costID, refreshCfg.costCount),
			"refreshBalance": bagStr(bag, refreshCfg.costID),
			"canRefresh":    canRefresh,
			"refreshNote": fmt.Sprintf("每日免费 %d 次，之后每次 %d 点券，今日还可付费刷新 %d 次。点券不足时不刷新。",
				refreshCfg.freeLimit, refreshCfg.costCount, paidRefreshRemaining),
		},
		"treasures":          treasures,
		"compensationCount":  strconv.FormatInt(state.GetPlunder().GetPlunderCompensationCount(), 10),
		"hasUnreadPlundered": state.GetPlunder().GetHasUnreadPlunderedLog(),
		"battleCount":        battle.GetBattleCount(),
		"battleLimit":        fightLimit,
		"skipBattle":         battle.GetIsSkipBattleCg(),
		"shop":               goods,
		"solarTerms":         solarDTO,
		"plants":             plantRows,
	}, nil
}

func bagStr(bag map[int64]int64, id int64) any {
	if bag == nil {
		return nil
	}
	return strconv.FormatInt(bag[id], 10)
}

func treasureRulesFromDesc(headDesc map[string]any) []string {
	tips2, _ := headDesc["tips2"].(map[string]any)
	if tips2 == nil {
		return []string{}
	}
	txt, _ := tips2["txt"].([]any)
	out := []string{}
	for _, entry := range txt {
		if s, ok := entry.(string); ok && strings.TrimSpace(s) != "" {
			out = append(out, strings.TrimSpace(s))
		}
	}
	return out
}
