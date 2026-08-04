package rbac_test

import (
	"testing"

	"github.com/MQEnergy/go-skeleton/internal/app/model"
	"github.com/MQEnergy/go-skeleton/pkg/rbac"
)

func TestCanAssignSubtree(t *testing.T) {
	roles := []model.SysRole{
		{ID: 3, ParentID: 0, Level: 0},
		{ID: 4, ParentID: 3, Level: 1},
		{ID: 5, ParentID: 4, Level: 2},
		{ID: 1, ParentID: 0, Level: 0},
	}
	if !rbac.CanAssign(roles, []uint64{3}, []uint64{3, 4}) {
		t.Fatal("tenant admin should assign self and child")
	}
	if rbac.CanAssign(roles, []uint64{4}, []uint64{3}) {
		t.Fatal("staff must not assign parent role")
	}
	if rbac.CanAssign(roles, []uint64{3}, []uint64{1}) {
		t.Fatal("must not assign unrelated branch")
	}
}
