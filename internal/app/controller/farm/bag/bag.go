package bag

import (
	"github.com/MQEnergy/go-skeleton/internal/app/controller"
	bagsvc "github.com/MQEnergy/go-skeleton/internal/app/service/farm/bag"
	farmtypes "github.com/MQEnergy/go-skeleton/internal/types/farm"
	"github.com/MQEnergy/go-skeleton/pkg/response"
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
