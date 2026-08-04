package common

import (
	"github.com/MQEnergy/go-skeleton/internal/vars"
	"github.com/MQEnergy/go-skeleton/pkg/jwtauth"
	"github.com/MQEnergy/go-skeleton/pkg/response"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
)

// Controller 公共接口
type Controller struct{}

var Common = &Controller{}

// Index 根路径
//
//	@Summary		服务首页
//	@Description	服务根路径健康探测
//	@Tags			公共接口
//	@Produce		json
//	@Success		200	{object}	response.JSONResponse	"成功"
//	@Router			/ [get]
func (c *Controller) Index(ctx fiber.Ctx) error {
	return response.SuccessJSON(ctx, "", "")
}

// Ping 公共 ping
//
//	@Summary		健康检查
//	@Description	公共健康检查接口
//	@Tags			公共接口
//	@Produce		json
//	@Success		200	{object}	response.JSONResponse	"成功"
//	@Router			/ping [get]
func (c *Controller) Ping(ctx fiber.Ctx) error {
	return response.SuccessJSON(ctx, "", "pong")
}

// TokenCreate 创建 demo JWT
//
//	@Summary		创建 Demo Token
//	@Description	演示用 JWT 签发，固定账号 john/doe
//	@Tags			公共接口
//	@Accept			x-www-form-urlencoded
//	@Produce		json
//	@Param			user	formData	string					true	"用户名"	default(john)
//	@Param			pass	formData	string					true	"密码"	default(doe)
//	@Success		200		{object}	response.JSONResponse	"成功，data 为 token 字符串"
//	@Failure		401		{object}	response.JSONResponse	"认证失败"
//	@Router			/token/create [post]
func (c *Controller) TokenCreate(ctx fiber.Ctx) error {
	user := ctx.FormValue("user")
	pass := ctx.FormValue("pass")
	if user != "john" || pass != "doe" {
		return response.UnauthorizedException(ctx, "")
	}
	token, err := jwtauth.New(&vars.Config).WithClaims(jwt.MapClaims{
		"name": user,
	}).GenerateToken()
	if err != nil {
		return response.UnauthorizedException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", token)
}

// TokenView 查看当前 Token 解析结果
//
//	@Summary		查看 Token Claims
//	@Description	解析 Authorization Bearer Token，返回 uid/uuid/role_ids
//	@Tags			公共接口
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Success		200	{object}	response.JSONResponse	"成功"
//	@Failure		401	{object}	response.JSONResponse	"未认证"
//	@Router			/token/view [post]
func (c *Controller) TokenView(ctx fiber.Ctx) error {
	return response.SuccessJSON(ctx, "", fiber.Map{
		"uid":      ctx.GetRespHeader("uid"),
		"uuid":     ctx.GetRespHeader("uuid"),
		"role_ids": ctx.GetRespHeader("role_ids"),
	})
}
