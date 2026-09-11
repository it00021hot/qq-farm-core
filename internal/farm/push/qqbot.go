// Package push — QQ 官方机器人渠道（对齐 rust services/qq_bot）：
// AccessToken 缓存 + Gateway WSS（intent 1<<25）+ C2C 单聊发送。
package push

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	qqBotTokenURL   = "https://api.bot.qq.com/app/getAppAccessToken"
	qqBotAPIBase    = "https://api.bot.qq.com"
	qqBotIntentC2C  = 1 << 25
	qqBotReadyWait  = 15 * time.Second
	qqBotReconnect  = 2 * time.Second
	qqBotTokenAhead = 60 * time.Second
)

// QqBotConfig mirrors rust QqBotConfig.
type QqBotConfig struct {
	AppID        string
	ClientSecret string
	UserOpenID   string
}

func (c QqBotConfig) complete() bool {
	return strings.TrimSpace(c.AppID) != "" &&
		strings.TrimSpace(c.ClientSecret) != "" &&
		strings.TrimSpace(c.UserOpenID) != ""
}

type qqBotToken struct {
	value     string
	expiresAt time.Time
}

func (t qqBotToken) fresh() bool {
	return time.Now().Add(qqBotTokenAhead).Before(t.expiresAt)
}

// QqBot is the process-wide QQ bot service singleton.
type QqBot struct {
	mu       sync.Mutex
	tokens   map[string]qqBotToken
	gateway  *qqBotGateway
	cfg      QqBotConfig
}

type qqBotGateway struct {
	cancel chan struct{}
	ready  chan struct{}
	key    string
}

var defaultQqBot = &QqBot{tokens: map[string]qqBotToken{}}

// QqBotShared returns the shared service.
func QqBotShared() *QqBot { return defaultQqBot }

// Configure swaps the credentials, restarting the gateway when they change.
func (b *QqBot) Configure(cfg QqBotConfig) {
	b.mu.Lock()
	defer b.mu.Unlock()
	key := cfg.AppID + ":" + cfg.ClientSecret
	if b.gateway != nil && b.gateway.key == key && cfg == b.cfg {
		return
	}
	if b.gateway != nil {
		close(b.gateway.cancel)
		b.gateway = nil
	}
	b.cfg = cfg
	if !cfg.complete() {
		return
	}
	gw := &qqBotGateway{cancel: make(chan struct{}), ready: make(chan struct{}), key: key}
	b.gateway = gw
	go b.gatewayLoop(cfg, gw)
}

func (b *QqBot) accessToken(cfg QqBotConfig, force bool) (string, error) {
	if strings.TrimSpace(cfg.AppID) == "" || strings.TrimSpace(cfg.ClientSecret) == "" {
		return "", fmt.Errorf("QQ Bot 配置不完整")
	}
	key := cfg.AppID + ":" + cfg.ClientSecret
	b.mu.Lock()
	if !force {
		if token, ok := b.tokens[key]; ok && token.fresh() {
			b.mu.Unlock()
			return token.value, nil
		}
	}
	b.mu.Unlock()

	payload, _ := json.Marshal(map[string]string{
		"appId":         strings.TrimSpace(cfg.AppID),
		"clientSecret":  strings.TrimSpace(cfg.ClientSecret),
	})
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Post(qqBotTokenURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("QQ Bot 网络错误: %w", err)
	}
	defer resp.Body.Close()
	var body struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
		Message     string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil || body.AccessToken == "" {
		msg := body.Message
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("获取 AccessToken 失败: %s", msg)
	}
	expiresIn := body.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 7200
	}
	b.mu.Lock()
	b.tokens[key] = qqBotToken{value: body.AccessToken, expiresAt: time.Now().Add(time.Duration(expiresIn) * time.Second)}
	b.mu.Unlock()
	return body.AccessToken, nil
}

func (b *QqBot) gatewayLoop(cfg QqBotConfig, gw *qqBotGateway) {
	for {
		select {
		case <-gw.cancel:
			return
		default:
		}
		b.gatewayOnce(cfg, gw)
		select {
		case <-gw.cancel:
			return
		case <-time.After(qqBotReconnect):
		}
	}
}

func (b *QqBot) gatewayOnce(cfg QqBotConfig, gw *qqBotGateway) {
	token, err := b.accessToken(cfg, false)
	if err != nil {
		slog.Warn("QQ Bot Gateway 初始化失败", "err", err)
		return
	}
	client := &http.Client{Timeout: 15 * time.Second}
	req, _ := http.NewRequest(http.MethodGet, qqBotAPIBase+"/gateway", nil)
	req.Header.Set("Authorization", "QQBot "+token)
	resp, err := client.Do(req)
	if err != nil {
		slog.Warn("QQ Bot Gateway 地址获取失败", "err", err)
		return
	}
	var gwBody struct {
		URL string `json:"url"`
	}
	err = json.NewDecoder(resp.Body).Decode(&gwBody)
	resp.Body.Close()
	if err != nil || gwBody.URL == "" {
		slog.Warn("QQ Bot Gateway 缺少 URL", "err", err)
		return
	}

	conn, _, err := websocket.DefaultDialer.Dial(gwBody.URL, nil)
	if err != nil {
		slog.Warn("QQ Bot Gateway 连接失败", "err", err)
		return
	}
	defer conn.Close()

	// Hello (op 10) → heartbeat interval
	var hello struct {
		Op int64 `json:"op"`
		D  struct {
			HeartbeatIntervalMs int64 `json:"heartbeat_interval"`
		} `json:"d"`
	}
	if err := conn.ReadJSON(&hello); err != nil || hello.Op != 10 {
		slog.Warn("QQ Bot Gateway 首包不是 Hello", "err", err)
		return
	}
	heartbeat := time.Duration(hello.D.HeartbeatIntervalMs) * time.Millisecond
	if heartbeat < time.Second {
		heartbeat = 41250 * time.Millisecond
	}

	identify := map[string]any{
		"op": 2,
		"d": map[string]any{
			"token":   "QQBot " + token,
			"intents": qqBotIntentC2C,
			"shard":   []int{0, 1},
			"properties": map[string]string{
				"$os": "windows", "$browser": "qq-farm-go", "$device": "qq-farm-go",
			},
		},
	}
	if err := conn.WriteJSON(identify); err != nil {
		return
	}

	heartbeatStop := make(chan struct{})
	go func() {
		defer close(heartbeatStop)
		ticker := time.NewTicker(heartbeat)
		defer ticker.Stop()
		var seq int64
		for {
			select {
			case <-gw.cancel:
				return
			case <-ticker.C:
				if err := conn.WriteJSON(map[string]any{"op": 1, "d": seq}); err != nil {
					return
				}
			case seq = <-gwSeq:
			}
		}
	}()
	defer close(heartbeatStop)

	for {
		select {
		case <-gw.cancel:
			return
		default:
		}
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var msg struct {
			Op int64  `json:"op"`
			T  string `json:"t"`
			S  int64  `json:"s"`
		}
		if json.Unmarshal(raw, &msg) != nil {
			continue
		}
		if msg.S > 0 {
			select {
			case gwSeq <- msg.S:
			default:
			}
		}
		switch msg.Op {
		case 0:
			if msg.T == "READY" || msg.T == "RESUMED" {
				select {
				case <-gw.ready:
				default:
					close(gw.ready)
				}
			}
			if msg.T == "C2C_MESSAGE_CREATE" {
				b.handleC2CMessage(raw, cfg)
			}
		case 7, 9:
			return
		}
	}
}

// gwSeq carries the latest gateway sequence for heartbeats.
var gwSeq = make(chan int64, 1)

// ensureReady waits up to qqBotReadyWait for the gateway READY.
func (b *QqBot) ensureReady() bool {
	b.mu.Lock()
	gw := b.gateway
	complete := b.cfg.complete()
	b.mu.Unlock()
	if gw == nil || !complete {
		return false
	}
	select {
	case <-gw.ready:
		return true
	case <-time.After(qqBotReadyWait):
		return false
	}
}

// SendText sends a C2C text to the configured user openid.
func (b *QqBot) SendText(title, content string) error {
	b.mu.Lock()
	cfg := b.cfg
	b.mu.Unlock()
	if !cfg.complete() {
		return fmt.Errorf("QQ Bot 配置不完整")
	}
	if !b.ensureReady() {
		return fmt.Errorf("QQ Bot Gateway 未就绪")
	}
	token, err := b.accessToken(cfg, false)
	if err != nil {
		return err
	}
	text := strings.TrimSpace(title) + "\n" + strings.TrimSpace(content)
	if strings.TrimSpace(title) == "" {
		text = strings.TrimSpace(content)
	}
	payload, _ := json.Marshal(map[string]any{"msg_type": 0, "content": text})
	url := qqBotAPIBase + "/v2/users/" + strings.TrimSpace(cfg.UserOpenID) + "/messages"
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	req.Header.Set("Authorization", "QQBot "+token)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return fmt.Errorf("QQ Bot 发送失败: %w", err)
	}
	defer resp.Body.Close()
	var body map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if resp.StatusCode == http.StatusUnauthorized {
		// token 过期：强制刷新后由下一次发送重试
		if _, err := b.accessToken(cfg, true); err != nil {
			return err
		}
		return fmt.Errorf("QQ Bot token 已刷新，请重试")
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("QQ Bot 发送失败: HTTP %d", resp.StatusCode)
	}
	if code, _ := body["code"].(float64); code != 0 {
		msg, _ := body["message"].(string)
		return fmt.Errorf("QQ Bot 发送失败: code=%v %s", body["code"], msg)
	}
	return nil
}

// handleC2CMessage applies the bind protocol to a private message and replies.
func (b *QqBot) handleC2CMessage(raw []byte, cfg QqBotConfig) {
	var payload struct {
		ID      string `json:"id"`
		Content string `json:"content"`
		Author  struct {
			UserOpenID string `json:"user_openid"`
			ID         string `json:"id"`
			Username   string `json:"username"`
		} `json:"author"`
	}
	if json.Unmarshal(raw, &payload) != nil {
		return
	}
	openid := payload.Author.UserOpenID
	if openid == "" {
		openid = payload.Author.ID
	}
	reply, handled := HandleC2CMessage(openid, payload.Author.Username, payload.Content)
	if !handled {
		return
	}
	// 有 msg_id 时走被动回复（官方推荐），否则主动下发。
	target := openid
	if err := b.sendTextTo(cfg, target, reply); err != nil {
		slog.Warn("QQ Bot 绑定回执失败", "err", err)
	}
}

// sendTextTo sends a C2C text to an arbitrary openid (binding receipts).
func (b *QqBot) sendTextTo(cfg QqBotConfig, userOpenID, text string) error {
	token, err := b.accessToken(cfg, false)
	if err != nil {
		return err
	}
	payload, _ := json.Marshal(map[string]any{"msg_type": 0, "content": text})
	url := qqBotAPIBase + "/v2/users/" + strings.TrimSpace(userOpenID) + "/messages"
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	req.Header.Set("Authorization", "QQBot "+token)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}
