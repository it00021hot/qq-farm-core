package model

const TableNameSysCasbinRule = "cn_sys_casbin_rule"

// SysCasbinRule Casbin策略表
type SysCasbinRule struct {
	ID    uint64  `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID" json:"id"`
	Ptype *string `gorm:"column:ptype;size:100;uniqueIndex:idx_casbin_rule;comment:策略类型 p/g" json:"ptype"`
	V0    *string `gorm:"column:v0;size:100;uniqueIndex:idx_casbin_rule;comment:主体(角色ID)" json:"v0"`
	V1    *string `gorm:"column:v1;size:100;uniqueIndex:idx_casbin_rule;comment:对象(路径)" json:"v1"`
	V2    *string `gorm:"column:v2;size:100;uniqueIndex:idx_casbin_rule;comment:动作(方法)" json:"v2"`
	V3    *string `gorm:"column:v3;size:100;uniqueIndex:idx_casbin_rule;comment:扩展字段v3" json:"v3"`
	V4    *string `gorm:"column:v4;size:100;uniqueIndex:idx_casbin_rule;comment:扩展字段v4" json:"v4"`
	V5    *string `gorm:"column:v5;size:100;uniqueIndex:idx_casbin_rule;comment:扩展字段v5" json:"v5"`
	V6    string  `gorm:"column:v6;size:25;not null;default:'';comment:扩展字段v6" json:"v6"`
	V7    string  `gorm:"column:v7;size:25;not null;default:'';comment:扩展字段v7" json:"v7"`
}

func (*SysCasbinRule) TableName() string { return TableNameSysCasbinRule }
