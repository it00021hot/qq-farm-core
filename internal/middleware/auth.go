package middleware

import (
	"errors"
	"strings"

	"github.com/MQEnergy/go-skeleton/internal/vars"
	"github.com/MQEnergy/go-skeleton/pkg/helper"
	"github.com/MQEnergy/go-skeleton/pkg/jwtauth"
	"github.com/MQEnergy/go-skeleton/pkg/response"
	"github.com/MQEnergy/go-skeleton/pkg/tenant"
	"github.com/spf13/cast"

	jwtware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
)

var NoAuthPaths = []string{
	"/backend/auth/login",
	"/backend/auth/refresh",
}

// AuthMiddleware jwt authentication middleware
func AuthMiddleware() fiber.Handler {
	return jwtware.New(jwtware.Config{
		SigningKey: jwtware.SigningKey{Key: []byte(vars.Config.GetString("jwt.secret"))},
		ErrorHandler: func(c fiber.Ctx, err error) error {
			msg := err.Error()
			if errors.Is(err, jwt.ErrTokenExpired) || strings.Contains(msg, "token is expired") || strings.Contains(msg, "exp") {
				return response.AuthExpiredException(c, "会话已过期，请刷新令牌")
			}
			return response.UnauthorizedException(c, "未认证: "+msg)
		},
		SuccessHandler: func(ctx fiber.Ctx) error {
			if user := jwtware.FromContext(ctx); user != nil {
				if claims, ok := user.Claims.(jwt.MapClaims); ok && user.Valid {
					if typ := cast.ToString(claims["typ"]); typ != "" && typ != jwtauth.TokenTypeAccess {
						return response.UnauthorizedException(ctx, "token 类型错误")
					}
					if sub, ok := claims["sub"].(map[string]interface{}); ok {
						uid := cast.ToString(sub["id"])
						uuid := cast.ToString(sub["uuid"])
						roleIDs := cast.ToString(sub["role_ids"])
						tenantID := cast.ToUint64(sub["tenant_id"])

						ctx.Locals(tenant.LocalUID, cast.ToUint64(uid))
						ctx.Locals(tenant.LocalUUID, uuid)
						ctx.Locals(tenant.LocalRoleIDs, roleIDs)
						ctx.Locals(tenant.LocalTenantID, tenantID)
						ctx.Locals(tenant.LocalIsPlatform, tenantID == 0)

						ctx.Set("uuid", uuid)
						ctx.Set("uid", uid)
						ctx.Set("role_ids", roleIDs)
						ctx.Set("tenant_id", cast.ToString(tenantID))
						return ctx.Next()
					}
				}
			}
			return response.UnauthorizedException(ctx, "token is invalid")
		},
		Next: func(ctx fiber.Ctx) bool {
			return helper.InAnySlice(NoAuthPaths, ctx.Path())
		},
	})
}
