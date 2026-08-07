package lands

import (
	"github.com/MQEnergy/go-skeleton/internal/app/controller"
	landssvc "github.com/MQEnergy/go-skeleton/internal/app/service/farm/lands"
	farmtypes "github.com/MQEnergy/go-skeleton/internal/types/farm"
	"github.com/MQEnergy/go-skeleton/pkg/response"
	"github.com/gofiber/fiber/v3"
)

type Controller struct {
	controller.Controller
}

var Lands = &Controller{}

// Get returns the current lands from a running account session.
func (c *Controller) Get(ctx fiber.Ctx) error {
	var req farmtypes.LandsReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	result, err := landssvc.Lands.Get(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", result)
}

// Operate runs a manual farm operation. Quiet hours do not apply here.
func (c *Controller) Operate(ctx fiber.Ctx) error {
	var req farmtypes.OperateReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	result, err := landssvc.Lands.Operate(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", result)
}
