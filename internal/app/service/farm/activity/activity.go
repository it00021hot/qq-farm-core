package activity

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/MQEnergy/go-skeleton/internal/app/model"
	"github.com/MQEnergy/go-skeleton/internal/app/service"
	"github.com/MQEnergy/go-skeleton/internal/farm/activitycenter"
	"github.com/MQEnergy/go-skeleton/internal/farm/game"
	farmruntime "github.com/MQEnergy/go-skeleton/internal/farm/runtime"
	farmtypes "github.com/MQEnergy/go-skeleton/internal/types/farm"
	"github.com/MQEnergy/go-skeleton/internal/vars"
	"github.com/MQEnergy/go-skeleton/pkg/tenant"
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
	db := tenant.Scope(vars.DB, tenant.TenantCtx(ctx))
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
	snap := activitycenter.BuildSnapshot(callCtx, api)
	return s.attachSnapshot(ctx, req.AccountID, snap, map[string]any{
		"pass": snap.Season["pass"],
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
			})
		}
		return nil, err
	}
	if reply != nil && reply.Data != nil && reply.Data.Constellation != nil {
		activitycenter.RememberConstellationNodes(act.ActivityId, reply.Data.Constellation)
	}
	snap := activitycenter.BuildSnapshot(callCtx, api)
	return s.attachSnapshot(ctx, req.AccountID, snap, map[string]any{
		"outcome":       "lighted",
		"constellation": snap.Constellation,
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
		"shop": snap.Shop,
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
	if _, err := api.ClaimSolarTerms(callCtx, termID); err != nil {
		return nil, err
	}
	snap := activitycenter.BuildSnapshot(callCtx, api)
	return s.attachSnapshot(ctx, req.AccountID, snap, map[string]any{
		"termId": req.TermID,
	})
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
	db := tenant.Scope(vars.DB, tenant.TenantCtx(ctx))
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
	if err := tenant.Scope(vars.DB, tenant.TenantCtx(ctx)).Where("id = ?", accountID).First(&account).Error; err != nil {
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
	db := tenant.Scope(vars.DB, tenant.TenantCtx(ctx))
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
