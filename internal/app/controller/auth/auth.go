package auth

import (
	"github.com/MQEnergy/go-skeleton/internal/app/controller"
	auth2 "github.com/MQEnergy/go-skeleton/internal/app/service/auth"
	"github.com/MQEnergy/go-skeleton/internal/types/user/auth"
	"github.com/MQEnergy/go-skeleton/internal/vars"
	"github.com/MQEnergy/go-skeleton/pkg/response"

	"github.com/gofiber/fiber/v3"
)

type AuthController struct {
	controller.Controller
}

var Auth = &AuthController{}

type (
	LoginReq           = auth.LoginReq
	RefreshReq         = auth.RefreshReq
	ChangePasswordReq  = auth.ChangePasswordReq
)

// Login 用户登录
//
//	@Summary		用户登录
//	@Tags			鉴权管理
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		LoginReq				true	"登录请求参数"
//	@Success		200		{object}	response.JSONResponse	"成功"
//	@Failure		400		{object}	response.JSONResponse	"请求错误"
//	@Router			/auth/login [post]
func (c *AuthController) Login(ctx fiber.Ctx) error {
	var reqParams auth.LoginReq
	if err := c.Validate(ctx, &reqParams); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := auth2.Auth.Login(reqParams)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

// Refresh 刷新 token
//
//	@Summary		刷新访问令牌
//	@Tags			鉴权管理
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		RefreshReq				true	"refreshToken"
//	@Success		200		{object}	response.JSONResponse	"成功"
//	@Router			/auth/refresh [post]
func (c *AuthController) Refresh(ctx fiber.Ctx) error {
	var reqParams auth.RefreshReq
	if err := c.Validate(ctx, &reqParams); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := auth2.Auth.Refresh(reqParams)
	if err != nil {
		return response.UnauthorizedException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

// Logout 退出登录
//
//	@Summary		用户退出登录
//	@Tags			鉴权管理
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Success		200	{object}	response.JSONResponse	"成功"
//	@Router			/auth/logout [post]
func (c *AuthController) Logout(ctx fiber.Ctx) error {
	uuid := ctx.GetRespHeader("uuid")
	if err := auth2.Auth.Logout(uuid); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", "")
}

// ChangePassword 修改当前用户密码
//
//	@Summary		修改当前用户密码
//	@Tags			鉴权管理
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			payload	body		ChangePasswordReq		true	"旧密码/新密码"
//	@Success		200		{object}	response.JSONResponse	"成功"
//	@Router			/auth/password [post]
func (c *AuthController) ChangePassword(ctx fiber.Ctx) error {
	var reqParams auth.ChangePasswordReq
	if err := c.Validate(ctx, &reqParams); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	if err := auth2.Auth.ChangePassword(ctx, reqParams); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "密码修改成功，请重新登录", nil)
}

// Info 当前用户权限信息
//
//	@Summary		当前用户信息与权限
//	@Tags			鉴权管理
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Success		200	{object}	response.JSONResponse	"成功"
//	@Router			/auth/info [get]
func (c *AuthController) Info(ctx fiber.Ctx) error {
	info, err := auth2.Auth.Info(ctx)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

// UserRoutes soybean 动态路由
//
//	@Summary		获取当前用户前端路由
//	@Tags			鉴权管理
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Success		200	{object}	response.JSONResponse	"成功"
//	@Router			/auth/user-routes [get]
func (c *AuthController) UserRoutes(ctx fiber.Ctx) error {
	info, err := auth2.Auth.UserRoutes(ctx)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

// Routes 获取所有后端路由
//
//	@Summary		获取所有后端路由
//	@Tags			鉴权管理
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Success		200	{object}	response.JSONResponse	"成功"
//	@Router			/auth/routes [get]
func (c *AuthController) Routes(ctx fiber.Ctx) error {
	routeList := make([]fiber.Route, 0)
	for _, route := range vars.Routes {
		if route.Method != "GET" && route.Method != "POST" {
			continue
		}
		routeList = append(routeList, route)
	}
	return response.SuccessJSON(ctx, "", routeList)
}
