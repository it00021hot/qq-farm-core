package tenant

import (
	"github.com/MQEnergy/go-skeleton/internal/app/controller"
	tenantsvc "github.com/MQEnergy/go-skeleton/internal/app/service/backend/tenant"
	tenanttypes "github.com/MQEnergy/go-skeleton/internal/types/tenant"
	"github.com/MQEnergy/go-skeleton/pkg/response"
	"github.com/MQEnergy/go-skeleton/pkg/tenant"
	"github.com/gofiber/fiber/v3"
)

type Controller struct {
	controller.Controller
}

var Tenant = &Controller{}

func (c *Controller) assertPlatform(ctx fiber.Ctx) error {
	isPlatform, _ := ctx.Locals(tenant.LocalIsPlatform).(bool)
	if !isPlatform {
		return response.ForbiddenException(ctx, "仅平台用户可管理租户")
	}
	return nil
}

func (c *Controller) Create(ctx fiber.Ctx) error {
	if err := c.assertPlatform(ctx); err != nil {
		return err
	}
	var req tenanttypes.CreateReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := tenantsvc.Tenant.Create(req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

func (c *Controller) Update(ctx fiber.Ctx) error {
	if err := c.assertPlatform(ctx); err != nil {
		return err
	}
	var req tenanttypes.UpdateReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	if err := tenantsvc.Tenant.Update(req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", nil)
}

func (c *Controller) Status(ctx fiber.Ctx) error {
	if err := c.assertPlatform(ctx); err != nil {
		return err
	}
	var req tenanttypes.StatusReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	if err := tenantsvc.Tenant.UpdateStatus(req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", nil)
}

func (c *Controller) List(ctx fiber.Ctx) error {
	if err := c.assertPlatform(ctx); err != nil {
		return err
	}
	var req tenanttypes.ListReq
	_ = c.Validate(ctx, &req)
	info, err := tenantsvc.Tenant.List(req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

func (c *Controller) Detail(ctx fiber.Ctx) error {
	if err := c.assertPlatform(ctx); err != nil {
		return err
	}
	var req tenanttypes.IDReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := tenantsvc.Tenant.Detail(req.ID)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

func (c *Controller) Usage(ctx fiber.Ctx) error {
	if err := c.assertPlatform(ctx); err != nil {
		return err
	}
	var req tenanttypes.IDReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := tenantsvc.Tenant.Usage(req.ID)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

func (c *Controller) Bind(ctx fiber.Ctx) error {
	if err := c.assertPlatform(ctx); err != nil {
		return err
	}
	var req tenanttypes.BindTenantReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	if err := tenantsvc.Tenant.BindAdminTenants(req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", nil)
}
