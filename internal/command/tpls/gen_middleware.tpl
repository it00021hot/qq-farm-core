package middleware

import (
	"github.com/gofiber/fiber/v3"
)

// {{.CmdName}}Middleware ...
func {{.CmdName}}Middleware() fiber.Handler {
	return func(ctx fiber.Ctx) error {
		// Todo implement ...
		return ctx.Next()
	}
}
