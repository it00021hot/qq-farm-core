package activitycenter

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/farm/game"
	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/activitypb"
)

// buildGreenPlum reports the type-12 brew activity (青梅 / 青酿换万金 and
// successors). The recurring event gets a fresh id every run. The registry is
// the primary source of schedule info; expired or unrecognized runs leave
// known=false so the UI can hide the tab.
func buildGreenPlum(ctx context.Context, api *game.API) map[string]any {
	out := map[string]any{
		"known":        false,
		"active":       false,
		"name":         "青梅",
		"balance":      "0",
		"balanceKnown": false,
		"ingredients":  []map[string]any{},
		"dailySeed": map[string]any{
			"claimed": false,
			"grantId": strconv.FormatInt(game.GreenPlumDailyGrantID, 10),
			"reward":  activityItemDTO(0, 0),
		},
		"currentRound": int64(0),
		"maxRounds":    int64(3),
		"started":      false,
		"finished":     false,
		"baseGold":     "0",
		"basePrice":    "0",
		"guaranteedPrice": "0",
		"quotePrices":  []string{},
		"quoteTotals":  []string{},
		"quote":        nil,
		"actions": map[string]any{
			"claimSeed": map[string]any{"enabled": false, "available": false},
			"start":     map[string]any{"enabled": false, "available": false},
			"continue":  map[string]any{"enabled": false, "available": false},
			"settle":    map[string]any{"enabled": false, "available": false},
		},
	}

	now := time.Now().Unix()
	items := logic.GreenPlumActivities()
	for i := range items {
		item := items[i]
		if item.BeginTime > 0 || item.EndTime > 0 || item.Name != "" {
			out["known"] = true
			out["activityId"] = item.ActivityID
			out["typeCode"] = item.Type
			if item.Name != "" {
				out["name"] = item.Name
			}
			out["startTime"] = item.BeginTime
			out["endTime"] = item.EndTime
			out["active"] = (item.BeginTime <= 0 || now >= item.BeginTime) &&
				(item.EndTime <= 0 || now <= item.EndTime)
			break
		}
	}
	if v, ok := out["known"].(bool); ok && v {
		refreshGreenPlumActions(out)
	}
	if api == nil {
		return out
	}

	// Only query registry-recognized entries. Do not fall back to hard-coded
	// dated activity ids — those expire each run and would keep the panel
	// "recognized" after the event ends.
	for i := range items {
		item := items[i]
		id, err := strconv.ParseInt(item.ActivityID, 10, 64)
		if err != nil || id <= 0 {
			continue
		}
		reply, err := api.QueryGreenPlum(ctx, id)
		if err != nil {
			continue
		}
		mergeGreenPlumReply(out, item, reply)
	}
	if v, ok := out["known"].(bool); ok && v {
		// The server's daily_seed.claimed flag is not always reliable; a claim
		// recorded by this process today is authoritative.
		if api.GreenPlumSeedClaimedToday() {
			if daily, ok := out["dailySeed"].(map[string]any); ok {
				daily["claimed"] = true
			}
		}
		ingredients, balance, balanceKnown := greenPlumIngredients(ctx, api)
		out["ingredients"] = ingredients
		out["balance"], out["balanceKnown"] = balance, balanceKnown
		now := time.Now().Unix()
		start := int64Value(out["startTime"])
		end := int64Value(out["endTime"])
		out["active"] = (start <= 0 || now >= start) && (end <= 0 || now <= end)
		refreshGreenPlumActions(out)
	}
	return out
}

// mergeGreenPlumReply folds one activity query reply into the green plum DTO.
// The daily seed entry carries qingmei_daily_seed; the brew entry carries
// qingmei_brew and, after an operate, qingmei_quote / qingmei_settlement.
func mergeGreenPlumReply(out map[string]any, item logic.ActivityRegistryItem, reply *activitypb.ActivityOperateReply) {
	if reply == nil || reply.Data == nil {
		return
	}
	if act := reply.Data.Activity; act != nil {
		out["known"] = true
		out["activityId"] = item.ActivityID
		if act.BeginTime > 0 || act.EndTime > 0 {
			out["startTime"] = strconv.FormatInt(act.BeginTime, 10)
			out["endTime"] = strconv.FormatInt(act.EndTime, 10)
		}
		if act.Name != "" {
			out["name"] = act.Name
		}
	}
	if seed := reply.Data.QingmeiDailySeed; seed != nil {
		out["dailyActivityId"] = item.ActivityID
		daily := map[string]any{
			"claimed": seed.Claimed,
			"grantId": strconv.FormatInt(game.GreenPlumDailyGrantID, 10),
			"reward":  activityItemDTO(0, 0),
		}
		if seed.Grant != nil {
			if seed.Grant.GrantId > 0 {
				daily["grantId"] = strconv.FormatInt(seed.Grant.GrantId, 10)
			}
			if seed.Grant.Item != nil {
				daily["reward"] = activityItemDTO(seed.Grant.Item.ItemId, seed.Grant.Item.Count)
			}
		}
		out["dailySeed"] = daily
	}
	if brew := reply.Data.QingmeiBrew; brew != nil {
		out["brewActivityId"] = item.ActivityID
		out["currentRound"] = brew.CurrentRound
		out["maxRounds"] = brew.MaxRounds
		if brew.MaxRounds <= 0 {
			out["maxRounds"] = int64(3)
		}
		out["started"] = brew.BaseGold > 0
		out["finished"] = brew.Finished
		out["baseGold"] = strconv.FormatInt(brew.BaseGold, 10)
		out["basePrice"] = strconv.FormatInt(brew.BasePrice, 10)
		out["guaranteedPrice"] = strconv.FormatInt(brew.GuaranteedPrice, 10)
		out["quotePrices"] = int64Strings(brew.QuotePrices)
		out["quoteTotals"] = int64Strings(brew.QuoteTotals)
	}
	if quote := reply.QingmeiQuote; quote != nil {
		out["quote"] = map[string]any{
			"round":     quote.Round,
			"unitPrice": strconv.FormatInt(quote.UnitPrice, 10),
			"totalGold": strconv.FormatInt(quote.TotalGold, 10),
			"doubled":   quote.Doubled,
		}
	}
	if settlement := reply.QingmeiSettlement; settlement != nil {
		out["settlement"] = map[string]any{
			"mode":      settlement.SettlementMode,
			"totalGold": strconv.FormatInt(settlement.TotalGold, 10),
		}
	}
}

func greenPlumIngredients(ctx context.Context, api *game.API) ([]map[string]any, string, bool) {
	bag, err := api.Bag(ctx)
	if err != nil {
		return nil, "0", false
	}
	out := make([]map[string]any, 0)
	var total int64
	for _, item := range game.GetBagItems(bag) {
		if item.Id != game.GreenPlumItemID || item.Count <= 0 {
			continue
		}
		uid := item.Uid
		if uid <= 0 {
			// System/balance entries without a UID are not concrete stacks and
			// cannot be invested, mirroring the reference bot.
			continue
		}
		total += item.Count
		mutantTypes := make([]string, 0, len(item.MutantTypes))
		for _, mt := range item.MutantTypes {
			if mt > 0 {
				mutantTypes = append(mutantTypes, strconv.FormatInt(mt, 10))
			}
		}
		dto := activityItemDTO(item.Id, item.Count)
		dto["uid"] = strconv.FormatInt(uid, 10)
		dto["key"] = fmt.Sprintf("%d:%s", uid, strings.Join(mutantTypes, ","))
		dto["mutantTypes"] = mutantTypes
		out = append(out, dto)
	}
	return out, strconv.FormatInt(total, 10), true
}

// refreshGreenPlumActions recomputes action availability from the merged DTO.
func refreshGreenPlumActions(g map[string]any) {
	currentRound := int64Value(g["currentRound"])
	maxRounds := int64Value(g["maxRounds"])
	if maxRounds <= 0 {
		maxRounds = 3
	}
	started := boolValue(g["started"])
	finished := boolValue(g["finished"])
	balance := int64Value(g["balance"])
	balanceKnown := boolValue(g["balanceKnown"])
	quoteTotals, _ := g["quoteTotals"].([]string)
	ingredients, _ := g["ingredients"].([]map[string]any)

	daily, _ := g["dailySeed"].(map[string]any)
	dailyClaimed := false
	if daily != nil {
		dailyClaimed = boolValue(daily["claimed"])
	}

	startEnabled := !started && (!balanceKnown || len(ingredients) > 0 || balance > 0)
	continueEnabled := started && !finished && currentRound < maxRounds
	settleEnabled := len(quoteTotals) > 0 || finished

	actions := map[string]any{
		"claimSeed": map[string]any{
			"enabled":   !dailyClaimed,
			"available": !dailyClaimed,
		},
		"start": map[string]any{
			"enabled":   startEnabled,
			"available": startEnabled,
		},
		"continue": map[string]any{
			"enabled":   continueEnabled,
			"available": continueEnabled,
		},
		"settle": map[string]any{
			"enabled":   settleEnabled,
			"available": settleEnabled,
		},
	}
	g["actions"] = actions
}

// QingMeiSeedRewardDTOs builds UI reward rows from a claim reply. The seed
// grant item lives in data.qingmei_daily_seed.grant.item; the flat rewards
// list is a fallback.
func QingMeiSeedRewardDTOs(reply *activitypb.ActivityOperateReply) []map[string]any {
	if reply == nil {
		return []map[string]any{}
	}
	if reply.Data != nil && reply.Data.QingmeiDailySeed != nil &&
		reply.Data.QingmeiDailySeed.Grant != nil && reply.Data.QingmeiDailySeed.Grant.Item != nil {
		g := reply.Data.QingmeiDailySeed.Grant.Item
		if g.ItemId > 0 {
			return []map[string]any{activityItemDTO(g.ItemId, g.Count)}
		}
	}
	rows := make([]map[string]any, 0, len(reply.Rewards))
	for _, r := range reply.Rewards {
		if r == nil || r.Id <= 0 {
			continue
		}
		rows = append(rows, activityItemDTO(r.Id, r.Count))
	}
	return rows
}

// QingMeiBrewDTO extracts the brew sub-state from an operate reply.
func QingMeiBrewDTO(reply *activitypb.ActivityOperateReply) map[string]any {
	out := map[string]any{
		"currentRound": int64(0),
		"maxRounds":    int64(3),
		"started":      false,
		"finished":     false,
		"baseGold":     "0",
		"quotePrices":  []string{},
		"quoteTotals":  []string{},
	}
	if reply == nil {
		return out
	}
	if brew := reply.Data.QingmeiBrew; brew != nil {
		out["currentRound"] = brew.CurrentRound
		out["maxRounds"] = brew.MaxRounds
		if brew.MaxRounds <= 0 {
			out["maxRounds"] = int64(3)
		}
		out["started"] = brew.BaseGold > 0
		out["finished"] = brew.Finished
		out["baseGold"] = strconv.FormatInt(brew.BaseGold, 10)
		out["quotePrices"] = int64Strings(brew.QuotePrices)
		out["quoteTotals"] = int64Strings(brew.QuoteTotals)
	}
	if reply.QingmeiQuote != nil {
		out["quote"] = map[string]any{
			"round":     reply.QingmeiQuote.Round,
			"unitPrice": strconv.FormatInt(reply.QingmeiQuote.UnitPrice, 10),
			"totalGold": strconv.FormatInt(reply.QingmeiQuote.TotalGold, 10),
			"doubled":   reply.QingmeiQuote.Doubled,
		}
	}
	return out
}

// QingMeiQuoteDTO extracts the latest quote from an operate reply.
func QingMeiQuoteDTO(reply *activitypb.ActivityOperateReply) map[string]any {
	if reply == nil || reply.QingmeiQuote == nil {
		return nil
	}
	return map[string]any{
		"round":     reply.QingmeiQuote.Round,
		"unitPrice": strconv.FormatInt(reply.QingmeiQuote.UnitPrice, 10),
		"totalGold": strconv.FormatInt(reply.QingmeiQuote.TotalGold, 10),
		"doubled":   reply.QingmeiQuote.Doubled,
	}
}

// QingMeiQuoteMessage renders a human message for the latest quote.
func QingMeiQuoteMessage(quote map[string]any) string {
	if quote == nil {
		return "酿造进度已更新"
	}
	round := int64Value(quote["round"])
	total := int64Value(quote["totalGold"])
	if total > 0 {
		return fmt.Sprintf("第 %d 轮报价：%d 金币", round, total)
	}
	return fmt.Sprintf("第 %d 轮报价已确认", round)
}

// QingMeiSettlementDTO extracts the settlement from an operate reply.
func QingMeiSettlementDTO(reply *activitypb.ActivityOperateReply) map[string]any {
	if reply == nil || reply.QingmeiSettlement == nil {
		return map[string]any{"mode": game.GreenPlumSharedSettlementMode, "totalGold": "0"}
	}
	return map[string]any{
		"mode":      reply.QingmeiSettlement.SettlementMode,
		"totalGold": strconv.FormatInt(reply.QingmeiSettlement.TotalGold, 10),
	}
}

// QingMeiSettlementRewardDTOs extracts reward rows from a settle reply.
func QingMeiSettlementRewardDTOs(reply *activitypb.ActivityOperateReply) []map[string]any {
	if reply == nil {
		return []map[string]any{}
	}
	if reply.QingmeiSettlement != nil && reply.QingmeiSettlement.Reward != nil &&
		reply.QingmeiSettlement.Reward.Id > 0 {
		return []map[string]any{activityItemDTO(reply.QingmeiSettlement.Reward.Id, reply.QingmeiSettlement.Reward.Count)}
	}
	rows := make([]map[string]any, 0, len(reply.Rewards))
	for _, r := range reply.Rewards {
		if r == nil || r.Id <= 0 {
			continue
		}
		rows = append(rows, activityItemDTO(r.Id, r.Count))
	}
	return rows
}

// QingMeiSettlementMessage renders a human message for a settlement reply.
func QingMeiSettlementMessage(reply *activitypb.ActivityOperateReply) string {
	if reply != nil && reply.QingmeiSettlement != nil && reply.QingmeiSettlement.TotalGold > 0 {
		return fmt.Sprintf("分享出售成功（1.5倍），获得 %d 金币", reply.QingmeiSettlement.TotalGold)
	}
	return "青梅酿已按分享奖励出售（1.5倍）"
}

func int64Strings(values []int64) []string {
	if len(values) == 0 {
		return []string{}
	}
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = strconv.FormatInt(v, 10)
	}
	return out
}

func int64Value(v any) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case float64:
		return int64(t)
	case string:
		n, _ := strconv.ParseInt(t, 10, 64)
		return n
	default:
		return 0
	}
}

func boolValue(v any) bool {
	b, _ := v.(bool)
	return b
}
