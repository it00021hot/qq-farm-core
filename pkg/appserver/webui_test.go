package appserver

import "testing"

func TestIsReservedAPIPath(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"/", false},
		{"", false},
		{"/index.html", false},
		{"/assets/app.js", false},
		{"/ping", true},
		{"/auth/login", true},
		{"/farm/ws", true},
		{"/system/admin/list", true},
		{"/api/ping", true},
		{"/game-config/foo.png", true},
		{"/token/create", true},
		{"/docs", true},
	}
	for _, tc := range cases {
		if got := isReservedAPIPath(tc.path); got != tc.want {
			t.Fatalf("isReservedAPIPath(%q)=%v want %v", tc.path, got, tc.want)
		}
	}
}
