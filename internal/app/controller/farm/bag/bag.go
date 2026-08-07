package bag

import (
	"github.com/it00021hot/qq-farm-core/internal/app/controller"
	bagsvc "github.com/it00021hot/qq-farm-core/internal/app/service/farm/bag"
	farmtypes "github.com/it00021hot/qq-farm-core/internal/types/farm"
	"github.com/it00021hot/qq-farm-core/pkg/response"
	"github.com/gofiber/fiber/v3"
)

type Controller struct {
	controller.Controller
}

var Bag = &Controller{}

func (c *Controller) Get(ctx fiber.Ctx) error {
	var req farmtypes.BagReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	result, err := bagsvc.Bag.Get(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", result)
}

func (c *Controller) Seeds(ctx fiber.Ctx) error {
	var req farmtypes.BagReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	result, err := bagsvc.Bag.Seeds(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", result)
}

func (c *Controller) Sell(ctx fiber.Ctx) error {
	var req farmtypes.BagSellReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	result, err := bagsvc.Bag.Sell(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", result)
}

func (c *Controller) Use(ctx fiber.Ctx) error {
	var req farmtypes.BagUseReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	result, err := bagsvc.Bag.Use(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", result)
}
