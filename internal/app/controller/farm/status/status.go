package status

import (
	"github.com/gofiber/fiber/v3"
	"github.com/it00021hot/qq-farm-core/internal/app/controller"
	statussvc "github.com/it00021hot/qq-farm-core/internal/app/service/farm/status"
	farmpush "github.com/it00021hot/qq-farm-core/internal/farm/push"
	farmruntime "github.com/it00021hot/qq-farm-core/internal/farm/runtime"
	farmtypes "github.com/it00021hot/qq-farm-core/internal/types/farm"
	"github.com/it00021hot/qq-farm-core/pkg/response"
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

// QqBotBindStatus returns the current binding + pending session state.
func (c *Controller) QqBotBindStatus(ctx fiber.Ctx) error {
	binding := farmpush.CurrentBinding()
	pending := farmpush.PendingBindUser()
	return response.SuccessJSON(ctx, "", map[string]any{
		"binding":       binding,
		"pendingUser":   pending,
		"pendingActive": pending != "",
	})
}

// QqBotBindStart opens a pending bind session (user then DMs the bot any message).
func (c *Controller) QqBotBindStart(ctx fiber.Ctx) error {
	sessionID := farmpush.StartBindSession("admin")
	return response.SuccessJSON(ctx, "", map[string]any{
		"sessionId": sessionID,
		"message":   "已发起绑定，请在 5 分钟内用你的 QQ 私聊机器人发送任意消息完成绑定",
	})
}

// FertilizerCheckBuy runs one event-driven fertilizer threshold check immediately.
func (c *Controller) FertilizerCheckBuy(ctx fiber.Ctx) error {
	var req struct {
		AccountID uint64 `json:"accountId" query:"accountId" validate:"required"`
	}
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	if err := farmruntime.Default.CheckFertilizerBuyNowByAccount(req.AccountID); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", map[string]any{"started": true,
		"message": "已触发化肥阈值检测，结果见看板日志"})
}

// QqBotBindPoll polls one pending bind session (rust poll_qq_bot_bind).
func (c *Controller) QqBotBindPoll(ctx fiber.Ctx) error {
	var req struct {
		SessionID string `json:"sessionId" query:"sessionId" validate:"required"`
	}
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	status, binding := farmpush.PollBindSession(req.SessionID)
	return response.SuccessJSON(ctx, "", map[string]any{
		"status":  status,
		"binding": binding,
	})
}

// QqBotBindUnbind clears the stored QQ bot binding.
func (c *Controller) QqBotBindUnbind(ctx fiber.Ctx) error {
	removed := farmpush.Unbind()
	return response.SuccessJSON(ctx, "", map[string]any{"unbound": removed})
}
