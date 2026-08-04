package boots

import (
	"github.com/MQEnergy/go-skeleton/internal/vars"
	"github.com/MQEnergy/go-skeleton/pkg/tenant"
)

// InitTenantPlugin 注册 GORM 租户行级隔离插件
func InitTenantPlugin() error {
	return tenant.Register(vars.DB)
}
