package permission

import (
	"github.com/MQEnergy/go-skeleton/internal/app/controller"
	permsvc "github.com/MQEnergy/go-skeleton/internal/app/service/platform/permission"
	"github.com/MQEnergy/go-skeleton/pkg/response"
	"github.com/gofiber/fiber/v3"
	"github.com/spf13/cast"
)

type Controller struct {
	controller.Controller
}

var Permission = &Controller{}

// ListAPIs 可绑定 API 列表
//
//	@Summary		可绑定后端 API 列表
//	@Description	仅平台用户；供菜单/按钮配置 b_url 与 methods
//	@Tags			权限管理
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Success		200	{object}	response.JSONResponse	"成功"
//	@Failure		400	{object}	response.JSONResponse	"请求错误"
//	@Router			/platform/permission/apis [get]
func (c *Controller) ListAPIs(ctx fiber.Ctx) error {
	list, err := permsvc.Permission.ListAPIs(ctx)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", list)
}

// RolePolicies 角色 Casbin 策略只读
//
//	@Summary		角色 Casbin 策略
//	@Description	仅平台用户；排障用
//	@Tags			权限管理
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			id	query		int						true	"角色ID"
//	@Success		200	{object}	response.JSONResponse	"成功"
//	@Failure		400	{object}	response.JSONResponse	"请求错误"
//	@Router			/platform/permission/role-policies [get]
func (c *Controller) RolePolicies(ctx fiber.Ctx) error {
	roleID := cast.ToUint64(ctx.Query("id"))
	if roleID == 0 {
		return response.BadRequestException(ctx, "角色ID无效")
	}
	list, err := permsvc.Permission.RolePolicies(ctx, roleID)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", list)
}

// Reload 重载 Casbin
//
//	@Summary		重载 Casbin 策略
//	@Description	仅平台用户
//	@Tags			权限管理
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Success		200	{object}	response.JSONResponse	"成功"
//	@Failure		400	{object}	response.JSONResponse	"请求错误"
//	@Router			/platform/permission/reload [post]
func (c *Controller) Reload(ctx fiber.Ctx) error {
	if err := permsvc.Permission.Reload(ctx); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", nil)
}
