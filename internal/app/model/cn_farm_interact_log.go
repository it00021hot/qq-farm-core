package model

const TableNameFarmInteractLog = "cn_farm_interact_log"

// FarmInteractLog 好友互动记录
type FarmInteractLog struct {
	ID         uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID" json:"id"`
	AccountID  uint64 `gorm:"column:account_id;not null;index;comment:账号ID" json:"accountId"`
	TargetGid  int64  `gorm:"column:target_gid;not null;default:0;index;comment:目标GID" json:"targetGid"`
	Action     string `gorm:"column:action;size:32;not null;default:'';index;comment:动作 steal/help/..." json:"action"`
	Result     string `gorm:"column:result;size:64;not null;default:'';comment:结果" json:"result"`
	DetailJSON string `gorm:"column:detail_json;type:text;not null;default:'{}';comment:详情JSON" json:"detailJson"`
	CreatedAt  uint   `gorm:"column:created_at;not null;index;comment:创建时间" json:"createdAt"`
}

func (*FarmInteractLog) TableName() string { return TableNameFarmInteractLog }
