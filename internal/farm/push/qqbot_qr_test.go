package push

// QQ Bot 富媒体（二维码图片）推送单测：用 httptest mock 官方接口
// （token / gateway / ws / files / messages），对齐 rust qq_bot/mod.rs 的
// sends_text_and_qr_through_official_flow 测试——文本 msg_type=0、二维码
// msg_type=7（先 /files 上传拿 file_info 再下发）、token 缓存、401 重试、
// openid 路径 query 转义。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gorilla/websocket"
)

// mockQqBotServer 官方接口 mock：捕获请求供断言。
type mockQqBotServer struct {
	mu            sync.Mutex
	wsBase        string // httptest 创建后再填充（gateway 路由拼 ws:// 地址）
	tokenRequests int
	paths         []string
	files         []map[string]any
	messages      []map[string]any
	failFirstMsg  int // 前 N 次 /messages 返回 401 鉴权失败（验证重试）
}

func (m *mockQqBotServer) snapshot() (paths []string, files, messages []map[string]any, tokenRequests, failFirstMsg int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.paths...),
		append([]map[string]any(nil), m.files...),
		append([]map[string]any(nil), m.messages...),
		m.tokenRequests, m.failFirstMsg
}

func newMockQqBotServer(t *testing.T) (*httptest.Server, *mockQqBotServer) {
	t.Helper()
	m := &mockQqBotServer{}
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	mux := http.NewServeMux()

	mux.HandleFunc("/app/getAppAccessToken", func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		m.tokenRequests++
		m.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"tok-1","expires_in":7200}`))
	})

	mux.HandleFunc("/gateway", func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		wsURL := m.wsBase + "/ws"
		m.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"url": wsURL})
	})

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		// Hello (op 10) → Identify (op 2) → READY (op 0)，与官方握手一致。
		_ = conn.WriteJSON(map[string]any{"op": 10, "d": map[string]any{"heartbeat_interval": 1000}})
		for {
			var msg map[string]any
			if conn.ReadJSON(&msg) != nil {
				return
			}
			if op, _ := msg["op"].(float64); op == 2 {
				_ = conn.WriteJSON(map[string]any{"op": 0, "s": 1, "t": "READY", "d": map[string]any{}})
			}
			// 心跳等其余消息静默。
		}
	})

	mux.HandleFunc("/v2/users/", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		m.mu.Lock()
		defer m.mu.Unlock()
		m.paths = append(m.paths, r.URL.EscapedPath())
		switch {
		case strings.HasSuffix(r.URL.EscapedPath(), "/files"):
			m.files = append(m.files, body)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"file_info":"file-info-1","ttl":300}`))
		case strings.HasSuffix(r.URL.EscapedPath(), "/messages"):
			if m.failFirstMsg > 0 {
				m.failFirstMsg--
				// 401 + 鉴权业务码：应触发 rust retry_auth 语义的一次重试。
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"code":11243,"message":"token expired"}`))
				return
			}
			m.messages = append(m.messages, body)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"msg-1"}`))
		default:
			http.NotFound(w, r)
		}
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	m.mu.Lock()
	m.wsBase = "ws" + strings.TrimPrefix(srv.URL, "http")
	m.mu.Unlock()
	return srv, m
}

func newTestQqBot(t *testing.T) (*QqBot, *mockQqBotServer) {
	t.Helper()
	srv, m := newMockQqBotServer(t)
	b := &QqBot{
		tokens:   map[string]qqBotToken{},
		tokenURL: srv.URL + "/app/getAppAccessToken",
		apiBase:  srv.URL,
	}
	b.Configure(QqBotConfig{AppID: "app", ClientSecret: "sec", UserOpenID: "user/open id"})
	t.Cleanup(func() { b.Configure(QqBotConfig{}) })
	return b, m
}

// TestSendQrImageThroughOfficialFlow 对齐 rust sends_text_and_qr_through_official_flow：
// 文本（msg_type=0）+ 二维码（msg_type=7，先 /files 上传拿 file_info），token 只取一次。
func TestSendQrImageThroughOfficialFlow(t *testing.T) {
	b, m := newTestQqBot(t)
	if !b.CurrentConfig().Complete() {
		t.Fatal("config should be complete")
	}
	if err := b.SendText("标题", "内容"); err != nil {
		t.Fatalf("SendText: %v", err)
	}
	if err := b.SendQrImage("https://example.com/qr.png"); err != nil {
		t.Fatalf("SendQrImage: %v", err)
	}

	paths, files, messages, tokenRequests, _ := m.snapshot()
	if tokenRequests != 1 {
		t.Fatalf("token requests = %d, want 1（token 需缓存）", tokenRequests)
	}
	if len(files) != 1 {
		t.Fatalf("files uploads = %d, want 1", len(files))
	}
	upload := files[0]
	if upload["file_type"] != float64(1) || upload["srv_send_msg"] != false || upload["url"] != "https://example.com/qr.png" {
		t.Fatalf("upload payload = %v", upload)
	}
	if len(messages) != 2 {
		t.Fatalf("messages = %d, want 2", len(messages))
	}
	if messages[0]["msg_type"] != float64(0) {
		t.Fatalf("text msg_type = %v", messages[0]["msg_type"])
	}
	media := messages[1]
	if media["msg_type"] != float64(7) {
		t.Fatalf("qr msg_type = %v", media["msg_type"])
	}
	mediaMap, _ := media["media"].(map[string]any)
	if mediaMap == nil || mediaMap["file_info"] != "file-info-1" {
		t.Fatalf("qr media = %v", media["media"])
	}
	// openid 需 query 转义（rust encode_path：'/' → %2F、空格 → +）。
	wantPaths := []string{
		"/v2/users/user%2Fopen+id/messages",
		"/v2/users/user%2Fopen+id/files",
		"/v2/users/user%2Fopen+id/messages",
	}
	if strings.Join(paths, "|") != strings.Join(wantPaths, "|") {
		t.Fatalf("paths = %v, want %v", paths, wantPaths)
	}
}

// TestSendQrImageValidations 入参校验对齐 rust send_qr_image 前置分支。
func TestSendQrImageValidations(t *testing.T) {
	b, _ := newTestQqBot(t)
	if err := b.SendQrImage(""); err == nil || !strings.Contains(err.Error(), "二维码地址为空") {
		t.Fatalf("empty url err = %v", err)
	}
	if err := b.SendQrImage("data:image/png;base64,xxx"); err == nil ||
		!strings.Contains(err.Error(), "公网 HTTP(S)") {
		t.Fatalf("data url err = %v", err)
	}
}

// TestSendMessageRetriesOnceOnAuthFailure rust request_json retry_auth：401 /
// 鉴权业务码（11243）时强制刷新 token 重试一次。
func TestSendMessageRetriesOnceOnAuthFailure(t *testing.T) {
	b, m := newTestQqBot(t)
	m.mu.Lock()
	m.failFirstMsg = 1
	m.mu.Unlock()

	if err := b.SendText("", "重试内容"); err != nil {
		t.Fatalf("SendText should recover via retry: %v", err)
	}
	_, _, messages, tokenRequests, _ := m.snapshot()
	if tokenRequests != 2 {
		t.Fatalf("token requests = %d, want 2（401 后强制刷新）", tokenRequests)
	}
	if len(messages) != 1 {
		t.Fatalf("messages = %d, want 1", len(messages))
	}
}

// TestUserMessagesPathEncoding 路径转义对齐 rust encode_path。
func TestUserMessagesPathEncoding(t *testing.T) {
	if got := userMessagesPath("user/open id", false); got != "/v2/users/user%2Fopen+id/messages" {
		t.Fatalf("messages path = %q", got)
	}
	if got := userMessagesPath("user", true); got != "/v2/users/user/files" {
		t.Fatalf("files path = %q", got)
	}
}

// TestQqBotAPIBodyFailed 鉴权码与业务失败判定对齐 rust auth_failed / api_body_failed。
func TestQqBotAPIBodyFailed(t *testing.T) {
	if !qqBotAuthFailed(map[string]any{"code": float64(11243)}) {
		t.Fatal("11243 should be auth failed")
	}
	if qqBotAuthFailed(map[string]any{"code": float64(304050)}) {
		t.Fatal("304050 should not be auth failed")
	}
	if qqBotAPIBodyFailed(map[string]any{"id": "ok"}) {
		t.Fatal("plain success body should not fail")
	}
	if !qqBotAPIBodyFailed(map[string]any{"code": "40001"}) {
		t.Fatal("numeric-string non-zero code should fail")
	}
	if !qqBotAPIBodyFailed(map[string]any{"error": "x"}) {
		t.Fatal("error key should fail")
	}
}
