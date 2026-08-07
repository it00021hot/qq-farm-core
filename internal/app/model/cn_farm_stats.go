package model

const TableNameFarmStats = "cn_farm_stats"

// FarmStats 按日统计
type FarmStats struct {
	ID           uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID" json:"id"`
	TenantID     uint64 `gorm:"column:tenant_id;not null;default:0;index;uniqueIndex:uk_tenant_account_day;comment:租户ID" json:"tenantId"`
	AccountID    uint64 `gorm:"column:account_id;not null;uniqueIndex:uk_tenant_account_day;index;comment:账号ID" json:"accountId"`
	StatDate     string `gorm:"column:stat_date;size:10;not null;uniqueIndex:uk_tenant_account_day;comment:统计日YYYY-MM-DD" json:"statDate"`
	Gold         int64  `gorm:"column:gold;not null;default:0;comment:金币增量" json:"gold"`
	Exp          int64  `gorm:"column:exp;not null;default:0;comment:经验增量" json:"exp"`
	HarvestCount int64  `gorm:"column:harvest_count;not null;default:0;comment:收获次数" json:"harvestCount"`
	StealCount   int64  `gorm:"column:steal_count;not null;default:0;comment:偷取次数" json:"stealCount"`
	HelpCount    int64  `gorm:"column:help_count;not null;default:0;comment:帮忙次数" json:"helpCount"`
	PlantCount   int64  `gorm:"column:plant_count;not null;default:0;comment:种植次数" json:"plantCount"`
	ExtraJSON    string `gorm:"column:extra_json;type:text;not null;default:'{}';comment:扩展JSON" json:"extraJson"`
	CreatedAt    uint   `gorm:"column:created_at;not null;comment:创建时间" json:"createdAt"`
	UpdatedAt    uint   `gorm:"column:updated_at;not null;comment:更新时间" json:"updatedAt"`
}

func (*FarmStats) TableName() string { return TableNameFarmStats }

func (m *FarmStats) GetTenantID() uint64   { return m.TenantID }
func (m *FarmStats) SetTenantID(id uint64) { m.TenantID = id }
