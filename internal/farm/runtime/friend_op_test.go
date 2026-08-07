package runtime

import (
	"context"
	"fmt"
	"testing"

	"github.com/MQEnergy/go-skeleton/internal/farm/logic"
	"github.com/MQEnergy/go-skeleton/internal/farm/proto/corepb"
	"github.com/MQEnergy/go-skeleton/internal/farm/proto/friendpb"
)

func TestSummarizeHarvestRewards(t *testing.T) {
	score, value := summarizeHarvestRewards([]*corepb.Item{
		{Id: 1022, Count: 5},
		{Id: 1019, Count: 3},
	})
	if score != 8 {
		t.Fatalf("score=%d want 8", score)
	}
	_ = value
}

func TestFormatStealSummaryWithScoreValue(t *testing.T) {
	got := formatStealSummary(3, []string{"月草", "艾草"}, 12, 90)
	want := "偷3(月草/艾草)+积分x12，价值90金"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestIsTransientNetworkError(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{nil, false},
		{fmt.Errorf("protocol: client closed"), true},
		{fmt.Errorf("protocol: connection closed"), true},
		{fmt.Errorf("protocol: not connected"), true},
		{context.Canceled, true},
		{fmt.Errorf("连接关闭"), true},
		{fmt.Errorf("code=1002003 banned"), false},
		{fmt.Errorf("rpc code=12345 invalid friend"), false},
	}
	for _, tc := range cases {
		if got := isTransientNetworkError(tc.err); got != tc.want {
			t.Fatalf("isTransientNetworkError(%v)=%v want %v", tc.err, got, tc.want)
		}
	}
	if friendlyNetworkError(fmt.Errorf("protocol: client closed")) != "连接关闭" {
		t.Fatal("friendlyNetworkError")
	}
}

func TestGetPatrolBatchSize(t *testing.T) {
	cases := []struct {
		n, want int
	}{
		{0, 0},
		{1, 1},
		{3, 1},
		{4, 1},
		{5, 2},
		{8, 2},
		{9, 3},
	}
	for _, tc := range cases {
		if got := getPatrolBatchSize(tc.n); got != tc.want {
			t.Fatalf("getPatrolBatchSize(%d)=%d want %d", tc.n, got, tc.want)
		}
	}
}

func TestSelectUnvisitedPatrolResetsWhenExhausted(t *testing.T) {
	visited := map[int64]struct{}{1: {}, 2: {}, 3: {}}
	candidates := []int64{1, 2, 3, 4}
	got := selectUnvisitedPatrol(candidates, 2, visited)
	if len(got) != 1 || got[0] != 4 {
		t.Fatalf("expected remaining unvisited [4], got %v", got)
	}

	visited = map[int64]struct{}{1: {}, 2: {}, 3: {}, 4: {}}
	got = selectUnvisitedPatrol(candidates, 2, visited)
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("expected reset restart [1 2], got %v visited=%v", got, visited)
	}
	if len(visited) != 0 {
		t.Fatalf("visited should be cleared on reset, got %v", visited)
	}
}

func TestBuildStealPatrolTargetsBubbleThenProbe(t *testing.T) {
	friends := []friendpb.GameFriend{
		{Gid: 10, Plant: &friendpb.Plant{StealPlantNum: 2}},
		{Gid: 11, Plant: &friendpb.Plant{StealPlantNum: 5}},
		{Gid: 12, Plant: &friendpb.Plant{}},
		{Gid: 13, Plant: &friendpb.Plant{}},
		{Gid: 14, Plant: &friendpb.Plant{}},
		{Gid: 15, Plant: &friendpb.Plant{}},
		{Gid: 99}, // self skipped
	}
	visited := map[int64]struct{}{}
	targets := buildStealPatrolTargets(friends, 99, nil, visited)
	// pool=6 → ceil(6/4)=2 probes; bubble sorted 11 then 10
	if len(targets) < 3 || targets[0] != 11 || targets[1] != 10 {
		t.Fatalf("bubble order wrong: %v", targets)
	}
	if len(targets) != 4 {
		t.Fatalf("want 2 bubble + 2 probe, got %v", targets)
	}
}

func TestBuildBadFriendTargetsTopByLevel(t *testing.T) {
	friends := make([]friendpb.GameFriend, 0, 25)
	for i := 1; i <= 25; i++ {
		friends = append(friends, friendpb.GameFriend{
			Gid: int64(i), Level: int64(i),
			Plant: &friendpb.Plant{},
		})
	}
	// friend with steal bubble should be excluded
	friends = append(friends, friendpb.GameFriend{
		Gid: 100, Level: 999, Plant: &friendpb.Plant{StealPlantNum: 1},
	})
	got := buildBadFriendTargets(friends, 0, nil, 20)
	if len(got) != 20 {
		t.Fatalf("want 20, got %d %v", len(got), got)
	}
	if got[0] != 25 || got[19] != 6 {
		t.Fatalf("expected levels 25..6, got %v", got)
	}
	for _, gid := range got {
		if gid == 100 {
			t.Fatal("steal-bubble friend must be excluded")
		}
	}
}

func TestCollectBadLandTargetsOwnersLimit(t *testing.T) {
	myGID := int64(7)
	lands := []logic.LandInfo{
		{
			ID: 1, Unlocked: true,
			Plant: &logic.PlantInfo{
				Phases:       []logic.PlantPhaseInfo{{Phase: logic.PhaseSmallLeaves}},
				WeedOwners:   []int64{1},
				InsectOwners: []int64{1, 2},
			},
		},
		{
			ID: 2, Unlocked: true,
			Plant: &logic.PlantInfo{
				Phases:     []logic.PlantPhaseInfo{{Phase: logic.PhaseBlooming}},
				WeedOwners: []int64{myGID},
			},
		},
		{
			ID: 3, Unlocked: true,
			Plant: &logic.PlantInfo{
				Phases: []logic.PlantPhaseInfo{{Phase: logic.PhaseMature}},
			},
		},
	}
	weed, bug := collectBadLandTargets(lands, myGID)
	if len(weed) != 1 || weed[0] != 1 {
		t.Fatalf("weed targets=%v", weed)
	}
	// land1 insects full; land2 insects empty → can put bug on 2; land3 mature skipped
	if len(bug) != 1 || bug[0] != 2 {
		t.Fatalf("bug targets=%v", bug)
	}
	// land 2: weed already by me → no weed
	if weeds, _ := collectBadLandTargets([]logic.LandInfo{lands[1]}, myGID); len(weeds) != 0 {
		t.Fatalf("expected no weed on land 2, got %v", weeds)
	}
}

func TestIsActivityPlant(t *testing.T) {
	if logic.IsActivityPlant(nil) {
		t.Fatal("nil plant")
	}
	if logic.IsActivityPlant(&logic.PlantInfo{}) {
		t.Fatal("no activity")
	}
	if !logic.IsActivityPlant(&logic.PlantInfo{Activity: &logic.PlantActivityInfo{Param1: 10}}) {
		t.Fatal("param1 score")
	}
	if !logic.IsActivityPlant(&logic.PlantInfo{Activity: &logic.PlantActivityInfo{Param2: 5}}) {
		t.Fatal("param2 fallback")
	}
	if logic.IsActivityPlant(&logic.PlantInfo{Activity: &logic.PlantActivityInfo{ActivityID: 1019}}) {
		t.Fatal("id alone is not activity")
	}
}
