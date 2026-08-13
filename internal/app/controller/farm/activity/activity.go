package activity

import (
	"github.com/it00021hot/qq-farm-core/internal/app/controller"
	activitysvc "github.com/it00021hot/qq-farm-core/internal/app/service/farm/activity"
	farmtypes "github.com/it00021hot/qq-farm-core/internal/types/farm"
	"github.com/it00021hot/qq-farm-core/pkg/response"
	"github.com/gofiber/fiber/v3"
)

type Controller struct {
	controller.Controller
}

var Activity = &Controller{}

func (c *Controller) Snapshot(ctx fiber.Ctx) error {
	var req farmtypes.ActivitySnapshotReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := activitysvc.Activity.Snapshot(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

func (c *Controller) ClaimPass(ctx fiber.Ctx) error {
	var req farmtypes.ActivityActionReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := activitysvc.Activity.ClaimPass(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

func (c *Controller) LightConstellation(ctx fiber.Ctx) error {
	var req farmtypes.ActivityActionReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := activitysvc.Activity.LightConstellation(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

func (c *Controller) ShopExchange(ctx fiber.Ctx) error {
	var req farmtypes.ActivityActionReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := activitysvc.Activity.ShopExchange(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

func (c *Controller) ClaimSolarTerm(ctx fiber.Ctx) error {
	var req farmtypes.ActivityActionReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := activitysvc.Activity.ClaimSolarTerm(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

func (c *Controller) ClaimTask(ctx fiber.Ctx) error {
	var req farmtypes.ActivityActionReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := activitysvc.Activity.ClaimTask(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

func (c *Controller) ClaimGift(ctx fiber.Ctx) error {
	var req farmtypes.ActivityActionReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := activitysvc.Activity.ClaimGift(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

func (c *Controller) Registry(ctx fiber.Ctx) error {
	info, err := activitysvc.Activity.Registry(ctx)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

func (c *Controller) ClaimGreenPlum(ctx fiber.Ctx) error {
	var req farmtypes.ActivityActionReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := activitysvc.Activity.ClaimGreenPlum(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

func (c *Controller) StartGreenPlumBrew(ctx fiber.Ctx) error {
	var req farmtypes.ActivityActionReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := activitysvc.Activity.StartGreenPlumBrew(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

func (c *Controller) ContinueGreenPlumBrew(ctx fiber.Ctx) error {
	var req farmtypes.ActivityActionReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := activitysvc.Activity.ContinueGreenPlumBrew(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

func (c *Controller) SettleGreenPlumBrew(ctx fiber.Ctx) error {
	var req farmtypes.ActivityActionReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := activitysvc.Activity.SettleGreenPlumBrew(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}
