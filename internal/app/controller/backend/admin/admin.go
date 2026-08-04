package admin

import (
	"github.com/MQEnergy/go-skeleton/internal/app/controller"
	adminsvc "github.com/MQEnergy/go-skeleton/internal/app/service/backend/admin"
	admintypes "github.com/MQEnergy/go-skeleton/internal/types/admin"
	"github.com/MQEnergy/go-skeleton/pkg/response"
	"github.com/gofiber/fiber/v3"
)

type Controller struct {
	controller.Controller
}

var Admin = &Controller{}

func (c *Controller) List(ctx fiber.Ctx) error {
	var req admintypes.ListReq
	_ = c.Validate(ctx, &req)
	info, err := adminsvc.Admin.List(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

func (c *Controller) Create(ctx fiber.Ctx) error {
	var req admintypes.CreateReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := adminsvc.Admin.Create(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

func (c *Controller) Update(ctx fiber.Ctx) error {
	var req admintypes.UpdateReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	if err := adminsvc.Admin.Update(ctx, req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", nil)
}

func (c *Controller) Status(ctx fiber.Ctx) error {
	var req admintypes.StatusReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	if err := adminsvc.Admin.UpdateStatus(ctx, req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", nil)
}

func (c *Controller) CreatePlatform(ctx fiber.Ctx) error {
	var req admintypes.PlatformCreateReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := adminsvc.Admin.CreatePlatform(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}
