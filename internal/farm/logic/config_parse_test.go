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
