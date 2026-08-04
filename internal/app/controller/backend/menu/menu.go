package menu

import (
	"github.com/MQEnergy/go-skeleton/internal/app/controller"
	menusvc "github.com/MQEnergy/go-skeleton/internal/app/service/backend/menu"
	menutypes "github.com/MQEnergy/go-skeleton/internal/types/menu"
	"github.com/MQEnergy/go-skeleton/pkg/response"
	"github.com/gofiber/fiber/v3"
)

type Controller struct {
	controller.Controller
}

var Menu = &Controller{}

func (c *Controller) Tree(ctx fiber.Ctx) error {
	list, err := menusvc.Menu.Tree(ctx)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", list)
}

func (c *Controller) Create(ctx fiber.Ctx) error {
	var req menutypes.CreateReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := menusvc.Menu.Create(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

func (c *Controller) Update(ctx fiber.Ctx) error {
	var req menutypes.UpdateReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	if err := menusvc.Menu.Update(ctx, req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", nil)
}

func (c *Controller) Delete(ctx fiber.Ctx) error {
	var req menutypes.IDReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	if err := menusvc.Menu.Delete(ctx, req.ID); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", nil)
}
