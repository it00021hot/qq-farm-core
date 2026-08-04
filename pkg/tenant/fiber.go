package tenant

import (
	"context"

	"github.com/gofiber/fiber/v3"
)

// Fiber Locals keys
const (
	LocalTenantID   = "tenant_id"
	LocalUID        = "uid"
	LocalUUID       = "uuid"
	LocalRoleIDs    = "role_ids"
	LocalIsPlatform = "is_platform"
	LocalIsSuper    = "is_super"
)

// FromFiber 从 Fiber Locals 构建带租户的 context
func FromFiber(ctx fiber.Ctx) context.Context {
	base := context.Background()
	tid, _ := ctx.Locals(LocalTenantID).(uint64)
	base = WithTenantID(base, tid)
	if isPlatform, _ := ctx.Locals(LocalIsPlatform).(bool); isPlatform && tid == 0 {
		// 平台未切租户时默认 skip，由业务显式 Scope
		base = WithSkip(base)
	}
	return base
}

// TenantCtx 当前数据隔离用的 context（平台已切租户时带 tenant_id）
func TenantCtx(ctx fiber.Ctx) context.Context {
	base := context.Background()
	tid, _ := ctx.Locals(LocalTenantID).(uint64)
	isPlatform, _ := ctx.Locals(LocalIsPlatform).(bool)
	if isPlatform && tid == 0 {
		return WithSkip(base)
	}
	return WithTenantID(base, tid)
}
