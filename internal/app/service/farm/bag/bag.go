package bag

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/MQEnergy/go-skeleton/internal/app/model"
	"github.com/MQEnergy/go-skeleton/internal/app/service"
	"github.com/MQEnergy/go-skeleton/internal/farm/logic"
	"github.com/MQEnergy/go-skeleton/internal/farm/proto/corepb"
	farmruntime "github.com/MQEnergy/go-skeleton/internal/farm/runtime"
	farmtypes "github.com/MQEnergy/go-skeleton/internal/types/farm"
	"github.com/MQEnergy/go-skeleton/internal/vars"
	"github.com/MQEnergy/go-skeleton/pkg/tenant"
	"github.com/gofiber/fiber/v3"
)

type Service struct {
	service.Service
}

var Bag = &Service{}

func (s *Service) Seeds(ctx fiber.Ctx, req farmtypes.BagReq) ([]logic.AvailableShopSeed, error) {
	var account model.FarmAccount
	if err := tenant.Scope(vars.DB, tenant.TenantCtx(ctx)).Where("id = ?", req.AccountID).First(&account).Error; err != nil {
		return nil, errors.New("账号不存在")
	}
	session, ok := farmruntime.Default.Session(req.AccountID)
	if ok && session.Status() == farmruntime.StatusRunning {
		callCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		list, err := session.GetAvailableSeeds(callCtx)
		if err == nil && len(list) > 0 {
			return list, nil
		}
		// Shop/gateway failure: fall through to catalog so settings page still works.
	}
	return logic.CatalogAvailableSeeds(0), nil
}

func (s *Service) Sell(ctx fiber.Ctx, req farmtypes.BagSellReq) (map[string]any, error) {
	session, err := s.session(ctx, req.AccountID)
	if err != nil {
		return nil, friendlyFarmErr(err)
	}
	sellItems := make([]corepb.Item, 0, len(req.Items))
	for _, it := range req.Items {
		if it.ID <= 0 || it.Count <= 0 {
			continue
		}
		sellItems = append(sellItems, corepb.Item{ID: it.ID, Count: it.Count, UID: it.UID})
	}
	if len(sellItems) == 0 {
		return nil, errors.New("缺少出售物品")
	}
	callCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := session.SellBagItems(callCtx, sellItems); err != nil {
		return nil, friendlyFarmErr(err)
	}
	return map[string]any{
		"accountId": req.AccountID,
		"count":     len(sellItems),
		"ok":        true,
	}, nil
}

func (s *Service) Get(ctx fiber.Ctx, req farmtypes.BagReq) (logic.BagUIResponse, error) {
	session, err := s.session(ctx, req.AccountID)
	if err != nil {
		return logic.BagUIResponse{}, friendlyFarmErr(err)
	}
	callCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	items, err := session.GetBagItems(callCtx)
	if err != nil {
		return logic.BagUIResponse{}, friendlyFarmErr(err)
	}
	return logic.FormatBagResponse(items), nil
}

func friendlyFarmErr(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "EOF") ||
		strings.Contains(msg, "i/o timeout") {
		return errors.New("游戏连接已断开，请重新启动账号")
	}
	return err
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
