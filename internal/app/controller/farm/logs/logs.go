package logs

import (
	"github.com/MQEnergy/go-skeleton/internal/app/controller"
	logssvc "github.com/MQEnergy/go-skeleton/internal/app/service/farm/logs"
	farmtypes "github.com/MQEnergy/go-skeleton/internal/types/farm"
	"github.com/MQEnergy/go-skeleton/pkg/response"
	"github.com/gofiber/fiber/v3"
)

type Controller struct {
	controller.Controller
}

var Logs = &Controller{}

// List 运行日志（内存环缓冲）
//
//	@Summary		运行日志列表
//	@Tags			农场日志
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			accountId	query		int						false	"账号ID"
//	@Param			module		query		string					false	"模块 farm|friend|system"
//	@Param			keyword		query		string					false	"关键词"
//	@Param			limit		query		int						false	"条数上限"
//	@Success		200			{object}	response.JSONResponse	"成功"
//	@Failure		400			{object}	response.JSONResponse	"请求错误"
//	@Router			/farm/logs [get]
func (c *Controller) List(ctx fiber.Ctx) error {
	var req farmtypes.LogsListReq
	_ = c.Validate(ctx, &req)
	list, err := logssvc.Logs.List(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", list)
}

// Clear 清空运行日志
//
//	@Summary		清空运行日志
//	@Tags			农场日志
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			accountId	query		int						false	"账号ID（空则清空全部）"
//	@Success		200			{object}	response.JSONResponse	"成功"
//	@Failure		400			{object}	response.JSONResponse	"请求错误"
//	@Router			/farm/logs [delete]
func (c *Controller) Clear(ctx fiber.Ctx) error {
	var req farmtypes.LogsClearReq
	_ = c.Validate(ctx, &req)
	// Also accept JSON body accountId for clients that POST-style delete.
	if req.AccountID == 0 {
		var body farmtypes.LogsClearReq
		_ = ctx.Bind().Body(&body)
		if body.AccountID != 0 {
			req.AccountID = body.AccountID
		}
	}
	info, err := logssvc.Logs.Clear(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}
