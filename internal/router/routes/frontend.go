package routes

import (
	"github.com/MQEnergy/go-skeleton/internal/app/controller/frontend"

	"github.com/gofiber/fiber/v3"
)

func InitFrontendGroup(r fiber.Router, handles ...any) {
	router := r.Group("api", handles...)
	{
		router.Get("/ping", frontend.Frontend.Ping)
	}
}
