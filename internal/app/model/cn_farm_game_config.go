package model

const TableNameFarmGameConfig = "cn_farm_game_config"

// FarmGameConfig 游戏元数据（种子/物品等，可热更）
type FarmGameConfig struct {
	ID         uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID" json:"id"`
	TenantID   uint64 `gorm:"column:tenant_id;not null;default:0;index;uniqueIndex:uk_tenant_cat_key;comment:租户ID 0：全局" json:"tenantId"`
	Category   string `gorm:"column:category;size:64;not null;default:'';uniqueIndex:uk_tenant_cat_key;comment:分类 seed/item/..." json:"category"`
	ConfigKey  string `gorm:"column:config_key;size:64;not null;uniqueIndex:uk_tenant_cat_key;comment:配置键" json:"configKey"`
	ConfigJSON string `gorm:"column:config_json;type:text;not null;default:'{}';comment:配置JSON" json:"configJson"`
	Version    uint   `gorm:"column:version;not null;default:1;comment:版本号" json:"version"`
	Status     uint8  `gorm:"column:status;not null;default:1;comment:状态 1：正常 2：禁用" json:"status"`
	CreatedAt  uint   `gorm:"column:created_at;not null;comment:创建时间" json:"createdAt"`
	UpdatedAt  uint   `gorm:"column:updated_at;not null;comment:更新时间" json:"updatedAt"`
}

func (*FarmGameConfig) TableName() string { return TableNameFarmGameConfig }

func (m *FarmGameConfig) GetTenantID() uint64   { return m.TenantID }
func (m *FarmGameConfig) SetTenantID(id uint64) { m.TenantID = id }
