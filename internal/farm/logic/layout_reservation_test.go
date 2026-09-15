package logic_test

import (
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
)

// 与 rust layout_reservation.rs / bot farm-multiland-reservation.test.js
// 场景对齐的黄金用例（依赖真实 Land 网格）。
func TestSelectFutureLayoutReservation(t *testing.T) {
	loadTestGameConfig(t)
	all := []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}

	t.Run("场景3：选最早可收敛布局，预留已空出部分", func(t *testing.T) {
		r, ok := logic.SelectFutureLayoutReservation([]int64{1, 2, 3}, all, 2)
		if !ok {
			t.Fatal("reservation expected")
		}
		if r.Layout.AnchorLandID != 5 {
			t.Fatalf("anchor = %d, want 5", r.Layout.AnchorLandID)
		}
		if !slices.Equal(r.Layout.LandIDs, []int64{5, 6, 1, 2}) {
			t.Fatalf("landIds = %v, want [5 6 1 2]", r.Layout.LandIDs)
		}
		if !slices.Equal(r.ReservedLandIDs, []int64{1, 2}) {
			t.Fatalf("reserved = %v, want [1 2]", r.ReservedLandIDs)
		}
	})

	t.Run("场景4：同一布局继续累积已空出土地", func(t *testing.T) {
		r, ok := logic.SelectFutureLayoutReservation([]int64{1, 2, 5}, all, 2)
		if !ok {
			t.Fatal("reservation expected")
		}
		if r.Layout.AnchorLandID != 5 {
			t.Fatalf("anchor = %d, want 5", r.Layout.AnchorLandID)
		}
		if !slices.Equal(r.ReservedLandIDs, []int64{5, 1, 2}) {
			t.Fatalf("reserved = %v, want [5 1 2]", r.ReservedLandIDs)
		}
	})

	t.Run("场景5：防振荡——更满的更晚布局不选", func(t *testing.T) {
		r, ok := logic.SelectFutureLayoutReservation([]int64{1, 3, 4, 7}, all, 2)
		if !ok {
			t.Fatal("reservation expected")
		}
		if r.Layout.AnchorLandID != 5 {
			t.Fatalf("anchor = %d, want 5", r.Layout.AnchorLandID)
		}
		if !slices.Equal(r.ReservedLandIDs, []int64{1}) {
			t.Fatalf("reserved = %v, want [1]", r.ReservedLandIDs)
		}
	})

	t.Run("场景6：空地已能连成完整布局则直接种", func(t *testing.T) {
		if r, ok := logic.SelectFutureLayoutReservation([]int64{1, 2, 5, 6}, all, 2); ok {
			t.Fatalf("unexpected reservation: %+v", r)
		}
	})

	t.Run("场景7：单格与凑不出的布局不预留", func(t *testing.T) {
		if r, ok := logic.SelectFutureLayoutReservation([]int64{1, 2, 3}, all, 1); ok {
			t.Fatalf("single land should not reserve: %+v", r)
		}
		if r, ok := logic.SelectFutureLayoutReservation([]int64{1, 2, 3}, []int64{1, 2, 3, 4}, 2); ok {
			t.Fatalf("impossible layout should not reserve: %+v", r)
		}
	})

	t.Run("空地列表为空不预留", func(t *testing.T) {
		if r, ok := logic.SelectFutureLayoutReservation(nil, all, 2); ok {
			t.Fatalf("empty current lands should not reserve: %+v", r)
		}
	})
}

func loadTestGameConfig(t *testing.T) {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	dir := filepath.Join(root, "resource", "farm", "gameConfig")
	if err := logic.LoadGameConfig(dir); err != nil {
		t.Fatal(err)
	}
}
