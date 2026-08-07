package routes

import (
	"github.com/MQEnergy/go-skeleton/internal/app/controller/auth"
	"github.com/gofiber/fiber/v3"
)

// InitAuthGroup 鉴权相关（需登录）
func InitAuthGroup(r fiber.Router, handles ...any) {
	router := r.Group("auth", handles...)
	{
		router.Get("/routes", auth.Auth.Routes).Name("获取所有后端路由")
		router.Get("/info", auth.Auth.Info).Name("当前用户权限信息")
		router.Post("/logout", auth.Auth.Logout).Name("退出登录")
		router.Post("/password", auth.Auth.ChangePassword).Name("修改当前用户密码")
	}
}
