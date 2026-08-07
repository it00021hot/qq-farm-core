package taskpb_test

import (
	"bytes"
	"testing"

	"github.com/MQEnergy/go-skeleton/internal/farm/proto/taskpb"
)

func TestTaskInfoRequestRoundTrip(t *testing.T) {
	orig := &taskpb.TaskInfoRequest{}
	raw := orig.Marshal()
	var back taskpb.TaskInfoRequest
	if err := back.Unmarshal(raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(back.Marshal()) != 0 {
		t.Fatalf("TaskInfoRequest should marshal to empty")
	}
}

func TestClaimTaskRewardRequestRoundTrip(t *testing.T) {
	orig := &taskpb.ClaimTaskRewardRequest{
		ID:       12345,
		DoShared: true,
	}
	raw := orig.Marshal()
	var back taskpb.ClaimTaskRewardRequest
	if err := back.Unmarshal(raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	raw2 := back.Marshal()
	if !bytes.Equal(raw, raw2) {
		t.Fatalf("round-trip bytes differ:\n  orig=% x\n  back=% x", raw, raw2)
	}
	if back.ID != 12345 || !back.DoShared {
		t.Fatalf("decoded=%+v", back)
	}
}

func TestClaimDailyRewardRequestRoundTrip(t *testing.T) {
	orig := &taskpb.ClaimDailyRewardRequest{
		Type:     1,
		PointIDs: []int64{10, 20, 30},
	}
	raw := orig.Marshal()
	var back taskpb.ClaimDailyRewardRequest
	if err := back.Unmarshal(raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.Type != 1 || len(back.PointIDs) != 3 {
		t.Fatalf("decoded=%+v", back)
	}
}
