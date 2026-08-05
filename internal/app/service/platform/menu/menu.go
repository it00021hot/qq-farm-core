package menu

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/MQEnergy/go-skeleton/internal/app/model"
	"github.com/MQEnergy/go-skeleton/internal/app/pkg/redis"
	"github.com/MQEnergy/go-skeleton/internal/app/service"
	menutypes "github.com/MQEnergy/go-skeleton/internal/types/menu"
	"github.com/MQEnergy/go-skeleton/internal/vars"
	"github.com/MQEnergy/go-skeleton/pkg/rbac"
	"github.com/MQEnergy/go-skeleton/pkg/tenant"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
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

func validateResourceFields(resourceType uint8, alias, bURL, methods string) error {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return errors.New("alias 不能为空")
	}
	bURL = strings.TrimSpace(bURL)
	methods = strings.TrimSpace(methods)
	if bURL != "" {
		if !strings.HasPrefix(bURL, "/") {
			return errors.New("b_url 必须以 / 开头")
		}
		if strings.Contains(bURL, "*") {
			return errors.New("b_url 禁止使用通配符，请配置精确路径")
		}
		if rbac.NormalizeMethods(methods) == "" {
			return errors.New("配置 b_url 时必须填写 methods")
		}
		norm := rbac.NormalizeMethods(methods)
		for _, m := range strings.Split(norm, ",") {
			if m != "GET" && m != "POST" {
				return errors.New("methods 仅允许 GET、POST")
			}
		}
	}
	_ = resourceType
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
	if err := validateResourceFields(req.ResourceType, req.Alias, req.BURL, req.Methods); err != nil {
		return nil, err
	}
	now := uint64(time.Now().Unix())
	row := &model.SysResource{
		Name:         req.Name,
		Alias:        req.Alias,
		Desc:         req.Desc,
		FURL:         req.FURL,
		BURL:         strings.TrimSpace(req.BURL),
		Methods:      rbac.NormalizeMethods(req.Methods),
		ParentID:     req.ParentID,
		Path:         req.Path,
		ResourceType: req.ResourceType,
		Icon:         req.Icon,
		HideInMenu:   req.HideInMenu,
		Status:       req.Status,
		SortOrder:    req.SortOrder,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if row.HideInMenu == 0 {
		row.HideInMenu = 1
	}
	if row.SortOrder == 0 {
		row.SortOrder = 50
	}
	if err := tenant.Global(vars.DB, context.Background()).Create(row).Error; err != nil {
		return nil, err
	}
	redis.InvalidatePermsCache(context.Background())
	return row, nil
}

func (s *Service) Update(ctx fiber.Ctx, req menutypes.UpdateReq) error {
	if err := s.assertPlatform(ctx); err != nil {
		return err
	}
	if err := validateResourceFields(req.ResourceType, req.Alias, req.BURL, req.Methods); err != nil {
		return err
	}
	db := tenant.Global(vars.DB, context.Background())
	var row model.SysResource
	if err := db.Where("id = ?", req.ID).First(&row).Error; err != nil {
		return errors.New("资源不存在")
	}
	if err := db.Model(&row).Updates(map[string]any{
		"name":          req.Name,
		"alias":         req.Alias,
		"desc":          req.Desc,
		"f_url":         req.FURL,
		"b_url":         strings.TrimSpace(req.BURL),
		"methods":       rbac.NormalizeMethods(req.Methods),
		"parent_id":     req.ParentID,
		"path":          req.Path,
		"resource_type": req.ResourceType,
		"icon":          req.Icon,
		"hide_in_menu":  req.HideInMenu,
		"status":        req.Status,
		"sort_order":    req.SortOrder,
		"updated_at":    uint64(time.Now().Unix()),
	}).Error; err != nil {
		return err
	}
	return s.resyncAffectedRoles(db, req.ID)
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
	roleIDs, err := rbac.FindRoleIDsByResource(db, id)
	if err != nil {
		return err
	}
	if err := db.Where("id = ?", id).Delete(&model.SysResource{}).Error; err != nil {
		return err
	}
	if err := rbac.SyncRolesCasbin(db, roleIDs); err != nil {
		return err
	}
	_ = rbac.ReloadPolicy()
	redis.InvalidatePermsCache(context.Background())
	return nil
}

func (s *Service) resyncAffectedRoles(db *gorm.DB, resourceID uint64) error {
	roleIDs, err := rbac.FindRoleIDsByResource(db, resourceID)
	if err != nil {
		return err
	}
	if err := rbac.SyncRolesCasbin(db, roleIDs); err != nil {
		return err
	}
	_ = rbac.ReloadPolicy()
	redis.InvalidatePermsCache(context.Background())
	return nil
}
