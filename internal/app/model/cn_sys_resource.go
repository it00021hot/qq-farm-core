package model

const TableNameSysResource = "cn_sys_resource"

// SysResource 后台配置资源菜单表
type SysResource struct {
	ID           uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID" json:"id"`
	Name         string `gorm:"column:name;size:64;not null;comment:名称" json:"name"`
	Alias        string `gorm:"column:alias;size:64;not null;uniqueIndex:backend_menu_alias_title_unique;comment:别名" json:"alias"`
	Desc         string `gorm:"column:desc;size:64;not null;comment:描述" json:"desc"`
	FURL         string `gorm:"column:f_url;size:64;not null;default:'';comment:前端路由" json:"fUrl"`
	BURL         string `gorm:"column:b_url;size:128;not null;default:'';comment:后端接口路径模式" json:"bUrl"`
	Methods      string `gorm:"column:methods;size:64;not null;default:'';comment:HTTP方法 GET,POST" json:"methods"`
	Icon         string `gorm:"column:icon;size:64;not null;default:'';comment:菜单icon" json:"icon"`
	ParentID     uint64 `gorm:"column:parent_id;not null;default:0;comment:父级ID" json:"parentId"`
	Path         string `gorm:"column:path;size:64;not null;default:'';comment:ID路径 1-2-3..." json:"path"`
	ResourceType uint8  `gorm:"column:resource_type;not null;default:1;comment:类型 1：目录 2：菜单 3：操作/按钮" json:"resourceType"`
	HideInMenu   uint8  `gorm:"column:hide_in_menu;not null;default:1;comment:侧栏显示：1显示 2隐藏" json:"hideInMenu"`
	Status       uint8  `gorm:"column:status;not null;comment:状态：1：正常 2：停用" json:"status"`
	SortOrder    uint16 `gorm:"column:sort_order;not null;default:50;comment:排序" json:"sortOrder"`
	CreatedAt    uint64 `gorm:"column:created_at;not null;comment:创建时间" json:"createdAt"`
	UpdatedAt    uint64 `gorm:"column:updated_at;not null;comment:更新时间" json:"updatedAt"`
}

func (*SysResource) TableName() string { return TableNameSysResource }
