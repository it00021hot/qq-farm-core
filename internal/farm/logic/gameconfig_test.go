package logic_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
)

func TestLoadGameConfig(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	dir := filepath.Join(root, "resource", "farm", "gameConfig")
	if err := logic.LoadGameConfig(dir); err != nil {
		t.Fatal(err)
	}
	land := logic.GetLandConfigByID(1)
	if land == nil {
		t.Fatal("land 1 missing")
	}
	if land.GridX != 0 || land.GridY != 5 {
		t.Fatalf("land1 grid=%d,%d", land.GridX, land.GridY)
	}
	seeds := logic.GetAllSeeds()
	if len(seeds) == 0 {
		t.Fatal("expected seeds from Plant.json + ItemInfo.json")
	}
	found := false
	for _, s := range seeds {
		if s.SeedID == 29999 {
			found = true
			if s.Name == "" {
				t.Fatal("seed 29999 name empty")
			}
			if s.Image == "" {
				t.Fatal("seed 29999 image empty")
			}
			break
		}
	}
	if !found {
		t.Fatal("seed 29999 missing")
	}
	if len(logic.GetAllFruits()) == 0 {
		t.Fatal("expected fruits")
	}
	if len(logic.GetAllItems(0)) == 0 {
		t.Fatal("expected items")
	}
	if len(logic.GetAllPlants()) == 0 {
		t.Fatal("expected plants")
	}
	if logic.GetItemByID(29999) == nil || logic.GetItemByID(29999).Type != 5 {
		t.Fatal("item 29999 should be seed type=5")
	}
}
