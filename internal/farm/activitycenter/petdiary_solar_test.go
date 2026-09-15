package activitycenter

import (
	"testing"

	"github.com/it00021hot/qq-farm-core/internal/farm/proto/activitypb"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/solartermspb"
)

func TestPetSolarTermClaimable(t *testing.T) {
	// 萌宠活动窗口 [100, 200]。
	head := &activitypb.PetDiaryActivityHead{StartTime: 100, EndTime: 200}
	cases := []struct {
		name string
		term *solartermspb.SolarTermInfo
		want bool
	}{
		{"overlap claimable", &solartermspb.SolarTermInfo{TermId: 7, BeginTime: 50, EndTime: 150, Status: 2}, true},
		{"window edge inclusive", &solartermspb.SolarTermInfo{TermId: 7, BeginTime: 100, EndTime: 200, Status: 2}, true},
		{"ends before window", &solartermspb.SolarTermInfo{TermId: 7, BeginTime: 10, EndTime: 99, Status: 2}, false},
		{"starts after window", &solartermspb.SolarTermInfo{TermId: 7, BeginTime: 201, EndTime: 300, Status: 2}, false},
		{"already claimed", &solartermspb.SolarTermInfo{TermId: 7, BeginTime: 50, EndTime: 150, Status: 3}, false},
		{"not yet claimable", &solartermspb.SolarTermInfo{TermId: 7, BeginTime: 50, EndTime: 150, Status: 1}, false},
		{"nil term", nil, false},
	}
	for _, c := range cases {
		if got := petSolarTermClaimable(c.term, head); got != c.want {
			t.Fatalf("%s: got %v want %v", c.name, got, c.want)
		}
	}
	if petSolarTermClaimable(&solartermspb.SolarTermInfo{Status: 2}, nil) {
		t.Fatal("nil head should not be claimable")
	}
}

func TestPetSolarClaimReplyMatches(t *testing.T) {
	if petSolarClaimReplyMatches(nil, 7) {
		t.Fatal("nil reply should not match")
	}
	if petSolarClaimReplyMatches(&solartermspb.ClaimSolarTermsReply{Term: &solartermspb.SolarTermInfo{TermId: 7, Status: 2}}, 7) {
		t.Fatal("status 2 echo should not match")
	}
	if petSolarClaimReplyMatches(&solartermspb.ClaimSolarTermsReply{Term: &solartermspb.SolarTermInfo{TermId: 8, Status: 3}}, 7) {
		t.Fatal("term id mismatch should not match")
	}
	if !petSolarClaimReplyMatches(&solartermspb.ClaimSolarTermsReply{Term: &solartermspb.SolarTermInfo{TermId: 7, Status: 3}}, 7) {
		t.Fatal("status 3 same id echo should match")
	}
}
