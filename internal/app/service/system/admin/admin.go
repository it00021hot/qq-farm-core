package admin

import (
	"context"
	"errors"
	"time"

	"github.com/MQEnergy/go-skeleton/internal/app/model"
	"github.com/MQEnergy/go-skeleton/internal/app/pkg/pagination"
	"github.com/MQEnergy/go-skeleton/internal/app/service"
	rolesvc "github.com/MQEnergy/go-skeleton/internal/app/service/platform/role"
	admintypes "github.com/MQEnergy/go-skeleton/internal/types/admin"
	"github.com/MQEnergy/go-skeleton/internal/vars"
	"github.com/MQEnergy/go-skeleton/pkg/helper"
	"github.com/MQEnergy/go-skeleton/pkg/response"
	"github.com/MQEnergy/go-skeleton/pkg/tenant"
	"github.com/gofiber/fiber/v3"
	"github.com/spf13/cast"
)

type Service struct {
	service.Service
}

var Admin = &Service{}

func (s *Service) checkQuota(tenantID uint64) error {
	if tenantID == 0 {
		return nil
	}
	db := tenant.Global(vars.DB, context.Background())
	var t model.SysTenant
	if err := db.Where("id = ?", tenantID).First(&t).Error; err != nil {
		return errors.New("租户不存在")
	}
	if t.MaxUsers == 0 {
		return nil
	}
	var used int64
	if err := db.Model(&model.SysAdmin{}).Where("tenant_id = ?", tenantID).Count(&used).Error; err != nil {
		return err
	}
	if uint(used) >= t.MaxUsers {
		return errors.New("已达到租户用户数上限")
	}
	return nil
}

func (s *Service) List(ctx fiber.Ctx, req admintypes.ListReq) (response.PageData, error) {
	tctx := tenant.TenantCtx(ctx)
	tid := tenant.MustID(tctx)
	db := tenant.Scope(vars.DB, tctx).Model(&model.SysAdmin{})
	if tid == 0 {
		// 平台未切租户时不应走到这里；兜底只查租户用户需 Skip+显式条件
		db = tenant.Global(vars.DB, context.Background()).Model(&model.SysAdmin{}).Where("tenant_id = ?", cast.ToUint64(ctx.Locals(tenant.LocalTenantID)))
	}
	if req.Keyword != "" {
		kw := "%" + req.Keyword + "%"
		db = db.Where("account ILIKE ? OR nick_name ILIKE ?", kw, kw)
	}
	if req.Status > 0 {
		db = db.Where("status = ?", req.Status)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return response.PageData{}, err
	}
	page := pagination.New().ParsePage(req.Current, req.Size)
	page.Total = total
	page.GetLastPage()
	var list []model.SysAdmin
	if err := db.Order("id DESC").Offset(page.GetOffset()).Limit(page.GetLimit()).Find(&list).Error; err != nil {
		return response.PageData{}, err
	}
	for i := range list {
		list[i].Password = ""
		list[i].Salt = ""
	}
	return response.NewPageData(list, req.Current, req.Size, total), nil
}

func (s *Service) Create(ctx fiber.Ctx, req admintypes.CreateReq) (*model.SysAdmin, error) {
	tid := cast.ToUint64(ctx.Locals(tenant.LocalTenantID))
	if tid == 0 {
		return nil, errors.New("缺少租户上下文")
	}
	if err := s.checkQuota(tid); err != nil {
		return nil, err
	}
	if err := rolesvc.Role.ValidateAssign(ctx, req.RoleIDs); err != nil {
		return nil, err
	}
	db := tenant.Global(vars.DB, context.Background())
	var exists int64
	if err := db.Model(&model.SysAdmin{}).Where("tenant_id = ? AND account = ?", tid, req.Account).Count(&exists).Error; err != nil {
		return nil, err
	}
	if exists > 0 {
		return nil, errors.New("账号已存在")
	}
	salt := helper.GenerateUuid(32)
	now := uint(time.Now().Unix())
	admin := &model.SysAdmin{
		UUID:      cast.ToString(helper.GenerateUUID()),
		TenantID:  tid,
		NickName:  req.NickName,
		RealName:  req.RealName,
		Account:   req.Account,
		Password:  helper.GeneratePasswordHash(req.Password, salt),
		Salt:      salt,
		Phone:     req.Phone,
		Email:     req.Email,
		RoleIds:   req.RoleIDs,
		Status:    req.Status,
		CreatedAt: now,
		UpdatedAt: now,
	}
	tctx := tenant.WithTenantID(context.Background(), tid)
	if err := tenant.Scope(vars.DB, tctx).Create(admin).Error; err != nil {
		return nil, err
	}
	admin.Password = ""
	admin.Salt = ""
	return admin, nil
}

func (s *Service) Update(ctx fiber.Ctx, req admintypes.UpdateReq) error {
	tid := cast.ToUint64(ctx.Locals(tenant.LocalTenantID))
	if err := rolesvc.Role.ValidateAssign(ctx, req.RoleIDs); err != nil {
		return err
	}
	tctx := tenant.WithTenantID(context.Background(), tid)
	db := tenant.Scope(vars.DB, tctx)
	var admin model.SysAdmin
	if err := db.Where("id = ?", req.ID).First(&admin).Error; err != nil {
		return errors.New("用户不存在")
	}
	updates := map[string]any{
		"nick_name":  req.NickName,
		"real_name":  req.RealName,
		"phone":      req.Phone,
		"email":      req.Email,
		"role_ids":   req.RoleIDs,
		"status":     req.Status,
		"updated_at": uint(time.Now().Unix()),
	}
	if req.Password != "" {
		salt := helper.GenerateUuid(32)
		updates["salt"] = salt
		updates["password"] = helper.GeneratePasswordHash(req.Password, salt)
	}
	return db.Model(&admin).Updates(updates).Error
}

func (s *Service) UpdateStatus(ctx fiber.Ctx, req admintypes.StatusReq) error {
	tid := cast.ToUint64(ctx.Locals(tenant.LocalTenantID))
	tctx := tenant.WithTenantID(context.Background(), tid)
	res := tenant.Scope(vars.DB, tctx).Model(&model.SysAdmin{}).Where("id = ?", req.ID).Updates(map[string]any{
		"status":     req.Status,
		"updated_at": uint(time.Now().Unix()),
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("用户不存在")
	}
	return nil
}

func (s *Service) CreatePlatform(ctx fiber.Ctx, req admintypes.PlatformCreateReq) (*model.SysAdmin, error) {
	isPlatform, _ := ctx.Locals(tenant.LocalIsPlatform).(bool)
	isSuper, _ := ctx.Locals(tenant.LocalIsSuper).(bool)
	if !isPlatform || !isSuper {
		return nil, errors.New("仅平台超管可创建平台用户")
	}
	db := tenant.Global(vars.DB, context.Background())
	var exists int64
	if err := db.Model(&model.SysAdmin{}).Where("tenant_id = 0 AND account = ?", req.Account).Count(&exists).Error; err != nil {
		return nil, err
	}
	if exists > 0 {
		return nil, errors.New("账号已存在")
	}
	salt := helper.GenerateUuid(32)
	now := uint(time.Now().Unix())
	admin := &model.SysAdmin{
		UUID:      cast.ToString(helper.GenerateUUID()),
		TenantID:  0,
		NickName:  req.NickName,
		Account:   req.Account,
		Password:  helper.GeneratePasswordHash(req.Password, salt),
		Salt:      salt,
		RoleIds:   req.RoleIDs,
		Status:    req.Status,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := db.Create(admin).Error; err != nil {
		return nil, err
	}
	for _, tid := range req.TenantIDs {
		if tid == 0 {
			continue
		}
		_ = db.Create(&model.SysAdminTenant{AdminID: admin.ID, TenantID: tid, CreatedAt: now}).Error
	}
	admin.Password = ""
	admin.Salt = ""
	return admin, nil
}
