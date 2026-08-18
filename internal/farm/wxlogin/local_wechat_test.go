package wxlogin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

func TestParseLocalWechatResponsePlainAndDouble(t *testing.T) {
	plain := []byte(`{"errcode":0,"jsdata":{"authorize_uuid":"uuid-1","nickname":"测试号"}}`)
	p, err := parseLocalWechatResponse(plain)
	if err != nil {
		t.Fatal(err)
	}
	if p.ErrCode != 0 || profileFromPayload(14013, p) == nil {
		t.Fatalf("payload=%+v", p)
	}
	wrapped, _ := json.Marshal(string(plain))
	p2, err := parseLocalWechatResponse(wrapped)
	if err != nil {
		t.Fatal(err)
	}
	if profileFromPayload(1, p2).AuthorizeUUID != "uuid-1" {
		t.Fatalf("double encoded uuid=%v", p2.JSData)
	}
	if _, err := parseLocalWechatResponse([]byte("not-json")); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestDetectAndAuthorizeLoopbackHTTP(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/check-login", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errcode": 0,
			"jsdata": map[string]any{
				"authorize_uuid": "uuid-1",
				"nickname":       "测试号",
				"headimgurl":     "https://img.example/a.png",
			},
		})
	})
	mux.HandleFunc("/api/authorize", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errcode": 0,
			"jsdata": map[string]any{
				"redirect_url": "https://yybadaccess.3g.qq.com/pc_yyb/pcyyb_oauth?login_type=WX&state=web&code=abc",
			},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	port, _ := strconv.Atoi(u.Port())
	client := LocalWechatClient{Scheme: "http", Host: "127.0.0.1"}
	profile, err := client.Detect(context.Background(), []uint16{uint16(port)})
	if err != nil {
		t.Fatal(err)
	}
	if profile.AuthorizeUUID != "uuid-1" || profile.Nickname != "测试号" {
		t.Fatalf("profile=%+v", profile)
	}
	redirect, err := client.Authorize(context.Background(), uint16(port), "uuid-1", 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(redirect, "code=abc") {
		t.Fatalf("redirect=%s", redirect)
	}
}

func TestDetectSelfSignedTLS(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errcode": 0,
			"jsdata":  map[string]any{"authorize_uuid": "tls-uuid", "nickname": "tls"},
		})
	}))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	port, _ := strconv.Atoi(u.Port())
	client := LocalWechatClient{
		Scheme:             "https",
		Host:               localWechatHost,
		DialIP:             "127.0.0.1",
		InsecureSkipVerify: true,
	}
	profile, err := client.Detect(context.Background(), []uint16{uint16(port)})
	if err != nil {
		t.Fatal(err)
	}
	if profile.AuthorizeUUID != "tls-uuid" {
		t.Fatalf("profile=%+v", profile)
	}
}

func TestAuthorizeMapsReject(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"errcode": 10050, "jsdata": map[string]any{}})
	}))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	port, _ := strconv.Atoi(u.Port())
	client := LocalWechatClient{Scheme: "http", Host: "127.0.0.1"}
	_, err := client.Authorize(context.Background(), uint16(port), "uuid-1", 0, 0)
	if err == nil || !strings.Contains(err.Error(), "拒绝授权") {
		t.Fatalf("err=%v", err)
	}
}

func TestQuickPeekDoesNotConsume(t *testing.T) {
	store := NewQuickStore()
	session, err := store.Create("owner")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Peek("owner", session.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Peek("owner", session.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Take("owner", session.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Peek("owner", session.ID); err == nil {
		t.Fatal("expected consumed session")
	}
}
