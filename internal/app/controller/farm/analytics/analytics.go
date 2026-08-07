package analytics

import (
	"github.com/it00021hot/qq-farm-core/internal/app/controller"
	analyticssvc "github.com/it00021hot/qq-farm-core/internal/app/service/farm/analytics"
	farmtypes "github.com/it00021hot/qq-farm-core/internal/types/farm"
	"github.com/it00021hot/qq-farm-core/pkg/response"
	"github.com/gofiber/fiber/v3"
)

type Controller struct {
	controller.Controller
}

var Analytics = &Controller{}

func (c *Controller) Detail(ctx fiber.Ctx) error {
	var req farmtypes.AnalyticsDetailReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := analyticssvc.Analytics.Detail(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}
