package game

import (
	"testing"

	"github.com/it00021hot/qq-farm-core/internal/farm/proto/plantpb"
	"google.golang.org/protobuf/proto"
)

func TestHarvestRequestMarshalNonEmpty(t *testing.T) {
	body, err := proto.Marshal(&plantpb.HarvestRequest{
		LandIds: []int64{1, 2, 3},
		HostGid: 12345,
		IsAll:   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(body) == 0 {
		t.Fatal("expected non-empty HarvestRequest body")
	}

	var got plantpb.HarvestRequest
	if err := proto.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.HostGid != 12345 || !got.IsAll || len(got.LandIds) != 3 {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}
