package automation

import (
	"encoding/json"
	"testing"

	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
)

// TestDetailMapExposesConfigFields 校验 detail 回显包含 runtime 已生效但
// 此前 API 面缺失的字段（兜种土地类型、好友申请自动接受过滤、运行时管理状态）。
func TestDetailMapExposesConfigFields(t *testing.T) {
	cfg := logic.DefaultAccountConfig()
	cfg.BagSeedLandTypes = map[string][]string{"29003": {"gold", "black"}}
	cfg.AutoAcceptFriendMinLevel = 30
	cfg.AutoAcceptRequireOwnLevel = true
	cfg.AutoAcceptHarvestStealEnabled = false
	cfg.AutoAcceptHarvestStealHarvest = 8
	cfg.AutoAcceptHarvestStealSteal = 1
	cfg.KnownFriendGids = []int64{10001, 10002}
	cfg.KnownFriendGidSyncCooldownSec = 300
	cfg.FriendsListCacheTtlSec = 60

	m := detailMap(42, cfg, mustJSON(cfg))
	want := map[string]any{
		"bagSeedLandTypes":              cfg.BagSeedLandTypes,
		"autoAcceptFriendMinLevel":      cfg.AutoAcceptFriendMinLevel,
		"autoAcceptRequireOwnLevel":     cfg.AutoAcceptRequireOwnLevel,
		"autoAcceptHarvestStealEnabled": cfg.AutoAcceptHarvestStealEnabled,
		"autoAcceptHarvestStealHarvest": cfg.AutoAcceptHarvestStealHarvest,
		"autoAcceptHarvestStealSteal":   cfg.AutoAcceptHarvestStealSteal,
		"knownFriendGids":               cfg.KnownFriendGids,
		"knownFriendGidSyncCooldownSec": cfg.KnownFriendGidSyncCooldownSec,
		"friendsListCacheTtlSec":        cfg.FriendsListCacheTtlSec,
	}
	for k, exp := range want {
		got, ok := m[k]
		if !ok {
			t.Fatalf("detailMap missing key %q", k)
		}
		// 统一序列化比对（map/slice 值不可直接 ==）。
		a, _ := json.Marshal(got)
		b, _ := json.Marshal(exp)
		if string(a) != string(b) {
			t.Fatalf("%s = %s, want %s", k, a, b)
		}
	}
	// configJson 序列化后也应包含这些键（供旧客户端整包读写）。
	var raw map[string]any
	if err := json.Unmarshal([]byte(m["configJson"].(string)), &raw); err != nil {
		t.Fatalf("configJson invalid: %v", err)
	}
	for k := range want {
		if _, ok := raw[k]; !ok {
			t.Fatalf("configJson missing key %q", k)
		}
	}
}

// TestModifyNormalizeRoundTrip 校验 Modify 的归一化路径：合并请求后统一走
// ParseAccountConfigJSON（Load+Normalize），落库与运行时拿到 clamp 后的值。
func TestModifyNormalizeRoundTrip(t *testing.T) {
	cfg := logic.DefaultAccountConfig()
	cfg.AutoAcceptFriendMinLevel = 9999
	cfg.AutoAcceptHarvestStealSteal = 0
	cfg.BagSeedLandTypes = map[string][]string{"29003": {"gold", "black", "red", "normal", "purple-gold"}}
	cfg = logic.ParseAccountConfigJSON(mustJSON(cfg))
	if cfg.AutoAcceptFriendMinLevel != 200 {
		t.Fatalf("autoAcceptFriendMinLevel=%d, want clamped 200", cfg.AutoAcceptFriendMinLevel)
	}
	if cfg.AutoAcceptHarvestStealSteal != 1 {
		t.Fatalf("autoAcceptHarvestStealSteal=%d, want clamped 1", cfg.AutoAcceptHarvestStealSteal)
	}
	if _, ok := cfg.BagSeedLandTypes["29003"]; ok {
		t.Fatalf("full-type bagSeedLandTypes entry should be dropped")
	}
}
