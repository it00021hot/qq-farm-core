package middleware

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/MQEnergy/go-skeleton/internal/app/model"
	"github.com/MQEnergy/go-skeleton/internal/vars"
	"github.com/MQEnergy/go-skeleton/pkg/helper"
	"github.com/MQEnergy/go-skeleton/pkg/response"
	"github.com/MQEnergy/go-skeleton/pkg/tenant"
	"github.com/gofiber/fiber/v3"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

// 不需要选定业务租户上下文的平台配置类路径前缀
var platformConfigPrefixes = []string{
	"/platform",
	"/auth",
	"/system/ping",
	"/system/platform-user",
}

// TenantMiddleware 解析租户上下文：校验过期/禁用，平台多租户切换
func TenantMiddleware(db *gorm.DB) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		uid := cast.ToUint64(ctx.Locals(tenant.LocalUID))
		if uid == 0 {
			uid = cast.ToUint64(ctx.GetRespHeader("uid"))
		}
		roleIDs := cast.ToString(ctx.Locals(tenant.LocalRoleIDs))
		if roleIDs == "" {
			roleIDs = ctx.GetRespHeader("role_ids")
		}
		homeTenantID := cast.ToUint64(ctx.Locals(tenant.LocalTenantID))
		// JWT 写入的 home tenant；若 Locals 未设则从 header 兼容
		if v := ctx.GetRespHeader("tenant_id"); v != "" && homeTenantID == 0 {
			homeTenantID = cast.ToUint64(v)
		}

		isSuper := helper.InAnySlice(strings.Split(roleIDs, ","), vars.Config.GetString("server.superRoleId"))
		isPlatform := homeTenantID == 0
		ctx.Locals(tenant.LocalIsPlatform, isPlatform)
		ctx.Locals(tenant.LocalIsSuper, isSuper)
		ctx.Locals(tenant.LocalUID, uid)
		ctx.Locals(tenant.LocalRoleIDs, roleIDs)

		path := ctx.Path()
		needTenantData := !isPlatformConfigPath(path)

		var activeTenantID uint64
		if isPlatform {
			if needTenantData {
				headerTID := cast.ToUint64(ctx.Get(vars.HeaderTenantID))
				if headerTID == 0 {
					// WebSocket clients pass tenant via query (cannot set custom headers).
					headerTID = cast.ToUint64(ctx.Query("tenantId"))
				}
				if headerTID == 0 {
					return response.BadRequestException(ctx, "请通过 "+vars.HeaderTenantID+" 指定操作租户")
				}
				if !isSuper {
					ok, err := platformCanAccessTenant(db, uid, headerTID)
					if err != nil {
						return response.InternalServerException(ctx, err.Error())
					}
					if !ok {
						return response.ForbiddenException(ctx, "无权操作该租户")
					}
				}
				if err := assertTenantActive(db, headerTID); err != nil {
					return response.ForbiddenException(ctx, err.Error())
				}
				activeTenantID = headerTID
			} else {
				activeTenantID = 0 // 全局配置
			}
		} else {
			// 租户用户：固定归属，忽略客户端切换
			if err := assertTenantActive(db, homeTenantID); err != nil {
				return response.ForbiddenException(ctx, err.Error())
			}
			activeTenantID = homeTenantID
		}

		ctx.Locals(tenant.LocalTenantID, activeTenantID)
		ctx.Set("tenant_id", cast.ToString(activeTenantID))
		return ctx.Next()
	}
}

func isPlatformConfigPath(path string) bool {
	for _, p := range platformConfigPrefixes {
		if path == p || strings.HasPrefix(path, p+"/") {
			return true
		}
	}
	// assignable 属于角色只读，平台配置前缀已覆盖 /backend/role
	return false
}

func platformCanAccessTenant(db *gorm.DB, adminID, tenantID uint64) (bool, error) {
	var count int64
	err := tenant.Global(db, context.Background()).
		Model(&model.SysAdminTenant{}).
		Where("admin_id = ? AND tenant_id = ?", adminID, tenantID).
		Count(&count).Error
	return count > 0, err
}

func assertTenantActive(db *gorm.DB, tenantID uint64) error {
	if tenantID == 0 {
		return errors.New("无效租户")
	}
	var t model.SysTenant
	err := tenant.Global(db, context.Background()).
		Where("id = ?", tenantID).
		First(&t).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("租户不存在")
		}
		return err
	}
	if t.Status != vars.StatusNormal {
		return errors.New("租户已禁用")
	}
	if t.ExpireAt > 0 && uint64(t.ExpireAt) < uint64(time.Now().Unix()) {
		return errors.New("租户已过期")
	}
	return nil
}
