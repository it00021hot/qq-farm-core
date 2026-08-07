package gameconfig

import (
	"errors"
	"time"

	"github.com/MQEnergy/go-skeleton/internal/app/model"
	"github.com/MQEnergy/go-skeleton/internal/app/pkg/pagination"
	"github.com/MQEnergy/go-skeleton/internal/app/service"
	"github.com/MQEnergy/go-skeleton/internal/farm/logic"
	farmtypes "github.com/MQEnergy/go-skeleton/internal/types/farm"
	"github.com/MQEnergy/go-skeleton/internal/vars"
	"github.com/MQEnergy/go-skeleton/pkg/tenant"
	"github.com/gofiber/fiber/v3"
)

type Service struct {
	service.Service
}

var GameConfig = &Service{}

func (s *Service) List(ctx fiber.Ctx, req farmtypes.GameConfigListReq) (map[string]any, error) {
	db := tenant.Global(vars.DB, ctx.Context()).Model(&model.FarmGameConfig{})
	if req.Category != "" {
		db = db.Where("category = ?", req.Category)
	}
	if req.Keyword != "" {
		db = db.Where("config_key LIKE ?", "%"+req.Keyword+"%")
	}
	var total int64
	_ = db.Count(&total).Error
	page := pagination.New().ParsePage(req.Current, req.Size)
	page.Total = total
	page.GetLastPage()
	var list []model.FarmGameConfig
	_ = db.Order("id desc").Offset(page.GetOffset()).Limit(page.GetLimit()).Find(&list).Error
	return map[string]any{"list": list, "total": total, "page": page}, nil
}

func (s *Service) Modify(ctx fiber.Ctx, req farmtypes.GameConfigModifyReq) error {
	db := tenant.Global(vars.DB, ctx.Context())
	var row model.FarmGameConfig
	if err := db.Where("id = ?", req.ID).First(&row).Error; err != nil {
		return errors.New("配置不存在")
	}
	updates := map[string]any{
		"config_json": req.ConfigJSON,
		"updated_at":  uint(time.Now().Unix()),
		"version":     row.Version + 1,
	}
	if req.Status > 0 {
		updates["status"] = req.Status
	}
	return db.Model(&row).Updates(updates).Error
}

func (s *Service) Seeds(_ fiber.Ctx) ([]logic.SeedInfo, error) {
	return logic.GetAllSeeds(), nil
}

func (s *Service) Fruits(_ fiber.Ctx) ([]map[string]any, error) {
	return logic.GetAllFruits(), nil
}

func (s *Service) Items(_ fiber.Ctx, req farmtypes.GameConfigItemsReq) ([]map[string]any, error) {
	return logic.GetAllItems(req.Type), nil
}

func (s *Service) Plants(_ fiber.Ctx) ([]map[string]any, error) {
	return logic.GetAllPlants(), nil
}

func (s *Service) ItemTypes(_ fiber.Ctx) ([]map[string]any, error) {
	return logic.ItemTypes(), nil
}

func (s *Service) SeedAdd(_ fiber.Ctx, req farmtypes.GameConfigSeedWriteReq) (map[string]any, error) {
	return logic.GlobalGameConfig.AddSeed(toSeedWrite(req))
}

func (s *Service) SeedModify(_ fiber.Ctx, req farmtypes.GameConfigSeedWriteReq) (map[string]any, error) {
	return logic.GlobalGameConfig.ModifySeed(toSeedWrite(req))
}

func (s *Service) SeedDelete(_ fiber.Ctx, req farmtypes.GameConfigSeedDeleteReq) (map[string]any, error) {
	return logic.GlobalGameConfig.DeleteSeed(req.SeedID)
}

func (s *Service) FruitAdd(_ fiber.Ctx, req farmtypes.GameConfigFruitWriteReq) (map[string]any, error) {
	return logic.GlobalGameConfig.AddFruit(toFruitWrite(req))
}

func (s *Service) FruitModify(_ fiber.Ctx, req farmtypes.GameConfigFruitWriteReq) (map[string]any, error) {
	return logic.GlobalGameConfig.ModifyFruit(toFruitWrite(req))
}

func (s *Service) FruitDelete(_ fiber.Ctx, req farmtypes.GameConfigFruitDeleteReq) (map[string]any, error) {
	return logic.GlobalGameConfig.DeleteFruit(req.ID)
}

func (s *Service) ItemAdd(_ fiber.Ctx, req farmtypes.GameConfigItemWriteReq) (map[string]any, error) {
	return logic.GlobalGameConfig.AddItem(toItemWrite(req))
}

func (s *Service) ItemModify(_ fiber.Ctx, req farmtypes.GameConfigItemWriteReq) (map[string]any, error) {
	return logic.GlobalGameConfig.ModifyItem(toItemWrite(req))
}

func (s *Service) ItemDelete(_ fiber.Ctx, req farmtypes.GameConfigItemDeleteReq) (map[string]any, error) {
	return logic.GlobalGameConfig.DeleteItem(req.ID)
}

func toSeedWrite(req farmtypes.GameConfigSeedWriteReq) logic.SeedWriteReq {
	return logic.SeedWriteReq{
		SeedID:        req.SeedID,
		Name:          req.Name,
		GrowPhases:    req.GrowPhases,
		LandLevelNeed: req.LandLevelNeed,
		Seasons:       req.Seasons,
		FruitCount:    req.FruitCount,
		Price:         req.Price,
		PriceID:       req.PriceID,
		Exp:           req.Exp,
		Size:          req.Size,
	}
}

func toFruitWrite(req farmtypes.GameConfigFruitWriteReq) logic.FruitWriteReq {
	return logic.FruitWriteReq{
		ID:         req.ID,
		PlantID:    req.PlantID,
		Name:       req.Name,
		Price:      req.Price,
		PriceID:    req.PriceID,
		Desc:       req.Desc,
		EffectDesc: req.EffectDesc,
		Rarity:     req.Rarity,
		MaxCount:   req.MaxCount,
		Level:      req.Level,
		FruitCount: req.FruitCount,
		AssetName:  req.AssetName,
	}
}

func toItemWrite(req farmtypes.GameConfigItemWriteReq) logic.ItemWriteReq {
	return logic.ItemWriteReq{
		ID:              req.ID,
		Type:            req.Type,
		Name:            req.Name,
		Price:           req.Price,
		PriceID:         req.PriceID,
		InteractionType: req.InteractionType,
		CanUse:          req.CanUse,
		Desc:            req.Desc,
		EffectDesc:      req.EffectDesc,
		Rarity:          req.Rarity,
		MaxCount:        req.MaxCount,
		Level:           req.Level,
		AssetName:       req.AssetName,
	}
}
