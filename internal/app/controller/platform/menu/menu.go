package menu

import (
	"github.com/MQEnergy/go-skeleton/internal/app/controller"
	menusvc "github.com/MQEnergy/go-skeleton/internal/app/service/platform/menu"
	menutypes "github.com/MQEnergy/go-skeleton/internal/types/menu"
	"github.com/MQEnergy/go-skeleton/pkg/response"
	"github.com/gofiber/fiber/v3"
)

type Controller struct {
	controller.Controller
}

var Menu = &Controller{}

type (
	CreateReq = menutypes.CreateReq
	UpdateReq = menutypes.UpdateReq
	IDReq     = menutypes.IDReq
)

// Tree 菜单树
//
//	@Summary		菜单树
//	@Description	仅平台用户
//	@Tags			菜单管理
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Success		200	{object}	response.JSONResponse	"成功"
//	@Failure		400	{object}	response.JSONResponse	"请求错误"
//	@Failure		403	{object}	response.JSONResponse	"无权限"
//	@Router			/platform/menu/tree [get]
func (c *Controller) Tree(ctx fiber.Ctx) error {
	list, err := menusvc.Menu.Tree(ctx)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", list)
}

// Create 创建菜单
//
//	@Summary		创建菜单/按钮
//	@Description	仅平台用户；resource_type 1目录 2菜单 3操作
//	@Tags			菜单管理
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			payload	body		CreateReq				true	"创建参数"
//	@Success		200		{object}	response.JSONResponse	"成功"
//	@Failure		400		{object}	response.JSONResponse	"请求错误"
//	@Failure		403		{object}	response.JSONResponse	"无权限"
//	@Router			/platform/menu/add [post]
func (c *Controller) Create(ctx fiber.Ctx) error {
	var req menutypes.CreateReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := menusvc.Menu.Create(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

// Update 更新菜单
//
//	@Summary		更新菜单/按钮
//	@Description	仅平台用户
//	@Tags			菜单管理
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			payload	body		UpdateReq				true	"更新参数"
//	@Success		200		{object}	response.JSONResponse	"成功"
//	@Failure		400		{object}	response.JSONResponse	"请求错误"
//	@Failure		403		{object}	response.JSONResponse	"无权限"
//	@Router			/platform/menu/modify [post]
func (c *Controller) Update(ctx fiber.Ctx) error {
	var req menutypes.UpdateReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	if err := menusvc.Menu.Update(ctx, req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", nil)
}

// Delete 删除菜单
//
//	@Summary		删除菜单/按钮
//	@Description	仅平台用户；有子节点时不可删
//	@Tags			菜单管理
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			payload	body		IDReq					true	"资源ID"
//	@Success		200		{object}	response.JSONResponse	"成功"
//	@Failure		400		{object}	response.JSONResponse	"请求错误"
//	@Failure		403		{object}	response.JSONResponse	"无权限"
//	@Router			/platform/menu/delete [post]
func (c *Controller) Delete(ctx fiber.Ctx) error {
	var req menutypes.IDReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	if err := menusvc.Menu.Delete(ctx, req.ID); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", nil)
}
