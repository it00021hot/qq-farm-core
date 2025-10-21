package routes

import (
	"github.com/MQEnergy/go-skeleton/internal/app/controller/backend/attachment"
	"github.com/MQEnergy/go-skeleton/internal/app/controller/backend/user/auth"
	"github.com/MQEnergy/go-skeleton/pkg/response"
	"github.com/gofiber/fiber/v2"
)

// InitBackendGroup 初始化后台接口路由
func InitBackendGroup(r fiber.Router, handles ...fiber.Handler) {
	router := r.Group("backend", handles...)
	{
		router.Get("/ping", func(ctx *fiber.Ctx) error {
			return response.SuccessJSON(ctx, "", "backend pong")
		})

		router.Get("/auth/routes", auth.Auth.Routes).Name("获取所有后端路由")
		router.Post("/auth/logout", auth.Auth.Logout).Name("退出登录")
		router.Post("/attachment/upload", attachment.Attachment.Upload).Name("上传资源")
	}

	// casbin中间件可根据不同的数据库进行单独配置 示例如下：
	// demo数据库中存在casbin_rule
	//prefix := vars.Config.Get("database.mysql.sources.demo.prefix")
	//demoHandles := append(handles, middleware.CasbinMiddleware(vars.MDB["demo"], prefix.(string), ""))
	//router1 := r.Group("demo", demoHandles...)
	//{
	//	router1.Get("/", func(ctx *fiber.Ctx) error { return nil })
	//}

	// demo1数据库中存在casbin_rule
	//prefix := vars.Config.Get("database.mysql.sources.demo1.prefix")
	//demo1Handles := append(handles, middleware.CasbinMiddleware(vars.MDB["demo1"], prefix.(string), ""))
	//router2 := r.Group("demo1", demo1Handles...)
	//{
	//	router2.Get("/", func(ctx *fiber.Ctx) error { return nil })
	//}
}
