package tenant

import (
	"context"
)

type (
	ctxKey  struct{}
	skipKey struct{}
)

// WithTenantID 将租户 ID 写入 context
func WithTenantID(ctx context.Context, id uint64) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// IDFrom 读取租户 ID
func IDFrom(ctx context.Context) (uint64, bool) {
	if ctx == nil {
		return 0, false
	}
	v, ok := ctx.Value(ctxKey{}).(uint64)
	return v, ok
}

// MustID 读取租户 ID，不存在则返回 0
func MustID(ctx context.Context) uint64 {
	id, _ := IDFrom(ctx)
	return id
}

// WithSkip 跳过租户自动过滤（平台全局查询）
func WithSkip(ctx context.Context) context.Context {
	return context.WithValue(ctx, skipKey{}, true)
}

// IsSkip 是否跳过租户过滤
func IsSkip(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	v, ok := ctx.Value(skipKey{}).(bool)
	return ok && v
}

// Model 需租户隔离的模型
type Model interface {
	GetTenantID() uint64
	SetTenantID(uint64)
}
