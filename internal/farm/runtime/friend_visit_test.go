package runtime

import (
	"testing"

	"github.com/it00021hot/qq-farm-core/internal/farm/proto/friendpb"
)

func friendListFixture() []friendpb.GameFriend {
	mk := func(gid int64, name string, level int64, steal, dry, weed, insect int64) friendpb.GameFriend {
		return friendpb.GameFriend{
			Gid: gid, Name: name, Level: level,
			Plant: &friendpb.Plant{
				StealPlantNum: steal, DryNum: dry, WeedNum: weed, InsectNum: insect,
			},
		}
	}
	return []friendpb.GameFriend{
		mk(101, "steal-only", 10, 3, 0, 0, 0),
		mk(102, "help-only", 20, 0, 1, 2, 0),
		mk(103, "steal-and-help", 30, 2, 0, 0, 1),
		mk(104, "idle-high-level", 99, 0, 0, 0, 0),
		mk(105, "idle-low-level", 1, 0, 0, 0, 0),
		{Gid: 106, Name: "no-plant", Level: 5},
		mk(107, "dup", 5, 1, 0, 0, 0),
		mk(107, "dup", 5, 1, 0, 0, 0),
		// self and blacklist entries below
		mk(999, "self", 50, 9, 0, 0, 0),
		mk(108, "blacklisted", 88, 9, 0, 0, 0),
	}
}

func TestBuildFriendVisitPlanBasics(t *testing.T) {
	blacklist := map[int64]struct{}{108: {}}
	plan := buildFriendVisitPlan(friendListFixture(), 999, blacklist, friendPlanOptions{
		StealEnabled: true, HelpEnabled: true, BadEnabled: true,
		HelpAllowedForAll: true,
		BadBudget:         10,
		MaxBadOnlyVisits:  20,
	})

	if plan.StealCount != 3 {
		t.Fatalf("expected 3 steal targets, got %d", plan.StealCount)
	}
	if plan.HelpCount != 2 {
		t.Fatalf("expected 2 help targets, got %d", plan.HelpCount)
	}
	// idle-high-level + idle-low-level + no-plant are bad-only candidates; dup counts once.
	if plan.BadOnlyCount != 3 {
		t.Fatalf("expected 3 bad-only targets, got %d", plan.BadOnlyCount)
	}
	if len(plan.Visits) != 4+3 {
		t.Fatalf("expected 7 visits, got %d", len(plan.Visits))
	}
	// Blacklisted and self must never appear.
	for _, v := range plan.Visits {
		if v.GID == 108 || v.GID == 999 {
			t.Fatalf("blacklisted/self friend planned: %+v", v)
		}
	}
	// Primary first (steal desc), bad-only tail sorted by level desc.
	if plan.Visits[0].GID != 103 && plan.Visits[0].GID != 101 {
		t.Fatalf("expected steal leader first, got %d", plan.Visits[0].GID)
	}
}

func TestBuildFriendVisitPlanBadQuotaAndCap(t *testing.T) {
	blacklist := map[int64]struct{}{}
	plan := buildFriendVisitPlan(friendListFixture(), 999, blacklist, friendPlanOptions{
		StealEnabled: true, HelpEnabled: true, BadEnabled: true,
		HelpAllowedForAll: true,
		BadBudget:         1,
		MaxBadOnlyVisits:  2,
	})
	if plan.BadOnlyCount != 2 {
		t.Fatalf("expected bad-only capped at 2, got %d", plan.BadOnlyCount)
	}
	// Highest level bad-only first: the tail is level desc, so last <= second-last.
	if plan.Visits[len(plan.Visits)-1].Level > plan.Visits[len(plan.Visits)-2].Level {
		t.Fatal("bad-only must be sorted by level desc")
	}

	plan = buildFriendVisitPlan(friendListFixture(), 999, blacklist, friendPlanOptions{
		StealEnabled: true, HelpEnabled: true, BadEnabled: true,
		HelpAllowedForAll: true,
		BadBudget:         0,
		MaxBadOnlyVisits:  20,
	})
	if plan.BadOnlyCount != 0 {
		t.Fatalf("no bad budget → no bad-only visits, got %d", plan.BadOnlyCount)
	}
}

func TestBuildFriendVisitPlanExpLimitProtectDog(t *testing.T) {
	blacklist := map[int64]struct{}{}
	dogs := func(gid int64) friendDogState {
		switch gid {
		case 101:
			return dogStateProtect
		case 102:
			return dogStateNone
		default:
			return dogStateUnknown
		}
	}
	plan := buildFriendVisitPlan(friendListFixture(), 999, blacklist, friendPlanOptions{
		StealEnabled: true, HelpEnabled: true, BadEnabled: false,
		// 经验已满：只有护主犬好友值得进农场帮忙。
		HelpAllowedForAll:       false,
		ProtectDogBypassEnabled: true,
		GetDogState:             dogs,
		BadBudget:               10,
		MaxBadOnlyVisits:        20,
	})
	// 101 has steal so it's primary anyway; 102 (help-only, dog=none) must not want help.
	for _, v := range plan.Visits {
		if v.GID == 102 && v.WantHelp {
			t.Fatal("help-only friend without protect dog must be skipped when exp full")
		}
	}
	// 102 (dog=none) and 103 (dog=unknown) both lose help when exp is full.
	if plan.SkippedExpLimit != 2 {
		t.Fatalf("expected 2 exp-limit skips, got %d", plan.SkippedExpLimit)
	}
	// 103's dog state is unknown → counted separately.
	if plan.SkippedUnknownDog != 1 {
		t.Fatalf("expected 1 unknown-dog skip, got %d", plan.SkippedUnknownDog)
	}

	// Without the bypass switch, unknown-dog friends are counted as skipped-unknown.
	plan = buildFriendVisitPlan(friendListFixture(), 999, blacklist, friendPlanOptions{
		StealEnabled: true, HelpEnabled: true, BadEnabled: false,
		HelpAllowedForAll:       false,
		ProtectDogBypassEnabled: true,
		GetDogState:             func(int64) friendDogState { return dogStateUnknown },
		BadBudget:               10,
		MaxBadOnlyVisits:        20,
	})
	if plan.SkippedUnknownDog == 0 {
		t.Fatal("unknown dog state should be counted separately when bypass is on")
	}
}

func TestBuildFriendVisitPlanStealSortOrder(t *testing.T) {
	blacklist := map[int64]struct{}{108: {}}
	plan := buildFriendVisitPlan(friendListFixture(), 999, blacklist, friendPlanOptions{
		StealEnabled: true, HelpEnabled: false, BadEnabled: false,
		HelpAllowedForAll: true,
	})
	// stealNum desc: 101 (steal 3) > 103 (steal 2) > 107 (steal 1).
	first := plan.Visits[0]
	if first.GID != 101 {
		t.Fatalf("expected gid 101 (steal=3) first, got %d (steal=%d)", first.GID, first.StealNum)
	}
	if plan.HelpCount != 0 {
		t.Fatalf("help disabled → 0 help visits, got %d", plan.HelpCount)
	}
}
