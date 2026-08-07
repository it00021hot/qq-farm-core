package status

import (
	"github.com/MQEnergy/go-skeleton/internal/app/controller"
	statussvc "github.com/MQEnergy/go-skeleton/internal/app/service/farm/status"
	farmtypes "github.com/MQEnergy/go-skeleton/internal/types/farm"
	"github.com/MQEnergy/go-skeleton/pkg/response"
	"github.com/gofiber/fiber/v3"
)

type Controller struct {
	controller.Controller
}

var Status = &Controller{}

type (
	StatusDetailReq = farmtypes.StatusDetailReq
	StatusListReq   = farmtypes.StatusListReq
)

// Detail 账号运行状态详情
//
//	@Summary		账号运行状态详情
//	@Tags			农场状态
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			accountId	query		int						true	"账号ID"
//	@Success		200			{object}	response.JSONResponse	"成功"
//	@Failure		400			{object}	response.JSONResponse	"请求错误"
//	@Router			/farm/status/detail [get]
func (c *Controller) Detail(ctx fiber.Ctx) error {
	var req farmtypes.StatusDetailReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := statussvc.Status.Detail(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

// List 账号运行状态列表
//
//	@Summary		账号运行状态列表
//	@Tags			农场状态
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			current		query		int						false	"页码"
//	@Param			size		query		int						false	"每页条数"
//	@Param			keyword		query		string					false	"名称/编码关键词"
//	@Success		200			{object}	response.JSONResponse	"成功"
//	@Failure		400			{object}	response.JSONResponse	"请求错误"
//	@Router			/farm/status/list [get]
func (c *Controller) List(ctx fiber.Ctx) error {
	var req farmtypes.StatusListReq
	_ = c.Validate(ctx, &req)
	info, err := statussvc.Status.List(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}
