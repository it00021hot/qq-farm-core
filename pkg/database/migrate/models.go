package migrate

// PostgreSQL 兼容的迁移结构（含字段注释；GORM 会生成 COMMENT ON COLUMN）

type sysTenant struct {
	ID           uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID"`
	Code         string `gorm:"column:code;size:64;not null;uniqueIndex;comment:租户编码"`
	Name         string `gorm:"column:name;size:128;not null;comment:租户名称"`
	Status       uint8  `gorm:"column:status;not null;default:1;index;comment:状态 1：正常 2：禁用"`
	MaxUsers     uint   `gorm:"column:max_users;not null;default:0;comment:用户上限 0：不限制"`
	MaxAccounts  uint   `gorm:"column:max_accounts;not null;default:0;comment:农场账号上限 0：不限制"`
	ExpireAt     uint   `gorm:"column:expire_at;not null;default:0;comment:过期时间Unix秒 0：永不过期"`
	ContactName  string `gorm:"column:contact_name;size:64;not null;default:'';comment:联系人"`
	ContactPhone string `gorm:"column:contact_phone;size:32;not null;default:'';comment:联系电话"`
	Remark       string `gorm:"column:remark;size:255;not null;default:'';comment:备注"`
	CreatedAt    uint   `gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt    uint   `gorm:"column:updated_at;not null;comment:更新时间"`
}

func (*sysTenant) TableName() string { return "cn_sys_tenant" }

type sysAdminTenant struct {
	ID        uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID"`
	AdminID   uint64 `gorm:"column:admin_id;not null;uniqueIndex:uk_admin_tenant;comment:平台用户ID"`
	TenantID  uint64 `gorm:"column:tenant_id;not null;uniqueIndex:uk_admin_tenant;index;comment:可管理的租户ID"`
	CreatedAt uint   `gorm:"column:created_at;not null;comment:创建时间"`
}

func (*sysAdminTenant) TableName() string { return "cn_sys_admin_tenant" }

type sysAdmin struct {
	ID        uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID"`
	UUID      string `gorm:"column:uuid;size:32;not null;comment:唯一id号"`
	TenantID  uint64 `gorm:"column:tenant_id;not null;default:0;index;uniqueIndex:uk_tenant_account;comment:租户ID 0：平台账号"`
	NickName  string `gorm:"column:nick_name;size:64;not null;comment:昵称"`
	RealName  string `gorm:"column:real_name;size:64;not null;default:'';comment:真实姓名"`
	Account   string `gorm:"column:account;size:64;not null;uniqueIndex:uk_tenant_account;comment:账号"`
	Password  string `gorm:"column:password;size:64;not null;default:'';comment:密码"`
	Phone     string `gorm:"column:phone;size:16;not null;default:'';comment:手机号"`
	Email     string `gorm:"column:email;size:128;not null;default:'';comment:邮箱"`
	Salt      string `gorm:"column:salt;size:32;not null;comment:密码盐"`
	RoleIds   string `gorm:"column:role_ids;size:64;not null;default:'';comment:角色IDs"`
	Status    uint8  `gorm:"column:status;not null;default:0;comment:状态 1：正常 2：禁用"`
	CreatedAt uint   `gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt uint   `gorm:"column:updated_at;not null;comment:更新时间"`
}

func (*sysAdmin) TableName() string { return "cn_sys_admin" }

type sysRole struct {
	ID        uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID"`
	ParentID  uint64 `gorm:"column:parent_id;not null;default:0;index;comment:上级角色ID 0为根"`
	Level     uint16 `gorm:"column:level;not null;default:0;comment:角色层级深度 根为0"`
	Name      string `gorm:"column:name;size:64;not null;uniqueIndex;comment:角色名称"`
	Code      string `gorm:"column:code;size:32;not null;uniqueIndex;comment:角色唯一code"`
	Desc      string `gorm:"column:desc;size:64;not null;default:'';comment:角色描述"`
	IsSys     uint8  `gorm:"column:is_sys;not null;default:0;comment:是否系统角色 1：是 0：否"`
	RoleType  uint8  `gorm:"column:role_type;not null;default:1;comment:角色类型 1：平台专用 2：可赋给租户用户"`
	Status    uint8  `gorm:"column:status;not null;comment:状态：1正常(默认) 2停用"`
	CreatedAt uint   `gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt uint   `gorm:"column:updated_at;not null;comment:更新时间"`
}

func (*sysRole) TableName() string { return "cn_sys_role" }

type sysResource struct {
	ID           uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID"`
	Name         string `gorm:"column:name;size:64;not null;comment:名称"`
	Alias_       string `gorm:"column:alias;size:64;not null;uniqueIndex:backend_menu_alias_title_unique;comment:别名"`
	Desc         string `gorm:"column:desc;size:64;not null;comment:描述"`
	FURL         string `gorm:"column:f_url;size:64;not null;default:'';comment:前端路由"`
	BURL         string `gorm:"column:b_url;size:128;not null;default:'';comment:后端接口路径模式"`
	Methods      string `gorm:"column:methods;size:64;not null;default:'';comment:HTTP方法 GET,POST"`
	Icon         string `gorm:"column:icon;size:64;not null;default:'';comment:菜单icon"`
	ParentID     uint64 `gorm:"column:parent_id;not null;default:0;comment:父级ID"`
	Path         string `gorm:"column:path;size:64;not null;default:'';comment:ID路径 1-2-3..."`
	ResourceType uint8  `gorm:"column:resource_type;not null;default:1;comment:类型 1：目录 2：菜单 3：操作/按钮"`
	HideInMenu   uint8  `gorm:"column:hide_in_menu;not null;default:1;comment:侧栏显示：1显示 2隐藏"`
	Status       uint8  `gorm:"column:status;not null;comment:状态：1：正常 2：停用"`
	SortOrder    uint16 `gorm:"column:sort_order;not null;default:50;comment:排序"`
	CreatedAt    uint64 `gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt    uint64 `gorm:"column:updated_at;not null;comment:更新时间"`
}

func (*sysResource) TableName() string { return "cn_sys_resource" }

type sysRoleAuth struct {
	ID          uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID"`
	RoleID      uint64 `gorm:"column:role_id;not null;comment:角色ID"`
	ResourceIds string `gorm:"column:resource_ids;type:text;not null;comment:菜单id列表 1,2,3..."`
}

func (*sysRoleAuth) TableName() string { return "cn_sys_role_auth" }

type sysCasbinRule struct {
	ID    uint64  `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID"`
	Ptype *string `gorm:"column:ptype;size:100;uniqueIndex:idx_casbin_rule;comment:策略类型 p/g"`
	V0    *string `gorm:"column:v0;size:100;uniqueIndex:idx_casbin_rule;comment:主体(角色ID)"`
	V1    *string `gorm:"column:v1;size:100;uniqueIndex:idx_casbin_rule;comment:对象(路径)"`
	V2    *string `gorm:"column:v2;size:100;uniqueIndex:idx_casbin_rule;comment:动作(方法)"`
	V3    *string `gorm:"column:v3;size:100;uniqueIndex:idx_casbin_rule;comment:扩展字段v3"`
	V4    *string `gorm:"column:v4;size:100;uniqueIndex:idx_casbin_rule;comment:扩展字段v4"`
	V5    *string `gorm:"column:v5;size:100;uniqueIndex:idx_casbin_rule;comment:扩展字段v5"`
	V6    string  `gorm:"column:v6;size:25;not null;default:'';comment:扩展字段v6"`
	V7    string  `gorm:"column:v7;size:25;not null;default:'';comment:扩展字段v7"`
}

func (*sysCasbinRule) TableName() string { return "cn_sys_casbin_rule" }

type attachment struct {
	ID               uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID"`
	TenantID         uint64 `gorm:"column:tenant_id;not null;default:0;index;comment:租户ID 0：平台"`
	UserID           uint64 `gorm:"column:user_id;not null;default:0;comment:附件上传的用户id"`
	AttachName       string `gorm:"column:attach_name;size:64;not null;default:'';comment:附件新名称"`
	AttachOriginName string `gorm:"column:attach_origin_name;size:255;not null;default:'';comment:附件原名称"`
	AttachURL        string `gorm:"column:attach_url;size:255;not null;comment:附件地址"`
	AttachType       uint8  `gorm:"column:attach_type;not null;default:1;comment:附件类型 1：图片 2：视频 3：文件"`
	AttachMimetype   string `gorm:"column:attach_mimetype;size:128;not null;default:'';comment:附件mime类型"`
	AttachExtension  string `gorm:"column:attach_extension;size:16;not null;default:'';comment:附件后缀名"`
	AttachSize       string `gorm:"column:attach_size;size:32;not null;default:'';comment:附件大小"`
	Status           uint8  `gorm:"column:status;not null;comment:状态 1：正常 0：删除"`
	CreatedAt        uint   `gorm:"column:created_at;not null;default:0;comment:创建时间"`
	UpdatedAt        uint   `gorm:"column:updated_at;not null;default:0;comment:更新时间"`
}

func (*attachment) TableName() string { return "cn_attachment" }

type farmAccount struct {
	ID           uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID"`
	TenantID     uint64 `gorm:"column:tenant_id;not null;default:0;index;uniqueIndex:uk_farm_account_code;comment:租户ID"`
	Name         string `gorm:"column:name;size:64;not null;default:'';comment:显示名称"`
	Code         string `gorm:"column:code;size:512;not null;default:'';uniqueIndex:uk_farm_account_code;comment:网关登录Code(一次性)"`
	Platform     string `gorm:"column:platform;size:32;not null;default:'qq';comment:平台 qq/wx"`
	LoginOS      string `gorm:"column:login_os;size:64;not null;default:'';comment:登录URL中的os"`
	ClientVer    string `gorm:"column:client_ver;size:64;not null;default:'';comment:登录URL中的ver"`
	Uin          string `gorm:"column:uin;size:32;not null;default:'';index;comment:UIN"`
	QQ           string `gorm:"column:qq;size:32;not null;default:'';comment:QQ号"`
	Avatar       string `gorm:"column:avatar;size:512;not null;default:'';comment:头像URL"`
	Username     string `gorm:"column:username;size:64;not null;default:'';comment:归属用户名(兼容)"`
	Remark       string `gorm:"column:remark;size:255;not null;default:'';comment:备注"`
	RunStatus    uint8  `gorm:"column:run_status;not null;default:0;index;comment:运行状态 0：停止 1：运行中 2：异常"`
	LastOnlineAt uint   `gorm:"column:last_online_at;not null;default:0;comment:最近在线Unix秒"`
	Status       uint8  `gorm:"column:status;not null;default:1;comment:状态 1：正常 2：禁用"`
	CreatedAt    uint   `gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt    uint   `gorm:"column:updated_at;not null;comment:更新时间"`
}

func (*farmAccount) TableName() string { return "cn_farm_account" }

type farmAccountConfig struct {
	ID         uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID"`
	TenantID   uint64 `gorm:"column:tenant_id;not null;default:0;index;uniqueIndex:uk_farm_cfg_account;comment:租户ID"`
	AccountID  uint64 `gorm:"column:account_id;not null;uniqueIndex:uk_farm_cfg_account;index;comment:账号ID"`
	ConfigJSON string `gorm:"column:config_json;type:text;not null;default:'{}';comment:配置JSON"`
	CreatedAt  uint   `gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt  uint   `gorm:"column:updated_at;not null;comment:更新时间"`
}

func (*farmAccountConfig) TableName() string { return "cn_farm_account_config" }

type farmCard struct {
	ID           uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID"`
	Code         string `gorm:"column:code;size:32;not null;uniqueIndex;comment:卡密编码"`
	CardType     uint8  `gorm:"column:card_type;not null;default:1;index;comment:类型 1：时长 2：账号额度"`
	Value        int    `gorm:"column:value;not null;default:0;comment:面值 时长天数或账号数 -1：永久"`
	Description  string `gorm:"column:description;size:255;not null;default:'';comment:描述"`
	Status       uint8  `gorm:"column:status;not null;default:1;index;comment:状态 1：未用 2：已用 3：禁用"`
	UsedByTenant uint64 `gorm:"column:used_by_tenant;not null;default:0;index;comment:使用租户ID"`
	UsedAt       uint   `gorm:"column:used_at;not null;default:0;comment:使用时间Unix秒"`
	CreatedAt    uint   `gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt    uint   `gorm:"column:updated_at;not null;comment:更新时间"`
}

func (*farmCard) TableName() string { return "cn_farm_card" }

type farmCardClaim struct {
	ID        uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID"`
	TenantID  uint64 `gorm:"column:tenant_id;not null;default:0;index;comment:租户ID"`
	CardID    uint64 `gorm:"column:card_id;not null;index;comment:卡密ID"`
	CardCode  string `gorm:"column:card_code;size:32;not null;default:'';comment:卡密编码快照"`
	CardType  uint8  `gorm:"column:card_type;not null;default:1;comment:类型快照"`
	Value     int    `gorm:"column:value;not null;default:0;comment:面值快照"`
	ClaimedBy uint64 `gorm:"column:claimed_by;not null;default:0;comment:兑换人adminID"`
	CreatedAt uint   `gorm:"column:created_at;not null;comment:兑换时间"`
}

func (*farmCardClaim) TableName() string { return "cn_farm_card_claim" }

type farmFriendGid struct {
	ID        uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID"`
	TenantID  uint64 `gorm:"column:tenant_id;not null;default:0;index;uniqueIndex:uk_tenant_account_gid;comment:租户ID"`
	AccountID uint64 `gorm:"column:account_id;not null;uniqueIndex:uk_tenant_account_gid;index;comment:账号ID"`
	Gid       int64  `gorm:"column:gid;not null;uniqueIndex:uk_tenant_account_gid;comment:好友GID"`
	Nickname  string `gorm:"column:nickname;size:128;not null;default:'';comment:昵称"`
	SyncedAt  uint   `gorm:"column:synced_at;not null;default:0;comment:同步时间Unix秒"`
	CreatedAt uint   `gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt uint   `gorm:"column:updated_at;not null;comment:更新时间"`
}

func (*farmFriendGid) TableName() string { return "cn_farm_friend_gid" }

type farmStats struct {
	ID           uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID"`
	TenantID     uint64 `gorm:"column:tenant_id;not null;default:0;index;uniqueIndex:uk_tenant_account_day;comment:租户ID"`
	AccountID    uint64 `gorm:"column:account_id;not null;uniqueIndex:uk_tenant_account_day;index;comment:账号ID"`
	StatDate     string `gorm:"column:stat_date;size:10;not null;uniqueIndex:uk_tenant_account_day;comment:统计日YYYY-MM-DD"`
	Gold         int64  `gorm:"column:gold;not null;default:0;comment:金币增量"`
	Exp          int64  `gorm:"column:exp;not null;default:0;comment:经验增量"`
	HarvestCount int64  `gorm:"column:harvest_count;not null;default:0;comment:收获次数"`
	StealCount   int64  `gorm:"column:steal_count;not null;default:0;comment:偷取次数"`
	HelpCount    int64  `gorm:"column:help_count;not null;default:0;comment:帮忙次数"`
	PlantCount   int64  `gorm:"column:plant_count;not null;default:0;comment:种植次数"`
	ExtraJSON    string `gorm:"column:extra_json;type:text;not null;default:'{}';comment:扩展JSON"`
	CreatedAt    uint   `gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt    uint   `gorm:"column:updated_at;not null;comment:更新时间"`
}

func (*farmStats) TableName() string { return "cn_farm_stats" }

type farmInteractLog struct {
	ID         uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID"`
	TenantID   uint64 `gorm:"column:tenant_id;not null;default:0;index;comment:租户ID"`
	AccountID  uint64 `gorm:"column:account_id;not null;index;comment:账号ID"`
	TargetGid  int64  `gorm:"column:target_gid;not null;default:0;index;comment:目标GID"`
	Action     string `gorm:"column:action;size:32;not null;default:'';index;comment:动作 steal/help/..."`
	Result     string `gorm:"column:result;size:64;not null;default:'';comment:结果"`
	DetailJSON string `gorm:"column:detail_json;type:text;not null;default:'{}';comment:详情JSON"`
	CreatedAt  uint   `gorm:"column:created_at;not null;index;comment:创建时间"`
}

func (*farmInteractLog) TableName() string { return "cn_farm_interact_log" }

type farmSystemConfig struct {
	ID         uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID"`
	TenantID   uint64 `gorm:"column:tenant_id;not null;default:0;index;uniqueIndex:uk_tenant_key;comment:租户ID 0：平台"`
	ConfigKey  string `gorm:"column:config_key;size:64;not null;uniqueIndex:uk_tenant_key;comment:配置键"`
	ConfigJSON string `gorm:"column:config_json;type:text;not null;default:'{}';comment:配置JSON"`
	Remark     string `gorm:"column:remark;size:255;not null;default:'';comment:备注"`
	CreatedAt  uint   `gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt  uint   `gorm:"column:updated_at;not null;comment:更新时间"`
}

func (*farmSystemConfig) TableName() string { return "cn_farm_system_config" }

type farmGameConfig struct {
	ID         uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID"`
	TenantID   uint64 `gorm:"column:tenant_id;not null;default:0;index;uniqueIndex:uk_tenant_cat_key;comment:租户ID 0：全局"`
	Category   string `gorm:"column:category;size:64;not null;default:'';uniqueIndex:uk_tenant_cat_key;comment:分类 seed/item/..."`
	ConfigKey  string `gorm:"column:config_key;size:64;not null;uniqueIndex:uk_tenant_cat_key;comment:配置键"`
	ConfigJSON string `gorm:"column:config_json;type:text;not null;default:'{}';comment:配置JSON"`
	Version    uint   `gorm:"column:version;not null;default:1;comment:版本号"`
	Status     uint8  `gorm:"column:status;not null;default:1;comment:状态 1：正常 2：禁用"`
	CreatedAt  uint   `gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt  uint   `gorm:"column:updated_at;not null;comment:更新时间"`
}

func (*farmGameConfig) TableName() string { return "cn_farm_game_config" }

type farmActivityState struct {
	ID         uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID"`
	TenantID   uint64 `gorm:"column:tenant_id;not null;default:0;index;uniqueIndex:uk_tenant_account_act;comment:租户ID"`
	AccountID  uint64 `gorm:"column:account_id;not null;uniqueIndex:uk_tenant_account_act;index;comment:账号ID"`
	ActivityID string `gorm:"column:activity_id;size:64;not null;uniqueIndex:uk_tenant_account_act;comment:活动ID"`
	StateJSON  string `gorm:"column:state_json;type:text;not null;default:'{}';comment:状态JSON"`
	SyncedAt   uint   `gorm:"column:synced_at;not null;default:0;comment:同步时间Unix秒"`
	CreatedAt  uint   `gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt  uint   `gorm:"column:updated_at;not null;comment:更新时间"`
}

func (*farmActivityState) TableName() string { return "cn_farm_activity_state" }
