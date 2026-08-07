package model

const TableNameFarmAccount = "cn_farm_account"

// FarmAccount 农场挂机账号
type FarmAccount struct {
	ID           uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID" json:"id"`
	TenantID     uint64 `gorm:"column:tenant_id;not null;default:0;index;uniqueIndex:uk_farm_account_code;comment:租户ID" json:"tenantId"`
	Name         string `gorm:"column:name;size:64;not null;default:'';comment:显示名称" json:"name"`
	Code         string `gorm:"column:code;size:512;not null;default:'';uniqueIndex:uk_farm_account_code;comment:网关登录Code(一次性)" json:"code"`
	Platform     string `gorm:"column:platform;size:32;not null;default:'qq';comment:平台 qq/wx" json:"platform"`
	LoginOS      string `gorm:"column:login_os;size:64;not null;default:'';comment:登录URL中的os" json:"loginOs"`
	ClientVer    string `gorm:"column:client_ver;size:64;not null;default:'';comment:登录URL中的ver" json:"clientVer"`
	Uin          string `gorm:"column:uin;size:32;not null;default:'';index;comment:UIN" json:"uin"`
	QQ           string `gorm:"column:qq;size:32;not null;default:'';comment:QQ号" json:"qq"`
	Avatar       string `gorm:"column:avatar;size:512;not null;default:'';comment:头像URL" json:"avatar"`
	Username     string `gorm:"column:username;size:64;not null;default:'';comment:归属用户名(兼容)" json:"username"`
	Remark       string `gorm:"column:remark;size:255;not null;default:'';comment:备注" json:"remark"`
	RunStatus    uint8  `gorm:"column:run_status;not null;default:0;index;comment:运行状态 0：停止 1：运行中 2：异常" json:"runStatus"`
	LastOnlineAt uint   `gorm:"column:last_online_at;not null;default:0;comment:最近在线Unix秒" json:"lastOnlineAt"`
	Status       uint8  `gorm:"column:status;not null;default:1;comment:状态 1：正常 2：禁用" json:"status"`
	CreatedAt    uint   `gorm:"column:created_at;not null;comment:创建时间" json:"createdAt"`
	UpdatedAt    uint   `gorm:"column:updated_at;not null;comment:更新时间" json:"updatedAt"`
}

func (*FarmAccount) TableName() string { return TableNameFarmAccount }

func (m *FarmAccount) GetTenantID() uint64   { return m.TenantID }
func (m *FarmAccount) SetTenantID(id uint64) { m.TenantID = id }
