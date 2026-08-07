package logic

import "testing"

func TestToTimeSec(t *testing.T) {
	t.Parallel()
	if ToTimeSec(0) != 0 || ToTimeSec(-1) != 0 {
		t.Fatal("non-positive")
	}
	if ToTimeSec(1_700_000_000) != 1_700_000_000 {
		t.Fatal("seconds passthrough")
	}
	if ToTimeSec(1_700_000_000_000) != 1_700_000_000 {
		t.Fatal("ms → sec")
	}
}

func TestGetServerTimeSecUsesSyncedClock(t *testing.T) {
	SyncServerTime(1_700_000_000_000)
	got := GetServerTimeSec()
	if got < 1_700_000_000 || got > 1_700_000_000+2 {
		t.Fatalf("GetServerTimeSec=%d", got)
	}
}

func TestInQuietHours(t *testing.T) {
	t.Parallel()

	if InQuietHours(false, "01:00", "07:00", "03:00") {
		t.Fatal("disabled should never be quiet")
	}
	if !InQuietHours(true, "01:00", "07:00", "03:00") {
		t.Fatal("inside window")
	}
	if InQuietHours(true, "01:00", "07:00", "08:00") {
		t.Fatal("outside window")
	}
	// start==end → all-day quiet (bot visit-strategy)
	if !InQuietHours(true, "23:00", "23:00", "12:00") {
		t.Fatal("start==end should be all-day quiet")
	}
	// wrap overnight
	if !InQuietHours(true, "23:00", "07:00", "01:00") {
		t.Fatal("overnight inside")
	}
	if InQuietHours(true, "23:00", "07:00", "12:00") {
		t.Fatal("overnight outside")
	}
}
