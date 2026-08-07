package model

const TableNameFarmAccountConfig = "cn_farm_account_config"

// FarmAccountConfig 账号自动化配置（JSON TEXT）
type FarmAccountConfig struct {
	ID         uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID" json:"id"`
	TenantID   uint64 `gorm:"column:tenant_id;not null;default:0;index;uniqueIndex:uk_farm_cfg_account;comment:租户ID" json:"tenantId"`
	AccountID  uint64 `gorm:"column:account_id;not null;uniqueIndex:uk_farm_cfg_account;index;comment:账号ID" json:"accountId"`
	ConfigJSON string `gorm:"column:config_json;type:text;not null;default:'{}';comment:配置JSON" json:"configJson"`
	CreatedAt  uint   `gorm:"column:created_at;not null;comment:创建时间" json:"createdAt"`
	UpdatedAt  uint   `gorm:"column:updated_at;not null;comment:更新时间" json:"updatedAt"`
}

func (*FarmAccountConfig) TableName() string { return TableNameFarmAccountConfig }

func (m *FarmAccountConfig) GetTenantID() uint64   { return m.TenantID }
func (m *FarmAccountConfig) SetTenantID(id uint64) { m.TenantID = id }
