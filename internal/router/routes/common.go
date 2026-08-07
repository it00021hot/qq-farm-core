package routes

import (
	"github.com/it00021hot/qq-farm-core/internal/app/controller/auth"
	"github.com/it00021hot/qq-farm-core/internal/app/controller/common"
	"github.com/it00021hot/qq-farm-core/internal/middleware"

	"github.com/gofiber/fiber/v3"
)

func InitCommonGroup(r fiber.Router, handles ...any) {
	router := r.Group("/", handles...)
	{
		router.Get("/", common.Common.Index)
		router.Get("/ping", common.Common.Ping)
		router.Post("/token/create", common.Common.TokenCreate)
		router.Post("/token/view", middleware.AuthMiddleware(), common.Common.TokenView)

		router.Post("/auth/login", auth.Auth.Login).Name("登录")
		router.Post("/auth/refresh", auth.Auth.Refresh).Name("刷新令牌")
	}
}
