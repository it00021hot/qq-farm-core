package account

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/MQEnergy/go-skeleton/internal/app/model"
	"github.com/MQEnergy/go-skeleton/internal/app/pkg/pagination"
	"github.com/MQEnergy/go-skeleton/internal/app/service"
	"github.com/MQEnergy/go-skeleton/internal/farm/logic"
	"github.com/MQEnergy/go-skeleton/internal/farm/loginurl"
	farmruntime "github.com/MQEnergy/go-skeleton/internal/farm/runtime"
	farmtypes "github.com/MQEnergy/go-skeleton/internal/types/farm"
	"github.com/MQEnergy/go-skeleton/internal/vars"
	"github.com/MQEnergy/go-skeleton/pkg/response"
	"github.com/MQEnergy/go-skeleton/pkg/tenant"
	"github.com/gofiber/fiber/v3"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

type Service struct {
	service.Service
}

var Account = &Service{}

func normalizeLoginInput(rawCode, rawPlatform string) (code, platform, loginOS, clientVer string, err error) {
	rawCode = strings.TrimSpace(rawCode)
	if rawCode == "" {
		return "", "", "", "", errors.New("请输入登录 Code 或完整登录 URL")
	}
	code = loginurl.ExtractCode(rawCode)
	if code == "" {
		return "", "", "", "", errors.New("无法解析登录 Code，请粘贴裸 Code 或含 code= 的登录 URL")
	}
	if len(code) > 512 {
		return "", "", "", "", errors.New("登录 Code 过长")
	}

	hints := loginurl.ExtractClientHints(rawCode)
	platform = loginurl.NormalizePlatform(hints.Platform)
	if platform == "" {
		platform = loginurl.NormalizePlatform(rawPlatform)
	}
	if platform == "" {
		platform = "qq"
	}
	loginOS = strings.TrimSpace(hints.OS)
	clientVer = strings.TrimSpace(hints.Ver)
	return code, platform, loginOS, clientVer, nil
}

func (s *Service) List(ctx fiber.Ctx, req farmtypes.AccountListReq) (response.PageData, error) {
	if cast.ToUint64(ctx.Locals(tenant.LocalTenantID)) == 0 {
		return response.PageData{}, errors.New("缺少租户上下文")
	}
	db := tenant.Scope(vars.DB, tenant.TenantCtx(ctx)).Model(&model.FarmAccount{})
	if req.Keyword != "" {
		kw := "%" + req.Keyword + "%"
		db = db.Where("name LIKE ? OR code LIKE ? OR qq LIKE ? OR uin LIKE ?", kw, kw, kw, kw)
	}
	if req.Status > 0 {
		db = db.Where("status = ?", req.Status)
	}
	// Empty query runStatus= must not filter: binders may coerce "" -> 0 (stopped-only).
	if raw := strings.TrimSpace(ctx.Query("runStatus")); raw != "" && req.RunStatus != nil {
		db = db.Where("run_status = ?", *req.RunStatus)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return response.PageData{}, err
	}
	page := pagination.New().ParsePage(req.Current, req.Size)
	page.Total = total
	page.GetLastPage()
	var list []model.FarmAccount
	if err := db.Order("id DESC").Offset(page.GetOffset()).Limit(page.GetLimit()).Find(&list).Error; err != nil {
		return response.PageData{}, err
	}
	// Overlay live runtime status (DB can lag after crash; boot resets, but prefer truth).
	for i := range list {
		st, _ := farmruntime.Default.GetStatus(list[i].ID)
		list[i].RunStatus = st
	}
	return response.NewPageData(list, req.Current, req.Size, total), nil
}

func (s *Service) Detail(ctx fiber.Ctx, id uint64) (*model.FarmAccount, error) {
	if cast.ToUint64(ctx.Locals(tenant.LocalTenantID)) == 0 {
		return nil, errors.New("缺少租户上下文")
	}
	var acc model.FarmAccount
	if err := tenant.Scope(vars.DB, tenant.TenantCtx(ctx)).Where("id = ?", id).First(&acc).Error; err != nil {
		return nil, errors.New("账号不存在")
	}
	return &acc, nil
}

func (s *Service) Create(ctx fiber.Ctx, req farmtypes.AccountCreateReq) (*model.FarmAccount, error) {
	tid := cast.ToUint64(ctx.Locals(tenant.LocalTenantID))
	if err := farmruntime.EnsureTenantActive(tid); err != nil {
		return nil, err
	}
	if err := farmruntime.EnsureAccountQuota(tid); err != nil {
		return nil, err
	}

	code, platform, loginOS, clientVer, err := normalizeLoginInput(req.Code, req.Platform)
	if err != nil {
		return nil, err
	}

	status := req.Status
	if status == 0 {
		status = vars.StatusNormal
	}
	name := strings.TrimSpace(req.Name)
	now := uint(time.Now().Unix())
	acc := &model.FarmAccount{
		TenantID:  tid,
		Name:      name,
		Code:      code,
		Platform:  platform,
		LoginOS:   loginOS,
		ClientVer: clientVer,
		Uin:       req.Uin,
		QQ:        req.QQ,
		Avatar:    req.Avatar,
		Remark:    req.Remark,
		RunStatus: farmruntime.RunStopped,
		Status:    status,
		CreatedAt: now,
		UpdatedAt: now,
	}
	db := tenant.Scope(vars.DB, tenant.TenantCtx(ctx))
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(acc).Error; err != nil {
			return err
		}
		if acc.Name == "" {
			acc.Name = fmt.Sprintf("账号%d", acc.ID)
			if err := tx.Model(acc).Update("name", acc.Name).Error; err != nil {
				return err
			}
		}
		cfgJSON, _ := json.Marshal(logic.DefaultAccountConfig())
		cfg := &model.FarmAccountConfig{
			TenantID:   tid,
			AccountID:  acc.ID,
			ConfigJSON: string(cfgJSON),
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		return tx.Create(cfg).Error
	}); err != nil {
		return nil, err
	}
	return acc, nil
}

func (s *Service) Update(ctx fiber.Ctx, req farmtypes.AccountUpdateReq) error {
	db := tenant.Scope(vars.DB, tenant.TenantCtx(ctx))
	var acc model.FarmAccount
	if err := db.Where("id = ?", req.ID).First(&acc).Error; err != nil {
		return errors.New("账号不存在")
	}

	code, platform, loginOS, clientVer, err := normalizeLoginInput(req.Code, req.Platform)
	if err != nil {
		return err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = acc.Name
	}
	if name == "" {
		name = fmt.Sprintf("账号%d", acc.ID)
	}

	codeChanged := code != strings.TrimSpace(acc.Code)

	updates := map[string]any{
		"name":       name,
		"code":       code,
		"platform":   platform,
		"uin":        req.Uin,
		"qq":         req.QQ,
		"avatar":     req.Avatar,
		"remark":     req.Remark,
		"status":     req.Status,
		"updated_at": uint(time.Now().Unix()),
	}
	if loginOS != "" {
		updates["login_os"] = loginOS
	}
	if clientVer != "" {
		updates["client_ver"] = clientVer
	}
	if err := db.Model(&acc).Updates(updates).Error; err != nil {
		return err
	}

	// Refreshing login code implies reconnect: start (or restart) when account stays enabled.
	if codeChanged && req.Status == vars.StatusNormal {
		if err := s.Start(ctx, req.ID); err != nil {
			return fmt.Errorf("账号已更新，自动启动失败: %w", err)
		}
	}
	return nil
}

func (s *Service) Delete(ctx fiber.Ctx, id uint64) error {
	_ = farmruntime.Default.Stop(id)
	db := tenant.Scope(vars.DB, tenant.TenantCtx(ctx))
	var acc model.FarmAccount
	if err := db.Where("id = ?", id).First(&acc).Error; err != nil {
		return errors.New("账号不存在")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("account_id = ?", id).Delete(&model.FarmAccountConfig{}).Error; err != nil {
			return err
		}
		return tx.Delete(&acc).Error
	})
}

func (s *Service) Start(ctx fiber.Ctx, id uint64) error {
	tid := cast.ToUint64(ctx.Locals(tenant.LocalTenantID))
	if err := farmruntime.EnsureTenantActive(tid); err != nil {
		return err
	}
	db := tenant.Scope(vars.DB, tenant.TenantCtx(ctx))
	var acc model.FarmAccount
	if err := db.Where("id = ?", id).First(&acc).Error; err != nil {
		return errors.New("账号不存在")
	}
	if acc.Status != vars.StatusNormal {
		return errors.New("账号已禁用")
	}
	if strings.TrimSpace(acc.Code) == "" {
		return errors.New("账号缺少登录 Code，请先编辑并填入 Code / 登录 URL")
	}
	// Blocks until gateway connect + Login succeed or fail; only then is run_status=running.
	if err := farmruntime.Default.Start(id); err != nil {
		return err
	}
	return nil
}

func (s *Service) Stop(ctx fiber.Ctx, id uint64) error {
	db := tenant.Scope(vars.DB, tenant.TenantCtx(ctx))
	var acc model.FarmAccount
	if err := db.Where("id = ?", id).First(&acc).Error; err != nil {
		return errors.New("账号不存在")
	}
	if err := farmruntime.Default.Stop(id); err != nil {
		// Still force DB stopped when session already gone
		_ = db.Model(&acc).Updates(map[string]any{
			"run_status": farmruntime.RunStopped,
			"updated_at": uint(time.Now().Unix()),
		}).Error
		if strings.Contains(err.Error(), "not found") {
			return nil
		}
		return err
	}
	return nil
}
