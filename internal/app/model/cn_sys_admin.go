package model

const TableNameSysAdmin = "cn_sys_admin"

// SysAdmin 后台管理员表
type SysAdmin struct {
	ID        uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID" json:"id"`
	UUID      string `gorm:"column:uuid;size:32;not null;comment:唯一id号" json:"uuid"`
	TenantID  uint64 `gorm:"column:tenant_id;not null;default:0;index;uniqueIndex:uk_tenant_account;comment:租户ID 0：平台账号" json:"tenant_id"`
	NickName  string `gorm:"column:nick_name;size:64;not null;comment:昵称" json:"nick_name"`
	RealName  string `gorm:"column:real_name;size:64;not null;default:'';comment:真实姓名" json:"real_name"`
	Account   string `gorm:"column:account;size:64;not null;uniqueIndex:uk_tenant_account;comment:账号" json:"account"`
	Password  string `gorm:"column:password;size:64;not null;default:'';comment:密码" json:"password"`
	Phone     string `gorm:"column:phone;size:16;not null;default:'';comment:手机号" json:"phone"`
	Email     string `gorm:"column:email;size:128;not null;default:'';comment:邮箱" json:"email"`
	Salt      string `gorm:"column:salt;size:32;not null;comment:密码盐" json:"salt"`
	RoleIds   string `gorm:"column:role_ids;size:64;not null;default:'';comment:角色IDs" json:"role_ids"`
	Status    uint8  `gorm:"column:status;not null;default:0;comment:状态 1：正常 2：禁用" json:"status"`
	CreatedAt uint   `gorm:"column:created_at;not null;comment:创建时间" json:"created_at"`
	UpdatedAt uint   `gorm:"column:updated_at;not null;comment:更新时间" json:"updated_at"`
}

func (*SysAdmin) TableName() string { return TableNameSysAdmin }

func (m *SysAdmin) GetTenantID() uint64   { return m.TenantID }
func (m *SysAdmin) SetTenantID(id uint64) { m.TenantID = id }
