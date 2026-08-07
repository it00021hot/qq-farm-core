package logic

import "testing"

func TestResolvePlantDisplayNamePrefersMutantConfig(t *testing.T) {
	goldName := "黄金·艾草"
	base := &PlantItem{ID: 1021135, Name: "艾草", Fruit: struct {
		ID    int64 `json:"id"`
		Count int64 `json:"count"`
	}{ID: 41135, Count: 24}}
	gold := &PlantItem{ID: 1121135, Name: goldName, Fruit: struct {
		ID    int64 `json:"id"`
		Count int64 `json:"count"`
	}{ID: 1041135, Count: 10}}
	goldFruit := &ItemInfo{ID: 1041135, Name: goldName, Type: 17}
	GlobalGameConfig.mu.Lock()
	GlobalGameConfig.plantByID[1021135] = base
	GlobalGameConfig.plantByID[1121135] = gold
	GlobalGameConfig.plantByFruit[41135] = base
	GlobalGameConfig.plantByFruit[1041135] = gold
	GlobalGameConfig.itemByID[1041135] = goldFruit
	GlobalGameConfig.mu.Unlock()
	t.Cleanup(func() {
		GlobalGameConfig.mu.Lock()
		delete(GlobalGameConfig.plantByID, 1021135)
		delete(GlobalGameConfig.plantByID, 1121135)
		delete(GlobalGameConfig.plantByFruit, 41135)
		delete(GlobalGameConfig.plantByFruit, 1041135)
		delete(GlobalGameConfig.itemByID, 1041135)
		GlobalGameConfig.mu.Unlock()
	})

	id, name := ResolvePlantDisplayName(&PlantInfo{
		ID: 1021135, Name: "艾草", MutantConfigIDs: []int64{1121135},
	})
	if id != 1121135 || name != goldName {
		t.Fatalf("mutant_config got id=%d name=%q want 1121135/%q", id, name, goldName)
	}

	// Server still sends base name/id; fruit_id reveals golden mutant.
	id, name = ResolvePlantDisplayName(&PlantInfo{
		ID: 1021135, Name: "艾草", FruitID: 1041135,
	})
	if id != 1121135 || name != goldName {
		t.Fatalf("fruit_id got id=%d name=%q want 1121135/%q", id, name, goldName)
	}

	id, name = ResolvePlantDisplayName(&PlantInfo{ID: 1021135, Name: "现场名"})
	if name != "现场名" {
		t.Fatalf("runtime name=%q", name)
	}
	_ = id
}

func TestMatureInSecFormula(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		matureBegin int64
		nowSec      int64
		want        int64
	}{
		{name: "future", matureBegin: 1_700_000_100, nowSec: 1_700_000_000, want: 100},
		{name: "exact now", matureBegin: 1_700_000_000, nowSec: 1_700_000_000, want: 0},
		{name: "past", matureBegin: 1_699_999_900, nowSec: 1_700_000_000, want: 0},
		{name: "missing", matureBegin: 0, nowSec: 1_700_000_000, want: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := MatureInSec(tc.matureBegin, tc.nowSec); got != tc.want {
				t.Fatalf("MatureInSec(%d,%d)=%d want %d", tc.matureBegin, tc.nowSec, got, tc.want)
			}
		})
	}
}

func TestFormatLandsResponseMatureInSecAndCareFlags(t *testing.T) {
	const nowMs = int64(1_700_000_000_000)
	SyncServerTime(nowMs)
	nowSec := GetServerTimeSec()
	// Keep assertions stable if a few ms elapse between sync and format.
	if nowSec < nowMs/1000 || nowSec > nowMs/1000+1 {
		t.Fatalf("serverNow=%d unexpected after sync", nowSec)
	}

	matureAt := nowMs/1000 + 120
	dryDue := nowMs/1000 - 10
	weedFuture := nowMs/1000 + 60
	bugDue := nowMs/1000 - 5

	lands := []LandInfo{
		{
			ID: 1, Unlocked: true,
			Plant: &PlantInfo{
				ID: 10, Name: "番茄", DryNum: 0,
				Phases: []PlantPhaseInfo{
					{Phase: PhaseGermination, BeginTime: nowMs/1000 - 100, DryTime: dryDue, WeedsTime: weedFuture, InsectTime: bugDue},
					{Phase: PhaseMature, BeginTime: matureAt},
				},
			},
		},
		{
			ID: 2, Unlocked: true,
			Plant: &PlantInfo{
				ID: 11, Name: "已熟",
				Phases: []PlantPhaseInfo{
					{Phase: PhaseMature, BeginTime: nowMs / 1000},
				},
			},
		},
		{
			ID: 3, Unlocked: true,
			Plant: &PlantInfo{
				ID: 12, Name: "有草虫",
				WeedOwners:   []int64{99},
				InsectOwners: []int64{88},
				Phases: []PlantPhaseInfo{
					{Phase: PhaseSmallLeaves, BeginTime: nowMs/1000 - 50, DryTime: 0, WeedsTime: 0, InsectTime: 0},
					{Phase: PhaseMature, BeginTime: matureAt + 500},
				},
			},
		},
	}

	res := FormatLandsResponse(lands)
	if len(res.Lands) != 3 {
		t.Fatalf("lands=%d", len(res.Lands))
	}

	row1 := res.Lands[0]
	wantMature := MatureInSec(matureAt, GetServerTimeSec())
	if row1.MatureInSec != wantMature {
		t.Fatalf("land1 matureInSec=%d want %d", row1.MatureInSec, wantMature)
	}
	if row1.Status != "growing" {
		t.Fatalf("land1 status=%s", row1.Status)
	}
	if !row1.NeedWater {
		t.Fatal("land1 needWater=false")
	}
	if row1.NeedWeed {
		t.Fatal("land1 needWeed should be false (weeds_time in future, no owners)")
	}
	if !row1.NeedBug {
		t.Fatal("land1 needBug=false")
	}

	row2 := res.Lands[1]
	if row2.Status != "harvestable" {
		t.Fatalf("land2 status=%s", row2.Status)
	}
	if row2.MatureInSec != 0 {
		t.Fatalf("land2 matureInSec=%d want 0", row2.MatureInSec)
	}

	row3 := res.Lands[2]
	if !row3.NeedWeed || !row3.NeedBug {
		t.Fatalf("land3 needWeed=%v needBug=%v", row3.NeedWeed, row3.NeedBug)
	}
	if row3.NeedWater {
		t.Fatal("land3 needWater should be false")
	}
	if row3.Status != "growing" {
		t.Fatalf("land3 status=%s (stealable must not override status)", row3.Status)
	}
}

func TestFormatLandsResponseStealableKeepsGrowingStatus(t *testing.T) {
	SyncServerTime(1_700_000_000_000)
	now := GetServerTimeSec()
	lands := []LandInfo{{
		ID: 1, Unlocked: true,
		Plant: &PlantInfo{
			ID: 1, Name: "可偷", Stealable: true,
			Phases: []PlantPhaseInfo{
				{Phase: PhaseBlooming, BeginTime: now - 10},
				{Phase: PhaseMature, BeginTime: now + 60},
			},
		},
	}}
	res := FormatLandsResponse(lands)
	if len(res.Lands) != 1 {
		t.Fatalf("lands=%d", len(res.Lands))
	}
	row := res.Lands[0]
	if row.Status != "growing" {
		t.Fatalf("status=%s want growing (bot getLandsDetail)", row.Status)
	}
	if !row.Stealable {
		t.Fatal("stealable flag missing")
	}
	want := MatureInSec(now+60, GetServerTimeSec())
	if row.MatureInSec != want {
		t.Fatalf("matureInSec=%d want %d", row.MatureInSec, want)
	}
}

func TestFormatFriendLandsResponseStealableOnlyWhenMature(t *testing.T) {
	SyncServerTime(1_700_000_000_000)
	now := GetServerTimeSec()
	lands := []LandInfo{
		{
			ID: 1, Unlocked: true,
			Plant: &PlantInfo{
				ID: 1, Name: "未熟可偷标记", Stealable: true,
				Phases: []PlantPhaseInfo{
					{Phase: PhaseBlooming, BeginTime: now - 10},
					{Phase: PhaseMature, BeginTime: now + 300},
				},
			},
		},
		{
			ID: 2, Unlocked: true,
			Plant: &PlantInfo{
				ID: 2, Name: "已熟可偷", Stealable: true,
				Phases: []PlantPhaseInfo{
					{Phase: PhaseBlooming, BeginTime: now - 100},
					{Phase: PhaseMature, BeginTime: now - 1},
				},
			},
		},
		{
			ID: 3, Unlocked: true,
			Plant: &PlantInfo{
				ID: 3, Name: "已熟不可偷", Stealable: false,
				Phases: []PlantPhaseInfo{
					{Phase: PhaseMature, BeginTime: now - 1},
				},
			},
		},
	}
	res := FormatFriendLandsResponse(lands)
	if len(res.Lands) != 3 {
		t.Fatalf("lands=%d", len(res.Lands))
	}
	immature := res.Lands[0]
	if immature.Status != "growing" {
		t.Fatalf("immature status=%s want growing", immature.Status)
	}
	if immature.Stealable {
		t.Fatal("immature must not advertise stealable (bot getFriendLandsDetail)")
	}
	if immature.MatureInSec <= 0 {
		t.Fatal("immature should have mature countdown")
	}
	ripe := res.Lands[1]
	if ripe.Status != "stealable" {
		t.Fatalf("ripe stealable status=%s want stealable", ripe.Status)
	}
	if !ripe.Stealable {
		t.Fatal("ripe stealable flag missing")
	}
	harvested := res.Lands[2]
	if harvested.Status != "harvested" {
		t.Fatalf("ripe non-stealable status=%s want harvested", harvested.Status)
	}
	if harvested.Stealable {
		t.Fatal("harvested must not be stealable")
	}
}

func TestFormatLandsResponseMultiCellSlaveInheritsMaster(t *testing.T) {
	SyncServerTime(1_700_000_000_000)
	seedID := int64(20001)
	size := int64(2)
	plant := &PlantItem{ID: 100, Name: "合种测试", SeedID: &seedID, Size: &size, Seasons: 1}

	GlobalGameConfig.mu.Lock()
	GlobalGameConfig.plantByID[100] = plant
	GlobalGameConfig.plantBySeed[seedID] = plant
	GlobalGameConfig.mu.Unlock()
	t.Cleanup(func() {
		GlobalGameConfig.mu.Lock()
		delete(GlobalGameConfig.plantByID, 100)
		delete(GlobalGameConfig.plantBySeed, seedID)
		GlobalGameConfig.mu.Unlock()
	})

	lands := []LandInfo{
		{
			ID: 1, Unlocked: true, SlaveLandIDs: []int64{2, 3, 4},
			Plant: &PlantInfo{
				ID: 100, Name: "合种测试", Season: 1,
				Phases: []PlantPhaseInfo{
					{Phase: PhaseGermination, BeginTime: 1_699_000_000},
					{Phase: PhaseMature, BeginTime: 1_800_000_000},
				},
			},
		},
		{ID: 2, Unlocked: true, MasterLandID: 1},
		{ID: 3, Unlocked: true, MasterLandID: 1},
		{ID: 4, Unlocked: true, MasterLandID: 1},
		{ID: 5, Unlocked: true},
	}

	res := FormatLandsResponse(lands)
	if len(res.Lands) != 5 {
		t.Fatalf("lands=%d", len(res.Lands))
	}
	for _, id := range []int64{2, 3, 4} {
		var row *UILandRow
		for i := range res.Lands {
			if res.Lands[i].ID == id {
				row = &res.Lands[i]
				break
			}
		}
		if row == nil {
			t.Fatalf("missing land %d", id)
		}
		if row.Status == "empty" {
			t.Fatalf("slave %d still empty", id)
		}
		if !row.OccupiedByMaster {
			t.Fatalf("slave %d occupiedByMaster=false", id)
		}
		if row.PlantName == "" || row.PlantName == "-" {
			t.Fatalf("slave %d plantName=%q", id, row.PlantName)
		}
		if row.SeedImage == "" {
			t.Fatalf("slave %d seedImage empty", id)
		}
		if row.MasterLandID != 1 {
			t.Fatalf("slave %d master=%d", id, row.MasterLandID)
		}
		if row.PlantSize != 2 {
			t.Fatalf("slave %d plantSize=%d", id, row.PlantSize)
		}
	}
	if res.Summary.Empty != 1 {
		t.Fatalf("summary.empty=%d want 1", res.Summary.Empty)
	}
	if res.Summary.Growing < 4 {
		t.Fatalf("summary.growing=%d want >=4", res.Summary.Growing)
	}
}
