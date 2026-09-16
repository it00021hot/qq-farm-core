package activitycenter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/it00021hot/qq-farm-core/internal/vars"
)

// Regression: the catalog used to load from the CWD-relative resource/farm
// path, which broke the installed desktop shell (unrelated working directory).
func TestLoadPetDiaryCatalogIgnoresWorkingDirectory(t *testing.T) {
	savedBase, savedData, savedAssets := vars.BasePath, petDiaryData, petDiaryAssetMap
	t.Cleanup(func() {
		vars.BasePath = savedBase
		petDiaryData, petDiaryAssetMap = savedData, savedAssets
	})

	root := t.TempDir()
	dir := filepath.Join(root, "resource", "farm", "activity-data")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	catalog := `{"ActivityPetTreasureHuntBase":[{"daily_feed_limit":3}]}`
	if err := os.WriteFile(filepath.Join(dir, "pet-diary-2026090101.json"), []byte(catalog), 0o644); err != nil {
		t.Fatal(err)
	}
	vars.BasePath = root

	// Unrelated working directory — a CWD-relative lookup cannot find the file.
	t.Chdir(t.TempDir())

	petDiaryData, petDiaryAssetMap = nil, nil
	data, _, err := loadPetDiaryCatalog()
	if err != nil {
		t.Fatalf("loadPetDiaryCatalog: %v", err)
	}
	if len(data.ActivityPetTreasureHuntBase) != 1 || data.ActivityPetTreasureHuntBase[0].DailyFeedLimit != 3 {
		t.Fatalf("unexpected catalog: %+v", data)
	}
}
