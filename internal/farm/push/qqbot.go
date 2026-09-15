// Package push — QQ 官方机器人渠道（对齐 rust services/qq_bot）：
// AccessToken 缓存 + Gateway WSS（intent 1<<25）+ C2C 单聊发送。
package push

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
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

// Complete reports whether appId/secret/userOpenid are all set
// (rust QqBotConfig::is_complete，供跨包调用方在推送前判断绑定状态)。
func (c QqBotConfig) Complete() bool { return c.complete() }

type qqBotToken struct {
	value     string
	expiresAt time.Time
}

func (t qqBotToken) fresh() bool {
	return time.Now().Add(qqBotTokenAhead).Before(t.expiresAt)
}

// QqBot is the process-wide QQ bot service singleton.
type QqBot struct {
	mu      sync.Mutex
	tokens  map[string]qqBotToken
	gateway *qqBotGateway
	cfg     QqBotConfig
	// tokenURL / apiBase 端点可注入（单测指向 httptest 服务），空值为生产地址。
	tokenURL string
	apiBase  string
}

type qqBotGateway struct {
	cancel chan struct{}
	ready  chan struct{}
	key    string
}

var defaultQqBot = &QqBot{tokens: map[string]qqBotToken{}}

// QqBotShared returns the shared service.
func QqBotShared() *QqBot { return defaultQqBot }

// endpoints 返回生效端点（空字段回退生产地址；rust with_endpoints 注入点同语义）。
func (b *QqBot) endpoints() (tokenURL, apiBase string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	tokenURL, apiBase = b.tokenURL, b.apiBase
	if tokenURL == "" {
		tokenURL = qqBotTokenURL
	}
	if apiBase == "" {
		apiBase = qqBotAPIBase
	}
	return tokenURL, apiBase
}

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
		"appId":        strings.TrimSpace(cfg.AppID),
		"clientSecret": strings.TrimSpace(cfg.ClientSecret),
	})
	tokenURL, _ := b.endpoints()
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Post(tokenURL, "application/json", bytes.NewReader(payload))
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
	_, apiBase := b.endpoints()
	req, _ := http.NewRequest(http.MethodGet, apiBase+"/gateway", nil)
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

// apiJSON 调用 QQ Bot API（rust request_json）：带 AccessToken 鉴权，401 或
// 鉴权类业务码时强制刷新 token 重试一次（rust retry_auth 语义）。
// 返回（HTTP 状态码, 响应 JSON, err）；err 仅表示网络 / 组装错误。
func (b *QqBot) apiJSON(cfg QqBotConfig, path string, payload map[string]any, retryAuth bool) (int, map[string]any, error) {
	token, err := b.accessToken(cfg, false)
	if err != nil {
		return 0, nil, err
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, err
	}
	_, apiBase := b.endpoints()
	req, err := http.NewRequest(http.MethodPost, apiBase+path, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", "QQBot "+token)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("QQ Bot 网络错误: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var value map[string]any
	if json.Unmarshal(raw, &value) != nil {
		value = map[string]any{"message": string(raw)}
	}
	if retryAuth && (resp.StatusCode == http.StatusUnauthorized || qqBotAuthFailed(value)) {
		// rust：invalidate + ensure_gateway 后重试一次；go 侧强制刷新 token 后重试。
		if _, refreshErr := b.accessToken(cfg, true); refreshErr == nil {
			return b.apiJSON(cfg, path, payload, false)
		}
		// 刷新失败：按原响应返回，由调用方按失败处理。
	}
	return resp.StatusCode, value, nil
}

// qqBotCodeOf 提取响应里的业务码（数值或数字字符串，rust api_body_failed 同口径）。
func qqBotCodeOf(value map[string]any) int64 {
	if value == nil {
		return 0
	}
	switch c := value["code"].(type) {
	case float64:
		return int64(c)
	case json.Number:
		n, _ := c.Int64()
		return n
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(c), 10, 64)
		return n
	}
	return 0
}

// qqBotAuthFailed 鉴权类业务码（rust auth_failed：11242/11243/11251/11261/11275）。
func qqBotAuthFailed(value map[string]any) bool {
	switch qqBotCodeOf(value) {
	case 11242, 11243, 11251, 11261, 11275:
		return true
	}
	return false
}

// qqBotAPIBodyFailed 业务失败判定（rust api_body_failed：code 存在且非 0，或带 error 键）。
func qqBotAPIBodyFailed(value map[string]any) bool {
	if value == nil {
		return false
	}
	if _, has := value["error"]; has {
		return true
	}
	if _, has := value["code"]; !has {
		return false
	}
	return qqBotCodeOf(value) != 0
}

// qqBotAPIMessage 取响应错误文案（rust api_message：message / msg，回退 fallback）。
func qqBotAPIMessage(value map[string]any, fallback string) string {
	if value != nil {
		for _, key := range []string{"message", "msg"} {
			if s, ok := value[key].(string); ok && s != "" {
				return s
			}
		}
	}
	return fallback
}

// sendResultError 把发送响应归一成 error（rust send_message_payload 的失败分支）。
func sendResultError(value map[string]any, err error) error {
	if err != nil {
		return fmt.Errorf("QQ Bot 发送失败: %w", err)
	}
	if qqBotAPIBodyFailed(value) {
		return fmt.Errorf("QQ Bot 发送失败: code=%v %s", value["code"], qqBotAPIMessage(value, ""))
	}
	return nil
}

// userMessagesPath 构造 C2C 消息 / 富媒体上传路径（openid 做 query 转义，
// 对齐 rust encode_path 的 form_urlencoded：'/' → %2F、空格 → +）。
func userMessagesPath(openID string, file bool) string {
	escaped := url.QueryEscape(strings.TrimSpace(openID))
	if file {
		return "/v2/users/" + escaped + "/files"
	}
	return "/v2/users/" + escaped + "/messages"
}

// SendText sends a C2C text to the configured user openid.
func (b *QqBot) SendText(title, content string) error {
	cfg := b.CurrentConfig()
	if !cfg.complete() {
		return fmt.Errorf("QQ Bot 配置不完整")
	}
	if !b.ensureReady() {
		return fmt.Errorf("QQ Bot Gateway 未就绪")
	}
	text := strings.TrimSpace(title) + "\n" + strings.TrimSpace(content)
	if strings.TrimSpace(title) == "" {
		text = strings.TrimSpace(content)
	}
	_, value, err := b.apiJSON(cfg, userMessagesPath(cfg.UserOpenID, false),
		map[string]any{"msg_type": 0, "content": text}, true)
	return sendResultError(value, err)
}

// SendQrImage 推送二维码图片（rust send_qr_image）：先把公网图片 URL 提交
// /v2/users/{openid}/files 富媒体上传（file_type=1，srv_send_msg=false）拿
// file_info，再以 msg_type=7 + media 下发。调用方先发文本、图片失败仅记录
// （rust trigger_offline_reminder：文本成功后才推图，图片失败不再回退重发文本）。
func (b *QqBot) SendQrImage(imageURL string) error {
	cfg := b.CurrentConfig()
	if !cfg.complete() {
		return fmt.Errorf("QQ Bot 配置不完整")
	}
	imageURL = strings.TrimSpace(imageURL)
	if imageURL == "" {
		return fmt.Errorf("二维码地址为空")
	}
	if !strings.HasPrefix(imageURL, "http://") && !strings.HasPrefix(imageURL, "https://") {
		return fmt.Errorf("QQ Bot 富媒体接口要求公网 HTTP(S) 二维码地址")
	}
	if !b.ensureReady() {
		return fmt.Errorf("QQ Bot Gateway 未就绪")
	}
	// 1) 富媒体上传（不直接下发，srv_send_msg=false）。
	_, value, err := b.apiJSON(cfg, userMessagesPath(cfg.UserOpenID, true),
		map[string]any{"file_type": 1, "url": imageURL, "srv_send_msg": false}, true)
	if err != nil {
		return fmt.Errorf("QQ Bot 二维码上传失败: %v", err)
	}
	fileInfo, _ := value["file_info"].(string)
	if strings.TrimSpace(fileInfo) == "" {
		if msg := qqBotAPIMessage(value, ""); msg != "" {
			return fmt.Errorf("QQ Bot 未返回 file_info: %s", msg)
		}
		return fmt.Errorf("QQ Bot 未返回 file_info")
	}
	// 2) msg_type=7 rich media 消息。
	_, value, err = b.apiJSON(cfg, userMessagesPath(cfg.UserOpenID, false),
		map[string]any{"msg_type": 7, "media": map[string]any{"file_info": fileInfo}}, true)
	return sendResultError(value, err)
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
