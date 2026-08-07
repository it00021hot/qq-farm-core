package gameconfig

import (
	"github.com/it00021hot/qq-farm-core/internal/app/controller"
	gameconfigsvc "github.com/it00021hot/qq-farm-core/internal/app/service/farm/gameconfig"
	farmtypes "github.com/it00021hot/qq-farm-core/internal/types/farm"
	"github.com/it00021hot/qq-farm-core/pkg/response"
	"github.com/gofiber/fiber/v3"
)

type Controller struct {
	controller.Controller
}

var GameConfig = &Controller{}

func (c *Controller) List(ctx fiber.Ctx) error {
	var req farmtypes.GameConfigListReq
	_ = c.Validate(ctx, &req)
	info, err := gameconfigsvc.GameConfig.List(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

func (c *Controller) Modify(ctx fiber.Ctx) error {
	var req farmtypes.GameConfigModifyReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	if err := gameconfigsvc.GameConfig.Modify(ctx, req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", nil)
}

func (c *Controller) Seeds(ctx fiber.Ctx) error {
	list, err := gameconfigsvc.GameConfig.Seeds(ctx)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", list)
}

func (c *Controller) Fruits(ctx fiber.Ctx) error {
	list, err := gameconfigsvc.GameConfig.Fruits(ctx)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", list)
}

func (c *Controller) Items(ctx fiber.Ctx) error {
	var req farmtypes.GameConfigItemsReq
	_ = c.Validate(ctx, &req)
	list, err := gameconfigsvc.GameConfig.Items(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", list)
}

func (c *Controller) Plants(ctx fiber.Ctx) error {
	list, err := gameconfigsvc.GameConfig.Plants(ctx)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", list)
}

func (c *Controller) ItemTypes(ctx fiber.Ctx) error {
	list, err := gameconfigsvc.GameConfig.ItemTypes(ctx)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", list)
}

func (c *Controller) SeedAdd(ctx fiber.Ctx) error {
	var req farmtypes.GameConfigSeedWriteReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := gameconfigsvc.GameConfig.SeedAdd(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

func (c *Controller) SeedModify(ctx fiber.Ctx) error {
	var req farmtypes.GameConfigSeedWriteReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := gameconfigsvc.GameConfig.SeedModify(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

func (c *Controller) SeedDelete(ctx fiber.Ctx) error {
	var req farmtypes.GameConfigSeedDeleteReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := gameconfigsvc.GameConfig.SeedDelete(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

func (c *Controller) FruitAdd(ctx fiber.Ctx) error {
	var req farmtypes.GameConfigFruitWriteReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := gameconfigsvc.GameConfig.FruitAdd(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

func (c *Controller) FruitModify(ctx fiber.Ctx) error {
	var req farmtypes.GameConfigFruitWriteReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := gameconfigsvc.GameConfig.FruitModify(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

func (c *Controller) FruitDelete(ctx fiber.Ctx) error {
	var req farmtypes.GameConfigFruitDeleteReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := gameconfigsvc.GameConfig.FruitDelete(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

func (c *Controller) ItemAdd(ctx fiber.Ctx) error {
	var req farmtypes.GameConfigItemWriteReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := gameconfigsvc.GameConfig.ItemAdd(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

func (c *Controller) ItemModify(ctx fiber.Ctx) error {
	var req farmtypes.GameConfigItemWriteReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := gameconfigsvc.GameConfig.ItemModify(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

func (c *Controller) ItemDelete(ctx fiber.Ctx) error {
	var req farmtypes.GameConfigItemDeleteReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := gameconfigsvc.GameConfig.ItemDelete(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}
