package lands

import (
	"context"
	"errors"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/app/model"
	"github.com/it00021hot/qq-farm-core/internal/app/service"
	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
	farmruntime "github.com/it00021hot/qq-farm-core/internal/farm/runtime"
	farmtypes "github.com/it00021hot/qq-farm-core/internal/types/farm"
	"github.com/it00021hot/qq-farm-core/internal/vars"
	"github.com/gofiber/fiber/v3"
)

type Service struct {
	service.Service
}

var Lands = &Service{}

func (s *Service) Get(ctx fiber.Ctx, req farmtypes.LandsReq) (logic.LandsUIResponse, error) {
	session, err := s.session(ctx, req.AccountID)
	if err != nil {
		return logic.LandsUIResponse{}, err
	}
	callCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	lands, err := session.GetLands(callCtx)
	if err != nil {
		return logic.LandsUIResponse{}, err
	}
	return logic.FormatLandsResponse(lands), nil
}

func (s *Service) Operate(ctx fiber.Ctx, req farmtypes.OperateReq) (map[string]any, error) {
	session, err := s.session(ctx, req.AccountID)
	if err != nil {
		return nil, err
	}
	callCtx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	hadWork, actions, err := session.RunFarmOp(callCtx, req.Op)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"accountId": req.AccountID,
		"op":        req.Op,
		"hadWork":   hadWork,
		"actions":   actions,
	}, nil
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
