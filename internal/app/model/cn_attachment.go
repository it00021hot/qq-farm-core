package model

const TableNameAttachment = "cn_attachment"

// Attachment 附件表
type Attachment struct {
	ID               uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID" json:"id"`
	TenantID         uint64 `gorm:"column:tenant_id;not null;default:0;index;comment:租户ID 0：平台" json:"tenantId"`
	UserID           uint64 `gorm:"column:user_id;not null;default:0;comment:附件上传的用户id" json:"userId"`
	AttachName       string `gorm:"column:attach_name;size:64;not null;default:'';comment:附件新名称" json:"attachName"`
	AttachOriginName string `gorm:"column:attach_origin_name;size:255;not null;default:'';comment:附件原名称" json:"attachOriginName"`
	AttachURL        string `gorm:"column:attach_url;size:255;not null;comment:附件地址" json:"attachUrl"`
	AttachType       uint8  `gorm:"column:attach_type;not null;default:1;comment:附件类型 1：图片 2：视频 3：文件" json:"attachType"`
	AttachMimetype   string `gorm:"column:attach_mimetype;size:128;not null;default:'';comment:附件mime类型" json:"attachMimetype"`
	AttachExtension  string `gorm:"column:attach_extension;size:16;not null;default:'';comment:附件后缀名" json:"attachExtension"`
	AttachSize       string `gorm:"column:attach_size;size:32;not null;default:'';comment:附件大小" json:"attachSize"`
	Status           uint8  `gorm:"column:status;not null;comment:状态 1：正常 0：删除" json:"status"`
	CreatedAt        uint   `gorm:"column:created_at;not null;default:0;comment:创建时间" json:"createdAt"`
	UpdatedAt        uint   `gorm:"column:updated_at;not null;default:0;comment:更新时间" json:"updatedAt"`
}

func (*Attachment) TableName() string { return TableNameAttachment }

func (m *Attachment) GetTenantID() uint64   { return m.TenantID }
func (m *Attachment) SetTenantID(id uint64) { m.TenantID = id }
