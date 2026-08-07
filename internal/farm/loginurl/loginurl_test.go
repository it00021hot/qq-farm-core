package loginurl

import "testing"

func TestExtractCodeBare(t *testing.T) {
	got := ExtractCode("abc123XYZ")
	if got != "abc123XYZ" {
		t.Fatalf("got %q", got)
	}
}

func TestExtractCodeFromURL(t *testing.T) {
	raw := "wss://gate-obt.nqf.qq.com/prod/ws?platform=qq&os=iOS&ver=1.0.0&code=hello%2Dcode"
	got := ExtractCode(raw)
	if got != "hello-code" {
		t.Fatalf("got %q", got)
	}
	hints := ExtractClientHints(raw)
	if hints.Platform != "qq" || hints.OS != "iOS" || hints.Ver != "1.0.0" {
		t.Fatalf("hints=%+v", hints)
	}
}
