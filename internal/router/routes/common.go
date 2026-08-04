package routes

import (
	"github.com/MQEnergy/go-skeleton/internal/app/controller/backend/user/auth"
	"github.com/MQEnergy/go-skeleton/internal/app/controller/common"
	"github.com/MQEnergy/go-skeleton/internal/middleware"

	"github.com/gofiber/fiber/v3"
)

func InitCommonGroup(r fiber.Router, handles ...any) {
	router := r.Group("/", handles...)
	{
		router.Get("/", common.Common.Index)
		router.Get("/ping", common.Common.Ping)
		router.Post("/token/create", common.Common.TokenCreate)
		router.Post("/token/view", middleware.AuthMiddleware(), common.Common.TokenView)

		router.Post("/backend/auth/login", auth.Auth.Login).Name("登录")
	}
}
