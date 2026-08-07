package dailygifts

import (
	"github.com/MQEnergy/go-skeleton/internal/app/controller"
	dailygiftssvc "github.com/MQEnergy/go-skeleton/internal/app/service/farm/dailygifts"
	farmtypes "github.com/MQEnergy/go-skeleton/internal/types/farm"
	"github.com/MQEnergy/go-skeleton/pkg/response"
	"github.com/gofiber/fiber/v3"
)

type Controller struct {
	controller.Controller
}

var DailyGifts = &Controller{}

func (c *Controller) Get(ctx fiber.Ctx) error {
	var req farmtypes.DailyGiftsReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	result, err := dailygiftssvc.DailyGifts.Get(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", result)
}
