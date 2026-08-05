package tenant

import (
	"github.com/MQEnergy/go-skeleton/internal/app/controller"
	tenantsvc "github.com/MQEnergy/go-skeleton/internal/app/service/platform/tenant"
	tenanttypes "github.com/MQEnergy/go-skeleton/internal/types/tenant"
	"github.com/MQEnergy/go-skeleton/pkg/response"
	"github.com/MQEnergy/go-skeleton/pkg/tenant"
	"github.com/gofiber/fiber/v3"
)

type Controller struct {
	controller.Controller
}

var Tenant = &Controller{}

// swagger 类型别名
type (
	CreateReq     = tenanttypes.CreateReq
	UpdateReq     = tenanttypes.UpdateReq
	StatusReq     = tenanttypes.StatusReq
	ListReq       = tenanttypes.ListReq
	IDReq         = tenanttypes.IDReq
	BindTenantReq = tenanttypes.BindTenantReq
)

func (c *Controller) assertPlatform(ctx fiber.Ctx) error {
	isPlatform, _ := ctx.Locals(tenant.LocalIsPlatform).(bool)
	if !isPlatform {
		return response.ForbiddenException(ctx, "仅平台用户可管理租户")
	}
	return nil
}

// Create 创建租户
//
//	@Summary		创建租户
//	@Description	平台用户创建租户；可选同时创建主账号。无需 X-Tenant-ID
//	@Tags			租户管理
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			payload	body		CreateReq				true	"创建参数"
//	@Success		200		{object}	response.JSONResponse	"成功"
//	@Failure		400		{object}	response.JSONResponse	"请求错误"
//	@Failure		401		{object}	response.JSONResponse	"未授权"
//	@Failure		403		{object}	response.JSONResponse	"无权限"
//	@Router			/platform/tenant/add [post]
func (c *Controller) Create(ctx fiber.Ctx) error {
	if err := c.assertPlatform(ctx); err != nil {
		return err
	}
	var req tenanttypes.CreateReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := tenantsvc.Tenant.Create(req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

// Update 更新租户
//
//	@Summary		更新租户
//	@Description	平台用户更新租户信息（含配额/过期/状态）
//	@Tags			租户管理
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			payload	body		UpdateReq				true	"更新参数"
//	@Success		200		{object}	response.JSONResponse	"成功"
//	@Failure		400		{object}	response.JSONResponse	"请求错误"
//	@Failure		403		{object}	response.JSONResponse	"无权限"
//	@Router			/platform/tenant/modify [post]
func (c *Controller) Update(ctx fiber.Ctx) error {
	if err := c.assertPlatform(ctx); err != nil {
		return err
	}
	var req tenanttypes.UpdateReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	if err := tenantsvc.Tenant.Update(req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", nil)
}

// Status 启停租户
//
//	@Summary		启停租户
//	@Description	平台用户启用或禁用租户
//	@Tags			租户管理
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			payload	body		StatusReq				true	"启停参数"
//	@Success		200		{object}	response.JSONResponse	"成功"
//	@Failure		400		{object}	response.JSONResponse	"请求错误"
//	@Failure		403		{object}	response.JSONResponse	"无权限"
//	@Router			/platform/tenant/status [post]
func (c *Controller) Status(ctx fiber.Ctx) error {
	if err := c.assertPlatform(ctx); err != nil {
		return err
	}
	var req tenanttypes.StatusReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	if err := tenantsvc.Tenant.UpdateStatus(req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", nil)
}

// List 租户列表
//
//	@Summary		租户列表
//	@Description	平台用户分页查询租户
//	@Tags			租户管理
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			current	query		int						false	"页码"
//	@Param			size	query		int						false	"每页条数"
//	@Param			keyword	query		string					false	"编码/名称关键词"
//	@Param			status	query		int						false	"状态 1正常 2禁用"
//	@Success		200		{object}	response.JSONResponse	"成功"
//	@Failure		403		{object}	response.JSONResponse	"无权限"
//	@Router			/platform/tenant/list [get]
func (c *Controller) List(ctx fiber.Ctx) error {
	if err := c.assertPlatform(ctx); err != nil {
		return err
	}
	var req tenanttypes.ListReq
	_ = c.Validate(ctx, &req)
	info, err := tenantsvc.Tenant.List(req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

// Detail 租户详情
//
//	@Summary		租户详情
//	@Description	含 used_users / expired
//	@Tags			租户管理
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			id	query		int						true	"租户ID"
//	@Success		200	{object}	response.JSONResponse	"成功"
//	@Failure		400	{object}	response.JSONResponse	"请求错误"
//	@Failure		403	{object}	response.JSONResponse	"无权限"
//	@Router			/platform/tenant/detail [get]
func (c *Controller) Detail(ctx fiber.Ctx) error {
	if err := c.assertPlatform(ctx); err != nil {
		return err
	}
	var req tenanttypes.IDReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := tenantsvc.Tenant.Detail(req.ID)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

// Bind 平台用户绑定租户
//
//	@Summary		平台用户绑定租户
//	@Description	为平台账号设置可管理的租户列表（全量覆盖）
//	@Tags			租户管理
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			payload	body		BindTenantReq			true	"绑定参数"
//	@Success		200		{object}	response.JSONResponse	"成功"
//	@Failure		400		{object}	response.JSONResponse	"请求错误"
//	@Failure		403		{object}	response.JSONResponse	"无权限"
//	@Router			/platform/tenant/bind [post]
func (c *Controller) Bind(ctx fiber.Ctx) error {
	if err := c.assertPlatform(ctx); err != nil {
		return err
	}
	var req tenanttypes.BindTenantReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	if err := tenantsvc.Tenant.BindAdminTenants(req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", nil)
}
