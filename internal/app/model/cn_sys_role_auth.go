package model

const TableNameSysRoleAuth = "cn_sys_role_auth"

// SysRoleAuth 角色权限表
type SysRoleAuth struct {
	ID          uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID" json:"id"`
	RoleID      uint64 `gorm:"column:role_id;not null;comment:角色ID" json:"role_id"`
	ResourceIds string `gorm:"column:resource_ids;type:text;not null;comment:菜单id列表 1,2,3..." json:"resource_ids"`
}

func (*SysRoleAuth) TableName() string { return TableNameSysRoleAuth }
