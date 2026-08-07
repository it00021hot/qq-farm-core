package model

const TableNameFarmFriendGid = "cn_farm_friend_gid"

// FarmFriendGid 已知好友 GID 缓存
type FarmFriendGid struct {
	ID        uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID" json:"id"`
	TenantID  uint64 `gorm:"column:tenant_id;not null;default:0;index;uniqueIndex:uk_tenant_account_gid;comment:租户ID" json:"tenantId"`
	AccountID uint64 `gorm:"column:account_id;not null;uniqueIndex:uk_tenant_account_gid;index;comment:账号ID" json:"accountId"`
	Gid       int64  `gorm:"column:gid;not null;uniqueIndex:uk_tenant_account_gid;comment:好友GID" json:"gid"`
	Nickname  string `gorm:"column:nickname;size:128;not null;default:'';comment:昵称" json:"nickname"`
	SyncedAt  uint   `gorm:"column:synced_at;not null;default:0;comment:同步时间Unix秒" json:"syncedAt"`
	CreatedAt uint   `gorm:"column:created_at;not null;comment:创建时间" json:"createdAt"`
	UpdatedAt uint   `gorm:"column:updated_at;not null;comment:更新时间" json:"updatedAt"`
}

func (*FarmFriendGid) TableName() string { return TableNameFarmFriendGid }

func (m *FarmFriendGid) GetTenantID() uint64   { return m.TenantID }
func (m *FarmFriendGid) SetTenantID(id uint64) { m.TenantID = id }
