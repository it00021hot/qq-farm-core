package logic

import "testing"

func TestParseAccountConfigJSONPreservesActivityOnlyFalse(t *testing.T) {
	raw := `{"plantingStrategy":"preferred","automation":{"friend_steal":true,"friend_steal_activity_only":false,"friend":true}}`
	cfg := ParseAccountConfigJSON(raw)
	if cfg.Automation.FriendStealActivityOnly {
		t.Fatalf("expected friend_steal_activity_only=false, got true")
	}
	if !cfg.Automation.FriendSteal {
		t.Fatalf("expected friend_steal=true")
	}
	if cfg.PlantingStrategy != StrategyPreferred {
		t.Fatalf("plantingStrategy=%q", cfg.PlantingStrategy)
	}
}

func TestParseAccountConfigJSONDefaultsActivityOnly(t *testing.T) {
	cfg := ParseAccountConfigJSON(`{}`)
	if !cfg.Automation.FriendStealActivityOnly {
		t.Fatalf("default should keep friend_steal_activity_only=true")
	}
}
