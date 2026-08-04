package model

const TableNameSysRole = "cn_sys_role"

// SysRole 角色表（全局，仅平台维护）
type SysRole struct {
	ID        uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID" json:"id"`
	ParentID  uint64 `gorm:"column:parent_id;not null;default:0;index;comment:上级角色ID 0为根" json:"parent_id"`
	Level     uint16 `gorm:"column:level;not null;default:0;comment:角色层级深度 根为0" json:"level"`
	Name      string `gorm:"column:name;size:64;not null;uniqueIndex;comment:角色名称" json:"name"`
	Code      string `gorm:"column:code;size:32;not null;uniqueIndex;comment:角色唯一code" json:"code"`
	Desc      string `gorm:"column:desc;size:64;not null;default:'';comment:角色描述" json:"desc"`
	IsSys     uint8  `gorm:"column:is_sys;not null;default:0;comment:是否系统角色 1：是 0：否" json:"is_sys"`
	RoleType  uint8  `gorm:"column:role_type;not null;default:1;comment:角色类型 1：平台专用 2：可赋给租户用户" json:"role_type"`
	Status    uint8  `gorm:"column:status;not null;comment:状态：1正常(默认) 2停用" json:"status"`
	CreatedAt uint   `gorm:"column:created_at;not null;comment:创建时间" json:"created_at"`
	UpdatedAt uint   `gorm:"column:updated_at;not null;comment:更新时间" json:"updated_at"`
}

func (*SysRole) TableName() string { return TableNameSysRole }
