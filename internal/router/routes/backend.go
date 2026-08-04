package routes

import (
	backendctl "github.com/MQEnergy/go-skeleton/internal/app/controller/backend"
	"github.com/MQEnergy/go-skeleton/internal/app/controller/backend/admin"
	"github.com/MQEnergy/go-skeleton/internal/app/controller/backend/attachment"
	"github.com/MQEnergy/go-skeleton/internal/app/controller/backend/menu"
	"github.com/MQEnergy/go-skeleton/internal/app/controller/backend/role"
	"github.com/MQEnergy/go-skeleton/internal/app/controller/backend/tenant"
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

		// 租户管理（平台）
		router.Post("/tenant", tenant.Tenant.Create).Name("创建租户")
		router.Get("/tenant", tenant.Tenant.List).Name("租户列表")
		router.Get("/tenant/detail", tenant.Tenant.Detail).Name("租户详情")
		router.Get("/tenant/usage", tenant.Tenant.Usage).Name("租户用量")
		router.Put("/tenant", tenant.Tenant.Update).Name("更新租户")
		router.Put("/tenant/status", tenant.Tenant.Status).Name("启停租户")
		router.Post("/tenant/bind", tenant.Tenant.Bind).Name("平台用户绑定租户")

		// 角色（平台维护；assignable 租户可读）
		router.Get("/role/tree", role.Role.Tree).Name("角色树")
		router.Get("/role/assignable", role.Role.Assignable).Name("可分配角色")
		router.Post("/role", role.Role.Create).Name("创建角色")
		router.Put("/role", role.Role.Update).Name("更新角色")
		router.Post("/role/delete", role.Role.Delete).Name("删除角色")
		router.Post("/role/auth", role.Role.SetAuth).Name("角色菜单授权")

		// 菜单/按钮（平台）
		router.Get("/menu/tree", menu.Menu.Tree).Name("菜单树")
		router.Post("/menu", menu.Menu.Create).Name("创建菜单")
		router.Put("/menu", menu.Menu.Update).Name("更新菜单")
		router.Post("/menu/delete", menu.Menu.Delete).Name("删除菜单")

		// 用户（租户上下文）
		router.Get("/admin/list", admin.Admin.List).Name("用户列表")
		router.Post("/admin", admin.Admin.Create).Name("创建用户")
		router.Put("/admin", admin.Admin.Update).Name("更新用户")
		router.Put("/admin/status", admin.Admin.Status).Name("启停用户")
		router.Post("/platform-user", admin.Admin.CreatePlatform).Name("创建平台用户")
	}
}
