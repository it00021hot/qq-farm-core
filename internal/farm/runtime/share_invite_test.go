package runtime

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseShareLink(t *testing.T) {
	p := parseShareLink("?uid=12345&openid=abc&share_source=7&doc_id=x")
	if p.uid != "12345" || p.openid != "abc" || p.shareCfgID != 7 {
		t.Fatalf("got %+v", p)
	}
	p = parseShareLink("https://example.com/path?uid=9&openid=o1")
	if p.uid != "9" || p.openid != "o1" || p.shareCfgID != 0 {
		t.Fatalf("url got %+v", p)
	}
}

func TestReadShareFileDedup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "share.txt")
	content := "uid=1&openid=a\nuid=1&openid=a\nuid=2&openid=b\nnoop\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	invites := readShareFile(path)
	if len(invites) != 2 {
		t.Fatalf("want 2 invites, got %d", len(invites))
	}
	if invites[0].uid != "1" || invites[1].uid != "2" {
		t.Fatalf("order/ids: %+v", invites)
	}
}

func TestResolveShareFilePath(t *testing.T) {
	t.Setenv("FARM_SHARE_FILE", "")
	got := resolveShareFilePath("/tmp/farmdata", "")
	want := filepath.Join("/tmp/farmdata", "share.txt")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	got = resolveShareFilePath("/tmp/farmdata", "/cfg/share.txt")
	if got != "/cfg/share.txt" {
		t.Fatalf("config override: %q", got)
	}
	t.Setenv("FARM_SHARE_FILE", "/custom/share.txt")
	if got := resolveShareFilePath("/tmp/farmdata", "/cfg/share.txt"); got != "/custom/share.txt" {
		t.Fatalf("env override: %q", got)
	}
}
