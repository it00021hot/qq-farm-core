package account

import (
	"github.com/MQEnergy/go-skeleton/internal/app/controller"
	accountsvc "github.com/MQEnergy/go-skeleton/internal/app/service/farm/account"
	farmtypes "github.com/MQEnergy/go-skeleton/internal/types/farm"
	"github.com/MQEnergy/go-skeleton/pkg/response"
	"github.com/gofiber/fiber/v3"
)

type Controller struct {
	controller.Controller
}

var Account = &Controller{}

type (
	AccountCreateReq = farmtypes.AccountCreateReq
	AccountUpdateReq = farmtypes.AccountUpdateReq
	AccountListReq   = farmtypes.AccountListReq
	AccountIDReq     = farmtypes.AccountIDReq
	AccountDeleteReq = farmtypes.AccountDeleteReq
)

// List 农场账号列表
//
//	@Summary		农场账号列表
//	@Tags			农场账号
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			current		query		int						false	"页码"
//	@Param			size		query		int						false	"每页条数"
//	@Param			keyword		query		string					false	"名称/编码/QQ关键词"
//	@Param			status		query		int						false	"状态 1正常 2禁用"
//	@Param			runStatus	query		int						false	"运行状态"
//	@Success		200			{object}	response.JSONResponse	"成功"
//	@Failure		400			{object}	response.JSONResponse	"请求错误"
//	@Router			/farm/account/list [get]
func (c *Controller) List(ctx fiber.Ctx) error {
	var req farmtypes.AccountListReq
	_ = c.Validate(ctx, &req)
	info, err := accountsvc.Account.List(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

// Detail 农场账号详情
//
//	@Summary		农场账号详情
//	@Tags			农场账号
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			id			query		int						true	"账号ID"
//	@Success		200			{object}	response.JSONResponse	"成功"
//	@Failure		400			{object}	response.JSONResponse	"请求错误"
//	@Router			/farm/account/detail [get]
func (c *Controller) Detail(ctx fiber.Ctx) error {
	var req farmtypes.AccountIDReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := accountsvc.Account.Detail(ctx, req.ID)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

// Create 创建农场账号
//
//	@Summary		创建农场账号
//	@Tags			农场账号
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			payload		body		AccountCreateReq		true	"创建参数"
//	@Success		200			{object}	response.JSONResponse	"成功"
//	@Failure		400			{object}	response.JSONResponse	"请求错误"
//	@Router			/farm/account/add [post]
func (c *Controller) Create(ctx fiber.Ctx) error {
	var req farmtypes.AccountCreateReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := accountsvc.Account.Create(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

// Update 更新农场账号
//
//	@Summary		更新农场账号
//	@Tags			农场账号
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			payload		body		AccountUpdateReq		true	"更新参数"
//	@Success		200			{object}	response.JSONResponse	"成功"
//	@Failure		400			{object}	response.JSONResponse	"请求错误"
//	@Router			/farm/account/modify [post]
func (c *Controller) Update(ctx fiber.Ctx) error {
	var req farmtypes.AccountUpdateReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	if err := accountsvc.Account.Update(ctx, req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", nil)
}

// Delete 删除农场账号
//
//	@Summary		删除农场账号
//	@Tags			农场账号
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			payload		body		AccountDeleteReq		true	"删除参数"
//	@Success		200			{object}	response.JSONResponse	"成功"
//	@Failure		400			{object}	response.JSONResponse	"请求错误"
//	@Router			/farm/account/delete [post]
func (c *Controller) Delete(ctx fiber.Ctx) error {
	var req farmtypes.AccountDeleteReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	if err := accountsvc.Account.Delete(ctx, req.ID); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", nil)
}

// Start 启动农场账号
//
//	@Summary		启动农场账号
//	@Tags			农场账号
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			payload		body		AccountIDReq			true	"账号ID"
//	@Success		200			{object}	response.JSONResponse	"成功"
//	@Failure		400			{object}	response.JSONResponse	"请求错误"
//	@Router			/farm/account/start [post]
func (c *Controller) Start(ctx fiber.Ctx) error {
	var req farmtypes.AccountIDReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	if err := accountsvc.Account.Start(ctx, req.ID); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", nil)
}

// Stop 停止农场账号
//
//	@Summary		停止农场账号
//	@Tags			农场账号
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			payload		body		AccountIDReq			true	"账号ID"
//	@Success		200			{object}	response.JSONResponse	"成功"
//	@Failure		400			{object}	response.JSONResponse	"请求错误"
//	@Router			/farm/account/stop [post]
func (c *Controller) Stop(ctx fiber.Ctx) error {
	var req farmtypes.AccountIDReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	if err := accountsvc.Account.Stop(ctx, req.ID); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", nil)
}
