package role

import (
	"context"
	"errors"
	"time"

	"github.com/MQEnergy/go-skeleton/internal/app/model"
	"github.com/MQEnergy/go-skeleton/internal/app/pkg/redis"
	"github.com/MQEnergy/go-skeleton/internal/app/service"
	roletypes "github.com/MQEnergy/go-skeleton/internal/types/role"
	"github.com/MQEnergy/go-skeleton/internal/vars"
	"github.com/MQEnergy/go-skeleton/pkg/rbac"
	"github.com/MQEnergy/go-skeleton/pkg/tenant"
	"github.com/gofiber/fiber/v3"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

type Service struct {
	service.Service
}

var Role = &Service{}

func (s *Service) assertPlatform(ctx fiber.Ctx) error {
	isPlatform, _ := ctx.Locals(tenant.LocalIsPlatform).(bool)
	if !isPlatform {
		return errors.New("仅平台用户可维护角色")
	}
	return nil
}

func (s *Service) Tree(ctx fiber.Ctx) ([]model.SysRole, error) {
	if err := s.assertPlatform(ctx); err != nil {
		return nil, err
	}
	var list []model.SysRole
	err := tenant.Global(vars.DB, context.Background()).Order("level ASC, id ASC").Find(&list).Error
	return list, err
}

func (s *Service) Create(ctx fiber.Ctx, req roletypes.CreateReq) (*model.SysRole, error) {
	if err := s.assertPlatform(ctx); err != nil {
		return nil, err
	}
	db := tenant.Global(vars.DB, context.Background())
	level := uint16(0)
	if req.ParentID > 0 {
		var parent model.SysRole
		if err := db.Where("id = ?", req.ParentID).First(&parent).Error; err != nil {
			return nil, errors.New("上级角色不存在")
		}
		level = parent.Level + 1
	}
	now := uint(time.Now().Unix())
	role := &model.SysRole{
		ParentID:  req.ParentID,
		Level:     level,
		Name:      req.Name,
		Code:      req.Code,
		Desc:      req.Desc,
		IsSys:     0,
		RoleType:  req.RoleType,
		Status:    req.Status,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := db.Create(role).Error; err != nil {
		return nil, err
	}
	return role, nil
}

func (s *Service) Update(ctx fiber.Ctx, req roletypes.UpdateReq) error {
	if err := s.assertPlatform(ctx); err != nil {
		return err
	}
	db := tenant.Global(vars.DB, context.Background())
	var role model.SysRole
	if err := db.Where("id = ?", req.ID).First(&role).Error; err != nil {
		return errors.New("角色不存在")
	}
	if role.IsSys == 1 && req.Code != role.Code {
		return errors.New("系统角色编码不可修改")
	}
	level := uint16(0)
	if req.ParentID > 0 {
		if req.ParentID == req.ID {
			return errors.New("上级角色不能是自己")
		}
		var parent model.SysRole
		if err := db.Where("id = ?", req.ParentID).First(&parent).Error; err != nil {
			return errors.New("上级角色不存在")
		}
		level = parent.Level + 1
	}
	return db.Model(&role).Updates(map[string]any{
		"parent_id":  req.ParentID,
		"level":      level,
		"name":       req.Name,
		"code":       req.Code,
		"desc":       req.Desc,
		"role_type":  req.RoleType,
		"status":     req.Status,
		"updated_at": uint(time.Now().Unix()),
	}).Error
}

func (s *Service) Delete(ctx fiber.Ctx, id uint64) error {
	if err := s.assertPlatform(ctx); err != nil {
		return err
	}
	db := tenant.Global(vars.DB, context.Background())
	var role model.SysRole
	if err := db.Where("id = ?", id).First(&role).Error; err != nil {
		return errors.New("角色不存在")
	}
	if role.IsSys == 1 {
		return errors.New("系统角色不可删除")
	}
	var child int64
	if err := db.Model(&model.SysRole{}).Where("parent_id = ?", id).Count(&child).Error; err != nil {
		return err
	}
	if child > 0 {
		return errors.New("请先删除子角色")
	}
	return db.Delete(&role).Error
}

func (s *Service) SetAuth(ctx fiber.Ctx, req roletypes.AuthReq) error {
	if err := s.assertPlatform(ctx); err != nil {
		return err
	}
	db := tenant.Global(vars.DB, context.Background())
	var role model.SysRole
	if err := db.Where("id = ?", req.RoleID).First(&role).Error; err != nil {
		return errors.New("角色不存在")
	}

	ids := rbac.ParseRoleIDs(req.ResourceIDs)
	if len(ids) > 0 {
		var count int64
		if err := db.Model(&model.SysResource{}).
			Where("id IN ? AND status = ?", ids, vars.StatusNormal).
			Count(&count).Error; err != nil {
			return err
		}
		if count != int64(len(ids)) {
			return errors.New("存在无效或已停用的资源ID")
		}
	}

	var auth model.SysRoleAuth
	err := db.Where("role_id = ?", req.RoleID).First(&auth).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		auth = model.SysRoleAuth{RoleID: req.RoleID, ResourceIds: req.ResourceIDs}
		if err := db.Create(&auth).Error; err != nil {
			return err
		}
	} else {
		if err := db.Model(&auth).Update("resource_ids", req.ResourceIDs).Error; err != nil {
			return err
		}
	}

	if err := rbac.SyncRoleCasbin(db, req.RoleID); err != nil {
		return err
	}
	_ = rbac.ReloadPolicy()
	redis.InvalidatePermsCache(context.Background())
	return nil
}

// GetAuth 查询角色已授权资源
func (s *Service) GetAuth(ctx fiber.Ctx, roleID uint64) (fiber.Map, error) {
	if err := s.assertPlatform(ctx); err != nil {
		return nil, err
	}
	db := tenant.Global(vars.DB, context.Background())
	var role model.SysRole
	if err := db.Where("id = ?", roleID).First(&role).Error; err != nil {
		return nil, errors.New("角色不存在")
	}
	var auth model.SysRoleAuth
	resourceIDs := ""
	idList := make([]uint64, 0)
	if err := db.Where("role_id = ?", roleID).First(&auth).Error; err == nil {
		resourceIDs = auth.ResourceIds
		idList = rbac.ParseRoleIDs(auth.ResourceIds)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	var resources []model.SysResource
	if len(idList) > 0 {
		if err := db.Where("id IN ?", idList).Order("sort_order ASC, id ASC").Find(&resources).Error; err != nil {
			return nil, err
		}
	}
	return fiber.Map{
		"roleId":         roleID,
		"resourceIds":    resourceIDs,
		"resourceIdList": idList,
		"resources":      resources,
	}, nil
}

// Assignable 租户侧可分配角色（按操作者子树裁剪）
func (s *Service) Assignable(ctx fiber.Ctx) ([]model.SysRole, error) {
	isSuper, _ := ctx.Locals(tenant.LocalIsSuper).(bool)
	roleIDsStr, _ := ctx.Locals(tenant.LocalRoleIDs).(string)
	operatorIDs := rbac.ParseRoleIDs(roleIDsStr)

	var all []model.SysRole
	if err := tenant.Global(vars.DB, context.Background()).Where("status = ?", vars.StatusNormal).Find(&all).Error; err != nil {
		return nil, err
	}
	if isSuper {
		out := make([]model.SysRole, 0)
		for _, r := range all {
			if r.RoleType == vars.RoleTypeTenant {
				out = append(out, r)
			}
		}
		return out, nil
	}
	return rbac.FilterAssignable(all, operatorIDs, vars.RoleTypeTenant), nil
}

// ValidateAssign 供用户服务调用
func (s *Service) ValidateAssign(ctx fiber.Ctx, targetRoleIDs string) error {
	isSuper, _ := ctx.Locals(tenant.LocalIsSuper).(bool)
	if isSuper {
		return nil
	}
	targets := rbac.ParseRoleIDs(targetRoleIDs)
	if len(targets) == 0 {
		return errors.New("请分配角色")
	}
	var all []model.SysRole
	if err := tenant.Global(vars.DB, context.Background()).Find(&all).Error; err != nil {
		return err
	}
	roleMap := make(map[uint64]model.SysRole, len(all))
	for _, r := range all {
		roleMap[r.ID] = r
	}
	for _, id := range targets {
		r, ok := roleMap[id]
		if !ok {
			return errors.New("角色不存在: " + cast.ToString(id))
		}
		if r.RoleType != vars.RoleTypeTenant {
			return errors.New("不能将平台专用角色分配给租户用户")
		}
	}
	operatorIDs := rbac.ParseRoleIDs(cast.ToString(ctx.Locals(tenant.LocalRoleIDs)))
	if !rbac.CanAssign(all, operatorIDs, targets) {
		return errors.New("不能分配高于自身权限的角色")
	}
	return nil
}
