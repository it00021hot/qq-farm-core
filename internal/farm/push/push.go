package push

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// Notify sends an offline reminder via HTTP webhook (Bark / WeCom robot etc).
func Notify(webhookURL, title, body string) error {
	if webhookURL == "" {
		return nil
	}
	payload, _ := json.Marshal(map[string]any{
		"title": title,
		"body":  body,
		"text":  fmt.Sprintf("%s\n%s", title, body),
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
