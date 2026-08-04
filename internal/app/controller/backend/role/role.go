package role

import (
	"github.com/MQEnergy/go-skeleton/internal/app/controller"
	rolesvc "github.com/MQEnergy/go-skeleton/internal/app/service/backend/role"
	roletypes "github.com/MQEnergy/go-skeleton/internal/types/role"
	"github.com/MQEnergy/go-skeleton/pkg/response"
	"github.com/gofiber/fiber/v3"
)

type Controller struct {
	controller.Controller
}

var Role = &Controller{}

func (c *Controller) Tree(ctx fiber.Ctx) error {
	list, err := rolesvc.Role.Tree(ctx)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", list)
}

func (c *Controller) Create(ctx fiber.Ctx) error {
	var req roletypes.CreateReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := rolesvc.Role.Create(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

func (c *Controller) Update(ctx fiber.Ctx) error {
	var req roletypes.UpdateReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	if err := rolesvc.Role.Update(ctx, req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", nil)
}

func (c *Controller) Delete(ctx fiber.Ctx) error {
	var req roletypes.IDReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	if err := rolesvc.Role.Delete(ctx, req.ID); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", nil)
}

func (c *Controller) SetAuth(ctx fiber.Ctx) error {
	var req roletypes.AuthReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	if err := rolesvc.Role.SetAuth(ctx, req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", nil)
}

func (c *Controller) Assignable(ctx fiber.Ctx) error {
	list, err := rolesvc.Role.Assignable(ctx)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", list)
}
