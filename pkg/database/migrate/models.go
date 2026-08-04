package migrate

// PostgreSQL 兼容的迁移结构（避免 gen model 中 MySQL 专有的 unsigned 类型标签）

type sysAdmin struct {
	ID        uint64 `gorm:"column:id;primaryKey;autoIncrement"`
	UUID      string `gorm:"column:uuid;size:32;not null"`
	DeptID    uint64 `gorm:"column:dept_id;not null;index"`
	NickName  string `gorm:"column:nick_name;size:64;not null"`
	RealName  string `gorm:"column:real_name;size:64;not null;default:''"`
	Desc      string `gorm:"column:desc;size:64;not null;default:''"`
	Gender    uint8  `gorm:"column:gender;not null;default:0"`
	Account   string `gorm:"column:account;size:64;not null;uniqueIndex:account_index"`
	Password  string `gorm:"column:password;size:64;not null;default:''"`
	Phone     string `gorm:"column:phone;size:16;not null;default:''"`
	Email     string `gorm:"column:email;size:128;not null;default:''"`
	Avatar    string `gorm:"column:avatar;size:128;not null;default:''"`
	Salt      string `gorm:"column:salt;size:32;not null"`
	RoleIds   string `gorm:"column:role_ids;size:32;not null;default:''"`
	Type      uint8  `gorm:"column:type;not null;default:1"`
	IsMain    uint8  `gorm:"column:is_main;not null;default:2"`
	IsAuth    uint8  `gorm:"column:is_auth;not null;default:1"`
	MfaSecret string `gorm:"column:mfa_secret;size:64;not null;default:''"`
	Status    uint8  `gorm:"column:status;not null;default:0"`
	CreatedBy string `gorm:"column:created_by;size:64;not null;default:''"`
	CreatedAt uint   `gorm:"column:created_at;not null"`
	UpdatedBy string `gorm:"column:updated_by;size:64;not null;default:''"`
	UpdatedAt uint   `gorm:"column:updated_at;not null"`
}

func (*sysAdmin) TableName() string { return "cn_sys_admin" }

type sysRole struct {
	ID        uint64 `gorm:"column:id;primaryKey;autoIncrement"`
	MchID     uint64 `gorm:"column:mch_id;not null"`
	Name      string `gorm:"column:name;size:64;not null;uniqueIndex"`
	Code      string `gorm:"column:code;size:16;not null"`
	Desc      string `gorm:"column:desc;size:64;not null;default:''"`
	IsSys     uint8  `gorm:"column:is_sys;not null;default:0"`
	RoleType  uint8  `gorm:"column:role_type;not null"`
	Status    uint8  `gorm:"column:status;not null"`
	CreatedAt uint   `gorm:"column:created_at;not null"`
	UpdatedAt uint   `gorm:"column:updated_at;not null"`
}

func (*sysRole) TableName() string { return "cn_sys_role" }

type sysResource struct {
	ID             uint64 `gorm:"column:id;primaryKey;autoIncrement"`
	Name           string `gorm:"column:name;size:64;not null"`
	Alias_         string `gorm:"column:alias;size:64;not null;uniqueIndex:backend_menu_alias_title_unique"`
	Desc           string `gorm:"column:desc;size:64;not null"`
	FURL           string `gorm:"column:f_url;size:64;not null;default:''"`
	BURL           string `gorm:"column:b_url;size:64;not null"`
	Redirect       string `gorm:"column:redirect;size:128;not null;default:''"`
	CompPath       string `gorm:"column:comp_path;size:128;not null;default:''"`
	Icon           string `gorm:"column:icon;size:32;not null;default:''"`
	CIcon          string `gorm:"column:c_icon;size:32;not null;default:''"`
	ParentID       uint64 `gorm:"column:parent_id;not null;default:0"`
	Path           string `gorm:"column:path;size:64;not null;default:''"`
	ResourceType   uint8  `gorm:"column:resource_type;not null;default:1"`
	IsHidden       uint8  `gorm:"column:is_hidden;not null;default:0"`
	IsCache        uint8  `gorm:"column:is_cache;not null;default:0"`
	IsExternal     uint8  `gorm:"column:is_external;not null;default:0"`
	AlwaysShow     uint8  `gorm:"column:always_show;not null;default:0"`
	BreadcrumbShow uint8  `gorm:"column:breadcrumb_show;not null;default:0"`
	IsAffix        uint8  `gorm:"column:is_affix;not null;default:0"`
	ResType        uint8  `gorm:"column:res_type;not null;default:1"`
	Status         uint8  `gorm:"column:status;not null"`
	SortOrder      uint16 `gorm:"column:sort_order;not null;default:50"`
	CreatedAt      uint64 `gorm:"column:created_at;not null"`
	UpdatedAt      uint64 `gorm:"column:updated_at;not null"`
}

func (*sysResource) TableName() string { return "cn_sys_resource" }

type sysRoleAuth struct {
	ID          uint64 `gorm:"column:id;primaryKey;autoIncrement"`
	RoleID      uint64 `gorm:"column:role_id;not null"`
	ResourceIds string `gorm:"column:resource_ids;type:text;not null"`
}

func (*sysRoleAuth) TableName() string { return "cn_sys_role_auth" }

type sysCasbinRule struct {
	ID    uint64  `gorm:"column:id;primaryKey;autoIncrement"`
	Ptype *string `gorm:"column:ptype;size:100"`
	V0    *string `gorm:"column:v0;size:100"`
	V1    *string `gorm:"column:v1;size:100"`
	V2    *string `gorm:"column:v2;size:100"`
	V3    *string `gorm:"column:v3;size:100"`
	V4    *string `gorm:"column:v4;size:100"`
	V5    *string `gorm:"column:v5;size:100"`
	V6    string  `gorm:"column:v6;size:25;not null;default:''"`
	V7    string  `gorm:"column:v7;size:25;not null;default:''"`
}

func (*sysCasbinRule) TableName() string { return "cn_sys_casbin_rule" }

type attachment struct {
	ID               uint64 `gorm:"column:id;primaryKey;autoIncrement"`
	UserID           uint64 `gorm:"column:user_id;not null;default:0"`
	AttachName       string `gorm:"column:attach_name;size:64;not null;default:''"`
	AttachOriginName string `gorm:"column:attach_origin_name;size:255;not null;default:''"`
	AttachURL        string `gorm:"column:attach_url;size:255;not null"`
	AttachType       uint8  `gorm:"column:attach_type;not null;default:1"`
	AttachMimetype   string `gorm:"column:attach_mimetype;size:128;not null;default:''"`
	AttachExtension  string `gorm:"column:attach_extension;size:16;not null;default:''"`
	AttachSize       string `gorm:"column:attach_size;size:32;not null;default:''"`
	Status           uint8  `gorm:"column:status;not null"`
	CreatedAt        uint   `gorm:"column:created_at;not null;default:0"`
	UpdatedAt        uint   `gorm:"column:updated_at;not null;default:0"`
}

func (*attachment) TableName() string { return "cn_attachment" }
