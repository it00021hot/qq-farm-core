package activity

// 新活动 HTTP 服务（公益小红花 / 雨落成诗 / 萌宠日记 / 图鉴 / 宠物）。

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/it00021hot/qq-farm-core/internal/app/model"
	"github.com/it00021hot/qq-farm-core/internal/farm/activitycenter"
	"github.com/it00021hot/qq-farm-core/internal/farm/game"
	farmruntime "github.com/it00021hot/qq-farm-core/internal/farm/runtime"
	"github.com/it00021hot/qq-farm-core/internal/types/farm"
	"github.com/it00021hot/qq-farm-core/internal/vars"
)

// Charity fetches the 公益小红花 snapshot.
func (s *Service) Charity(ctx fiber.Ctx, req farm.ActivitySnapshotReq) (map[string]any, error) {
	_, api, err := s.liveSession(req.AccountID)
	if err != nil {
		return nil, err
	}
	callCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	return activitycenter.BuildCharity(callCtx, api)
}

// CharityOperate runs one charity action: claimSeeds / donateLove / claimDailyGift / progressReward(target).
func (s *Service) CharityOperate(ctx fiber.Ctx, req farm.ActivityActionReq) (map[string]any, error) {
	_, api, err := s.liveSession(req.AccountID)
	if err != nil {
		return nil, err
	}
	callCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	switch req.Action {
	case "claimSeeds":
		return activitycenter.ClaimCharitySeeds(callCtx, api)
	case "donateLove":
		return activitycenter.DonateCharityLove(callCtx, api)
	case "claimDailyGift":
		return activitycenter.ClaimCharityDailyGift(callCtx, api)
	case "progressReward":
		target, _ := strconv.ParseInt(req.ItemID, 10, 64)
		return activitycenter.ClaimCharityProgressReward(callCtx, api, target)
	default:
		return nil, errors.New("未知公益操作")
	}
}

// Weather fetches the 雨落成诗 snapshot.
func (s *Service) Weather(ctx fiber.Ctx, req farm.ActivitySnapshotReq) (map[string]any, error) {
	session, api, err := s.liveSession(req.AccountID)
	if err != nil {
		return nil, err
	}
	callCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	return activitycenter.BuildWeather(callCtx, api, session.GID())
}

// WeatherOperate runs one weather action.
func (s *Service) WeatherOperate(ctx fiber.Ctx, req farm.ActivityActionReq) (map[string]any, error) {
	session, api, err := s.liveSession(req.AccountID)
	if err != nil {
		return nil, err
	}
	myGID := session.GID()
	callCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	friendGID, _ := strconv.ParseInt(req.FriendGID, 10, 64)
	switch req.Action {
	case "exchangeCollector":
		return activitycenter.ExchangeWeatherCollector(callCtx, api)
	case "collect":
		return activitycenter.CollectFriendWeather(callCtx, api, friendGID, myGID)
	case "summon":
		return activitycenter.SummonThunderstorm(callCtx, api, myGID)
	case "frog":
		return activitycenter.UseWeatherFarmBottle(callCtx, api, friendGID, myGID, 5005, nil)
	case "cloud":
		landID, _ := strconv.ParseInt(req.ItemID, 10, 64)
		var lands []int64
		if landID > 0 {
			lands = []int64{landID}
		}
		return activitycenter.UseWeatherFarmBottle(callCtx, api, friendGID, myGID, 5006, lands)
	case "advanceResearch":
		nodeID, _ := strconv.ParseInt(req.ItemID, 10, 64)
		return activitycenter.AdvanceWeatherResearch(callCtx, api, nodeID, myGID)
	case "scan":
		gids := make([]int64, 0, len(req.Gids))
		for _, gid := range req.Gids {
			if v, err := strconv.ParseInt(gid, 10, 64); err == nil && v > 0 {
				gids = append(gids, v)
			}
		}
		return activitycenter.ScanFriendWeather(callCtx, api, gids, myGID, func(c context.Context) bool {
			return waitForFriendCheckIdle(c, session)
		})
	default:
		return nil, errors.New("未知天气操作")
	}
}

// PetDiary fetches the 萌宠成长日记 snapshot.
func (s *Service) PetDiary(ctx fiber.Ctx, req farm.ActivitySnapshotReq) (map[string]any, error) {
	_, api, err := s.liveSession(req.AccountID)
	if err != nil {
		return nil, err
	}
	callCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return activitycenter.BuildPetDiary(callCtx, api)
}

// PetDiaryOperate runs one pet-diary action.
func (s *Service) PetDiaryOperate(ctx fiber.Ctx, req farm.ActivityActionReq) (map[string]any, error) {
	_, api, err := s.liveSession(req.AccountID)
	if err != nil {
		return nil, err
	}
	callCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	opts := map[string]any{}
	if v, err := strconv.ParseInt(req.ItemID, 10, 64); err == nil {
		opts["order"] = v
		opts["charmId"] = v
		opts["goodsId"] = v
		opts["nodeId"] = v
	}
	if v, err := strconv.ParseInt(req.TermID, 10, 64); err == nil {
		opts["termId"] = v
	}
	if v, err := strconv.ParseInt(req.FriendGID, 10, 64); err == nil {
		opts["gid"] = v
	}
	if v, err := strconv.ParseInt(req.TargetID, 10, 64); err == nil {
		opts["treasureId"] = v
		opts["challengeId"] = v
	}
	if req.Count > 0 {
		opts["count"] = req.Count
	}
	if req.SkipBattle {
		opts["skip"] = true
	}
	return activitycenter.OperatePetDiary(callCtx, api, req.Action, opts)
}

// PetDiaryRecords fetches interact (31) or plundered (44) logs.
func (s *Service) PetDiaryRecords(ctx fiber.Ctx, req farm.ActivityPetDiaryRecordsReq) ([]map[string]any, error) {
	_, api, err := s.liveSession(req.AccountID)
	if err != nil {
		return nil, err
	}
	callCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return activitycenter.GetPetDiaryRecords(callCtx, api, req.Kind)
}

// PetDiaryFriend queries one friend's treasures + defender charms (op 47).
func (s *Service) PetDiaryFriend(ctx fiber.Ctx, req farm.ActivityPetDiaryFriendReq) (map[string]any, error) {
	_, api, err := s.liveSession(req.AccountID)
	if err != nil {
		return nil, err
	}
	gid, err := strconv.ParseInt(req.GID, 10, 64)
	if err != nil || gid <= 0 {
		return nil, errors.New("好友 GID 必须是正十进制整数")
	}
	callCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return activitycenter.GetPetDiaryFriendInfo(callCtx, api, gid)
}

// liveSession resolves the account's running session + API.
func (s *Service) liveSession(accountID uint64) (*farmruntime.Session, *game.API, error) {
	var account model.FarmAccount
	if err := vars.DB.Where("id = ?", accountID).First(&account).Error; err != nil {
		return nil, nil, errors.New("账号不存在")
	}
	session, ok := farmruntime.Default.Session(accountID)
	if !ok || session.Status() != farmruntime.StatusRunning {
		return nil, nil, errors.New("账号未运行")
	}
	api := session.GameAPI()
	if api == nil {
		return nil, nil, errors.New("账号未连接游戏")
	}
	return session, api, nil
}

// waitForFriendCheckIdle mirrors bot waitForFriendTaskIdle: the friend patrol
// enters friend farms too, weather scans yield until it goes idle.
func waitForFriendCheckIdle(ctx context.Context, s *farmruntime.Session) bool {
	if s == nil {
		return true
	}
	deadline := time.Now().Add(10 * time.Second)
	for s.FriendCheckRunning() {
		if time.Now().After(deadline) {
			return false
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(250 * time.Millisecond):
		}
	}
	return true
}
