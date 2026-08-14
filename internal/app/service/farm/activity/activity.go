package activity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/app/model"
	"github.com/it00021hot/qq-farm-core/internal/app/service"
	"github.com/it00021hot/qq-farm-core/internal/farm/activitycenter"
	"github.com/it00021hot/qq-farm-core/internal/farm/game"
	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/activitypb"
	farmruntime "github.com/it00021hot/qq-farm-core/internal/farm/runtime"
	farmtypes "github.com/it00021hot/qq-farm-core/internal/types/farm"
	"github.com/it00021hot/qq-farm-core/internal/vars"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type Service struct {
	service.Service
}

var Activity = &Service{}

func (s *Service) Snapshot(ctx fiber.Ctx, req farmtypes.ActivitySnapshotReq) (map[string]any, error) {
	if req.AccountID == 0 {
		return nil, errors.New("accountId 必填")
	}
	db := vars.DB
	var states []model.FarmActivityState
	_ = db.Where("account_id = ?", req.AccountID).Find(&states).Error
	hydrateConstellationFromStates(states)

	result := map[string]any{
		"accountId":     req.AccountID,
		"states":        states,
		"season":        map[string]any{},
		"shop":          map[string]any{},
		"solarTerms":    map[string]any{},
		"constellation": map[string]any{},
		"greenPlum":     map[string]any{},
	}

	session, err := s.session(ctx, req.AccountID)
	if err != nil {
		return result, nil
	}
	callCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	api := session.GameAPI()
	if api == nil {
		return result, nil
	}
	snap := activitycenter.BuildSnapshot(callCtx, api)
	return s.attachSnapshot(ctx, req.AccountID, snap, nil)
}

// Registry returns the process-wide activity registry (debug/protocol sniffing).
func (s *Service) Registry(_ fiber.Ctx) (map[string]any, error) {
	return map[string]any{
		"activities": logic.ActivityRegistrySnapshot(),
	}, nil
}

func (s *Service) ClaimPass(ctx fiber.Ctx, req farmtypes.ActivityActionReq) (map[string]any, error) {
	_, api, callCtx, cancel, err := s.liveAPI(ctx, req.AccountID)
	if err != nil {
		return nil, err
	}
	defer cancel()

	season, err := api.GetSeasonInfo(callCtx)
	if err != nil {
		return nil, err
	}
	if season.SeasonInfo == nil || season.SeasonInfo.Pass == nil {
		return nil, errors.New("服务端未发现可用游记")
	}
	pass := season.SeasonInfo.Pass
	claimable := false
	for _, node := range pass.Nodes {
		if node.NodeId <= pass.ClaimedThroughLevel {
			continue
		}
		if node.NodeId <= pass.CurrentLevel {
			claimable = true
			break
		}
	}
	if !claimable {
		return nil, errors.New("当前没有可领取的游记奖励")
	}
	reply, err := api.ClaimBattlePassRewards(callCtx)
	if err != nil {
		return nil, err
	}
	if reply != nil && reply.Pass != nil {
		activitycenter.ApplySeasonPassNotify(reply.Pass)
	}
	var rewardPairs [][2]int64
	if reply != nil {
		for _, r := range reply.Rewards {
			if r == nil || r.ItemId <= 0 {
				continue
			}
			rewardPairs = append(rewardPairs, [2]int64{r.ItemId, r.Count})
		}
	}
	snap := activitycenter.BuildSnapshot(callCtx, api)
	return s.attachSnapshot(ctx, req.AccountID, snap, map[string]any{
		"pass":    snap.Season["pass"],
		"rewards": activitycenter.RewardDTOsFromPairs(rewardPairs),
	})
}

func (s *Service) LightConstellation(ctx fiber.Ctx, req farmtypes.ActivityActionReq) (map[string]any, error) {
	_, api, callCtx, cancel, err := s.liveAPI(ctx, req.AccountID)
	if err != nil {
		return nil, err
	}
	defer cancel()

	season, err := api.GetSeasonInfo(callCtx)
	if err != nil {
		return nil, err
	}
	act := game.FindSeasonActivity(season.SeasonInfo, game.ConstellationActivityType())
	if act == nil {
		return nil, errors.New("服务端未发现星座活动")
	}
	reply, err := api.LightConstellation(callCtx, act.ActivityId)
	if err != nil {
		if game.IsConstellationAlreadyClaimed(err) {
			snap := activitycenter.BuildSnapshot(callCtx, api)
			return s.attachSnapshot(ctx, req.AccountID, snap, map[string]any{
				"outcome":     "nothingToClaim",
				"noClaimable": true,
				"message":     "今日星宿奖励已经领取，无需重复操作",
				"rewards":     []map[string]any{},
			})
		}
		return nil, err
	}
	var dynamic *activitypb.ConstellationData
	if reply != nil && reply.Data != nil && reply.Data.Constellation != nil {
		dynamic = reply.Data.Constellation
		activitycenter.RememberConstellationNodes(act.ActivityId, dynamic)
	}
	serverTime := int64(0)
	if season.SeasonInfo != nil {
		serverTime = season.SeasonInfo.ServerTime
	}
	snap := activitycenter.BuildSnapshot(callCtx, api)
	return s.attachSnapshot(ctx, req.AccountID, snap, map[string]any{
		"outcome":       "lighted",
		"constellation": snap.Constellation,
		"rewards":       activitycenter.ConstellationClaimRewardDTOs(act, serverTime, dynamic),
	})
}

func (s *Service) ShopExchange(ctx fiber.Ctx, req farmtypes.ActivityActionReq) (map[string]any, error) {
	if req.ItemID == "" {
		return nil, errors.New("itemId 必填")
	}
	goodsID, err := game.ParsePositiveInt64(req.ItemID)
	if err != nil {
		return nil, err
	}
	count := req.Count
	if count <= 0 {
		count = 1
	}

	_, api, callCtx, cancel, err := s.liveAPI(ctx, req.AccountID)
	if err != nil {
		return nil, err
	}
	defer cancel()

	season, err := api.GetSeasonInfo(callCtx)
	if err != nil {
		return nil, err
	}
	shopAct := game.FindSeasonActivity(season.SeasonInfo, game.ShopActivityType())
	if shopAct == nil {
		return nil, errors.New("当前赛季未发现活动商店")
	}
	if _, err := api.ExchangeShopGoods(callCtx, shopAct.ActivityId, goodsID, count); err != nil {
		return nil, err
	}
	snap := activitycenter.BuildSnapshot(callCtx, api)
	return s.attachSnapshot(ctx, req.AccountID, snap, map[string]any{
		"shop":    snap.Shop,
		"rewards": activitycenter.ShopExchangeRewardDTOs(snap.Shop, req.ItemID, count),
	})
}

func (s *Service) ClaimSolarTerm(ctx fiber.Ctx, req farmtypes.ActivityActionReq) (map[string]any, error) {
	if req.TermID == "" {
		return nil, errors.New("termId 必填")
	}
	termID, err := game.ParsePositiveInt64(req.TermID)
	if err != nil {
		return nil, err
	}

	_, api, callCtx, cancel, err := s.liveAPI(ctx, req.AccountID)
	if err != nil {
		return nil, err
	}
	defer cancel()

	solar, err := api.GetSolarTerms(callCtx)
	if err != nil {
		return nil, err
	}
	var found *int64
	for i := range solar.Terms {
		if solar.Terms[i].TermId == termID {
			if solar.Terms[i].Status != 2 {
				return nil, errors.New("指定节令当前不可领取")
			}
			found = &termID
			break
		}
	}
	if found == nil {
		return nil, errors.New("服务端未发现指定节令")
	}
	reply, err := api.ClaimSolarTerms(callCtx, termID)
	if err != nil {
		return nil, err
	}
	var rewardPairs [][2]int64
	if reply != nil {
		for _, r := range reply.Rewards {
			if r == nil || r.ItemId <= 0 {
				continue
			}
			rewardPairs = append(rewardPairs, [2]int64{r.ItemId, r.Count})
		}
	}
	snap := activitycenter.BuildSnapshot(callCtx, api)
	return s.attachSnapshot(ctx, req.AccountID, snap, map[string]any{
		"termId":  req.TermID,
		"rewards": activitycenter.RewardDTOsFromPairs(rewardPairs),
	})
}

// ClaimGreenPlum claims the 青梅 seed reward. The activity id may be supplied
// explicitly or resolved by feature from the registry; already-claimed today is
// reported as an idempotent outcome instead of an error.
func (s *Service) ClaimGreenPlum(ctx fiber.Ctx, req farmtypes.ActivityActionReq) (map[string]any, error) {
	_, api, callCtx, cancel, err := s.liveAPI(ctx, req.AccountID)
	if err != nil {
		return nil, err
	}
	defer cancel()

	activityID, err := s.resolveGreenPlumActivity(callCtx, api, req.ActivityID, true)
	if err != nil {
		return nil, err
	}

	reply, err := api.ClaimGreenPlumSeed(callCtx, activityID)
	if err != nil {
		if game.IsAlreadyClaimedGreenPlum(err) {
			api.RememberGreenPlumSeedClaimed()
			snap := activitycenter.BuildSnapshot(callCtx, api)
			return s.attachSnapshot(ctx, req.AccountID, snap, map[string]any{
				"outcome":     "alreadyClaimed",
				"greenPlum":   snap.GreenPlum,
				"message":     "今日青梅种子已经领取，无需重复领取",
				"rewards":     []map[string]any{},
			})
		}
		return nil, err
	}
	snap := activitycenter.BuildSnapshot(callCtx, api)
	return s.attachSnapshot(ctx, req.AccountID, snap, map[string]any{
		"outcome":   "claimed",
		"greenPlum": snap.GreenPlum,
		"rewards":   activitycenter.QingMeiSeedRewardDTOs(reply),
	})
}

// StartGreenPlumBrew starts a brewing round with the given 青梅 ingredients.
// Ingredients are bag UID entries (uid+count pairs); when the request only
// carries a legacy count, a single bag entry holding at least that many 青梅
// is used instead.
func (s *Service) StartGreenPlumBrew(ctx fiber.Ctx, req farmtypes.ActivityActionReq) (map[string]any, error) {
	_, api, callCtx, cancel, err := s.liveAPI(ctx, req.AccountID)
	if err != nil {
		return nil, err
	}
	defer cancel()

	activityID, err := s.resolveGreenPlumActivity(callCtx, api, req.ActivityID, false)
	if err != nil {
		return nil, err
	}

	ingredients, err := buildGreenPlumIngredients(callCtx, api, req)
	if err != nil {
		return nil, err
	}
	reply, err := api.StartGreenPlumBrew(callCtx, activityID, ingredients)
	if err != nil {
		return nil, err
	}
	snap := activitycenter.BuildSnapshot(callCtx, api)
	return s.attachSnapshot(ctx, req.AccountID, snap, map[string]any{
		"outcome":     "started",
		"greenPlum":   snap.GreenPlum,
		"activity":    activitycenter.QingMeiBrewDTO(reply),
		"message":     "已开始青梅酿造",
	})
}

// buildGreenPlumIngredients turns the start-brew request into ingredient
// entries. UID entries from the request are validated against the live bag;
// a legacy scalar count is resolved to the single bag entry holding at least
// that many 青梅, matching the reference bot's fallback path.
func buildGreenPlumIngredients(ctx context.Context, api *game.API, req farmtypes.ActivityActionReq) ([]*activitypb.StartQingMeiBrewRequest_Ingredient, error) {
	if len(req.Ingredients) > 0 {
		bag, err := api.Bag(ctx)
		if err != nil {
			return nil, err
		}
		byUID := make(map[int64]int64, 8)
		for _, item := range game.GetBagItems(bag) {
			if item.Id == game.GreenPlumItemID && item.Uid > 0 {
				byUID[item.Uid] = item.Count
			}
		}
		seen := make(map[int64]bool, len(req.Ingredients))
		out := make([]*activitypb.StartQingMeiBrewRequest_Ingredient, 0, len(req.Ingredients))
		for _, ing := range req.Ingredients {
			uid, parseErr := strconv.ParseInt(strings.TrimSpace(ing.Uid), 10, 64)
			if parseErr != nil || uid <= 0 || ing.Count <= 0 {
				return nil, fmt.Errorf("青梅原料 UID/数量 无效: %q", ing.Uid)
			}
			if seen[uid] {
				return nil, fmt.Errorf("青梅 UID %d 重复", uid)
			}
			if byUID[uid] < ing.Count {
				return nil, fmt.Errorf("青梅 UID %d 数量不足", uid)
			}
			seen[uid] = true
			out = append(out, &activitypb.StartQingMeiBrewRequest_Ingredient{
				Uid:   uid,
				Count: ing.Count,
			})
		}
		return out, nil
	}

	// Legacy path: a single count investing from the bag entry holding the
	// most 青梅.
	if req.Count <= 0 {
		return nil, errors.New("count 或 ingredients 必填")
	}
	uid, err := findGreenPlumBagUID(ctx, api, req.Count)
	if err != nil {
		return nil, err
	}
	return []*activitypb.StartQingMeiBrewRequest_Ingredient{
		{Uid: uid, Count: req.Count},
	}, nil
}

// ContinueGreenPlumBrew continues to the next brewing round.
func (s *Service) ContinueGreenPlumBrew(ctx fiber.Ctx, req farmtypes.ActivityActionReq) (map[string]any, error) {
	_, api, callCtx, cancel, err := s.liveAPI(ctx, req.AccountID)
	if err != nil {
		return nil, err
	}
	defer cancel()

	activityID, err := s.resolveGreenPlumActivity(callCtx, api, req.ActivityID, false)
	if err != nil {
		return nil, err
	}
	reply, err := api.ContinueGreenPlumBrew(callCtx, activityID)
	if err != nil {
		return nil, err
	}
	snap := activitycenter.BuildSnapshot(callCtx, api)
	quote := activitycenter.QingMeiQuoteDTO(reply)
	return s.attachSnapshot(ctx, req.AccountID, snap, map[string]any{
		"outcome":     "continued",
		"greenPlum":   snap.GreenPlum,
		"quote":       quote,
		"message":     activitycenter.QingMeiQuoteMessage(quote),
	})
}

// SettleGreenPlumBrew reports the share and settles the brew at the boosted
// (1.5x) shared settlement mode.
func (s *Service) SettleGreenPlumBrew(ctx fiber.Ctx, req farmtypes.ActivityActionReq) (map[string]any, error) {
	_, api, callCtx, cancel, err := s.liveAPI(ctx, req.AccountID)
	if err != nil {
		return nil, err
	}
	defer cancel()

	activityID, err := s.resolveGreenPlumActivity(callCtx, api, req.ActivityID, false)
	if err != nil {
		return nil, err
	}
	reply, err := api.SettleGreenPlumBrew(callCtx, activityID)
	if err != nil {
		return nil, err
	}
	snap := activitycenter.BuildSnapshot(callCtx, api)
	return s.attachSnapshot(ctx, req.AccountID, snap, map[string]any{
		"outcome":    "settled",
		"greenPlum":  snap.GreenPlum,
		"settlement": activitycenter.QingMeiSettlementDTO(reply),
		"rewards":    activitycenter.QingMeiSettlementRewardDTOs(reply),
		"message":    activitycenter.QingMeiSettlementMessage(reply),
	})
}

// resolveGreenPlumActivity resolves the 青梅 activity id from req.ActivityID if
// provided, otherwise falls back to the reference 青梅 activity ids (daily seed
// entry …01 vs brew entry …02, matching liyangpengs/qq-farm-bot).
func (s *Service) resolveGreenPlumActivity(callCtx context.Context, api *game.API, rawID string, wantDaily bool) (int64, error) {
	activityID := int64(0)
	if rawID != "" {
		id, err := game.ParsePositiveInt64(rawID)
		if err != nil {
			return 0, err
		}
		activityID = id
	}
	if activityID <= 0 {
		activityID = api.FindGreenPlumActivityID(callCtx, wantDaily)
	}
	if activityID <= 0 {
		if wantDaily {
			return 0, errors.New("青梅种子活动尚未被识别，无法领取")
		}
		return 0, errors.New("青梅酿造活动尚未被识别，无法操作")
	}
	return activityID, nil
}

// findGreenPlumBagUID finds a bag entry holding at least count 青梅 and returns
// its uid for the start-brew ingredient.
func findGreenPlumBagUID(ctx context.Context, api *game.API, count int64) (int64, error) {
	bag, err := api.Bag(ctx)
	if err != nil {
		return 0, err
	}
	for _, item := range game.GetBagItems(bag) {
		if item.Id != game.GreenPlumItemID {
			continue
		}
		if item.Count >= count {
			return item.Uid, nil
		}
	}
	return 0, errors.New("青梅数量不足，或数量分散在多个背包条目中")
}

func (s *Service) ClaimTask(ctx fiber.Ctx, req farmtypes.ActivityActionReq) (map[string]any, error) {
	_, api, callCtx, cancel, err := s.liveAPI(ctx, req.AccountID)
	if err != nil {
		return nil, err
	}
	defer cancel()

	claimedTasks, claimedActives, err := api.ClaimAllTasks(callCtx)
	if err != nil {
		return nil, err
	}
	stateJSON, _ := json.Marshal(map[string]any{
		"claimedTasks":   claimedTasks,
		"claimedActives": claimedActives,
	})
	if err := s.touchState(ctx, req.AccountID, "task", string(stateJSON)); err != nil {
		return nil, err
	}
	return map[string]any{
		"accountId":      req.AccountID,
		"claimedTasks":   claimedTasks,
		"claimedActives": claimedActives,
	}, nil
}

func (s *Service) ClaimGift(ctx fiber.Ctx, req farmtypes.ActivityActionReq) (map[string]any, error) {
	_, api, callCtx, cancel, err := s.liveAPI(ctx, req.AccountID)
	if err != nil {
		return nil, err
	}
	defer cancel()

	reply, err := api.TaskInfo(callCtx)
	if err != nil {
		return nil, err
	}
	if reply.TaskInfo == nil {
		return map[string]any{"accountId": req.AccountID, "claimedActives": 0}, nil
	}
	claimed := 0
	for _, active := range reply.TaskInfo.Actives {
		pointIDs := active.ClaimablePointIDs()
		if len(pointIDs) == 0 {
			continue
		}
		if _, err := api.ClaimDailyReward(callCtx, active.Type, pointIDs); err == nil {
			claimed += len(pointIDs)
		}
	}
	stateJSON, _ := json.Marshal(map[string]any{"claimedActives": claimed})
	if err := s.touchState(ctx, req.AccountID, "gift", string(stateJSON)); err != nil {
		return nil, err
	}
	return map[string]any{
		"accountId":      req.AccountID,
		"claimedActives": claimed,
	}, nil
}

func (s *Service) attachSnapshot(ctx fiber.Ctx, accountID uint64, snap activitycenter.Snapshot, extra map[string]any) (map[string]any, error) {
	db := vars.DB
	now := uint(time.Now().Unix())
	_ = s.persistSnapshotStates(db, accountID, snap, now)

	var states []model.FarmActivityState
	_ = db.Where("account_id = ?", accountID).Find(&states).Error

	result := map[string]any{
		"accountId":     accountID,
		"states":        states,
		"season":        snap.Season,
		"shop":          snap.Shop,
		"solarTerms":    snap.SolarTerms,
		"constellation": snap.Constellation,
		"greenPlum":     snap.GreenPlum,
		"capabilities":  snap.Capabilities,
		"actions":       snap.Actions,
	}
	if len(snap.Errors) > 0 {
		result["errors"] = snap.Errors
	}
	for k, v := range extra {
		result[k] = v
	}
	// Nest under snapshot key as well for bot-style clients.
	result["snapshot"] = map[string]any{
		"season":        snap.Season,
		"shop":          snap.Shop,
		"solarTerms":    snap.SolarTerms,
		"constellation": snap.Constellation,
		"greenPlum":     snap.GreenPlum,
		"capabilities":  snap.Capabilities,
		"actions":       snap.Actions,
		"errors":        snap.Errors,
	}
	return result, nil
}

func hydrateConstellationFromStates(states []model.FarmActivityState) {
	for _, st := range states {
		if st.ActivityID != "constellation" {
			continue
		}
		activitycenter.HydrateConstellationConfirmedFromStateJSON(st.ActivityID, st.StateJSON)
	}
}

func (s *Service) liveAPI(ctx fiber.Ctx, accountID uint64) (*farmruntime.Session, *game.API, context.Context, context.CancelFunc, error) {
	session, err := s.session(ctx, accountID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	api := session.GameAPI()
	if api == nil {
		return nil, nil, nil, nil, errors.New("游戏连接未就绪")
	}
	callCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	return session, api, callCtx, cancel, nil
}

func (s *Service) session(ctx fiber.Ctx, accountID uint64) (*farmruntime.Session, error) {
	var account model.FarmAccount
	if err := vars.DB.Where("id = ?", accountID).First(&account).Error; err != nil {
		return nil, errors.New("账号不存在")
	}
	session, ok := farmruntime.Default.Session(accountID)
	if !ok || session.Status() != farmruntime.StatusRunning {
		return nil, errors.New("账号未运行")
	}
	return session, nil
}

func (s *Service) persistSnapshotStates(db *gorm.DB, accountID uint64, snap activitycenter.Snapshot, now uint) error {
	entries := []struct {
		id   string
		data map[string]any
	}{
		{"pass", snap.Season},
		{"constellation", snap.Constellation},
		{"shop", snap.Shop},
		{"greenPlum", snap.GreenPlum},
	}
	for _, e := range entries {
		if len(e.data) == 0 {
			continue
		}
		stateJSON := activitycenter.StateJSONFromSnapshot(e.id, e.data)
		if err := upsertState(db, accountID, e.id, stateJSON, now); err != nil {
			return err
		}
	}
	if terms, ok := snap.SolarTerms["terms"].([]map[string]any); ok {
		for _, term := range terms {
			if term["canClaim"] != true {
				continue
			}
			id, _ := term["id"].(string)
			if id == "" {
				continue
			}
			stateJSON := activitycenter.StateJSONFromSnapshot("solar", term)
			if err := upsertState(db, accountID, "solar:"+id, stateJSON, now); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) touchState(ctx fiber.Ctx, accountID uint64, activityID, stateJSON string) error {
	if accountID == 0 {
		return errors.New("accountId 必填")
	}
	db := vars.DB
	now := uint(time.Now().Unix())
	return upsertState(db, accountID, activityID, stateJSON, now)
}

func upsertState(db *gorm.DB, accountID uint64, activityID, stateJSON string, now uint) error {
	var st model.FarmActivityState
	err := db.Where("account_id = ? AND activity_id = ?", accountID, activityID).First(&st).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		st = model.FarmActivityState{
			AccountID:  accountID,
			ActivityID: activityID,
			StateJSON:  stateJSON,
			SyncedAt:   now,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		return db.Create(&st).Error
	}
	if err != nil {
		return err
	}
	return db.Model(&st).Updates(map[string]any{
		"state_json": stateJSON,
		"synced_at":  now,
		"updated_at": now,
	}).Error
}
