package routes

import (
	backendctl "github.com/MQEnergy/go-skeleton/internal/app/controller/backend"
	"github.com/MQEnergy/go-skeleton/internal/app/controller/system/admin"
	"github.com/gofiber/fiber/v3"
)

// InitSystemGroup 系统业务（用户管理）
// 约定：仅 GET/POST；动作 list|add|modify|delete（及 detail/status 等）
func InitSystemGroup(r fiber.Router, handles ...any) {
	router := r.Group("system", handles...)
	{
		router.Get("/ping", backendctl.Backend.Ping)

		router.Get("/admin/list", admin.Admin.List).Name("用户列表")
		router.Post("/admin/add", admin.Admin.Create).Name("创建用户")
		router.Post("/admin/modify", admin.Admin.Update).Name("更新用户")
		router.Post("/admin/status", admin.Admin.Status).Name("启停用户")
	}
}
