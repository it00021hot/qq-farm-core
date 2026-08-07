package farm

// AccountCreateReq 创建农场账号（code 可为裸登录 Code 或完整登录 URL）
type AccountCreateReq struct {
	Name     string `json:"name" validate:"omitempty,max=64"`
	Code     string `json:"code" validate:"required,max=2048"`
	Platform string `json:"platform" validate:"omitempty,oneof=qq wx"`
	Uin      string `json:"uin" validate:"max=32"`
	QQ       string `json:"qq" validate:"max=32"`
	Avatar   string `json:"avatar" validate:"max=512"`
	Remark   string `json:"remark" validate:"max=255"`
	Status   uint8  `json:"status" validate:"omitempty,oneof=1 2"`
}

// AccountUpdateReq 更新农场账号（可刷新登录 Code）
type AccountUpdateReq struct {
	ID       uint64 `json:"id" validate:"required"`
	Name     string `json:"name" validate:"omitempty,max=64"`
	Code     string `json:"code" validate:"required,max=2048"`
	Platform string `json:"platform" validate:"omitempty,oneof=qq wx"`
	Uin      string `json:"uin" validate:"max=32"`
	QQ       string `json:"qq" validate:"max=32"`
	Avatar   string `json:"avatar" validate:"max=512"`
	Remark   string `json:"remark" validate:"max=255"`
	Status   uint8  `json:"status" validate:"required,oneof=1 2"`
}

// AccountListReq 账号列表
type AccountListReq struct {
	Current   int    `json:"current" query:"current"`
	Size      int    `json:"size" query:"size"`
	Keyword   string `json:"keyword" query:"keyword"`
	Status    uint8  `json:"status" query:"status"`
	RunStatus *uint8 `json:"runStatus" query:"runStatus"`
}

// AccountIDReq 账号 ID
type AccountIDReq struct {
	ID uint64 `json:"id" query:"id" validate:"required"`
}

// LandsReq requests live lands for a running account.
type LandsReq struct {
	AccountID uint64 `json:"accountId" query:"accountId" validate:"required"`
}

// OperateReq requests a manual farm operation.
type OperateReq struct {
	AccountID uint64 `json:"accountId" validate:"required"`
	Op        string `json:"op" validate:"required,oneof=all harvest clear plant upgrade"`
}

// BagReq requests live bag items for a running account.
type BagReq struct {
	AccountID uint64 `json:"accountId" query:"accountId" validate:"required"`
}

// BagSellItem is one sell line.
type BagSellItem struct {
	ID    int64 `json:"id" validate:"required"`
	Count int64 `json:"count" validate:"required,min=1"`
	UID   int64 `json:"uid"`
}

// BagSellReq sells bag items.
type BagSellReq struct {
	AccountID uint64        `json:"accountId" validate:"required"`
	Items     []BagSellItem `json:"items" validate:"required,min=1,dive"`
}

// BagUseReq uses one bag item.
type BagUseReq struct {
	AccountID uint64 `json:"accountId" validate:"required"`
	ItemID    int64  `json:"itemId" validate:"required"`
	Count     int64  `json:"count" validate:"required,min=1"`
}

// DailyGiftsReq requests personal daily-gift / task overview.
type DailyGiftsReq struct {
	AccountID uint64 `json:"accountId" query:"accountId" validate:"required"`
}

// AccountDeleteReq 删除账号
type AccountDeleteReq struct {
	ID uint64 `json:"id" validate:"required"`
}

// CardGenerateReq 批量生成卡密
type CardGenerateReq struct {
	CardType    uint8  `json:"cardType" validate:"required,oneof=1 2"`
	Value       int    `json:"value" validate:"required"`
	Count       int    `json:"count" validate:"required,min=1,max=200"`
	Description string `json:"description" validate:"max=255"`
}

// CardRedeemReq 兑换卡密
type CardRedeemReq struct {
	Code string `json:"code" validate:"required,max=32"`
}

// CardListReq 卡密列表
type CardListReq struct {
	Current  int    `json:"current" query:"current"`
	Size     int    `json:"size" query:"size"`
	Keyword  string `json:"keyword" query:"keyword"`
	CardType uint8  `json:"cardType" query:"cardType"`
	Status   uint8  `json:"status" query:"status"`
}

// CardStatusReq 作废/启停卡密
type CardStatusReq struct {
	ID     uint64 `json:"id" validate:"required"`
	Status uint8  `json:"status" validate:"required,oneof=1 3"`
}

// AutomationDetailReq 自动化配置详情
type AutomationDetailReq struct {
	AccountID uint64 `json:"accountId" query:"accountId" validate:"required"`
}

// AutomationModifyReq 修改自动化配置（结构化，对齐 AccountConfig）
type AutomationModifyReq struct {
	AccountID                          uint64          `json:"accountId" validate:"required"`
	Automation                         *map[string]any `json:"automation"`
	Intervals                          *map[string]any `json:"intervals"`
	PlantingStrategy                   *string         `json:"plantingStrategy"`
	PreferredSeedID                    *int64          `json:"preferredSeedId"`
	BagSeedPriority                    []int64         `json:"bagSeedPriority"`
	BagSeedFallbackStrategy            *string         `json:"bagSeedFallbackStrategy"`
	PlantOrderRandom                   *bool           `json:"plantOrderRandom"`
	PlantDelaySeconds                  *int            `json:"plantDelaySeconds"`
	StealDelaySeconds                  *int            `json:"stealDelaySeconds"`
	FriendQuietHours                   *map[string]any `json:"friendQuietHours"`
	FriendBlacklist                    []int64         `json:"friendBlacklist"`
	PlantBlacklist                     []int64         `json:"plantBlacklist"`
	FertilizerBuyOrganicCount          *int            `json:"fertilizerBuyOrganicCount"`
	FertilizerBuyOrganicThresholdHours *int            `json:"fertilizerBuyOrganicThresholdHours"`
	FertilizerBuyNormalCount           *int            `json:"fertilizerBuyNormalCount"`
	FertilizerBuyNormalThresholdHours  *int            `json:"fertilizerBuyNormalThresholdHours"`
	FertilizerBuyCheckIntervalMinutes  *int            `json:"fertilizerBuyCheckIntervalMinutes"`
	ConfigJSON                         string          `json:"configJson"` // 兼容旧客户端
}

// StatusDetailReq 运行状态详情
type StatusDetailReq struct {
	AccountID uint64 `json:"accountId" query:"accountId" validate:"required"`
}

// LogsListReq 运行日志查询（内存环缓冲）
type LogsListReq struct {
	AccountID uint64 `json:"accountId" query:"accountId"`
	Module    string `json:"module" query:"module"`
	Keyword   string `json:"keyword" query:"keyword"`
	Limit     int    `json:"limit" query:"limit"`
}

// LogsClearReq 清空运行日志
type LogsClearReq struct {
	AccountID uint64 `json:"accountId" query:"accountId"`
}

// StatusListReq 运行状态列表
type StatusListReq struct {
	Current int    `json:"current" query:"current"`
	Size    int    `json:"size" query:"size"`
	Keyword string `json:"keyword" query:"keyword"`
}

// FriendListReq 好友/互动列表
type FriendListReq struct {
	Current   int    `json:"current" query:"current"`
	Size      int    `json:"size" query:"size"`
	AccountID uint64 `json:"accountId" query:"accountId"`
	Keyword   string `json:"keyword" query:"keyword"`
}

// FriendInteractRecordsReq 最近访客（游戏侧互动记录）
type FriendInteractRecordsReq struct {
	AccountID uint64 `json:"accountId" query:"accountId" validate:"required"`
}

// FriendSyncReq 同步好友列表
type FriendSyncReq struct {
	AccountID uint64 `json:"accountId" validate:"required"`
}

// FriendLandsReq 好友土地详情
type FriendLandsReq struct {
	AccountID uint64 `json:"accountId" query:"accountId" validate:"required"`
	Gid       int64  `json:"gid" query:"gid" validate:"required"`
}

// FriendOpReq 好友互动操作
type FriendOpReq struct {
	AccountID uint64 `json:"accountId" validate:"required"`
	Gid       int64  `json:"gid" validate:"required"`
	Op        string `json:"op" validate:"required,oneof=steal help water weed bug bad"`
}

// ActivitySnapshotReq 活动快照
type ActivitySnapshotReq struct {
	AccountID uint64 `json:"accountId" query:"accountId" validate:"required"`
}

// ActivityActionReq 活动操作
type ActivityActionReq struct {
	AccountID uint64 `json:"accountId" validate:"required"`
	TermID    string `json:"termId"`
	ItemID    string `json:"itemId"`
	Count     int64  `json:"count"`
}

// AnalyticsDetailReq 分析详情
type AnalyticsDetailReq struct {
	AccountID uint64 `json:"accountId" query:"accountId" validate:"required"`
	Days      int    `json:"days" query:"days"`
	Sort      string `json:"sort" query:"sort"`
}

// GameConfigListReq 游戏配置列表（DB 元数据表）
type GameConfigListReq struct {
	Current  int    `json:"current" query:"current"`
	Size     int    `json:"size" query:"size"`
	Category string `json:"category" query:"category"`
	Keyword  string `json:"keyword" query:"keyword"`
}

// GameConfigModifyReq 修改游戏配置（DB 元数据表）
type GameConfigModifyReq struct {
	ID         uint64 `json:"id" validate:"required"`
	ConfigJSON string `json:"configJson" validate:"required"`
	Status     uint8  `json:"status" validate:"omitempty,oneof=1 2"`
}

// GameConfigItemsReq 道具列表（可选按 type 过滤）
type GameConfigItemsReq struct {
	Type int64 `json:"type" query:"type"`
}

// GameConfigSeedWriteReq 种子录入/修改
type GameConfigSeedWriteReq struct {
	SeedID        int64  `json:"seedId" validate:"required"`
	Name          string `json:"name"`
	GrowPhases    string `json:"growPhases"`
	LandLevelNeed int64  `json:"landLevelNeed"`
	Seasons       int64  `json:"seasons"`
	FruitCount    int64  `json:"fruitCount"`
	Price         int64  `json:"price"`
	PriceID       int64  `json:"priceId"`
	Exp           int64  `json:"exp"`
	Size          int64  `json:"size"`
}

// GameConfigSeedDeleteReq 删除种子
type GameConfigSeedDeleteReq struct {
	SeedID int64 `json:"seedId" validate:"required"`
}

// GameConfigFruitWriteReq 果实录入/修改
type GameConfigFruitWriteReq struct {
	ID         int64  `json:"id"`
	PlantID    int64  `json:"plantId"`
	Name       string `json:"name"`
	Price      int64  `json:"price"`
	PriceID    int64  `json:"priceId"`
	Desc       string `json:"desc"`
	EffectDesc string `json:"effectDesc"`
	Rarity     int64  `json:"rarity"`
	MaxCount   int64  `json:"maxCount"`
	Level      int64  `json:"level"`
	FruitCount int64  `json:"fruitCount"`
	AssetName  string `json:"assetName"`
}

// GameConfigFruitDeleteReq 删除果实
type GameConfigFruitDeleteReq struct {
	ID int64 `json:"id" validate:"required"`
}

// GameConfigItemWriteReq 道具录入/修改
type GameConfigItemWriteReq struct {
	ID              int64  `json:"id" validate:"required"`
	Type            int64  `json:"type"`
	Name            string `json:"name"`
	Price           int64  `json:"price"`
	PriceID         int64  `json:"priceId"`
	InteractionType string `json:"interactionType"`
	CanUse          int64  `json:"canUse"`
	Desc            string `json:"desc"`
	EffectDesc      string `json:"effectDesc"`
	Rarity          int64  `json:"rarity"`
	MaxCount        int64  `json:"maxCount"`
	Level           int64  `json:"level"`
	AssetName       string `json:"assetName"`
}

// GameConfigItemDeleteReq 删除道具
type GameConfigItemDeleteReq struct {
	ID int64 `json:"id" validate:"required"`
}
