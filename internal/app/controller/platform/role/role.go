package role

import (
	"github.com/MQEnergy/go-skeleton/internal/app/controller"
	rolesvc "github.com/MQEnergy/go-skeleton/internal/app/service/platform/role"
	roletypes "github.com/MQEnergy/go-skeleton/internal/types/role"
	"github.com/MQEnergy/go-skeleton/pkg/response"
	"github.com/gofiber/fiber/v3"
)

type Controller struct {
	controller.Controller
}

var Role = &Controller{}

type (
	CreateReq = roletypes.CreateReq
	UpdateReq = roletypes.UpdateReq
	IDReq     = roletypes.IDReq
	AuthReq   = roletypes.AuthReq
)

// Tree 角色树
//
//	@Summary		角色树
//	@Description	仅平台用户可查看完整角色树
//	@Tags			角色管理
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Success		200	{object}	response.JSONResponse	"成功"
//	@Failure		400	{object}	response.JSONResponse	"请求错误"
//	@Failure		403	{object}	response.JSONResponse	"无权限"
//	@Router			/platform/role/tree [get]
func (c *Controller) Tree(ctx fiber.Ctx) error {
	list, err := rolesvc.Role.Tree(ctx)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", list)
}

// Create 创建角色
//
//	@Summary		创建角色
//	@Description	仅平台用户；role_type=1 平台专用，=2 可赋给租户用户
//	@Tags			角色管理
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			payload	body		CreateReq				true	"创建参数"
//	@Success		200		{object}	response.JSONResponse	"成功"
//	@Failure		400		{object}	response.JSONResponse	"请求错误"
//	@Failure		403		{object}	response.JSONResponse	"无权限"
//	@Router			/platform/role/add [post]
func (c *Controller) Create(ctx fiber.Ctx) error {
	var req roletypes.CreateReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := rolesvc.Role.Create(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

// Update 更新角色
//
//	@Summary		更新角色
//	@Description	仅平台用户
//	@Tags			角色管理
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			payload	body		UpdateReq				true	"更新参数"
//	@Success		200		{object}	response.JSONResponse	"成功"
//	@Failure		400		{object}	response.JSONResponse	"请求错误"
//	@Failure		403		{object}	response.JSONResponse	"无权限"
//	@Router			/platform/role/modify [post]
func (c *Controller) Update(ctx fiber.Ctx) error {
	var req roletypes.UpdateReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	if err := rolesvc.Role.Update(ctx, req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", nil)
}

// Delete 删除角色
//
//	@Summary		删除角色
//	@Description	仅平台用户；系统角色不可删
//	@Tags			角色管理
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			payload	body		IDReq					true	"角色ID"
//	@Success		200		{object}	response.JSONResponse	"成功"
//	@Failure		400		{object}	response.JSONResponse	"请求错误"
//	@Failure		403		{object}	response.JSONResponse	"无权限"
//	@Router			/platform/role/delete [post]
func (c *Controller) Delete(ctx fiber.Ctx) error {
	var req roletypes.IDReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	if err := rolesvc.Role.Delete(ctx, req.ID); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", nil)
}

// SetAuth 角色菜单授权
//
//	@Summary		角色菜单授权
//	@Description	仅平台用户；resource_ids 为逗号分隔菜单ID；同步写入 Casbin
//	@Tags			角色管理
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			payload	body		AuthReq					true	"授权参数"
//	@Success		200		{object}	response.JSONResponse	"成功"
//	@Failure		400		{object}	response.JSONResponse	"请求错误"
//	@Failure		403		{object}	response.JSONResponse	"无权限"
//	@Router			/platform/role/auth [post]
func (c *Controller) SetAuth(ctx fiber.Ctx) error {
	var req roletypes.AuthReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	if err := rolesvc.Role.SetAuth(ctx, req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", nil)
}

// GetAuth 查询角色授权
//
//	@Summary		查询角色菜单授权
//	@Description	仅平台用户
//	@Tags			角色管理
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			role_id	query		int						true	"角色ID"
//	@Success		200		{object}	response.JSONResponse	"成功"
//	@Failure		400		{object}	response.JSONResponse	"请求错误"
//	@Failure		403		{object}	response.JSONResponse	"无权限"
//	@Router			/platform/role/auth [get]
func (c *Controller) GetAuth(ctx fiber.Ctx) error {
	var req roletypes.GetAuthReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := rolesvc.Role.GetAuth(ctx, req.RoleID)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

// Assignable 可分配角色
//
//	@Summary		可分配角色列表
//	@Description	按操作者角色子树裁剪，且仅返回 role_type=租户 的角色。租户与平台均可调用
//	@Tags			角色管理
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Success		200	{object}	response.JSONResponse	"成功"
//	@Failure		400	{object}	response.JSONResponse	"请求错误"
//	@Router			/platform/role/assignable [get]
func (c *Controller) Assignable(ctx fiber.Ctx) error {
	list, err := rolesvc.Role.Assignable(ctx)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", list)
}
