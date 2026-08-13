package activitycenter

import (
	"testing"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/activitypb"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/seasonpb"
)

func TestApplySeasonPassNotifyPreservesNodes(t *testing.T) {
	liveTravelPassMu.Lock()
	liveTravelPass = nil
	liveTravelPassMu.Unlock()

	full := &seasonpb.SeasonPass{
		ActivityId:          100,
		CurrentLevel:        2,
		CurrentProgress:     3,
		ProgressTarget:      10,
		ClaimedThroughLevel: 1,
		Title:               []byte("游记"),
		Nodes: []*seasonpb.SeasonRewardNode{
			{NodeId: 1, IsKeyLevel: false},
			{NodeId: 2, IsKeyLevel: true},
			{NodeId: 3, IsKeyLevel: false},
		},
	}
	first := ApplySeasonPassNotify(full)
	if first == nil {
		t.Fatal("expected pass dto")
	}
	nodes, _ := first["nodes"].([]map[string]any)
	if len(nodes) != 3 {
		t.Fatalf("nodes=%d want 3", len(nodes))
	}
	if first["claimableCount"] != 1 {
		t.Fatalf("claimableCount=%v want 1", first["claimableCount"])
	}

	push := &seasonpb.SeasonPass{
		ActivityId:          0,
		CurrentLevel:        3,
		CurrentProgress:     1,
		ProgressTarget:      10,
		ClaimedThroughLevel: 1,
		Title:               nil,
		Nodes:               nil,
	}
	next := ApplySeasonPassNotify(push)
	if next["title"] != "游记" {
		t.Fatalf("title=%v want 游记", next["title"])
	}
	if next["activityId"] != "100" {
		t.Fatalf("activityId=%v want 100", next["activityId"])
	}
	nodes, _ = next["nodes"].([]map[string]any)
	if len(nodes) != 3 {
		t.Fatalf("preserved nodes=%d want 3", len(nodes))
	}
	if next["claimableCount"] != 2 {
		t.Fatalf("claimableCount after push=%v want 2", next["claimableCount"])
	}
}

func TestRememberConstellationNodesMonotonic(t *testing.T) {
	activityID := int64(9001)
	RememberConstellationNodes(activityID, &activitypb.ConstellationData{
		Nodes: []*activitypb.ConstellationNode{
			{NodeId: 1, Field_2: true, Field_3: false},
		},
	})
	RememberConstellationNodes(activityID, &activitypb.ConstellationData{
		Nodes: []*activitypb.ConstellationNode{
			{NodeId: 2, Field_2: true, Field_3: true},
		},
	})
	v, ok := lastConstellationConfirmed.Load("9001")
	if !ok {
		t.Fatal("expected confirmed cache")
	}
	confirmed := v.(*constellationConfirmed)
	if len(confirmed.Opened) < 2 {
		t.Fatalf("opened=%v", confirmed.Opened)
	}
	if len(confirmed.Lit) < 1 {
		t.Fatalf("lit=%v", confirmed.Lit)
	}
}

func TestBuildActionsClaimPass(t *testing.T) {
	actions := buildActions(
		map[string]any{
			"pass": map[string]any{
				"claimableCount": 2,
				"nodes": []map[string]any{
					{"claimable": true},
					{"claimable": true},
				},
			},
			"serverTime": "100",
		},
		map[string]any{},
		map[string]any{},
		map[string]any{},
		map[string]any{},
	)
	claim := actions["claimPass"]
	if claim["enabled"] != true {
		t.Fatalf("enabled=%v", claim["enabled"])
	}
	if claim["available"] != true {
		t.Fatalf("available=%v", claim["available"])
	}
	if claim["count"] != 2 {
		t.Fatalf("count=%v", claim["count"])
	}
}

func TestStateJSONConstellationPersistsNodes(t *testing.T) {
	raw := StateJSONFromSnapshot("constellation", map[string]any{
		"currentDay":             3,
		"activityId":             "9001",
		"confirmedOpenedNodeIds": []string{"1", "2"},
		"confirmedLitNodeIds":    []string{"2"},
	})
	if raw == "" {
		t.Fatal("empty json")
	}
	HydrateConstellationConfirmedFromStateJSON("constellation", raw)
	v, ok := lastConstellationConfirmed.Load("9001")
	if !ok {
		t.Fatal("hydrate missed activity id")
	}
	confirmed := v.(*constellationConfirmed)
	if len(confirmed.Lit) == 0 {
		t.Fatalf("lit empty after hydrate: %+v", confirmed)
	}
}

func TestBuildGreenPlum(t *testing.T) {
	logic.ResetActivityRegistry()

	// Unknown activity -> known=false, inactive, no activityId yet.
	gp := buildGreenPlum(nil, nil)
	if gp["known"] != false || gp["active"] != false {
		t.Fatalf("expected unknown/inactive: %+v", gp)
	}
	if _, ok := gp["activityId"]; ok {
		t.Fatalf("unexpected activityId for unknown activity: %+v", gp)
	}

	// Known + active (matched by type code 12, not by hard-coded id).
	logic.RegisterActivity(logic.ActivityRegistryItem{
		ActivityID: "2026081202",
		Type:       12,
		Name:       "青酿换万金",
		BeginTime:  time.Now().Unix() - 3600,
		EndTime:    time.Now().Unix() + 86400,
	})
	gp = buildGreenPlum(nil, nil)
	if gp["known"] != true || gp["active"] != true {
		t.Fatalf("expected known/active: %+v", gp)
	}
	if gp["activityId"] != "2026081202" {
		t.Fatalf("activityId=%v", gp["activityId"])
	}
	if gp["typeCode"] != int64(12) {
		t.Fatalf("typeCode=%v", gp["typeCode"])
	}

	// Claim action reflects activity state.
	actions := buildActions(map[string]any{}, map[string]any{}, map[string]any{}, map[string]any{}, gp)
	claim, ok := actions["claimGreenPlum"]
	if !ok {
		t.Fatal("missing claimGreenPlum action")
	}
	if claim["enabled"] != true || claim["available"] != true {
		t.Fatalf("claim action should be enabled: %+v", claim)
	}
}
