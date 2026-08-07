package commerce

import (
	"github.com/it00021hot/qq-farm-core/internal/app/controller"
	commercesvc "github.com/it00021hot/qq-farm-core/internal/app/service/farm/commerce"
	farmtypes "github.com/it00021hot/qq-farm-core/internal/types/farm"
	"github.com/it00021hot/qq-farm-core/pkg/response"
	"github.com/gofiber/fiber/v3"
)

type Controller struct {
	controller.Controller
}

var Commerce = &Controller{}

func (c *Controller) Mall(ctx fiber.Ctx) error {
	var req farmtypes.MallListReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	result, err := commercesvc.Commerce.Mall(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", result)
}

func (c *Controller) Purchase(ctx fiber.Ctx) error {
	var req farmtypes.MallPurchaseReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	result, err := commercesvc.Commerce.Purchase(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", result)
}

func (c *Controller) MysteryShop(ctx fiber.Ctx) error {
	var req farmtypes.CommerceAccountReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	result, err := commercesvc.Commerce.MysteryShop(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", result)
}

func (c *Controller) Diamond(ctx fiber.Ctx) error {
	var req farmtypes.CommerceAccountReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	result, err := commercesvc.Commerce.Diamond(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", result)
}
