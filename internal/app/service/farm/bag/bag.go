package bag

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/it00021hot/qq-farm-core/internal/app/model"
	"github.com/it00021hot/qq-farm-core/internal/app/service"
	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/corepb"
	farmruntime "github.com/it00021hot/qq-farm-core/internal/farm/runtime"
	farmtypes "github.com/it00021hot/qq-farm-core/internal/types/farm"
	"github.com/it00021hot/qq-farm-core/internal/vars"
)

type Service struct {
	service.Service
}

var Bag = &Service{}

func (s *Service) Seeds(ctx fiber.Ctx, req farmtypes.BagReq) ([]logic.AvailableShopSeed, error) {
	var account model.FarmAccount
	if err := vars.DB.Where("id = ?", req.AccountID).First(&account).Error; err != nil {
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
			return nil, errors.New("出售物品参数无效")
		}
		sellItems = append(sellItems, corepb.Item{Id: it.ID, Count: it.Count, Uid: it.UID})
	}
	if len(sellItems) == 0 {
		return nil, errors.New("没有可出售的物品")
	}
	callCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	reply, err := session.SellBagItems(callCtx, sellItems)
	if err != nil {
		return nil, friendlyFarmErr(err)
	}
	// 汇总文案对齐 rust bag_sell：`出售 白萝卜×12，获得 金币×100`。
	sold := logic.AggregateGains(reply.GetSellItems())
	gained := logic.AggregateGains(reply.GetGetItems())
	var parts []string
	if text := logic.FormatGains(sold); text != "" {
		parts = append(parts, "出售 "+text)
	}
	if text := logic.FormatGains(gained); text != "" {
		parts = append(parts, "获得 "+text)
	}
	return map[string]any{
		"accountId": req.AccountID,
		"count":     len(sellItems),
		"ok":        true,
		"sold":      gainRows(sold),
		"gained":    gainRows(gained),
		"summary":   strings.Join(parts, "，"),
	}, nil
}

func (s *Service) Use(ctx fiber.Ctx, req farmtypes.BagUseReq) (map[string]any, error) {
	session, err := s.session(ctx, req.AccountID)
	if err != nil {
		return nil, friendlyFarmErr(err)
	}
	callCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	reply, err := session.UseBagItem(callCtx, req.ItemID, req.Count)
	if err != nil {
		return nil, friendlyFarmErr(err)
	}
	// 汇总文案对齐 rust bag_use：`获得 金币×100`（回包 items + land_reward）。
	var gained []logic.GainEntry
	if reply != nil {
		gained = logic.AggregateGains(append(append([]*corepb.Item{}, reply.GetItems()...), reply.GetLandReward().GetItems()...))
	}
	summary := ""
	if text := logic.FormatGains(gained); text != "" {
		summary = "获得 " + text
	}
	return map[string]any{
		"accountId": req.AccountID,
		"itemId":    req.ItemID,
		"count":     req.Count,
		"ok":        true,
		"rewards":   gainRows(gained),
		"summary":   summary,
	}, nil
}

// gainRows renders aggregated gains as {id, count, name} rows (rust gain_dtos).
func gainRows(entries []logic.GainEntry) []map[string]any {
	rows := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		rows = append(rows, map[string]any{
			"id":    e.ID,
			"count": e.Count,
			"name":  logic.GainDisplayName(e.ID),
		})
	}
	return rows
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

// SeedsInBag returns the seed entries currently in the account bag. Seeds are
// resolved from the full Plant.json catalog instead of the shop goods list, so
// activity/legacy seeds that are not sold in the shop are still listed for the
// bag-priority planting strategy.
func (s *Service) SeedsInBag(ctx fiber.Ctx, req farmtypes.BagReq) ([]logic.BagSeed, error) {
	session, err := s.session(ctx, req.AccountID)
	if err != nil {
		return nil, friendlyFarmErr(err)
	}
	callCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	seeds, err := session.BagSeeds(callCtx)
	if err != nil {
		return nil, friendlyFarmErr(err)
	}
	return seeds, nil
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
	if err := vars.DB.Where("id = ?", accountID).First(&account).Error; err != nil {
		return nil, errors.New("账号不存在")
	}
	session, ok := farmruntime.Default.Session(accountID)
	if !ok || session.Status() != farmruntime.StatusRunning {
		return nil, errors.New("账号未运行")
	}
	return session, nil
}
