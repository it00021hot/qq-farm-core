package game

import (
	"testing"

	"github.com/MQEnergy/go-skeleton/internal/farm/proto/plantpb"
)

func TestHarvestRequestMarshalNonEmpty(t *testing.T) {
	body := (&plantpb.HarvestRequest{
		LandIDs: []int64{1, 2, 3},
		HostGID: 12345,
		IsAll:   true,
	}).Marshal()
	if len(body) == 0 {
		t.Fatal("expected non-empty HarvestRequest body")
	}

	var got plantpb.HarvestRequest
	if err := got.Unmarshal(body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.HostGID != 12345 || !got.IsAll || len(got.LandIDs) != 3 {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}
