package middleware

import (
	"strings"

	"github.com/MQEnergy/go-skeleton/internal/vars"
	"github.com/MQEnergy/go-skeleton/pkg/helper"
	"github.com/MQEnergy/go-skeleton/pkg/rbac"
	"github.com/MQEnergy/go-skeleton/pkg/response"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// 定义不需要权限校验的路径
var excludePaths = []string{
	"/auth/login",
	"/auth/refresh",
	"/auth/loginForTest",
	"/auth/shop-login",
	"/auth/wx-login",
	"/auth/wx-qr-register",
	"/auth/forget-pass",
}

// CasbinMiddleware casbin middleware（使用进程内单例 Enforcer）
func CasbinMiddleware(db *gorm.DB, prefix, tableName string) fiber.Handler {
	_ = db
	_ = prefix
	_ = tableName
	return func(ctx fiber.Ctx) error {
		if helper.InAnySlice[string](excludePaths, ctx.Path()) {
			return ctx.Next()
		}

		roleIds := ctx.GetRespHeader("role_ids")
		if roleIds == "" {
			return response.UnauthorizedException(ctx, "该用户还未分配角色权限")
		}
		roleList := strings.Split(roleIds, ",")
		if helper.InAnySlice[string](roleList, vars.Config.GetString("server.superRoleId")) {
			return ctx.Next()
		}
		if rbac.GetEnforcer() == nil {
			return response.InternalServerException(ctx, "casbin enforcer not ready")
		}

		obj := ctx.Path()
		act := ctx.Method()
		flag := false
		for _, sub := range roleList {
			sub = strings.TrimSpace(sub)
			if sub == "" {
				continue
			}
			if ok, _ := rbac.Enforce(sub, obj, act); ok {
				flag = true
				break
			}
		}
		if !flag {
			return response.ForbiddenException(ctx, "该用户未授权访问权限")
		}
		return ctx.Next()
	}
}
