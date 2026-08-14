package game

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/activitypb"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/seasonpb"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/solartermspb"
)

// beijingDateKey returns YYYY-MM-DD in UTC+8 using synced server time when
// available. Mirrors runtime.beijingDateKey so the game package can gate
// daily activity claims on the same calendar day.
func beijingDateKey() string {
	nowSec := logic.GetServerTimeSec()
	var t time.Time
	if nowSec > 0 {
		t = time.Unix(nowSec, 0).UTC()
	} else {
		t = time.Now().UTC()
	}
	bj := t.Add(8 * time.Hour)
	return fmt.Sprintf("%04d-%02d-%02d", bj.Year(), int(bj.Month()), bj.Day())
}

const (
	activityService = "gamepb.activitypb.ActivityService"
	seasonService   = "gamepb.seasonpb.SeasonService"
	solarService    = "gamepb.solartermspb.SolarTermsService"

	shopActivityType          int64 = 3
	constellationActivityType int64 = 13
	operateExchangeShop       int64 = 1
	operateQueryShop          int64 = 7
	operateLightConstellation int64 = 21

	// 青梅（青酿换万金）活动操作码，与抓包确认的协议一致：
	// 领每日种子(4) / 查询酿造(7) / 开始酿造(14) / 继续酿造(15) / 结算出售(16)。
	operateClaimGreenPlumSeed int64 = 4
	operateQueryGreenPlum     int64 = 7
	operateStartGreenPlumBrew int64 = 14
	operateContinueGreenPlum  int64 = 15
	operateSettleGreenPlum    int64 = 16

	// GreenPlumItemID 是青梅物品 id。
	GreenPlumItemID int64 = 41221
	// GreenPlumDailyActivityID / GreenPlumBrewActivityID 是青梅（青酿换万金）活动的
	// 每日种子领奖与酿造两个活动 id，与参考项目 liyangpengs/qq-farm-bot 对齐。
	// 每日种子活动为 …01，酿造活动为 …02。
	GreenPlumDailyActivityID int64 = 2026081201
	GreenPlumBrewActivityID   int64 = 2026081202
	// GreenPlumDailyGrantID 是每日种子礼包的 grant_id。
	GreenPlumDailyGrantID int64 = 3
	// greenPlumShareField1/4 是青梅结算前的分享上报场景参数。
	greenPlumShareField1 int32 = 11
	greenPlumShareField4 int32 = 215
	// GreenPlumSharedSettlementMode 是按分享奖励(1.5倍)结算的模式。
	GreenPlumSharedSettlementMode int64 = 2
	// greenPlumAlreadyClaimedCode 是每日种子已领取的错误码。
	greenPlumAlreadyClaimedCode = "1034014"
)

func ShopActivityType() int64          { return shopActivityType }
func ConstellationActivityType() int64 { return constellationActivityType }

func (a *API) sendActivity(ctx context.Context, method string, body []byte) ([]byte, error) {
	if err := a.requireSender(); err != nil {
		return nil, err
	}
	logActivityRequest(method, body)
	raw, _, err := a.Sender.Send(ctx, activityService, method, nonNilBody(body))
	if err == nil {
		logActivityReply(method, raw)
	}
	return raw, err
}

// logActivityRequest records the activity RPC selector for protocol sniffing.
func logActivityRequest(method string, body []byte) {
	req := &activitypb.QueryActivityRequest{}
	if err := unmarshalMessage(body, req); err != nil {
		return
	}
	slog.Info("activity rpc request",
		"method", method,
		"activityId", req.ActivityId,
		"operateType", req.OperateType,
	)
}

// logActivityReply records the activity RPC reply summary for protocol sniffing.
func logActivityReply(method string, raw []byte) {
	reply := &activitypb.ActivityOperateReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		slog.Debug("activity rpc reply decode", "method", method, "err", err)
		return
	}
	detail := map[string]any{
		"activityId":  reply.ActivityId,
		"operateType": reply.OperateType,
	}
	if reply.Data != nil {
		if reply.Data.Activity != nil {
			detail["name"] = reply.Data.Activity.Name
			detail["type"] = reply.Data.Activity.Type
			detail["groupId"] = reply.Data.Activity.GroupId
			detail["beginTime"] = reply.Data.Activity.BeginTime
			detail["endTime"] = reply.Data.Activity.EndTime
			detail["sortOrder"] = reply.Data.Activity.SortOrder
			detail["field20"] = reply.Data.Activity.Field_20
			detail["field23"] = reply.Data.Activity.Field_23
			if len(reply.Data.Activity.Extra) > 0 {
				detail["extra"] = reply.Data.Activity.Extra
			}
		}
		if reply.Data.Catalog != nil {
			detail["catalogGoods"] = len(reply.Data.Catalog.Goods)
		}
		if reply.Data.Constellation != nil {
			detail["constellationNodes"] = len(reply.Data.Constellation.Nodes)
			detail["constellationGroups"] = len(reply.Data.Constellation.Groups)
		}
		if reply.Data.QingmeiDailySeed != nil {
			seed := reply.Data.QingmeiDailySeed
			detail["qingmeiDailyClaimed"] = seed.Claimed
			if seed.Grant != nil {
				detail["qingmeiGrantId"] = seed.Grant.GrantId
				if seed.Grant.Item != nil {
					detail["qingmeiGrantItem"] = map[string]any{
						"itemId": seed.Grant.Item.ItemId,
						"count":  seed.Grant.Item.Count,
					}
				}
			}
		}
		if brew := reply.Data.QingmeiBrew; brew != nil {
			detail["qingmeiBrew"] = map[string]any{
				"baseGold":          brew.BaseGold,
				"currentRound":      brew.CurrentRound,
				"maxRounds":         brew.MaxRounds,
				"finished":          brew.Finished,
				"quotePrices":       brew.QuotePrices,
				"quoteTotals":       brew.QuoteTotals,
				"ingredientItemIds": brew.IngredientItemIds,
			}
		}
	}
	if reply.QingmeiBrewStarted != nil {
		detail["qingmeiBrewStarted"] = reply.QingmeiBrewStarted.BaseGold
	}
	if reply.QingmeiQuote != nil {
		detail["qingmeiQuote"] = map[string]any{
			"round":      reply.QingmeiQuote.Round,
			"unitPrice":  reply.QingmeiQuote.UnitPrice,
			"totalGold":  reply.QingmeiQuote.TotalGold,
			"doubled":    reply.QingmeiQuote.Doubled,
		}
	}
	if reply.QingmeiSettlement != nil {
		detail["qingmeiSettlement"] = map[string]any{
			"mode":      reply.QingmeiSettlement.SettlementMode,
			"totalGold": reply.QingmeiSettlement.TotalGold,
		}
	}
	if len(reply.Rewards) > 0 {
		rewards := make([]map[string]any, 0, len(reply.Rewards))
		for _, r := range reply.Rewards {
			if r == nil {
				continue
			}
			rewards = append(rewards, map[string]any{"itemId": r.Id, "count": r.Count})
		}
		detail["rewards"] = rewards
	}
	slog.Info("activity rpc reply", "method", method, "detail", detail)
}

func (a *API) sendSeason(ctx context.Context, method string, body []byte) ([]byte, error) {
	if err := a.requireSender(); err != nil {
		return nil, err
	}
	raw, _, err := a.Sender.Send(ctx, seasonService, method, nonNilBody(body))
	return raw, err
}

func (a *API) sendSolar(ctx context.Context, method string, body []byte) ([]byte, error) {
	if err := a.requireSender(); err != nil {
		return nil, err
	}
	raw, _, err := a.Sender.Send(ctx, solarService, method, nonNilBody(body))
	return raw, err
}

// GetSeasonInfo fetches current season and pass progress.
func (a *API) GetSeasonInfo(ctx context.Context) (*seasonpb.GetSeasonInfoReply, error) {
	raw, err := a.sendSeason(ctx, "GetSeasonInfo", marshalMessage(&seasonpb.GetSeasonInfoRequest{}))
	if err != nil {
		return nil, err
	}
	reply := &seasonpb.GetSeasonInfoReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	RegisterSeasonActivities(reply.SeasonInfo)
	return reply, nil
}

// RegisterSeasonActivities registers season sub-activities into the logic
// activity registry so activity-restricted logic (e.g. fruit selling) can gate
// on real activity schedules instead of defaulting to "unknown = active".
func RegisterSeasonActivities(season *seasonpb.SeasonInfo) {
	if season == nil {
		return
	}
	items := make([]logic.ActivityRegistryItem, 0, len(season.Activities))
	for i := range season.Activities {
		act := season.Activities[i]
		items = append(items, logic.ActivityRegistryItem{
			ActivityID: strconv.FormatInt(act.ActivityId, 10),
			Type:       act.Type,
			Name:       strings.TrimSpace(string(act.Name)),
			BeginTime:  act.BeginTime,
			EndTime:    act.EndTime,
		})
	}
	logic.RegisterActivities(items)
}

// ClaimBattlePassRewards claims all eligible pass nodes.
func (a *API) ClaimBattlePassRewards(ctx context.Context) (*seasonpb.ClaimBattlePassRewardsReply, error) {
	raw, err := a.sendSeason(ctx, "ClaimBattlePassRewards", marshalMessage(&seasonpb.ClaimBattlePassRewardsRequest{}))
	if err != nil {
		return nil, err
	}
	reply := &seasonpb.ClaimBattlePassRewardsReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// GetSolarTerms fetches solar term list.
func (a *API) GetSolarTerms(ctx context.Context) (*solartermspb.GetSolarTermsReply, error) {
	raw, err := a.sendSolar(ctx, "GetSolarTerms", marshalMessage(&solartermspb.GetSolarTermsRequest{}))
	if err != nil {
		return nil, err
	}
	reply := &solartermspb.GetSolarTermsReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// ClaimSolarTerms claims one solar term reward.
func (a *API) ClaimSolarTerms(ctx context.Context, termID int64) (*solartermspb.ClaimSolarTermsReply, error) {
	req := &solartermspb.ClaimSolarTermsRequest{TermId: termID}
	raw, err := a.sendSolar(ctx, "ClaimSolarTerms", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &solartermspb.ClaimSolarTermsReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// QueryActivityShop loads star-sand shop catalog.
func (a *API) QueryActivityShop(ctx context.Context, activityID int64) (*activitypb.ActivityOperateReply, error) {
	req := &activitypb.QueryActivityRequest{
		ActivityId:  activityID,
		OperateType: operateQueryShop,
	}
	raw, err := a.sendActivity(ctx, "Operate", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &activitypb.ActivityOperateReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	if reply.ActivityId != activityID || reply.OperateType != operateQueryShop {
		return nil, fmt.Errorf("activity shop query: unexpected reply activity=%d operate=%d", reply.ActivityId, reply.OperateType)
	}
	if reply.Data == nil || reply.Data.Catalog == nil {
		return nil, fmt.Errorf("activity shop query: missing catalog")
	}
	return reply, nil
}

// ExchangeShopGoods exchanges star-sand shop goods.
func (a *API) ExchangeShopGoods(ctx context.Context, activityID, goodsID, count int64) (*activitypb.ActivityOperateReply, error) {
	req := &activitypb.ExchangeShopRequest{
		ActivityId:  activityID,
		OperateType: operateExchangeShop,
		ExchangeShopOperate: &activitypb.ExchangeShopOperateParams{
			GoodsId: goodsID,
			Count:   count,
		},
	}
	raw, err := a.sendActivity(ctx, "Operate", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &activitypb.ActivityOperateReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	if reply.ActivityId != activityID || reply.OperateType != operateExchangeShop {
		return nil, fmt.Errorf("activity shop exchange: unexpected reply activity=%d operate=%d", reply.ActivityId, reply.OperateType)
	}
	if reply.Data == nil || reply.Data.Catalog == nil {
		return nil, fmt.Errorf("activity shop exchange: missing catalog")
	}
	return reply, nil
}

// LightConstellation lights today's constellation node.
func (a *API) LightConstellation(ctx context.Context, activityID int64) (*activitypb.ActivityOperateReply, error) {
	req := &activitypb.OperateConstellationRequest{
		ActivityId:  activityID,
		OperateType: operateLightConstellation,
		Field_119:   &activitypb.OperateConstellationRequest_Empty{},
	}
	raw, err := a.sendActivity(ctx, "Operate", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &activitypb.ActivityOperateReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	if reply.Data == nil || reply.Data.Constellation == nil {
		return nil, fmt.Errorf("activity constellation: missing constellation data")
	}
	return reply, nil
}

// IsConstellationAlreadyClaimed reports gateway error 1034038.
func IsConstellationAlreadyClaimed(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "error code=1034038")
}

// IsAlreadyClaimedGreenPlum reports gateway error 1034014, the confirmed
// "already claimed today" code for the daily 青梅 seed grant.
func IsAlreadyClaimedGreenPlum(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "error code="+greenPlumAlreadyClaimedCode)
}

// ClaimGreenPlumSeed claims today's 青梅 seed reward (operateType=4). On a
// successful reply the claim date is cached on the API so snapshot building
// can treat the daily seed as already claimed today.
func (a *API) ClaimGreenPlumSeed(ctx context.Context, activityID int64) (*activitypb.ActivityOperateReply, error) {
	req := &activitypb.ClaimQingMeiDailySeedRequest{
		ActivityId:  activityID,
		OperateType: operateClaimGreenPlumSeed,
		Params: &activitypb.ClaimQingMeiDailySeedRequest_Params{
			GrantId: GreenPlumDailyGrantID,
		},
	}
	raw, err := a.sendActivity(ctx, "Operate", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &activitypb.ActivityOperateReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	a.greenPlumSeedClaimedDate = beijingDateKey()
	return reply, nil
}

// GreenPlumSeedClaimedToday reports whether today's 青梅 seed was claimed
// according to the process cache. The server flag may be unreliable, so this
// acts as the authoritative "claimed today" source for the snapshot.
func (a *API) GreenPlumSeedClaimedToday() bool {
	return a.greenPlumSeedClaimedDate != "" && a.greenPlumSeedClaimedDate == beijingDateKey()
}

// RememberGreenPlumSeedClaimed records today's 青梅 seed as claimed without a
// fresh RPC. Used when the server replies with an "already claimed" error.
func (a *API) RememberGreenPlumSeedClaimed() {
	a.greenPlumSeedClaimedDate = beijingDateKey()
}

// QueryGreenPlum loads the full 青梅 brewing state (operateType=7).
func (a *API) QueryGreenPlum(ctx context.Context, activityID int64) (*activitypb.ActivityOperateReply, error) {
	req := &activitypb.QueryActivityRequest{
		ActivityId:  activityID,
		OperateType: operateQueryGreenPlum,
	}
	raw, err := a.sendActivity(ctx, "Operate", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &activitypb.ActivityOperateReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	if reply.Data == nil {
		return nil, fmt.Errorf("green plum query: missing data")
	}
	return reply, nil
}

// StartGreenPlumBrew starts a brewing round with the given 青梅 ingredient
// stacks (operateType=14). Each ingredient selects a concrete bag UID entry
// and the count to invest from it; multiple UIDs can be invested together.
func (a *API) StartGreenPlumBrew(ctx context.Context, activityID int64, ingredients []*activitypb.StartQingMeiBrewRequest_Ingredient) (*activitypb.ActivityOperateReply, error) {
	if len(ingredients) == 0 {
		return nil, fmt.Errorf("green plum brew: at least one ingredient required")
	}
	req := &activitypb.StartQingMeiBrewRequest{
		ActivityId:  activityID,
		OperateType: operateStartGreenPlumBrew,
		Params: &activitypb.StartQingMeiBrewRequest_Params{
			Ingredients: ingredients,
		},
	}
	raw, err := a.sendActivity(ctx, "Operate", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &activitypb.ActivityOperateReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// ContinueGreenPlumBrew continues to the next brewing round (operateType=15).
func (a *API) ContinueGreenPlumBrew(ctx context.Context, activityID int64) (*activitypb.ActivityOperateReply, error) {
	req := &activitypb.ContinueQingMeiBrewRequest{
		ActivityId:  activityID,
		OperateType: operateContinueGreenPlum,
		Params:      &activitypb.ContinueQingMeiBrewRequest_Empty{},
	}
	raw, err := a.sendActivity(ctx, "Operate", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &activitypb.ActivityOperateReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// SettleGreenPlumBrew reports the share event and settles the brew at the
// shared (1.5x) settlement mode (operateType=16). The share must be reported
// before settling to unlock the boosted payout.
func (a *API) SettleGreenPlumBrew(ctx context.Context, activityID int64) (*activitypb.ActivityOperateReply, error) {
	if _, err := a.ReportShareScene(ctx, greenPlumShareField1, greenPlumShareField4); err != nil {
		return nil, fmt.Errorf("green plum share report: %w", err)
	}
	req := &activitypb.SettleQingMeiBrewRequest{
		ActivityId:  activityID,
		OperateType: operateSettleGreenPlum,
		Params: &activitypb.SettleQingMeiBrewRequest_Params{
			SettlementMode: GreenPlumSharedSettlementMode,
		},
	}
	raw, err := a.sendActivity(ctx, "Operate", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &activitypb.ActivityOperateReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// FindGreenPlumActivityID returns the 青梅 activity id: wantDaily selects the
// daily seed entry (…01), otherwise the brew entry (…02). The recurring
// activity gets a fresh id every run, so the id is discovered from the live
// API (the daily entry carries data.qingmei_daily_seed, the brew entry
// carries data.qingmei_brew) and cached per API. The hard-coded ids aligned
// with the reference bot are used as a fallback when nothing is recognized.
func (a *API) FindGreenPlumActivityID(ctx context.Context, wantDaily bool) int64 {
	if wantDaily {
		if id := a.greenPlumDailyActivityID.Load(); id > 0 {
			return id
		}
	} else if id := a.greenPlumBrewActivityID.Load(); id > 0 {
		return id
	}

	for _, item := range logic.GreenPlumActivities() {
		id, err := strconv.ParseInt(item.ActivityID, 10, 64)
		if err != nil || id <= 0 {
			continue
		}
		reply, err := a.QueryGreenPlum(ctx, id)
		if err != nil || reply == nil || reply.Data == nil {
			continue
		}
		if reply.Data.QingmeiDailySeed != nil {
			a.greenPlumDailyActivityID.Store(id)
			if wantDaily {
				return id
			}
		}
		if reply.Data.QingmeiBrew != nil {
			a.greenPlumBrewActivityID.Store(id)
			if !wantDaily {
				return id
			}
		}
	}

	if wantDaily {
		a.greenPlumDailyActivityID.Store(GreenPlumDailyActivityID)
		return GreenPlumDailyActivityID
	}
	a.greenPlumBrewActivityID.Store(GreenPlumBrewActivityID)
	return GreenPlumBrewActivityID
}

// FindGreenPlumDailyActivityID returns the daily seed activity id.
func (a *API) FindGreenPlumDailyActivityID(ctx context.Context) int64 {
	return a.FindGreenPlumActivityID(ctx, true)
}

// FindSeasonActivity finds a sub-activity by type code.
func FindSeasonActivity(season *seasonpb.SeasonInfo, typeCode int64) *seasonpb.SeasonActivity {
	if season == nil {
		return nil
	}
	for i := range season.Activities {
		if season.Activities[i].Type == typeCode {
			return season.Activities[i]
		}
	}
	return nil
}

// ParsePositiveInt64 parses a decimal int64 string.
func ParsePositiveInt64(s string) (int64, error) {
	if s == "" {
		return 0, fmt.Errorf("empty value")
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil || v <= 0 {
		return 0, fmt.Errorf("invalid int64: %q", s)
	}
	return v, nil
}
