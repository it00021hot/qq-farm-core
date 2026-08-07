package runtime

import (
	"testing"
	"time"
)

func TestRemainSecFromDurationCeilLikeBot(t *testing.T) {
	t.Parallel()

	cases := []struct {
		d    time.Duration
		want int
	}{
		{d: 0, want: 0},
		{d: -time.Second, want: 0},
		{d: time.Millisecond, want: 1},
		{d: 999 * time.Millisecond, want: 1},
		{d: time.Second, want: 1},
		{d: 1001 * time.Millisecond, want: 2},
		{d: 1500 * time.Millisecond, want: 2},
		{d: 10 * time.Second, want: 10},
	}
	for _, tc := range cases {
		if got := remainSecFromDuration(tc.d); got != tc.want {
			t.Fatalf("remainSecFromDuration(%v)=%d want %d", tc.d, got, tc.want)
		}
	}
	if remainSecUntil(time.Time{}) != 0 {
		t.Fatal("zero time → 0")
	}
}

func TestNextChecksSnapshotFields(t *testing.T) {
	s := &Session{
		id:          "1",
		status:      StatusRunning,
		nextFarmAt:  time.Now().Add(10 * time.Second),
		nextHelpAt:  time.Now().Add(3 * time.Second),
		nextStealAt: time.Now().Add(7 * time.Second),
	}
	snap := s.Snapshot()
	nc := snap.NextChecks
	if nc.FarmRemainSec < 9 || nc.FarmRemainSec > 10 {
		t.Fatalf("farmRemainSec=%d", nc.FarmRemainSec)
	}
	if nc.HelpRemainSec < 2 || nc.HelpRemainSec > 3 {
		t.Fatalf("helpRemainSec=%d", nc.HelpRemainSec)
	}
	if nc.StealRemainSec < 6 || nc.StealRemainSec > 7 {
		t.Fatalf("stealRemainSec=%d", nc.StealRemainSec)
	}
	wantFriend := nc.HelpRemainSec
	if nc.StealRemainSec > wantFriend {
		wantFriend = nc.StealRemainSec
	}
	if nc.FriendRemainSec != wantFriend {
		t.Fatalf("friendRemainSec=%d want max(help,steal)=%d", nc.FriendRemainSec, wantFriend)
	}
}
