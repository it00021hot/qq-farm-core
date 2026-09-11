package logic

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Config defaults mirror qq-farm-bot/core/src/models/store/shared-state.ts
// DEFAULT_ACCOUNT_CONFIG and types/config.ts.

// PlantingStrategy values.
const (
	StrategyPreferred     = "preferred"
	StrategyLevel         = "level"
	StrategyMaxExp        = "max_exp"
	StrategyMaxFertExp    = "max_fert_exp"
	StrategyMaxProfit     = "max_profit"
	StrategyMaxFertProfit = "max_fert_profit"
	StrategyBagPriority   = "bag_priority"
)

// AllowedPlantingStrategies mirrors ALLOWED_PLANTING_STRATEGIES.
var AllowedPlantingStrategies = []string{
	StrategyPreferred, StrategyLevel, StrategyMaxExp, StrategyMaxFertExp,
	StrategyMaxProfit, StrategyMaxFertProfit, StrategyBagPriority,
}

// FertilizerMode values.
const (
	FertilizerBoth    = "both"
	FertilizerNormal  = "normal"
	FertilizerOrganic = "organic"
	FertilizerSmart   = "smart"
	FertilizerNone    = "none"
)

// Fertilizer land type keys.
const (
	LandTypePurpleGold = "purple-gold"
	LandTypeGold       = "gold"
	LandTypeBlack      = "black"
	LandTypeRed        = "red"
	LandTypeNormal     = "normal"
)

// AllFertilizerLandTypes is the full selectable set (land-analysis.ts).
var AllFertilizerLandTypes = []string{
	LandTypePurpleGold, LandTypeGold, LandTypeBlack, LandTypeRed, LandTypeNormal,
}

// FertilizerLandTypeLabels maps type keys to Chinese labels.
var FertilizerLandTypeLabels = map[string]string{
	LandTypePurpleGold: "紫金土地",
	LandTypeGold:       "金土地",
	LandTypeBlack:      "黑土地",
	LandTypeRed:        "红土地",
	LandTypeNormal:     "普通土地",
}

// AutomationConfig mirrors types/config.ts AutomationConfig.
type AutomationConfig struct {
	Farm                               bool     `json:"farm"`
	FarmPush                           bool     `json:"farm_push"`
	LandUpgrade                        bool     `json:"land_upgrade"`
	Friend                             bool     `json:"friend"`
	FriendAutoAccept                   bool     `json:"friend_auto_accept"`
	FriendHelpExpLimit                 bool     `json:"friend_help_exp_limit"`
	FriendSteal                        bool     `json:"friend_steal"`
	FriendHelp                         bool     `json:"friend_help"`
	FriendBad                          bool     `json:"friend_bad"`
	FriendHelpProtectDogIgnoreExpLimit bool     `json:"friend_help_protect_dog_ignore_exp_limit"`
	Task                               bool     `json:"task"`
	FertilizerGift                     bool     `json:"fertilizer_gift"`
	FertilizerBuyOrganic               bool     `json:"fertilizer_buy_organic"`
	FertilizerBuyNormal                bool     `json:"fertilizer_buy_normal"`
	MysteryShopAutoBuy                 bool     `json:"mystery_shop_auto_buy"`
	MysteryShopAllowGold               bool     `json:"mystery_shop_allow_gold"`
	MysteryShopAllowCoupon             bool     `json:"mystery_shop_allow_coupon"`
	MysteryShopAllowGoldBean           bool     `json:"mystery_shop_allow_gold_bean"`
	MysteryShopAllowDiamond            bool     `json:"mystery_shop_allow_diamond"`
	MysteryShopArrivalNotify           bool     `json:"mystery_shop_arrival_notify"`
	MysteryShopPurchaseNotify          bool     `json:"mystery_shop_purchase_notify"`
	Sell                               bool     `json:"sell"`
	Fertilizer                         string   `json:"fertilizer"`
	FertilizerMultiSeason              bool     `json:"fertilizer_multi_season"`
	FertilizerLandTypes                []string `json:"fertilizer_land_types"`
	FertilizerSmartSeconds             int      `json:"fertilizer_smart_seconds"`
	SkipOwnWeedBug                     bool     `json:"skip_own_weed_bug"`
	ShowManualFertilizer               bool     `json:"show_manual_fertilizer"`
	// Legacy keys from older UI forks; ignored by settings and daily routines (bot always-on).
	FarmManage     bool `json:"farm_manage,omitempty"`
	FarmWater      bool `json:"farm_water,omitempty"`
	FarmWeed       bool `json:"farm_weed,omitempty"`
	FarmBug        bool `json:"farm_bug,omitempty"`
	Email          bool `json:"email,omitempty"`
	FreeGifts      bool `json:"free_gifts,omitempty"`
	ShareReward    bool `json:"share_reward,omitempty"`
	VipGift        bool `json:"vip_gift,omitempty"`
	MonthCard      bool `json:"month_card,omitempty"`
	OpenServerGift bool `json:"open_server_gift,omitempty"`
}

// IntervalConfig mirrors types/config.ts IntervalConfig (seconds).
type IntervalConfig struct {
	Farm      int `json:"farm"`
	FarmMin   int `json:"farmMin"`
	FarmMax   int `json:"farmMax"`
	FriendMin int `json:"friendMin"`
	FriendMax int `json:"friendMax"`
	HelpMin   int `json:"helpMin"`
	HelpMax   int `json:"helpMax"`
	StealMin  int `json:"stealMin"`
	StealMax  int `json:"stealMax"`
}

// QuietHoursConfig mirrors types/config.ts QuietHoursConfig.
type QuietHoursConfig struct {
	Enabled      bool   `json:"enabled"`
	Start        string `json:"start"`
	End          string `json:"end"`
	ContinueFarm bool   `json:"continueFarm"`
}

// AccountConfig mirrors types/config.ts AccountConfig.
type AccountConfig struct {
	Automation                         AutomationConfig    `json:"automation"`
	PlantingStrategy                   string              `json:"plantingStrategy"`
	PreferredSeedID                    int64               `json:"preferredSeedId"`
	Intervals                          IntervalConfig      `json:"intervals"`
	FriendQuietHours                   QuietHoursConfig    `json:"friendQuietHours"`
	KnownFriendGids                    []int64             `json:"knownFriendGids"`
	KnownFriendGidSyncCooldownSec      int                 `json:"knownFriendGidSyncCooldownSec"`
	FriendsListCacheTtlSec             int                 `json:"friendsListCacheTtlSec"`
	FriendBlacklist                    []int64             `json:"friendBlacklist"`
	PlantBlacklist                     []int64             `json:"plantBlacklist"`
	StealDelaySeconds                  int                 `json:"stealDelaySeconds"`
	PlantOrderRandom                   bool                `json:"plantOrderRandom"`
	PlantDelaySeconds                  int                 `json:"plantDelaySeconds"`
	FertilizerBuyOrganicCount          int                 `json:"fertilizerBuyOrganicCount"`
	FertilizerBuyOrganicThresholdHours int                 `json:"fertilizerBuyOrganicThresholdHours"`
	FertilizerBuyNormalCount           int                 `json:"fertilizerBuyNormalCount"`
	FertilizerBuyNormalThresholdHours  int                 `json:"fertilizerBuyNormalThresholdHours"`
	FertilizerBuyCheckIntervalMinutes  int                 `json:"fertilizerBuyCheckIntervalMinutes"`
	BagSeedPriority                    []int64             `json:"bagSeedPriority"`
	BagSeedLandTypes                   map[string][]string `json:"bagSeedLandTypes"`
	BagSeedFallbackStrategy            string              `json:"bagSeedFallbackStrategy"`
	AutoAcceptFriendMinLevel           int                 `json:"autoAcceptFriendMinLevel"`
	AutoAcceptRequireOwnLevel          bool                `json:"autoAcceptRequireOwnLevel"`
	AutoAcceptHarvestStealEnabled      bool                `json:"autoAcceptHarvestStealEnabled"`
	AutoAcceptHarvestStealHarvest      int                 `json:"autoAcceptHarvestStealHarvest"`
	AutoAcceptHarvestStealSteal        int                 `json:"autoAcceptHarvestStealSteal"`
}

// DefaultAccountConfig returns a deep copy of DEFAULT_ACCOUNT_CONFIG.
func DefaultAccountConfig() AccountConfig {
	return AccountConfig{
		Automation: AutomationConfig{
			Farm:                               true,
			FarmPush:                           true,
			LandUpgrade:                        true,
			Friend:                             true,
			FriendAutoAccept:                   true,
			FriendHelpExpLimit:                 true,
			FriendSteal:                        true,
			FriendHelp:                         true,
			FriendBad:                          true,
			FriendHelpProtectDogIgnoreExpLimit: true,
			Task:                               true,
			FertilizerGift:                     false,
			FertilizerBuyOrganic:               false,
			FertilizerBuyNormal:                false,
			MysteryShopAutoBuy:                 false,
			MysteryShopAllowGold:               true,
			MysteryShopAllowCoupon:             false,
			MysteryShopAllowGoldBean:           false,
			MysteryShopAllowDiamond:            false,
			MysteryShopArrivalNotify:           false,
			MysteryShopPurchaseNotify:          false,
			Sell:                               true,
			Fertilizer:                         FertilizerSmart,
			FertilizerMultiSeason:              true,
			FertilizerLandTypes:                append([]string(nil), AllFertilizerLandTypes...),
			FertilizerSmartSeconds:             360,
			SkipOwnWeedBug:                     true,
			ShowManualFertilizer:               true,
		},
		// 新账号默认对齐本机账号 1（rust default_account_config，4fe322f）：
		// 背包优先种植、偷菜间隔 60–90、好友安静时段 01:00–08:30、兑种回退优先。
		PlantingStrategy: StrategyBagPriority,
		PreferredSeedID:  0,
		Intervals: IntervalConfig{
			Farm: 2, FarmMin: 20, FarmMax: 25,
			FriendMin: 20, FriendMax: 25,
			HelpMin: 20, HelpMax: 25,
			StealMin: 60, StealMax: 90,
		},
		FriendQuietHours: QuietHoursConfig{
			Enabled:      true,
			Start:        "01:00",
			End:          "08:30",
			ContinueFarm: true,
		},
		KnownFriendGids:                    nil,
		KnownFriendGidSyncCooldownSec:      300,
		FriendsListCacheTtlSec:             60,
		FriendBlacklist:                    nil,
		PlantBlacklist:                     []int64{20002, 20003, 20059, 20065, 20064, 20060, 20061},
		StealDelaySeconds:                  1,
		PlantOrderRandom:                   true,
		PlantDelaySeconds:                  2,
		FertilizerBuyOrganicCount:          1,
		FertilizerBuyOrganicThresholdHours: 10,
		FertilizerBuyNormalCount:           1,
		FertilizerBuyNormalThresholdHours:  10,
		FertilizerBuyCheckIntervalMinutes:  60,
		// 默认背包种子优先顺序（对齐 rust DEFAULT_BAG_SEED_PRIORITY；仅新账号初始值）。
		BagSeedPriority:               []int64{29003, 20129, 21380, 20108, 26032},
		BagSeedLandTypes:              map[string][]string{},
		BagSeedFallbackStrategy:       StrategyPreferred,
		AutoAcceptFriendMinLevel:      0,
		AutoAcceptRequireOwnLevel:     false,
		AutoAcceptHarvestStealEnabled: true,
		AutoAcceptHarvestStealHarvest: 8,
		AutoAcceptHarvestStealSteal:   1,
	}
}

// ParseAccountConfigJSON loads config on top of defaults so omitted JSON keys keep bot defaults,
// while explicit false values are preserved. All numeric windows are then clamped
// (bot shared-state normalize* helpers).
func ParseAccountConfigJSON(raw string) AccountConfig {
	cfg := DefaultAccountConfig()
	if strings.TrimSpace(raw) == "" {
		return cfg
	}
	_ = json.Unmarshal([]byte(raw), &cfg)
	normalizeAccountConfig(&cfg)
	return cfg
}

// clampInt bounds v into [min, max].
func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// validPlantingStrategy mirrors ALLOWED_PLANTING_STRATEGIES.
func validPlantingStrategy(s string) bool {
	for _, allowed := range AllowedPlantingStrategies {
		if s == allowed {
			return true
		}
	}
	return false
}

// normalizeAccountConfig mirrors bot normalizeAccountConfig clamping.
func normalizeAccountConfig(cfg *AccountConfig) {
	if !validPlantingStrategy(cfg.PlantingStrategy) {
		// rust 语义：非法值回退默认（bag_priority，4fe322f 新账号默认）。
		cfg.PlantingStrategy = StrategyBagPriority
	}
	if cfg.PreferredSeedID < 0 {
		cfg.PreferredSeedID = 0
	}
	// intervals：至少 1 秒，min>max 互换，统一好友区间取 help/steal 较快的一组。
	iv := &cfg.Intervals
	iv.Farm = clampInt(iv.Farm, 1, 86400)
	iv.FarmMin = clampInt(iv.FarmMin, 1, 86400)
	iv.FarmMax = clampInt(iv.FarmMax, 1, 86400)
	if iv.FarmMin > iv.FarmMax {
		iv.FarmMin, iv.FarmMax = iv.FarmMax, iv.FarmMin
	}
	iv.HelpMin = clampInt(iv.HelpMin, 1, 86400)
	iv.HelpMax = clampInt(iv.HelpMax, 1, 86400)
	if iv.HelpMin > iv.HelpMax {
		iv.HelpMin, iv.HelpMax = iv.HelpMax, iv.HelpMin
	}
	iv.StealMin = clampInt(iv.StealMin, 1, 86400)
	iv.StealMax = clampInt(iv.StealMax, 1, 86400)
	if iv.StealMin > iv.StealMax {
		iv.StealMin, iv.StealMax = iv.StealMax, iv.StealMin
	}
	iv.FriendMin = clampInt(iv.FriendMin, 1, 86400)
	iv.FriendMax = clampInt(iv.FriendMax, 1, 86400)
	if iv.FriendMin > iv.FriendMax {
		iv.FriendMin, iv.FriendMax = iv.FriendMax, iv.FriendMin
	}
	// 安静时段时间串 HH:MM。
	cfg.FriendQuietHours.Start = normalizeTimeString(cfg.FriendQuietHours.Start, "01:00")
	cfg.FriendQuietHours.End = normalizeTimeString(cfg.FriendQuietHours.End, "07:30")
	// GID 同步冷却与列表缓存 TTL。
	cfg.KnownFriendGidSyncCooldownSec = clampInt(cfg.KnownFriendGidSyncCooldownSec, 30, 86400)
	cfg.FriendsListCacheTtlSec = clampInt(cfg.FriendsListCacheTtlSec, 10, 86400)
	// 黑名单去重正数。
	cfg.FriendBlacklist = normalizeGIDList(cfg.FriendBlacklist)
	cfg.PlantBlacklist = normalizeGIDList(cfg.PlantBlacklist)
	cfg.KnownFriendGids = normalizeGIDList(cfg.KnownFriendGids)
	cfg.BagSeedPriority = normalizeGIDList(cfg.BagSeedPriority)
	// 偷菜/种植延迟。
	cfg.StealDelaySeconds = clampInt(cfg.StealDelaySeconds, 0, 300)
	cfg.PlantDelaySeconds = clampInt(cfg.PlantDelaySeconds, 0, 60)
	// 肥料购买。
	cfg.FertilizerBuyOrganicCount = clampInt(cfg.FertilizerBuyOrganicCount, 0, 10000)
	cfg.FertilizerBuyOrganicThresholdHours = clampInt(cfg.FertilizerBuyOrganicThresholdHours, 0, 990)
	cfg.FertilizerBuyNormalCount = clampInt(cfg.FertilizerBuyNormalCount, 0, 10000)
	cfg.FertilizerBuyNormalThresholdHours = clampInt(cfg.FertilizerBuyNormalThresholdHours, 0, 990)
	cfg.FertilizerBuyCheckIntervalMinutes = clampInt(cfg.FertilizerBuyCheckIntervalMinutes, 1, 1440)
	// 兜种回退策略不含 bag_priority；非法/缺省回退到默认（preferred，对齐 rust）。
	if cfg.BagSeedFallbackStrategy == StrategyBagPriority || cfg.BagSeedFallbackStrategy == "" {
		cfg.BagSeedFallbackStrategy = StrategyPreferred
	}
	// 好友申请过滤器。
	cfg.AutoAcceptFriendMinLevel = clampInt(cfg.AutoAcceptFriendMinLevel, 0, 200)
	cfg.AutoAcceptHarvestStealHarvest = clampInt(cfg.AutoAcceptHarvestStealHarvest, 0, 9999)
	cfg.AutoAcceptHarvestStealSteal = clampInt(cfg.AutoAcceptHarvestStealSteal, 1, 9999)
	// 肥料模式与土地类型。
	switch cfg.Automation.Fertilizer {
	case FertilizerBoth, FertilizerNormal, FertilizerOrganic, FertilizerSmart, FertilizerNone:
	default:
		cfg.Automation.Fertilizer = FertilizerSmart
	}
	cfg.Automation.FertilizerSmartSeconds = clampInt(cfg.Automation.FertilizerSmartSeconds, 30, 3600)
	cfg.Automation.FertilizerLandTypes = normalizeFertilizerLandTypes(cfg.Automation.FertilizerLandTypes)
	// 兜种土地类型：空/全类型等价不限制，统一省略。
	if cfg.BagSeedLandTypes == nil {
		cfg.BagSeedLandTypes = map[string][]string{}
	}
	for seedID, types := range cfg.BagSeedLandTypes {
		normalized := normalizeFertilizerLandTypes(types)
		if len(normalized) == 0 || len(normalized) == len(AllFertilizerLandTypes) {
			delete(cfg.BagSeedLandTypes, seedID)
			continue
		}
		cfg.BagSeedLandTypes[seedID] = normalized
	}
}

// normalizeTimeString mirrors bot normalizeTimeString (HH:MM).
func normalizeTimeString(v, fallback string) string {
	s := strings.TrimSpace(v)
	var hh, mm int
	if _, err := fmt.Sscanf(s, "%d:%d", &hh, &mm); err != nil {
		return fallback
	}
	if hh < 0 || hh > 23 || mm < 0 || mm > 59 {
		return fallback
	}
	return fmt.Sprintf("%02d:%02d", hh, mm)
}

// normalizeGIDList keeps unique positive ids.
func normalizeGIDList(values []int64) []int64 {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(values))
	out := make([]int64, 0, len(values))
	for _, v := range values {
		if v <= 0 {
			continue
		}
		if _, dup := seen[v]; dup {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

// normalizeFertilizerLandTypes keeps valid unique land types.
func normalizeFertilizerLandTypes(values []string) []string {
	valid := map[string]struct{}{}
	for _, t := range AllFertilizerLandTypes {
		valid[t] = struct{}{}
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.ToLower(strings.TrimSpace(v))
		if _, ok := valid[v]; !ok {
			continue
		}
		if _, dup := seen[v]; dup {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

// FriendIntervalBounds returns the unified friend-loop interval bounds:
// the tighter of the help/steal ranges (bot worker.ts friendMin/friendMax).
func (c AccountConfig) FriendIntervalBounds() (minSec, maxSec int) {
	minSec, maxSec = c.Intervals.FriendMin, c.Intervals.FriendMax
	if minSec <= 0 {
		minSec = minIntValue(c.Intervals.HelpMin, c.Intervals.StealMin)
	}
	if maxSec <= 0 {
		maxSec = minIntValue(c.Intervals.HelpMax, c.Intervals.StealMax)
	}
	if minSec <= 0 {
		minSec = 20
	}
	if maxSec < minSec {
		maxSec = minSec
	}
	return minSec, maxSec
}

func minIntValue(a, b int) int {
	if a <= 0 || (b > 0 && b < a) {
		if a <= 0 {
			return b
		}
		return b
	}
	return a
}
