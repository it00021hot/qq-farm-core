package activitycenter

import (
	"testing"

	"github.com/MQEnergy/go-skeleton/internal/farm/proto/seasonpb"
)

func TestLoadConstellationCatalog(t *testing.T) {
	c := LoadConstellationCatalog()
	if c == nil {
		t.Fatal("catalog nil")
	}
	if c.ActivityID != "2026072701" {
		t.Fatalf("activityId=%s", c.ActivityID)
	}
	if len(c.Groups) == 0 {
		t.Fatal("empty groups")
	}
	if c.Groups[0].NodeID.String() != "1" {
		t.Fatalf("nodeId=%q", c.Groups[0].NodeID.String())
	}
	if c.Groups[0].Name == "" {
		t.Fatal("empty name")
	}
}

func TestBuildConstellationUsesCatalog(t *testing.T) {
	act := &seasonpb.SeasonActivity{
		ActivityId: 2026072701,
		BeginTime:  1785290400,
		EndTime:    1787846399,
		Name:       []byte("千星同明"),
		Type:       13,
	}
	out := buildConstellation(act, 1786091400, nil, nil)
	if out["catalogStatus"] != "supported" {
		t.Fatalf("status=%v", out["catalogStatus"])
	}
	groups, _ := out["groups"].([]map[string]any)
	if len(groups) == 0 {
		t.Fatal("empty groups")
	}
	if groups[0]["name"] != "角宿" {
		t.Fatalf("name=%v", groups[0]["name"])
	}
	rewards, _ := groups[0]["rewards"].([]map[string]any)
	if len(rewards) == 0 {
		t.Fatal("empty rewards")
	}
	if rewards[0]["id"] == nil || rewards[0]["id"] == "0" {
		t.Fatalf("reward id=%v", rewards[0]["id"])
	}
}
