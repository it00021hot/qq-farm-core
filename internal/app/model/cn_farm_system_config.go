package model

const TableNameFarmSystemConfig = "cn_farm_system_config"

// FarmSystemConfig 全局系统配置
type FarmSystemConfig struct {
	ID         uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID" json:"id"`
	ConfigKey  string `gorm:"column:config_key;size:64;not null;uniqueIndex:uk_sys_config_key;comment:配置键" json:"configKey"`
	ConfigJSON string `gorm:"column:config_json;type:text;not null;default:'{}';comment:配置JSON" json:"configJson"`
	Remark     string `gorm:"column:remark;size:255;not null;default:'';comment:备注" json:"remark"`
	CreatedAt  uint   `gorm:"column:created_at;not null;comment:创建时间" json:"createdAt"`
	UpdatedAt  uint   `gorm:"column:updated_at;not null;comment:更新时间" json:"updatedAt"`
}

func (*FarmSystemConfig) TableName() string { return TableNameFarmSystemConfig }
