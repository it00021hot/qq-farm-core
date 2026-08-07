package card

import (
	"strings"

	"github.com/MQEnergy/go-skeleton/internal/app/controller"
	cardsvc "github.com/MQEnergy/go-skeleton/internal/app/service/farm/card"
	farmtypes "github.com/MQEnergy/go-skeleton/internal/types/farm"
	"github.com/MQEnergy/go-skeleton/pkg/response"
	"github.com/gofiber/fiber/v3"
)

func platformOnlyErr(ctx fiber.Ctx, err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "仅平台用户") {
		return response.ForbiddenException(ctx, err.Error())
	}
	return response.BadRequestException(ctx, err.Error())
}

type Controller struct {
	controller.Controller
}

var Card = &Controller{}

type (
	CardGenerateReq = farmtypes.CardGenerateReq
	CardRedeemReq   = farmtypes.CardRedeemReq
	CardListReq     = farmtypes.CardListReq
	CardStatusReq   = farmtypes.CardStatusReq
)

// List 卡密列表
//
//	@Summary		卡密列表
//	@Description	仅平台用户
//	@Tags			农场卡密
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			current		query		int						false	"页码"
//	@Param			size		query		int						false	"每页条数"
//	@Param			keyword		query		string					false	"卡密/描述关键词"
//	@Param			cardType	query		int						false	"类型 1时长 2额度"
//	@Param			status		query		int						false	"状态"
//	@Success		200			{object}	response.JSONResponse	"成功"
//	@Failure		400			{object}	response.JSONResponse	"请求错误"
//	@Router			/farm/card/list [get]
func (c *Controller) List(ctx fiber.Ctx) error {
	var req farmtypes.CardListReq
	_ = c.Validate(ctx, &req)
	info, err := cardsvc.Card.List(ctx, req)
	if err != nil {
		return platformOnlyErr(ctx, err)
	}
	return response.SuccessJSON(ctx, "", info)
}

// Generate 批量生成卡密
//
//	@Summary		批量生成卡密
//	@Description	仅平台用户
//	@Tags			农场卡密
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			payload	body		CardGenerateReq			true	"生成参数"
//	@Success		200		{object}	response.JSONResponse	"成功"
//	@Failure		400		{object}	response.JSONResponse	"请求错误"
//	@Router			/farm/card/add [post]
func (c *Controller) Generate(ctx fiber.Ctx) error {
	var req farmtypes.CardGenerateReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := cardsvc.Card.Generate(ctx, req)
	if err != nil {
		return platformOnlyErr(ctx, err)
	}
	return response.SuccessJSON(ctx, "", info)
}

// Redeem 兑换卡密
//
//	@Summary		兑换卡密
//	@Description	更新当前租户 expire_at / max_accounts
//	@Tags			农场卡密
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			X-Tenant-ID	header		string					false	"平台用户操作目标租户ID"
//	@Param			payload		body		CardRedeemReq			true	"兑换参数"
//	@Success		200			{object}	response.JSONResponse	"成功"
//	@Failure		400			{object}	response.JSONResponse	"请求错误"
//	@Router			/farm/card/redeem [post]
func (c *Controller) Redeem(ctx fiber.Ctx) error {
	var req farmtypes.CardRedeemReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := cardsvc.Card.Redeem(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

// Status 作废卡密
//
//	@Summary		作废/恢复卡密
//	@Description	仅平台用户；已使用卡密不可改
//	@Tags			农场卡密
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			payload	body		CardStatusReq			true	"状态参数"
//	@Success		200		{object}	response.JSONResponse	"成功"
//	@Failure		400		{object}	response.JSONResponse	"请求错误"
//	@Router			/farm/card/status [post]
func (c *Controller) Status(ctx fiber.Ctx) error {
	var req farmtypes.CardStatusReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	if err := cardsvc.Card.UpdateStatus(ctx, req); err != nil {
		return platformOnlyErr(ctx, err)
	}
	return response.SuccessJSON(ctx, "", nil)
}
