package routes

import (
	backendctl "github.com/MQEnergy/go-skeleton/internal/app/controller/backend"
	"github.com/MQEnergy/go-skeleton/internal/app/controller/backend/attachment"
	"github.com/MQEnergy/go-skeleton/internal/app/controller/backend/user/auth"
	"github.com/gofiber/fiber/v3"
)

// InitBackendGroup 初始化后台接口路由
func InitBackendGroup(r fiber.Router, handles ...any) {
	router := r.Group("backend", handles...)
	{
		router.Get("/ping", backendctl.Backend.Ping)

		router.Get("/auth/routes", auth.Auth.Routes).Name("获取所有后端路由")
		router.Post("/auth/logout", auth.Auth.Logout).Name("退出登录")
		router.Post("/attachment/upload", attachment.Attachment.Upload).Name("上传资源")
		router.Post("/attachment/access-url", attachment.Attachment.AccessURL).Name("置换临时访问地址")
		router.Get("/attachment/file/:file_path", attachment.Attachment.ReadFile).Name("读取文件数据内容")
	}

	// casbin中间件可根据不同的数据库进行单独配置 示例如下：
	// demo数据库中存在casbin_rule
	//prefix := vars.Config.Get("database.pgsql.sources.demo.prefix")
	//demoHandles := append(handles, middleware.CasbinMiddleware(vars.MDB["demo"], prefix.(string), ""))
	//router1 := r.Group("demo", demoHandles...)
	//{
	//	router1.Get("/", func(ctx fiber.Ctx) error { return nil })
	//}

	// demo1数据库中存在casbin_rule
	//prefix := vars.Config.Get("database.pgsql.sources.demo1.prefix")
	//demo1Handles := append(handles, middleware.CasbinMiddleware(vars.MDB["demo1"], prefix.(string), ""))
	//router2 := r.Group("demo1", demo1Handles...)
	//{
	//	router2.Get("/", func(ctx fiber.Ctx) error { return nil })
	//}
}
