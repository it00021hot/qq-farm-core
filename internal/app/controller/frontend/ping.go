package frontend

import (
	"github.com/MQEnergy/go-skeleton/pkg/response"
	"github.com/gofiber/fiber/v3"
)

// Controller 前台公共接口
type Controller struct{}

var Frontend = &Controller{}

// Ping 前台 ping
//
//	@Summary		前台健康检查
//	@Description	前台/API 模块健康检查，需要登录
//	@Tags			前台公共
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Success		200	{object}	response.JSONResponse	"成功"
//	@Failure		401	{object}	response.JSONResponse	"未认证"
//	@Router			/api/ping [get]
func (c *Controller) Ping(ctx fiber.Ctx) error {
	return response.SuccessJSON(ctx, "", "api pong")
}
