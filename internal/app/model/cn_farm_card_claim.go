package model

const TableNameFarmCardClaim = "cn_farm_card_claim"

// FarmCardClaim 卡密兑换记录
type FarmCardClaim struct {
	ID        uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID" json:"id"`
	TenantID  uint64 `gorm:"column:tenant_id;not null;default:0;index;comment:租户ID" json:"tenantId"`
	CardID    uint64 `gorm:"column:card_id;not null;index;comment:卡密ID" json:"cardId"`
	CardCode  string `gorm:"column:card_code;size:32;not null;default:'';comment:卡密编码快照" json:"cardCode"`
	CardType  uint8  `gorm:"column:card_type;not null;default:1;comment:类型快照" json:"cardType"`
	Value     int    `gorm:"column:value;not null;default:0;comment:面值快照" json:"value"`
	ClaimedBy uint64 `gorm:"column:claimed_by;not null;default:0;comment:兑换人adminID" json:"claimedBy"`
	CreatedAt uint   `gorm:"column:created_at;not null;comment:兑换时间" json:"createdAt"`
}

func (*FarmCardClaim) TableName() string { return TableNameFarmCardClaim }

func (m *FarmCardClaim) GetTenantID() uint64   { return m.TenantID }
func (m *FarmCardClaim) SetTenantID(id uint64) { m.TenantID = id }
