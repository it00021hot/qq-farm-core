package middleware

import (
	"strings"
	"time"

	"github.com/MQEnergy/go-skeleton/pkg/helper"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cache"
	"github.com/gofiber/utils/v2"
)

// CacheMiddleware http cache middleware
func CacheMiddleware() fiber.Handler {
	return cache.New(cache.Config{
		Next: func(c fiber.Ctx) bool {
			if c.Query("noCache") == "true" {
				return true
			}
			// Never cache WebSocket upgrade / realtime channel.
			if c.Path() == "/farm/ws" || strings.EqualFold(c.Get("Upgrade"), "websocket") {
				return true
			}
			return false
		},
		KeyGenerator: func(ctx fiber.Ctx) string {
			keyG := ctx.IP() + ":" + ctx.Path()
			switch ctx.Method() {
			case fiber.MethodGet:
				params := make([]string, 0)
				for k, v := range ctx.Queries() {
					params = append(params, k+"="+v)
				}
				keyG += ":" + strings.Join(params, "&")
			case fiber.MethodPost:
				keyG += ":" + string(ctx.BodyRaw())
			}
			return utils.CopyString(helper.GenerateHash(keyG))
		},
		Expiration:          60 * time.Second,
		DisableCacheControl: false, // client side caching enabled when false
		Methods:             []string{fiber.MethodGet, fiber.MethodHead, fiber.MethodPost},
	})
}
