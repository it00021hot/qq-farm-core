package runtime

import (
	"testing"
	"time"

	"github.com/MQEnergy/go-skeleton/internal/farm/proto/taskpb"
)

func TestGrowthTaskRowFromPB(t *testing.T) {
	row := growthTaskRowFromPB(&taskpb.Task{
		Id: 7, Desc: "农场提升至 54 级", Progress: 53, TotalProgress: 54, IsUnlocked: true,
	})
	if row.ID != 7 || row.IsCompleted || row.Desc == "" {
		t.Fatalf("row=%+v", row)
	}
	done := growthTaskRowFromPB(&taskpb.Task{Id: 1, Progress: 3, TotalProgress: 3})
	if !done.IsCompleted {
		t.Fatal("expected completed")
	}
}

func TestUnixMsZero(t *testing.T) {
	var zero time.Time
	if unixMs(zero) != 0 {
		t.Fatal("zero time")
	}
}
