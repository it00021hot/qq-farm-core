package rbac

import (
	"testing"

	"github.com/MQEnergy/go-skeleton/internal/app/model"
)

func TestAggregatePolicies(t *testing.T) {
	resources := []model.SysResource{
		{BURL: "/backend/admin/*", Methods: "GET,POST"},
		{BURL: "/backend/admin/*", Methods: "PUT"},
		{BURL: "", Methods: "GET"},
		{BURL: "/backend/role/assignable", Methods: "GET"},
	}
	got := AggregatePolicies(resources)
	if got["/backend/admin/*"] != "GET,POST,PUT" {
		t.Fatalf("admin methods = %q", got["/backend/admin/*"])
	}
	if got["/backend/role/assignable"] != "GET" {
		t.Fatalf("assignable methods = %q", got["/backend/role/assignable"])
	}
	if _, ok := got[""]; ok {
		t.Fatal("empty path should be skipped")
	}
}

func TestNormalizeMethods(t *testing.T) {
	if got := NormalizeMethods(" post,get ,POST "); got != "GET,POST" {
		t.Fatalf("got %q", got)
	}
}
