package wxlogin

// qrlogin 单测：对齐 rust services/qrlogin.rs tests —— URL 构造、请求头、
// 状态码归一化（Wait/OK/Used/Error）、业务码非 0 失败。HTTP 层用 httptest
// 注入端点（MpHTTPClient 接口 + 端点字段化，见 qrlogin.go）。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuildMpLoginURLFormat(t *testing.T) {
	// rust build_login_url_format。
	got := BuildMpLoginURL("abc123")
	want := "https://h5.qzone.qq.com/qqq/code/abc123?_proxy=1&from=ide"
	if got != want {
		t.Fatalf("BuildMpLoginURL = %q, want %q", got, want)
	}
}

func TestMpConstantsAlignRust(t *testing.T) {
	// rust qua_constant / chrome_ua_constant / watcher 预设。
	if MpQUA != "V1_HT5_QDT_0.70.2209190_x64_0_DEV_D" {
		t.Fatalf("QUA = %q", MpQUA)
	}
	if !strings.HasPrefix(MpChromeUA, "Mozilla") || !strings.Contains(MpChromeUA, "Chrome/120") {
		t.Fatalf("Chrome UA unexpected: %q", MpChromeUA)
	}
	if MpPresetFarmAppID != "1112386029" {
		t.Fatalf("farm preset appid = %q", MpPresetFarmAppID)
	}
}

func TestMpHeadersIncludeQuaAndUA(t *testing.T) {
	// rust headers_include_qua_and_ua：键保持原始小写（qua 为自定义头，
	// go 侧经 req.Header[key] 原样写入，不做 MIME 规范化）。
	h := mpHeaders()
	if len(h["qua"]) == 0 || h["qua"][0] != MpQUA {
		t.Fatalf("missing qua header: %v", h)
	}
	if len(h["user-agent"]) == 0 || h["user-agent"][0] != MpChromeUA {
		t.Fatalf("missing user-agent header: %v", h)
	}
	if len(h["host"]) == 0 || h["host"][0] != "q.qq.com" {
		t.Fatalf("missing host header: %v", h)
	}
}

func newTestMpSession(t *testing.T, handler http.HandlerFunc) *MiniProgramLoginSession {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return &MiniProgramLoginSession{
		client:           srv.Client(),
		loginCodeURL:     srv.URL + "/ide/devtoolAuth/GetLoginCode",
		syncStatusURL:    srv.URL + "/ide/devtoolAuth/syncScanSateGetTicket",
		loginExchangeURL: srv.URL + "/ide/login",
	}
}

func TestRequestLoginCodeOK(t *testing.T) {
	var gotUA, gotQua string
	s := newTestMpSession(t, func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("user-agent")
		gotQua = r.Header.Get("qua")
		if r.URL.Path != "/ide/devtoolAuth/GetLoginCode" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"data":{"code":"login-code-1"}}`))
	})
	res, err := s.RequestLoginCode(context.Background())
	if err != nil {
		t.Fatalf("RequestLoginCode err: %v", err)
	}
	if res.Code != "login-code-1" {
		t.Fatalf("code = %q", res.Code)
	}
	if res.URL != BuildMpLoginURL("login-code-1") {
		t.Fatalf("url = %q", res.URL)
	}
	if gotUA != MpChromeUA || gotQua != MpQUA {
		t.Fatalf("headers not applied: ua=%q qua=%q", gotUA, gotQua)
	}
}

func TestRequestLoginCodeBusinessCodeNonZero(t *testing.T) {
	// rust：业务码非 0 → Err("获取登录码失败")。
	s := newTestMpSession(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":1001,"message":"busy"}`))
	})
	if _, err := s.RequestLoginCode(context.Background()); err == nil || err.Error() != "获取登录码失败" {
		t.Fatalf("err = %v, want 获取登录码失败", err)
	}
	// 字段缺失按 -1 处理（rust unwrap_or(-1)）→ 同样失败。
	s2 := newTestMpSession(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	})
	if _, err := s2.RequestLoginCode(context.Background()); err == nil {
		t.Fatal("missing code field should fail")
	}
}

func TestQueryStatusNormalization(t *testing.T) {
	cases := []struct {
		name   string
		body   string
		status int
		want   MpStatusResult
	}{
		{"ok", `{"code":0,"data":{"ok":1,"ticket":"tk","uin":"10000","nick":"nick"}}`, 200,
			MpStatusResult{Status: MpStatusOK, Ticket: "tk", Uin: "10000", Nickname: "nick"}},
		{"waiting", `{"code":0,"data":{"ok":0}}`, 200, MpStatusResult{Status: MpStatusWait}},
		{"used", `{"code":-10003,"message":"used"}`, 200, MpStatusResult{Status: MpStatusUsed}},
		{"other code", `{"code":-2,"message":"x"}`, 200, MpStatusResult{Status: MpStatusError, Msg: "Code: -2"}},
		{"missing code", `{"message":"?"}`, 200, MpStatusResult{Status: MpStatusError, Msg: "Code: -1"}},
		{"non-200", `{"code":0,"data":{"ok":1}}`, 500, MpStatusResult{Status: MpStatusError, Msg: "non-200 status"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newTestMpSession(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/ide/devtoolAuth/syncScanSateGetTicket" {
					t.Errorf("unexpected path %q", r.URL.Path)
				}
				if r.URL.Query().Get("code") != "abc" {
					t.Errorf("code query = %q", r.URL.Query().Get("code"))
				}
				w.WriteHeader(c.status)
				_, _ = w.Write([]byte(c.body))
			})
			got, err := s.QueryStatus(context.Background(), "abc")
			if err != nil {
				t.Fatalf("QueryStatus err: %v", err)
			}
			if got != c.want {
				t.Fatalf("got %+v, want %+v", got, c.want)
			}
		})
	}
}

func TestGetAuthCode(t *testing.T) {
	var gotBody map[string]any
	s := newTestMpSession(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/ide/login" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		gotBody = decodeJSONBody(t, r)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":"game-code-9"}`))
	})
	got, err := s.GetAuthCode(context.Background(), "tk-1", MpPresetFarmAppID)
	if err != nil {
		t.Fatalf("GetAuthCode err: %v", err)
	}
	if got != "game-code-9" {
		t.Fatalf("code = %q", got)
	}
	if gotBody["appid"] != MpPresetFarmAppID || gotBody["ticket"] != "tk-1" {
		t.Fatalf("request body = %v", gotBody)
	}

	// rust：非 200 → Ok("")（调用方按失败处理）。
	s2 := newTestMpSession(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	})
	if got, err := s2.GetAuthCode(context.Background(), "tk", MpPresetFarmAppID); err != nil || got != "" {
		t.Fatalf("non-200: code=%q err=%v", got, err)
	}
}

func TestQueryStatusNetworkErrorPropagates(t *testing.T) {
	// rust：网络错误原样返回 Err（调用方等一轮再试）。
	s := &MiniProgramLoginSession{
		client: &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			return nil, context.DeadlineExceeded
		})},
		syncStatusURL: "https://q.qq.com/ide/devtoolAuth/syncScanSateGetTicket",
	}
	if _, err := s.QueryStatus(context.Background(), "abc"); err == nil {
		t.Fatal("network error should propagate")
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func decodeJSONBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.NewDecoder(r.Body).Decode(&out); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return out
}
