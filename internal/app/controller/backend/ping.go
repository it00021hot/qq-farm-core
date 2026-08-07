package backend

import (
	"github.com/it00021hot/qq-farm-core/pkg/response"
	"github.com/gofiber/fiber/v3"
)

// Controller 后台公共接口
type Controller struct{}

var Backend = &Controller{}

// Ping 后台 ping
//
//	@Summary		后台健康检查
//	@Description	后台模块健康检查，需要登录
//	@Tags			后台公共
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Success		200	{object}	response.JSONResponse	"成功"
//	@Failure		401	{object}	response.JSONResponse	"未认证"
//	@Router			/system/ping [get]
func (c *Controller) Ping(ctx fiber.Ctx) error {
	return response.SuccessJSON(ctx, "", "backend pong")
}
