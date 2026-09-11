package push

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

// Notify sends an offline reminder via HTTP webhook (Bark / WeCom robot etc).
func Notify(webhookURL, title, body string) error {
	if webhookURL == "" {
		return nil
	}
	payload, _ := json.Marshal(map[string]any{
		"title":   title,
		"body":    body,
		"text":    fmt.Sprintf("%s\n%s", title, body),
		"msgtype": "text",
		"markdown": map[string]string{
			"content": fmt.Sprintf("**%s**\n%s", title, body),
		},
	})
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(webhookURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		slog.Warn("farm push failed", "err", err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("push status %d", resp.StatusCode)
	}
	return nil
}

// ConfigureQqBot applies the credentials from config (called at boot).
func ConfigureQqBot(appID, clientSecret, userOpenID string) {
	QqBotShared().Configure(QqBotConfig{
		AppID: appID, ClientSecret: clientSecret, UserOpenID: userOpenID,
	})
}

// CurrentConfig returns the active QQ bot credentials.
func (b *QqBot) CurrentConfig() QqBotConfig {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.cfg
}

type dingTalkChannels struct {
	endpoint string
	token    string
	secret   string
}

var currentDingTalk atomic.Value

// ConfigureDingTalk applies the dingtalk channel from config (called at boot).
func ConfigureDingTalk(endpoint, token, secret string) {
	currentDingTalk.Store(dingTalkChannels{endpoint: endpoint, token: token, secret: secret})
}

// NotifyAll pushes through every configured channel:
// QQ 官方机器人 + 钉钉（rust 渠道面）+ 兼容旧 webhook。
func NotifyAll(webhookURL, title, body string) {
	if cfg := QqBotShared().CurrentConfig(); cfg.complete() {
		if err := QqBotShared().SendText(title, body); err != nil {
			slog.Warn("qq bot push failed", "err", err)
		}
	}
	if v := currentDingTalk.Load(); v != nil {
		c := v.(dingTalkChannels)
		if strings.TrimSpace(c.endpoint) != "" || strings.TrimSpace(c.token) != "" {
			if err := SendDingTalk(c.endpoint, c.token, c.secret, title, body); err != nil {
				slog.Warn("dingtalk push failed", "err", err)
			}
		}
	}
	if webhookURL != "" {
		_ = Notify(webhookURL, title, body)
	}
}
