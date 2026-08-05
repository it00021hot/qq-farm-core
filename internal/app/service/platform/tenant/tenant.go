package tenant

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/MQEnergy/go-skeleton/internal/app/model"
	"github.com/MQEnergy/go-skeleton/internal/app/pkg/pagination"
	"github.com/MQEnergy/go-skeleton/internal/app/service"
	tenanttypes "github.com/MQEnergy/go-skeleton/internal/types/tenant"
	"github.com/MQEnergy/go-skeleton/internal/vars"
	"github.com/MQEnergy/go-skeleton/pkg/helper"
	"github.com/MQEnergy/go-skeleton/pkg/response"
	"github.com/MQEnergy/go-skeleton/pkg/tenant"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

type Service struct {
	service.Service
}

var Tenant = &Service{}

func (s *Service) Create(req tenanttypes.CreateReq) (*model.SysTenant, error) {
	db := tenant.Global(vars.DB, context.Background())
	var exists int64
	if err := db.Model(&model.SysTenant{}).Where("code = ?", req.Code).Count(&exists).Error; err != nil {
		return nil, err
	}
	if exists > 0 {
		return nil, errors.New("租户编码已存在")
	}
	now := uint(time.Now().Unix())
	t := &model.SysTenant{
		Code:         req.Code,
		Name:         req.Name,
		Status:       vars.StatusNormal,
		MaxUsers:     req.MaxUsers,
		ExpireAt:     req.ExpireAt,
		ContactName:  req.ContactName,
		ContactPhone: req.ContactPhone,
		Remark:       req.Remark,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(t).Error; err != nil {
			return err
		}
		if req.AdminAccount == "" {
			return nil
		}
		if req.AdminPassword == "" {
			return errors.New("创建主账号时必须提供密码")
		}
		salt := helper.GenerateUuid(32)
		admin := &model.SysAdmin{
			UUID:      cast.ToString(helper.GenerateUUID()),
			TenantID:  t.ID,
			NickName:  req.AdminNickName,
			Account:   req.AdminAccount,
			Password:  helper.GeneratePasswordHash(req.AdminPassword, salt),
			Salt:      salt,
			RoleIds:   cast.ToString(vars.DefaultTenantAdminRoleID),
			Status:    vars.StatusNormal,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if admin.NickName == "" {
			admin.NickName = req.AdminAccount
		}
		return tx.Create(admin).Error
	})
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Service) Update(req tenanttypes.UpdateReq) error {
	db := tenant.Global(vars.DB, context.Background())
	var t model.SysTenant
	if err := db.Where("id = ?", req.ID).First(&t).Error; err != nil {
		return errors.New("租户不存在")
	}
	if req.MaxUsers > 0 {
		var used int64
		if err := db.Model(&model.SysAdmin{}).Where("tenant_id = ?", req.ID).Count(&used).Error; err != nil {
			return err
		}
		if uint(used) > req.MaxUsers {
			return fmt.Errorf("当前用户数 %d 已超过新配额 %d", used, req.MaxUsers)
		}
	}
	return db.Model(&t).Updates(map[string]any{
		"name":          req.Name,
		"max_users":     req.MaxUsers,
		"expire_at":     req.ExpireAt,
		"contact_name":  req.ContactName,
		"contact_phone": req.ContactPhone,
		"remark":        req.Remark,
		"status":        req.Status,
		"updated_at":    uint(time.Now().Unix()),
	}).Error
}

func (s *Service) UpdateStatus(req tenanttypes.StatusReq) error {
	db := tenant.Global(vars.DB, context.Background())
	res := db.Model(&model.SysTenant{}).Where("id = ?", req.ID).Updates(map[string]any{
		"status":     req.Status,
		"updated_at": uint(time.Now().Unix()),
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("租户不存在")
	}
	return nil
}

func (s *Service) Detail(id uint64) (map[string]any, error) {
	db := tenant.Global(vars.DB, context.Background())
	var t model.SysTenant
	if err := db.Where("id = ?", id).First(&t).Error; err != nil {
		return nil, errors.New("租户不存在")
	}
	var used int64
	_ = db.Model(&model.SysAdmin{}).Where("tenant_id = ?", id).Count(&used).Error
	return map[string]any{
		"tenant":     t,
		"used_users": used,
		"expired":    t.ExpireAt > 0 && uint64(t.ExpireAt) < uint64(time.Now().Unix()),
	}, nil
}

func (s *Service) List(req tenanttypes.ListReq) (response.PageData, error) {
	db := tenant.Global(vars.DB, context.Background()).Model(&model.SysTenant{})
	if req.Keyword != "" {
		kw := "%" + req.Keyword + "%"
		db = db.Where("code ILIKE ? OR name ILIKE ?", kw, kw)
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
	var list []model.SysTenant
	if err := db.Order("id DESC").Offset(page.GetOffset()).Limit(page.GetLimit()).Find(&list).Error; err != nil {
		return response.PageData{}, err
	}
	return response.NewPageData(list, req.Current, req.Size, total), nil
}

func (s *Service) BindAdminTenants(req tenanttypes.BindTenantReq) error {
	db := tenant.Global(vars.DB, context.Background())
	var admin model.SysAdmin
	if err := db.Where("id = ?", req.AdminID).First(&admin).Error; err != nil {
		return errors.New("用户不存在")
	}
	if admin.TenantID != 0 {
		return errors.New("仅平台用户可绑定多租户")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("admin_id = ?", req.AdminID).Delete(&model.SysAdminTenant{}).Error; err != nil {
			return err
		}
		now := uint(time.Now().Unix())
		for _, tid := range req.TenantIDs {
			if tid == 0 {
				continue
			}
			row := model.SysAdminTenant{AdminID: req.AdminID, TenantID: tid, CreatedAt: now}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
