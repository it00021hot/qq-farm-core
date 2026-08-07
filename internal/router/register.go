package router

import (
	"strings"

	"github.com/MQEnergy/go-skeleton/internal/middleware"
	"github.com/MQEnergy/go-skeleton/internal/router/routes"
	"github.com/MQEnergy/go-skeleton/internal/vars"
	"github.com/MQEnergy/go-skeleton/pkg/helper"
	"github.com/MQEnergy/go-skeleton/pkg/response"
	"github.com/goccy/go-json"
	"github.com/gofiber/contrib/v3/swagger"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/gofiber/fiber/v3/middleware/static"
)

// Register ...
func Register(appName string) *fiber.App {
	publicMiddleware := []any{
		middleware.LoggerMiddleware(),  // 日志
		middleware.WhiteIpMiddleware(), // 白名单
	}

	r := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			return response.NotFoundException(c, err.Error())
		},
		BodyLimit:      100 * 1024 * 1024, // Set the body limit to 100MB
		AppName:        appName,           // This allows to setup app name for the app
		JSONEncoder:    json.Marshal,      // If you're not happy with the performance of encoding/json, we recommend you to use these libraries
		JSONDecoder:    json.Unmarshal,
		ReadBufferSize: vars.Config.GetInt("server.bufferSize"), // fix: Request Header Fields Too Large
	})
	// middleware cors, compress, cache, X-Request-Id
	r.Use(
		recover.New(),
		cors.New(cors.Config{
			AllowOriginsFunc: func(origin string) bool {
				originList := vars.Config.GetString("server.originList")
				if originList == "*" {
					return true
				}
				originSlice := strings.Split(originList, ",")
				if helper.InAnySlice[string](originSlice, origin) {
					return true
				}
				return false
			},
			AllowCredentials: true,
		}),
		compress.New(),
		requestid.New(),
	)

	// swagger
	if vars.Config.GetBool("swagger.enabled") {
		r.Use(swagger.New(swagger.Config{
			BasePath: vars.Config.GetString("swagger.basePath"),
			FilePath: vars.Config.GetString("swagger.filePath"),
			Path:     vars.Config.GetString("swagger.path"),
			Title:    vars.Config.GetString("swagger.title"),
		}))
	}

	// common
	routes.InitCommonGroup(r, publicMiddleware...)

	// game-config static icons (Plant/Item images), public like qq-farm-bot /game-config/*
	gameConfigDir := vars.Config.GetString("farm.gameConfigDir")
	if gameConfigDir == "" {
		gameConfigDir = "resource/farm/gameConfig"
	}
	r.Get("/game-config/*", static.New(gameConfigDir))

	protected := append(publicMiddleware,
		middleware.AuthMiddleware(),
		middleware.CacheMiddleware(),
	)

	routes.InitAuthGroup(r, protected...)
	routes.InitSystemGroup(r, protected...)
	routes.InitFarmGroup(r, protected...)

	// frontend
	routes.InitFrontendGroup(r,
		append(publicMiddleware,
			middleware.AuthMiddleware(),
			middleware.CacheMiddleware(),
		)...,
	)

	vars.Routes = r.GetRoutes()
	return r
}
