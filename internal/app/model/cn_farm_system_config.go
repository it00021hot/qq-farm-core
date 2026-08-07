package model

const TableNameFarmSystemConfig = "cn_farm_system_config"

// FarmSystemConfig 全局系统配置（平台级 tenant_id=0）
type FarmSystemConfig struct {
	ID         uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID" json:"id"`
	TenantID   uint64 `gorm:"column:tenant_id;not null;default:0;index;uniqueIndex:uk_tenant_key;comment:租户ID 0：平台" json:"tenantId"`
	ConfigKey  string `gorm:"column:config_key;size:64;not null;uniqueIndex:uk_tenant_key;comment:配置键" json:"configKey"`
	ConfigJSON string `gorm:"column:config_json;type:text;not null;default:'{}';comment:配置JSON" json:"configJson"`
	Remark     string `gorm:"column:remark;size:255;not null;default:'';comment:备注" json:"remark"`
	CreatedAt  uint   `gorm:"column:created_at;not null;comment:创建时间" json:"createdAt"`
	UpdatedAt  uint   `gorm:"column:updated_at;not null;comment:更新时间" json:"updatedAt"`
}

func (*FarmSystemConfig) TableName() string { return TableNameFarmSystemConfig }

func (m *FarmSystemConfig) GetTenantID() uint64   { return m.TenantID }
func (m *FarmSystemConfig) SetTenantID(id uint64) { m.TenantID = id }
