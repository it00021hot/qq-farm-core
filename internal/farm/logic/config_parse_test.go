package logic

import "testing"

func TestParseAccountConfigJSONPreservesFriendStealFalse(t *testing.T) {
	raw := `{"plantingStrategy":"preferred","automation":{"friend_steal":false,"friend":true}}`
	cfg := ParseAccountConfigJSON(raw)
	if cfg.Automation.FriendSteal {
		t.Fatalf("expected friend_steal=false, got true")
	}
	if !cfg.Automation.Friend {
		t.Fatalf("expected friend=true")
	}
	if cfg.PlantingStrategy != StrategyPreferred {
		t.Fatalf("plantingStrategy=%q", cfg.PlantingStrategy)
	}
}

func TestParseAccountConfigJSONDefaultsFriendSteal(t *testing.T) {
	cfg := ParseAccountConfigJSON(`{}`)
	if !cfg.Automation.FriendSteal {
		t.Fatalf("default should keep friend_steal=true")
	}
}

func TestParseAccountConfigJSONNormalizesAutoAccept(t *testing.T) {
	raw := `{
		"autoAcceptFriendMinLevel": 999,
		"autoAcceptRequireOwnLevel": true,
		"autoAcceptHarvestStealEnabled": false,
		"autoAcceptHarvestStealHarvest": -3,
		"autoAcceptHarvestStealSteal": 0
	}`
	cfg := ParseAccountConfigJSON(raw)
	if cfg.AutoAcceptFriendMinLevel != 200 {
		t.Fatalf("autoAcceptFriendMinLevel=%d, want clamped 200", cfg.AutoAcceptFriendMinLevel)
	}
	if !cfg.AutoAcceptRequireOwnLevel {
		t.Fatalf("autoAcceptRequireOwnLevel should be preserved true")
	}
	if cfg.AutoAcceptHarvestStealEnabled {
		t.Fatalf("autoAcceptHarvestStealEnabled should be preserved false")
	}
	if cfg.AutoAcceptHarvestStealHarvest != 0 {
		t.Fatalf("autoAcceptHarvestStealHarvest=%d, want clamped 0", cfg.AutoAcceptHarvestStealHarvest)
	}
	if cfg.AutoAcceptHarvestStealSteal != 1 {
		t.Fatalf("autoAcceptHarvestStealSteal=%d, want clamped 1", cfg.AutoAcceptHarvestStealSteal)
	}
}

func TestParseAccountConfigJSONNormalizesBagSeedLandTypes(t *testing.T) {
	raw := `{
		"bagSeedLandTypes": {
			"29003": ["gold", "black"],
			"20129": ["normal", "invalid", "normal"],
			"21380": ["purple-gold", "gold", "black", "red", "normal"]
		}
	}`
	cfg := ParseAccountConfigJSON(raw)
	if got := cfg.BagSeedLandTypes["29003"]; len(got) != 2 {
		t.Fatalf("29003 types=%v, want gold+black preserved", got)
	}
	// 重复与非法类型被过滤。
	if got := cfg.BagSeedLandTypes["20129"]; len(got) != 1 || got[0] != "normal" {
		t.Fatalf("20129 types=%v, want [normal]", got)
	}
	// 勾满全部 5 类等价不限制，统一省略该 key。
	if _, ok := cfg.BagSeedLandTypes["21380"]; ok {
		t.Fatalf("21380 full-type entry should be dropped")
	}
}
