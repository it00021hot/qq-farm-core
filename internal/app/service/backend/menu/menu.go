package menu

import (
	"context"
	"errors"
	"time"

	"github.com/MQEnergy/go-skeleton/internal/app/model"
	"github.com/MQEnergy/go-skeleton/internal/app/service"
	menutypes "github.com/MQEnergy/go-skeleton/internal/types/menu"
	"github.com/MQEnergy/go-skeleton/internal/vars"
	"github.com/MQEnergy/go-skeleton/pkg/tenant"
	"github.com/gofiber/fiber/v3"
)

type Service struct {
	service.Service
}

var Menu = &Service{}

func (s *Service) assertPlatform(ctx fiber.Ctx) error {
	isPlatform, _ := ctx.Locals(tenant.LocalIsPlatform).(bool)
	if !isPlatform {
		return errors.New("仅平台用户可维护菜单")
	}
	return nil
}

func (s *Service) Tree(ctx fiber.Ctx) ([]model.SysResource, error) {
	if err := s.assertPlatform(ctx); err != nil {
		return nil, err
	}
	var list []model.SysResource
	err := tenant.Global(vars.DB, context.Background()).Order("sort_order ASC, id ASC").Find(&list).Error
	return list, err
}

func (s *Service) Create(ctx fiber.Ctx, req menutypes.CreateReq) (*model.SysResource, error) {
	if err := s.assertPlatform(ctx); err != nil {
		return nil, err
	}
	now := uint64(time.Now().Unix())
	row := &model.SysResource{
		Name:         req.Name,
		Alias:        req.Alias,
		Desc:         req.Desc,
		FURL:         req.FURL,
		BURL:         req.BURL,
		ParentID:     req.ParentID,
		Path:         req.Path,
		ResourceType: req.ResourceType,
		Icon:         req.Icon,
		Status:       req.Status,
		SortOrder:    req.SortOrder,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if row.SortOrder == 0 {
		row.SortOrder = 50
	}
	if err := tenant.Global(vars.DB, context.Background()).Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

func (s *Service) Update(ctx fiber.Ctx, req menutypes.UpdateReq) error {
	if err := s.assertPlatform(ctx); err != nil {
		return err
	}
	db := tenant.Global(vars.DB, context.Background())
	var row model.SysResource
	if err := db.Where("id = ?", req.ID).First(&row).Error; err != nil {
		return errors.New("资源不存在")
	}
	return db.Model(&row).Updates(map[string]any{
		"name":          req.Name,
		"alias":         req.Alias,
		"desc":          req.Desc,
		"f_url":         req.FURL,
		"b_url":         req.BURL,
		"parent_id":     req.ParentID,
		"path":          req.Path,
		"resource_type": req.ResourceType,
		"icon":          req.Icon,
		"status":        req.Status,
		"sort_order":    req.SortOrder,
		"updated_at":    uint64(time.Now().Unix()),
	}).Error
}

func (s *Service) Delete(ctx fiber.Ctx, id uint64) error {
	if err := s.assertPlatform(ctx); err != nil {
		return err
	}
	db := tenant.Global(vars.DB, context.Background())
	var child int64
	if err := db.Model(&model.SysResource{}).Where("parent_id = ?", id).Count(&child).Error; err != nil {
		return err
	}
	if child > 0 {
		return errors.New("请先删除子资源")
	}
	return db.Where("id = ?", id).Delete(&model.SysResource{}).Error
}
