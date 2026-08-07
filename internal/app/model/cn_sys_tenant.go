package model

const TableNameSysTenant = "cn_sys_tenant"

// SysTenant 租户表
type SysTenant struct {
	ID           uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID" json:"id"`
	Code         string `gorm:"column:code;size:64;not null;uniqueIndex;comment:租户编码" json:"code"`
	Name         string `gorm:"column:name;size:128;not null;comment:租户名称" json:"name"`
	Status       uint8  `gorm:"column:status;not null;default:1;index;comment:状态 1：正常 2：禁用" json:"status"`
	MaxUsers     uint   `gorm:"column:max_users;not null;default:0;comment:用户上限 0：不限制" json:"maxUsers"`
	MaxAccounts  uint   `gorm:"column:max_accounts;not null;default:0;comment:农场账号上限 0：不限制" json:"maxAccounts"`
	ExpireAt     uint   `gorm:"column:expire_at;not null;default:0;comment:过期时间Unix秒 0：永不过期" json:"expireAt"`
	ContactName  string `gorm:"column:contact_name;size:64;not null;default:'';comment:联系人" json:"contactName"`
	ContactPhone string `gorm:"column:contact_phone;size:32;not null;default:'';comment:联系电话" json:"contactPhone"`
	Remark       string `gorm:"column:remark;size:255;not null;default:'';comment:备注" json:"remark"`
	CreatedAt    uint   `gorm:"column:created_at;not null;comment:创建时间" json:"createdAt"`
	UpdatedAt    uint   `gorm:"column:updated_at;not null;comment:更新时间" json:"updatedAt"`
}

func (*SysTenant) TableName() string { return TableNameSysTenant }

const TableNameSysAdminTenant = "cn_sys_admin_tenant"

// SysAdminTenant 平台用户可管理的租户
type SysAdminTenant struct {
	ID        uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID" json:"id"`
	AdminID   uint64 `gorm:"column:admin_id;not null;uniqueIndex:uk_admin_tenant;comment:平台用户ID" json:"adminId"`
	TenantID  uint64 `gorm:"column:tenant_id;not null;uniqueIndex:uk_admin_tenant;index;comment:可管理的租户ID" json:"tenantId"`
	CreatedAt uint   `gorm:"column:created_at;not null;comment:创建时间" json:"createdAt"`
}

func (*SysAdminTenant) TableName() string { return TableNameSysAdminTenant }
