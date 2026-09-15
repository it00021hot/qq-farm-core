package runtime

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
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
	if nc.FarmQuiet || nc.HelpQuiet || nc.StealQuiet {
		t.Fatalf("quiet flags should default false: %+v", nc)
	}
}

func TestNextChecksQuietFlagsFollowConfig(t *testing.T) {
	// 静默窗口覆盖整天（start==end → 全天生效），用配置驱动三种标记。
	s := &Session{
		id:     "1",
		status: StatusRunning,
		cfg: SessionConfig{AccountConfig: logic.AccountConfig{
			FriendQuietHours: logic.QuietHoursConfig{Enabled: true, Start: "00:00", End: "00:00", ContinueFarm: true},
		}},
	}
	nc := s.Snapshot().NextChecks
	if !nc.HelpQuiet || !nc.StealQuiet {
		t.Fatalf("friend quiet should mark help/steal quiet: %+v", nc)
	}
	if nc.FarmQuiet {
		t.Fatalf("continueFarm=true keeps farm running: %+v", nc)
	}

	s.cfg.AccountConfig.FriendQuietHours.ContinueFarm = false
	nc = s.Snapshot().NextChecks
	if !nc.FarmQuiet {
		t.Fatalf("continueFarm=false should mark farm quiet: %+v", nc)
	}

	s.cfg.AccountConfig.FriendQuietHours.Enabled = false
	nc = s.Snapshot().NextChecks
	if nc.FarmQuiet || nc.HelpQuiet || nc.StealQuiet {
		t.Fatalf("disabled quiet hours should clear flags: %+v", nc)
	}
}

func TestNextChecksQuietJSONNames(t *testing.T) {
	raw, err := json.Marshal(NextChecksSnapshot{FarmQuiet: true, HelpQuiet: true, StealQuiet: true})
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"farmQuiet", "helpQuiet", "stealQuiet"} {
		if v, ok := decoded[key].(bool); !ok || !v {
			t.Fatalf("missing/invalid %s in %s", key, raw)
		}
	}
}
