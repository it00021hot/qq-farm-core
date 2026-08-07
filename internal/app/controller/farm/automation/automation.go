package automation

import (
	"github.com/MQEnergy/go-skeleton/internal/app/controller"
	automationsvc "github.com/MQEnergy/go-skeleton/internal/app/service/farm/automation"
	farmtypes "github.com/MQEnergy/go-skeleton/internal/types/farm"
	"github.com/MQEnergy/go-skeleton/pkg/response"
	"github.com/gofiber/fiber/v3"
)

type Controller struct {
	controller.Controller
}

var Automation = &Controller{}

type (
	AutomationDetailReq = farmtypes.AutomationDetailReq
	AutomationModifyReq = farmtypes.AutomationModifyReq
)

// Detail 自动化配置详情
//
//	@Summary		自动化配置详情
//	@Tags			农场自动化
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			X-Tenant-ID	header		string					false	"平台用户操作目标租户ID"
//	@Param			accountId	query		int						true	"账号ID"
//	@Success		200			{object}	response.JSONResponse	"成功"
//	@Failure		400			{object}	response.JSONResponse	"请求错误"
//	@Router			/farm/automation/detail [get]
func (c *Controller) Detail(ctx fiber.Ctx) error {
	var req farmtypes.AutomationDetailReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := automationsvc.Automation.Detail(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

// Modify 修改自动化配置
//
//	@Summary		修改自动化配置
//	@Tags			农场自动化
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			X-Tenant-ID	header		string					false	"平台用户操作目标租户ID"
//	@Param			payload		body		AutomationModifyReq		true	"配置参数"
//	@Success		200			{object}	response.JSONResponse	"成功"
//	@Failure		400			{object}	response.JSONResponse	"请求错误"
//	@Router			/farm/automation/modify [post]
func (c *Controller) Modify(ctx fiber.Ctx) error {
	var req farmtypes.AutomationModifyReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	if err := automationsvc.Automation.Modify(ctx, req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", nil)
}
