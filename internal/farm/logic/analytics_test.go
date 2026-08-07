package logic_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
)

func TestGetPlantRankings(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	dir := filepath.Join(root, "resource", "farm", "gameConfig")
	if err := logic.LoadGameConfig(dir); err != nil {
		t.Skipf("game config unavailable: %v", err)
	}

	for _, sortBy := range []string{"exp", "fert_exp", "profit", "fert_profit"} {
		rankings := logic.GetPlantRankings(sortBy)
		if len(rankings) == 0 {
			t.Fatalf("expected rankings for sort=%s", sortBy)
		}
		if rankings[0].SeedID <= 0 || rankings[0].Name == "" {
			t.Fatalf("invalid top ranking for sort=%s: %+v", sortBy, rankings[0])
		}
	}
}
