package routes

import (
	"github.com/MQEnergy/go-skeleton/internal/app/controller/platform/menu"
	"github.com/MQEnergy/go-skeleton/internal/app/controller/platform/permission"
	"github.com/MQEnergy/go-skeleton/internal/app/controller/platform/role"
	"github.com/MQEnergy/go-skeleton/internal/app/controller/platform/tenant"
	"github.com/gofiber/fiber/v3"
)

// InitPlatformGroup 平台配置（租户/角色/菜单/权限）
// 约定：仅 GET/POST；动作 list|add|modify|delete（及 detail/tree/status 等）
func InitPlatformGroup(r fiber.Router, handles ...any) {
	router := r.Group("platform", handles...)
	{
		router.Get("/tenant/list", tenant.Tenant.List).Name("租户列表")
		router.Get("/tenant/detail", tenant.Tenant.Detail).Name("租户详情")
		router.Post("/tenant/add", tenant.Tenant.Create).Name("创建租户")
		router.Post("/tenant/modify", tenant.Tenant.Update).Name("更新租户")
		router.Post("/tenant/status", tenant.Tenant.Status).Name("启停租户")
		router.Post("/tenant/bind", tenant.Tenant.Bind).Name("平台用户绑定租户")

		router.Get("/role/tree", role.Role.Tree).Name("角色树")
		router.Get("/role/assignable", role.Role.Assignable).Name("可分配角色")
		router.Post("/role/add", role.Role.Create).Name("创建角色")
		router.Post("/role/modify", role.Role.Update).Name("更新角色")
		router.Post("/role/delete", role.Role.Delete).Name("删除角色")
		router.Get("/role/auth", role.Role.GetAuth).Name("查询角色菜单授权")
		router.Post("/role/auth", role.Role.SetAuth).Name("角色菜单授权")

		router.Get("/menu/tree", menu.Menu.Tree).Name("菜单树")
		router.Post("/menu/add", menu.Menu.Create).Name("创建菜单")
		router.Post("/menu/modify", menu.Menu.Update).Name("更新菜单")
		router.Post("/menu/delete", menu.Menu.Delete).Name("删除菜单")

		router.Get("/permission/apis", permission.Permission.ListAPIs).Name("可绑定API列表")
		router.Get("/permission/role-policies", permission.Permission.RolePolicies).Name("角色Casbin策略")
		router.Post("/permission/reload", permission.Permission.Reload).Name("重载Casbin")
	}
}
