package admin

import (
	"errors"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/app/model"
	"github.com/it00021hot/qq-farm-core/internal/app/pkg/pagination"
	"github.com/it00021hot/qq-farm-core/internal/app/service"
	admintypes "github.com/it00021hot/qq-farm-core/internal/types/admin"
	"github.com/it00021hot/qq-farm-core/internal/vars"
	"github.com/it00021hot/qq-farm-core/pkg/helper"
	"github.com/it00021hot/qq-farm-core/pkg/response"
	"github.com/gofiber/fiber/v3"
	"github.com/spf13/cast"
)

type Service struct{ service.Service }

var Admin = &Service{}

const defaultRoleIDs = "1"

func (s *Service) List(_ fiber.Ctx, req admintypes.ListReq) (response.PageData, error) {
	db := vars.DB.Model(&model.SysAdmin{})
	if req.Keyword != "" {
		kw := "%" + req.Keyword + "%"
		db = db.Where("account LIKE ? OR nick_name LIKE ?", kw, kw)
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

func (s *Service) Create(_ fiber.Ctx, req admintypes.CreateReq) (*model.SysAdmin, error) {
	var exists int64
	if err := vars.DB.Model(&model.SysAdmin{}).Where("account = ?", req.Account).Count(&exists).Error; err != nil {
		return nil, err
	}
	if exists > 0 {
		return nil, errors.New("账号已存在")
	}
	roleIDs := req.RoleIDs
	if roleIDs == "" {
		roleIDs = defaultRoleIDs
	}
	salt, now := helper.GenerateUuid(32), uint(time.Now().Unix())
	admin := &model.SysAdmin{
		UUID:      cast.ToString(helper.GenerateUUID()),
		NickName:  req.NickName,
		RealName:  req.RealName,
		Account:   req.Account,
		Password:  helper.GeneratePasswordHash(req.Password, salt),
		Salt:      salt,
		Phone:     req.Phone,
		Email:     req.Email,
		RoleIds:   roleIDs,
		Status:    req.Status,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := vars.DB.Create(admin).Error; err != nil {
		return nil, err
	}
	admin.Password = ""
	admin.Salt = ""
	return admin, nil
}

func (s *Service) Update(_ fiber.Ctx, req admintypes.UpdateReq) error {
	var admin model.SysAdmin
	if err := vars.DB.Where("id = ?", req.ID).First(&admin).Error; err != nil {
		return errors.New("用户不存在")
	}
	roleIDs := req.RoleIDs
	if roleIDs == "" {
		roleIDs = defaultRoleIDs
	}
	updates := map[string]any{
		"nick_name":  req.NickName,
		"real_name":  req.RealName,
		"phone":      req.Phone,
		"email":      req.Email,
		"role_ids":   roleIDs,
		"status":     req.Status,
		"updated_at": uint(time.Now().Unix()),
	}
	if req.Password != "" {
		salt := helper.GenerateUuid(32)
		updates["salt"] = salt
		updates["password"] = helper.GeneratePasswordHash(req.Password, salt)
	}
	return vars.DB.Model(&admin).Updates(updates).Error
}

func (s *Service) UpdateStatus(_ fiber.Ctx, req admintypes.StatusReq) error {
	res := vars.DB.Model(&model.SysAdmin{}).Where("id = ?", req.ID).Updates(map[string]any{
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
