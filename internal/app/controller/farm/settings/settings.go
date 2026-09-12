package settings

// 设置面板扩展端点：系统配置 / 设备预设 / 离线提醒。

import (
	"github.com/gofiber/fiber/v3"

	"github.com/it00021hot/qq-farm-core/internal/app/controller"
	settingsvc "github.com/it00021hot/qq-farm-core/internal/app/service/farm/settings"
	"github.com/it00021hot/qq-farm-core/pkg/response"
)

// Controller holds the settings-panel endpoints.
type Controller struct {
	controller.Controller
}

var Settings = &Controller{}

// SystemConfig returns the stored system config merged over defaults.
func (c *Controller) SystemConfig(ctx fiber.Ctx) error {
	info, err := settingsvc.Service{}.SystemConfig(ctx)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

// SaveSystemConfig persists the system config payload.
func (c *Controller) SaveSystemConfig(ctx fiber.Ctx) error {
	var payload settingsvc.SystemConfigPayload
	if err := c.Validate(ctx, &payload); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := settingsvc.Service{}.SaveSystemConfig(ctx, payload)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

// ResetSystemConfig clears overrides and returns defaults.
func (c *Controller) ResetSystemConfig(ctx fiber.Ctx) error {
	info, err := settingsvc.Service{}.ResetSystemConfig(ctx)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

// DevicePresets lists built-in device fingerprints.
func (c *Controller) DevicePresets(ctx fiber.Ctx) error {
	return response.SuccessJSON(ctx, "", settingsvc.Service{}.DevicePresets(ctx))
}

// OfflineReminder returns the reminder config.
func (c *Controller) OfflineReminder(ctx fiber.Ctx) error {
	info, err := settingsvc.Service{}.OfflineReminder(ctx)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

// SaveOfflineReminder persists the reminder config.
func (c *Controller) SaveOfflineReminder(ctx fiber.Ctx) error {
	var cfg settingsvc.OfflineReminder
	if err := c.Validate(ctx, &cfg); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := settingsvc.Service{}.SaveOfflineReminder(ctx, cfg)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}

// TestOfflineReminder sends a test notification.
func (c *Controller) TestOfflineReminder(ctx fiber.Ctx) error {
	var cfg settingsvc.OfflineReminder
	if err := c.Validate(ctx, &cfg); err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	info, err := settingsvc.Service{}.TestOfflineReminder(ctx, cfg)
	if err != nil {
		return response.BadRequestException(ctx, err.Error())
	}
	return response.SuccessJSON(ctx, "", info)
}
