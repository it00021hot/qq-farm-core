package admin

import (
	"github.com/MQEnergy/go-skeleton/internal/app/controller"
	adminsvc "github.com/MQEnergy/go-skeleton/internal/app/service/system/admin"
	admintypes "github.com/MQEnergy/go-skeleton/internal/types/admin"
	"github.com/MQEnergy/go-skeleton/pkg/response"
	"github.com/gofiber/fiber/v3"
)

type Controller struct {
	controller.Controller
}

var Admin = &Controller{}

type (
	CreateReq         = admintypes.CreateReq
	UpdateReq         = admintypes.UpdateReq
	ListReq           = admintypes.ListReq
	StatusReq         = admintypes.StatusReq
	PlatformCreateReq = admintypes.PlatformCreateReq
)

// List 用户列表
//
//	@Summary		用户列表
//	@Description	租户上下文下列出用户。平台用户必须传 Header X-Tenant-ID
//	@Tags			用户管理
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			X-Tenant-ID	header		string					false	"平台用户操作目标租户ID"
//	@Param			current		query		int						false	"页码"
//	@Param			size		query		int						false	"每页条数"
//	@Param			keyword		query		string					false	"账号/昵称关键词"
//	@Param			status		query		int						false	"状态 1正常 2禁用"
//	@Success		200			{object}	response.JSONResponse	"成功"
//	@Failure		400			{object}	response.JSONResponse	"请求错误"
//	@Failure		401			{object}	response.JSONResponse	"未授权"
//	@Failure		403			{object}	response.JSONResponse	"无权限"
//	@Router			/system/admin/list [get]
func (c *Controller) List(ctx fiber.Ctx) error {
	var req admintypes.ListReq
	_ = c.Validate(ctx, &req)
	info, err := adminsvc.Admin.List(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

// Create 创建用户
//
//	@Summary		创建租户用户
//	@Description	在当前租户上下文创建用户并分配角色（不高于操作者）。平台用户必须传 X-Tenant-ID
//	@Tags			用户管理
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			X-Tenant-ID	header		string					false	"平台用户操作目标租户ID"
//	@Param			payload		body		CreateReq				true	"创建参数"
//	@Success		200			{object}	response.JSONResponse	"成功"
//	@Failure		400			{object}	response.JSONResponse	"请求错误"
//	@Failure		403			{object}	response.JSONResponse	"无权限"
//	@Router			/system/admin/add [post]
func (c *Controller) Create(ctx fiber.Ctx) error {
	var req admintypes.CreateReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := adminsvc.Admin.Create(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

// Update 更新用户
//
//	@Summary		更新租户用户
//	@Description	更新用户资料与角色。平台用户必须传 X-Tenant-ID
//	@Tags			用户管理
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			X-Tenant-ID	header		string					false	"平台用户操作目标租户ID"
//	@Param			payload		body		UpdateReq				true	"更新参数"
//	@Success		200			{object}	response.JSONResponse	"成功"
//	@Failure		400			{object}	response.JSONResponse	"请求错误"
//	@Failure		403			{object}	response.JSONResponse	"无权限"
//	@Router			/system/admin/modify [post]
func (c *Controller) Update(ctx fiber.Ctx) error {
	var req admintypes.UpdateReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	if err := adminsvc.Admin.Update(ctx, req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", nil)
}

// Status 启停用户
//
//	@Summary		启停租户用户
//	@Description	启用或禁用用户。平台用户必须传 X-Tenant-ID
//	@Tags			用户管理
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			X-Tenant-ID	header		string					false	"平台用户操作目标租户ID"
//	@Param			payload		body		StatusReq				true	"启停参数"
//	@Success		200			{object}	response.JSONResponse	"成功"
//	@Failure		400			{object}	response.JSONResponse	"请求错误"
//	@Failure		403			{object}	response.JSONResponse	"无权限"
//	@Router			/system/admin/status [post]
func (c *Controller) Status(ctx fiber.Ctx) error {
	var req admintypes.StatusReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	if err := adminsvc.Admin.UpdateStatus(ctx, req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", nil)
}

// CreatePlatform 创建平台用户
//
//	@Summary		创建平台用户
//	@Description	仅平台超管；可同时绑定可管理租户。无需 X-Tenant-ID
//	@Tags			用户管理
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			payload	body		PlatformCreateReq		true	"创建参数"
//	@Success		200		{object}	response.JSONResponse	"成功"
//	@Failure		400		{object}	response.JSONResponse	"请求错误"
//	@Failure		403		{object}	response.JSONResponse	"无权限"
//	@Router			/system/platform-user/add [post]
func (c *Controller) CreatePlatform(ctx fiber.Ctx) error {
	var req admintypes.PlatformCreateReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := adminsvc.Admin.CreatePlatform(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}
