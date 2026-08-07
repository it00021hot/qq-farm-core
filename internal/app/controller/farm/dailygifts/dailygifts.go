package dailygifts

import (
	"github.com/it00021hot/qq-farm-core/internal/app/controller"
	dailygiftssvc "github.com/it00021hot/qq-farm-core/internal/app/service/farm/dailygifts"
	farmtypes "github.com/it00021hot/qq-farm-core/internal/types/farm"
	"github.com/it00021hot/qq-farm-core/pkg/response"
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
