package friend

import (
	"github.com/it00021hot/qq-farm-core/internal/app/controller"
	friendsvc "github.com/it00021hot/qq-farm-core/internal/app/service/farm/friend"
	farmtypes "github.com/it00021hot/qq-farm-core/internal/types/farm"
	"github.com/it00021hot/qq-farm-core/pkg/response"
	"github.com/gofiber/fiber/v3"
)

type Controller struct {
	controller.Controller
}

var Friend = &Controller{}

func (c *Controller) List(ctx fiber.Ctx) error {
	var req farmtypes.FriendListReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := friendsvc.Friend.List(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

func (c *Controller) Sync(ctx fiber.Ctx) error {
	var req farmtypes.FriendSyncReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := friendsvc.Friend.Sync(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

func (c *Controller) Lands(ctx fiber.Ctx) error {
	var req farmtypes.FriendLandsReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := friendsvc.Friend.Lands(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

func (c *Controller) Op(ctx fiber.Ctx) error {
	var req farmtypes.FriendOpReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := friendsvc.Friend.Op(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

func (c *Controller) InteractLogs(ctx fiber.Ctx) error {
	var req farmtypes.FriendListReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := friendsvc.Friend.InteractLogs(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

func (c *Controller) InteractRecords(ctx fiber.Ctx) error {
	var req farmtypes.FriendInteractRecordsReq
	if err := c.Validate(ctx, &req); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := friendsvc.Friend.InteractRecords(ctx, req)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}
