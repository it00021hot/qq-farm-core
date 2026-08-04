package auth

import (
	"github.com/MQEnergy/go-skeleton/internal/app/controller"
	auth2 "github.com/MQEnergy/go-skeleton/internal/app/service/backend/user/auth"
	"github.com/MQEnergy/go-skeleton/internal/types/user/auth"
	"github.com/MQEnergy/go-skeleton/internal/vars"
	"github.com/MQEnergy/go-skeleton/pkg/response"

	"github.com/gofiber/fiber/v3"
)

type AuthController struct {
	controller.Controller
}

var Auth = &AuthController{}

// LoginReq swagger 登录参数（复用 types）
type LoginReq = auth.LoginReq

// Login 用户登录
//
//	@Summary		用户登录
//	@Description	用户登录接口
//	@Tags			鉴权管理
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		LoginReq				true	"登录请求参数"
//	@Success		200		{object}	response.JSONResponse	"成功"
//	@Failure		400		{object}	response.JSONResponse	"请求错误"
//	@Router			/backend/auth/login [post]
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

// Logout 退出登录
//
//	@Summary		退出登录
//	@Description	用户退出登录
//	@Tags			鉴权管理
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Success		200	{object}	response.JSONResponse	"成功"
//	@Failure		400	{object}	response.JSONResponse	"请求错误"
//	@Router			/backend/auth/logout [post]
func (c *AuthController) Logout(ctx fiber.Ctx) error {
	uuid := ctx.GetRespHeader("uuid")
	if err := auth2.Auth.Logout(uuid); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", "")
}

// Routes 获取所有后端路由
//
//	@Summary		获取所有后端路由
//	@Description	获取所有后端路由接口
//	@Tags			鉴权管理
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Success		200	{object}	response.JSONResponse	"成功"
//	@Failure		400	{object}	response.JSONResponse	"请求错误"
//	@Router			/backend/auth/routes [get]
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
