package runtime

import "testing"

// TestParseWsHTTPCode mirrors rust engine.rs parse_ws_http_code 测试语义：
// 从「unexpected server response:」后提取状态码，其余情况返回 0。
func TestParseWsHTTPCode(t *testing.T) {
	cases := []struct {
		msg  string
		want int64
	}{
		{"protocol: dial: unexpected server response: 400: websocket: bad handshake", 400},
		{"Protocol: Dial: UNEXPECTED SERVER RESPONSE: 503: x", 503},
		{"protocol: dial: websocket: bad handshake", 0},
		{"unexpected server response: ", 0},
		{"unexpected server response: abc", 0},
		{"", 0},
	}
	for _, c := range cases {
		if got := parseWsHTTPCode(c.msg); got != c.want {
			t.Fatalf("parseWsHTTPCode(%q) = %d, want %d", c.msg, got, c.want)
		}
	}
}

// TestLogWs400DialFailure400：400 时按分类与文案记录（有/无应用宝授权两种文案，
// 对齐 rust engine.rs ws_400 分支）。通过 appendRuntimeLog → hub 不可依赖 DB，
// 这里只校验不 panic 且非 400 不触发；文案分支依赖 logWs400DialFailure 内部实现。
func TestLogWs400DialFailureNoPanic(t *testing.T) {
	s := &Session{id: "0"}
	// 非 400：静默返回。
	s.logWs400DialFailure(errText("protocol: dial: websocket: bad handshake"))
	// 400：loadFarmAccount 在 vars.DB == nil 时安全返回 false。
	s.logWs400DialFailure(errText("protocol: dial: unexpected server response: 400: websocket: bad handshake"))
}

type errText string

func (e errText) Error() string { return string(e) }
