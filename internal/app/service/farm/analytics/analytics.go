package analytics

import (
	"errors"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/app/model"
	"github.com/it00021hot/qq-farm-core/internal/app/service"
	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
	farmstats "github.com/it00021hot/qq-farm-core/internal/farm/stats"
	farmtypes "github.com/it00021hot/qq-farm-core/internal/types/farm"
	"github.com/it00021hot/qq-farm-core/internal/vars"
	"github.com/gofiber/fiber/v3"
)

type Service struct {
	service.Service
}

var Analytics = &Service{}

func (s *Service) Detail(ctx fiber.Ctx, req farmtypes.AnalyticsDetailReq) (map[string]any, error) {
	if req.AccountID == 0 {
		return nil, errors.New("accountId 必填")
	}
	days := req.Days
	if days <= 0 {
		days = 7
	}
	since := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	db := vars.DB
	var stats []model.FarmStats
	_ = db.Where("account_id = ? AND stat_date >= ?", req.AccountID, since).Order("stat_date asc").Find(&stats).Error
	todayExp, todayGold := farmstats.TodayExpGold(req.AccountID, 0)
	out := map[string]any{
		"accountId":  req.AccountID,
		"days":       days,
		"stats":      stats,
		"operations": farmstats.TodayOperations(req.AccountID, 0),
		"todayExp":   todayExp,
		"todayGold":  todayGold,
	}
	if sortBy := req.Sort; sortBy != "" {
		out["sort"] = sortBy
		out["rankings"] = logic.GetPlantRankings(sortBy)
	}
	return out, nil
}
