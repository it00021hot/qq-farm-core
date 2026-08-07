package logs

import (
	"github.com/it00021hot/qq-farm-core/internal/app/model"
	"github.com/it00021hot/qq-farm-core/internal/app/service"
	"github.com/it00021hot/qq-farm-core/internal/farm/hub"
	farmtypes "github.com/it00021hot/qq-farm-core/internal/types/farm"
	"github.com/it00021hot/qq-farm-core/internal/vars"
	"github.com/gofiber/fiber/v3"
)

type Service struct {
	service.Service
}

var Logs = &Service{}

func (s *Service) List(ctx fiber.Ctx, req farmtypes.LogsListReq) ([]hub.LogEntry, error) {
	if req.AccountID != 0 {
		db := vars.DB
		var acc model.FarmAccount
		if err := db.Where("id = ?", req.AccountID).First(&acc).Error; err != nil {
			return nil, err
		}
	}
	return hub.Logs.Query(req.AccountID, req.Module, req.Keyword, req.Limit), nil
}

func (s *Service) Clear(ctx fiber.Ctx, req farmtypes.LogsClearReq) (map[string]any, error) {
	if req.AccountID != 0 {
		db := vars.DB
		var acc model.FarmAccount
		if err := db.Where("id = ?", req.AccountID).First(&acc).Error; err != nil {
			return nil, err
		}
	}
	n := hub.Logs.Clear(req.AccountID)
	out := map[string]any{"cleared": n}
	if req.AccountID != 0 {
		out["accountId"] = req.AccountID
	} else {
		out["cleared"] = "all"
	}
	// Notify WS clients so UIs can reset.
	hub.Default.Broadcast(hub.Event{
		Type:      "logs_cleared",
		AccountID: req.AccountID,
	})
	return out, nil
}
