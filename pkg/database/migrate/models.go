package migrate

// PostgreSQL 兼容的迁移结构（含字段注释；GORM 会生成 COMMENT ON COLUMN）

type sysAdmin struct {
	ID        uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID"`
	UUID      string `gorm:"column:uuid;size:32;not null;comment:唯一id号"`
	NickName  string `gorm:"column:nick_name;size:64;not null;comment:昵称"`
	RealName  string `gorm:"column:real_name;size:64;not null;default:'';comment:真实姓名"`
	Account   string `gorm:"column:account;size:64;not null;uniqueIndex:uk_admin_account;comment:账号"`
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

type farmAccount struct {
	ID            uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID"`
	Name          string `gorm:"column:name;size:64;not null;default:'';comment:显示名称"`
	Code          string `gorm:"column:code;size:512;not null;default:'';uniqueIndex:uk_farm_account_code;comment:网关登录Code(一次性)"`
	Platform      string `gorm:"column:platform;size:32;not null;default:'qq';comment:平台 qq/wx"`
	LoginOS       string `gorm:"column:login_os;size:64;not null;default:'';comment:登录URL中的os"`
	ClientVer     string `gorm:"column:client_ver;size:64;not null;default:'';comment:登录URL中的ver"`
	Uin           string `gorm:"column:uin;size:32;not null;default:'';index;comment:UIN"`
	QQ            string `gorm:"column:qq;size:32;not null;default:'';comment:QQ号"`
	Avatar        string `gorm:"column:avatar;size:512;not null;default:'';comment:头像URL"`
	Username      string `gorm:"column:username;size:64;not null;default:'';comment:归属用户名(兼容)"`
	Remark        string `gorm:"column:remark;size:255;not null;default:'';comment:备注"`
	RunStatus     uint8  `gorm:"column:run_status;not null;default:0;index;comment:运行状态 0：停止 1：运行中 2：异常"`
	LastOnlineAt  uint   `gorm:"column:last_online_at;not null;default:0;comment:最近在线Unix秒"`
	Status        uint8  `gorm:"column:status;not null;default:1;comment:状态 1：正常 2：禁用"`
	WxOpenID      string `gorm:"column:wx_openid;size:128;not null;default:'';comment:应用宝openid"`
	WxLoginBuffer string `gorm:"column:wx_login_buffer;type:text;not null;default:'';comment:应用宝login_buffer"`
	WxAccessToken string `gorm:"column:wx_access_token;size:512;not null;default:'';comment:应用宝accesstoken"`
	CreatedAt     uint   `gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt     uint   `gorm:"column:updated_at;not null;comment:更新时间"`
}

func (*farmAccount) TableName() string { return "cn_farm_account" }

type farmAccountConfig struct {
	ID         uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID"`
	AccountID  uint64 `gorm:"column:account_id;not null;uniqueIndex:uk_farm_cfg_account;index;comment:账号ID"`
	ConfigJSON string `gorm:"column:config_json;type:text;not null;default:'{}';comment:配置JSON"`
	CreatedAt  uint   `gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt  uint   `gorm:"column:updated_at;not null;comment:更新时间"`
}

func (*farmAccountConfig) TableName() string { return "cn_farm_account_config" }

type farmFriendGid struct {
	ID        uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID"`
	AccountID uint64 `gorm:"column:account_id;not null;uniqueIndex:uk_account_gid;index;comment:账号ID"`
	Gid       int64  `gorm:"column:gid;not null;uniqueIndex:uk_account_gid;comment:好友GID"`
	Nickname  string `gorm:"column:nickname;size:128;not null;default:'';comment:昵称"`
	SyncedAt  uint   `gorm:"column:synced_at;not null;default:0;comment:同步时间Unix秒"`
	CreatedAt uint   `gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt uint   `gorm:"column:updated_at;not null;comment:更新时间"`
}

func (*farmFriendGid) TableName() string { return "cn_farm_friend_gid" }

type farmStats struct {
	ID           uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID"`
	AccountID    uint64 `gorm:"column:account_id;not null;uniqueIndex:uk_account_day;index;comment:账号ID"`
	StatDate     string `gorm:"column:stat_date;size:10;not null;uniqueIndex:uk_account_day;comment:统计日YYYY-MM-DD"`
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
	ConfigKey  string `gorm:"column:config_key;size:64;not null;uniqueIndex:uk_sys_config_key;comment:配置键"`
	ConfigJSON string `gorm:"column:config_json;type:text;not null;default:'{}';comment:配置JSON"`
	Remark     string `gorm:"column:remark;size:255;not null;default:'';comment:备注"`
	CreatedAt  uint   `gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt  uint   `gorm:"column:updated_at;not null;comment:更新时间"`
}

func (*farmSystemConfig) TableName() string { return "cn_farm_system_config" }

type farmGameConfig struct {
	ID         uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID"`
	Category   string `gorm:"column:category;size:64;not null;default:'';uniqueIndex:uk_game_cat_key;comment:分类 seed/item/..."`
	ConfigKey  string `gorm:"column:config_key;size:64;not null;uniqueIndex:uk_game_cat_key;comment:配置键"`
	ConfigJSON string `gorm:"column:config_json;type:text;not null;default:'{}';comment:配置JSON"`
	Version    uint   `gorm:"column:version;not null;default:1;comment:版本号"`
	Status     uint8  `gorm:"column:status;not null;default:1;comment:状态 1：正常 2：禁用"`
	CreatedAt  uint   `gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt  uint   `gorm:"column:updated_at;not null;comment:更新时间"`
}

func (*farmGameConfig) TableName() string { return "cn_farm_game_config" }

type farmActivityState struct {
	ID         uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID"`
	AccountID  uint64 `gorm:"column:account_id;not null;uniqueIndex:uk_account_act;index;comment:账号ID"`
	ActivityID string `gorm:"column:activity_id;size:64;not null;uniqueIndex:uk_account_act;comment:活动ID"`
	StateJSON  string `gorm:"column:state_json;type:text;not null;default:'{}';comment:状态JSON"`
	SyncedAt   uint   `gorm:"column:synced_at;not null;default:0;comment:同步时间Unix秒"`
	CreatedAt  uint   `gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt  uint   `gorm:"column:updated_at;not null;comment:更新时间"`
}

func (*farmActivityState) TableName() string { return "cn_farm_activity_state" }
