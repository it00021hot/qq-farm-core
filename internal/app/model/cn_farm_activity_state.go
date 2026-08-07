package model

const TableNameFarmActivityState = "cn_farm_activity_state"

// FarmActivityState 活动中心本地合并状态
type FarmActivityState struct {
	ID         uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID" json:"id"`
	AccountID  uint64 `gorm:"column:account_id;not null;uniqueIndex:uk_account_act;index;comment:账号ID" json:"accountId"`
	ActivityID string `gorm:"column:activity_id;size:64;not null;uniqueIndex:uk_account_act;comment:活动ID" json:"activityId"`
	StateJSON  string `gorm:"column:state_json;type:text;not null;default:'{}';comment:状态JSON" json:"stateJson"`
	SyncedAt   uint   `gorm:"column:synced_at;not null;default:0;comment:同步时间Unix秒" json:"syncedAt"`
	CreatedAt  uint   `gorm:"column:created_at;not null;comment:创建时间" json:"createdAt"`
	UpdatedAt  uint   `gorm:"column:updated_at;not null;comment:更新时间" json:"updatedAt"`
}

func (*FarmActivityState) TableName() string { return TableNameFarmActivityState }
