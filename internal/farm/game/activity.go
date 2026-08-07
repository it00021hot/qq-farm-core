package game

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/MQEnergy/go-skeleton/internal/farm/proto/activitypb"
	"github.com/MQEnergy/go-skeleton/internal/farm/proto/seasonpb"
	"github.com/MQEnergy/go-skeleton/internal/farm/proto/solartermspb"
)

const (
	activityService = "gamepb.activitypb.ActivityService"
	seasonService   = "gamepb.seasonpb.SeasonService"
	solarService    = "gamepb.solartermspb.SolarTermsService"

	shopActivityType          int64 = 3
	constellationActivityType int64 = 13
)

func ShopActivityType() int64          { return shopActivityType }
func ConstellationActivityType() int64 { return constellationActivityType }

func (a *API) sendActivity(ctx context.Context, method string, body []byte) ([]byte, error) {
	if err := a.requireSender(); err != nil {
		return nil, err
	}
	raw, _, err := a.Sender.Send(ctx, activityService, method, nonNilBody(body))
	return raw, err
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
	raw, err := a.sendSeason(ctx, "GetSeasonInfo", (&seasonpb.GetSeasonInfoRequest{}).Marshal())
	if err != nil {
		return nil, err
	}
	reply := &seasonpb.GetSeasonInfoReply{}
	if err := reply.Unmarshal(raw); err != nil {
		return nil, err
	}
	return reply, nil
}

// ClaimBattlePassRewards claims all eligible pass nodes.
func (a *API) ClaimBattlePassRewards(ctx context.Context) (*seasonpb.ClaimBattlePassRewardsReply, error) {
	raw, err := a.sendSeason(ctx, "ClaimBattlePassRewards", (&seasonpb.ClaimBattlePassRewardsRequest{}).Marshal())
	if err != nil {
		return nil, err
	}
	reply := &seasonpb.ClaimBattlePassRewardsReply{}
	if err := reply.Unmarshal(raw); err != nil {
		return nil, err
	}
	return reply, nil
}

// GetSolarTerms fetches solar term list.
func (a *API) GetSolarTerms(ctx context.Context) (*solartermspb.GetSolarTermsReply, error) {
	raw, err := a.sendSolar(ctx, "GetSolarTerms", (&solartermspb.GetSolarTermsRequest{}).Marshal())
	if err != nil {
		return nil, err
	}
	reply := &solartermspb.GetSolarTermsReply{}
	if err := reply.Unmarshal(raw); err != nil {
		return nil, err
	}
	return reply, nil
}

// ClaimSolarTerms claims one solar term reward.
func (a *API) ClaimSolarTerms(ctx context.Context, termID int64) (*solartermspb.ClaimSolarTermsReply, error) {
	req := &solartermspb.ClaimSolarTermsRequest{TermID: termID}
	raw, err := a.sendSolar(ctx, "ClaimSolarTerms", req.Marshal())
	if err != nil {
		return nil, err
	}
	reply := &solartermspb.ClaimSolarTermsReply{}
	if err := reply.Unmarshal(raw); err != nil {
		return nil, err
	}
	return reply, nil
}

// QueryActivityShop loads star-sand shop catalog.
func (a *API) QueryActivityShop(ctx context.Context, activityID int64) (*activitypb.ActivityOperateReply, error) {
	req := &activitypb.QueryActivityRequest{
		ActivityID:  activityID,
		OperateType: activitypb.OperateQueryShop,
	}
	raw, err := a.sendActivity(ctx, "Operate", req.Marshal())
	if err != nil {
		return nil, err
	}
	reply := &activitypb.ActivityOperateReply{}
	if err := reply.Unmarshal(raw); err != nil {
		return nil, err
	}
	if reply.ActivityID != activityID || reply.OperateType != activitypb.OperateQueryShop {
		return nil, fmt.Errorf("activity shop query: unexpected reply activity=%d operate=%d", reply.ActivityID, reply.OperateType)
	}
	if reply.Data == nil || reply.Data.Catalog == nil {
		return nil, fmt.Errorf("activity shop query: missing catalog")
	}
	return reply, nil
}

// ExchangeShopGoods exchanges star-sand shop goods.
func (a *API) ExchangeShopGoods(ctx context.Context, activityID, goodsID, count int64) (*activitypb.ActivityOperateReply, error) {
	req := &activitypb.ExchangeShopRequest{
		ActivityID:  activityID,
		OperateType: activitypb.OperateExchangeShop,
		ExchangeShopOperate: &activitypb.ExchangeShopOperateParams{
			GoodsID: goodsID,
			Count:   count,
		},
	}
	raw, err := a.sendActivity(ctx, "Operate", req.Marshal())
	if err != nil {
		return nil, err
	}
	reply := &activitypb.ActivityOperateReply{}
	if err := reply.Unmarshal(raw); err != nil {
		return nil, err
	}
	if reply.ActivityID != activityID || reply.OperateType != activitypb.OperateExchangeShop {
		return nil, fmt.Errorf("activity shop exchange: unexpected reply activity=%d operate=%d", reply.ActivityID, reply.OperateType)
	}
	if reply.Data == nil || reply.Data.Catalog == nil {
		return nil, fmt.Errorf("activity shop exchange: missing catalog")
	}
	return reply, nil
}

// LightConstellation lights today's constellation node.
func (a *API) LightConstellation(ctx context.Context, activityID int64) (*activitypb.ActivityOperateReply, error) {
	req := &activitypb.OperateConstellationRequest{
		ActivityID:  activityID,
		OperateType: activitypb.OperateLightConstellation,
	}
	raw, err := a.sendActivity(ctx, "Operate", req.Marshal())
	if err != nil {
		return nil, err
	}
	reply := &activitypb.ActivityOperateReply{}
	if err := reply.Unmarshal(raw); err != nil {
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

// FindSeasonActivity finds a sub-activity by type code.
func FindSeasonActivity(season *seasonpb.SeasonInfo, typeCode int64) *seasonpb.SeasonActivity {
	if season == nil {
		return nil
	}
	for i := range season.Activities {
		if season.Activities[i].Type == typeCode {
			return &season.Activities[i]
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
