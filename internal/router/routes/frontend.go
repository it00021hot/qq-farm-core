package routes

import (
	"github.com/it00021hot/qq-farm-core/internal/app/controller/frontend"

	"github.com/gofiber/fiber/v3"
)

func InitFrontendGroup(r fiber.Router, handles ...any) {
	router := r.Group("api", handles...)
	{
		router.Get("/ping", frontend.Frontend.Ping)
	}
}
