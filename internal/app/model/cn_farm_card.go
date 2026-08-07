package model

const TableNameFarmCard = "cn_farm_card"

// 卡密类型
const (
	FarmCardTypeTime  uint8 = 1 // 时长（天）
	FarmCardTypeQuota uint8 = 2 // 账号额度
)

// 卡密状态
const (
	FarmCardStatusUnused   uint8 = 1 // 未使用
	FarmCardStatusUsed     uint8 = 2 // 已使用
	FarmCardStatusDisabled uint8 = 3 // 已禁用
)

// FarmCard 平台卡密（全局，非租户隔离）
type FarmCard struct {
	ID            uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID" json:"id"`
	Code          string `gorm:"column:code;size:32;not null;uniqueIndex;comment:卡密编码" json:"code"`
	CardType      uint8  `gorm:"column:card_type;not null;default:1;index;comment:类型 1：时长 2：账号额度" json:"cardType"`
	Value         int    `gorm:"column:value;not null;default:0;comment:面值 时长天数或账号数 -1：永久" json:"value"`
	Description   string `gorm:"column:description;size:255;not null;default:'';comment:描述" json:"description"`
	Status        uint8  `gorm:"column:status;not null;default:1;index;comment:状态 1：未用 2：已用 3：禁用" json:"status"`
	UsedByTenant  uint64 `gorm:"column:used_by_tenant;not null;default:0;index;comment:使用租户ID" json:"usedByTenant"`
	UsedAt        uint   `gorm:"column:used_at;not null;default:0;comment:使用时间Unix秒" json:"usedAt"`
	CreatedAt     uint   `gorm:"column:created_at;not null;comment:创建时间" json:"createdAt"`
	UpdatedAt     uint   `gorm:"column:updated_at;not null;comment:更新时间" json:"updatedAt"`
}

func (*FarmCard) TableName() string { return TableNameFarmCard }
