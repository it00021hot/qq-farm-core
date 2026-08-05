package vars

// Role type constants (cn_sys_role.role_type)
const (
	RoleTypePlatform uint8 = 1 // 仅平台账号可分配
	RoleTypeTenant   uint8 = 2 // 可赋给租户用户
)

// Resource type constants (cn_sys_resource.resource_type)
const (
	ResourceTypeDir    uint8 = 1 // 目录
	ResourceTypeMenu   uint8 = 2 // 菜单
	ResourceTypeButton uint8 = 3 // 操作/按钮
)

// CasbinSyncMarker 写入 cn_sys_casbin_rule.v3，标识由资源授权同步产生的策略
const CasbinSyncMarker = "resource_sync"

// Tenant / admin status
const (
	StatusNormal   uint8 = 1
	StatusDisabled uint8 = 2
)

// DefaultTenantAdminRoleID 开户默认绑定的租户管理员角色（seed id=3）
const DefaultTenantAdminRoleID uint64 = 3

// HeaderTenantID 平台用户切换租户的请求头
const HeaderTenantID = "X-Tenant-ID"
