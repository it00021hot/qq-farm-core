package logic

import (
	"encoding/json"
	"strings"
)

// Config defaults ported from qq-farm-bot/core/src/models/store/shared-state.ts
// and types/config.ts.

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
	Farm                    bool     `json:"farm"`
	FarmPush                bool     `json:"farm_push"`
	LandUpgrade             bool     `json:"land_upgrade"`
	Friend                  bool     `json:"friend"`
	FriendHelpExpLimit      bool     `json:"friend_help_exp_limit"`
	FriendSteal             bool     `json:"friend_steal"`
	FriendHelp              bool     `json:"friend_help"`
	FriendBad               bool     `json:"friend_bad"`
	Task                    bool     `json:"task"`
	FertilizerGift          bool     `json:"fertilizer_gift"`
	FertilizerBuyOrganic    bool     `json:"fertilizer_buy_organic"`
	FertilizerBuyNormal     bool     `json:"fertilizer_buy_normal"`
	Sell                    bool     `json:"sell"`
	Fertilizer              string   `json:"fertilizer"`
	FertilizerMultiSeason   bool     `json:"fertilizer_multi_season"`
	FertilizerLandTypes     []string `json:"fertilizer_land_types"`
	FertilizerSmartSeconds  int      `json:"fertilizer_smart_seconds"`
	SkipOwnWeedBug          bool     `json:"skip_own_weed_bug"`
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

// IntervalConfig mirrors types/config.ts IntervalConfig.
type IntervalConfig struct {
	Farm     int `json:"farm"`
	FarmMin  int `json:"farmMin"`
	FarmMax  int `json:"farmMax"`
	HelpMin  int `json:"helpMin"`
	HelpMax  int `json:"helpMax"`
	StealMin int `json:"stealMin"`
	StealMax int `json:"stealMax"`
}

// QuietHoursConfig mirrors types/config.ts QuietHoursConfig.
type QuietHoursConfig struct {
	Enabled bool   `json:"enabled"`
	Start   string `json:"start"`
	End     string `json:"end"`
}

// AccountConfig mirrors types/config.ts AccountConfig.
type AccountConfig struct {
	Automation                         AutomationConfig `json:"automation"`
	PlantingStrategy                   string           `json:"plantingStrategy"`
	PreferredSeedID                    int64            `json:"preferredSeedId"`
	Intervals                          IntervalConfig   `json:"intervals"`
	FriendQuietHours                   QuietHoursConfig `json:"friendQuietHours"`
	KnownFriendGids                    []int64          `json:"knownFriendGids"`
	KnownFriendGidSyncCooldownSec      int              `json:"knownFriendGidSyncCooldownSec"`
	FriendsListCacheTtlSec             int              `json:"friendsListCacheTtlSec"`
	FriendBlacklist                    []int64          `json:"friendBlacklist"`
	PlantBlacklist                     []int64          `json:"plantBlacklist"`
	StealDelaySeconds                  int              `json:"stealDelaySeconds"`
	PlantOrderRandom                   bool             `json:"plantOrderRandom"`
	PlantDelaySeconds                  int              `json:"plantDelaySeconds"`
	FertilizerBuyOrganicCount          int              `json:"fertilizerBuyOrganicCount"`
	FertilizerBuyOrganicThresholdHours int              `json:"fertilizerBuyOrganicThresholdHours"`
	FertilizerBuyNormalCount           int              `json:"fertilizerBuyNormalCount"`
	FertilizerBuyNormalThresholdHours  int              `json:"fertilizerBuyNormalThresholdHours"`
	FertilizerBuyCheckIntervalMinutes  int              `json:"fertilizerBuyCheckIntervalMinutes"`
	BagSeedPriority                    []int64          `json:"bagSeedPriority"`
	BagSeedFallbackStrategy            string           `json:"bagSeedFallbackStrategy"`
}

// DefaultAccountConfig returns a deep copy of DEFAULT_ACCOUNT_CONFIG.
func DefaultAccountConfig() AccountConfig {
	return AccountConfig{
		Automation: AutomationConfig{
			Farm:                    true,
			FarmPush:                true,
			LandUpgrade:             true,
			Friend:                  true,
			FriendHelpExpLimit:      false,
			FriendSteal:             true,
			FriendHelp:              false,
			FriendBad:               false,
			Task:                    true,
			FertilizerGift:          true,
			FertilizerBuyOrganic:    false,
			FertilizerBuyNormal:     false,
			Sell:                    true,
			Fertilizer:              FertilizerSmart,
			FertilizerMultiSeason:   true,
			FertilizerLandTypes:     append([]string(nil), AllFertilizerLandTypes...),
			FertilizerSmartSeconds:  360,
			SkipOwnWeedBug:          true,
		},
		PlantingStrategy: StrategyPreferred,
		PreferredSeedID:  0,
		Intervals: IntervalConfig{
			Farm: 2, FarmMin: 20, FarmMax: 25,
			HelpMin: 20, HelpMax: 25,
			StealMin: 10, StealMax: 15,
		},
		FriendQuietHours: QuietHoursConfig{
			Enabled: false,
			Start:   "01:00",
			End:     "07:30",
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
		BagSeedPriority:                    []int64{20329, 21037, 26032, 29003},
		BagSeedFallbackStrategy:            StrategyLevel,
	}
}

// ParseAccountConfigJSON loads config on top of defaults so omitted JSON keys keep bot defaults,
// while explicit false values are preserved.
func ParseAccountConfigJSON(raw string) AccountConfig {
	cfg := DefaultAccountConfig()
	if strings.TrimSpace(raw) == "" {
		return cfg
	}
	_ = json.Unmarshal([]byte(raw), &cfg)
	if cfg.PlantingStrategy == "" {
		cfg.PlantingStrategy = StrategyPreferred
	}
	if cfg.BagSeedFallbackStrategy == "" {
		cfg.BagSeedFallbackStrategy = StrategyLevel
	}
	if cfg.Automation.Fertilizer == "" {
		cfg.Automation.Fertilizer = FertilizerSmart
	}
	return cfg
}
