package push

// 钉钉群机器人 webhook（对齐 rust services/push.rs）：
// sign = base64(HMAC-SHA256(secret, "{timestamp}\n{secret}"))。

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const dingTalkWebhookPrefix = "https://oapi.dingtalk.com/robot/send?access_token="

// BuildDingTalkWebhook builds a signed dingtalk robot URL.
func BuildDingTalkWebhook(endpoint, token, secret string) (string, error) {
	endpoint = strings.TrimSpace(endpoint)
	token = strings.TrimSpace(token)
	var base string
	switch {
	case strings.HasPrefix(endpoint, "https://"):
		if !strings.HasPrefix(endpoint, dingTalkWebhookPrefix) {
			return "", fmt.Errorf("钉钉 Webhook 地址格式无效")
		}
		base = endpoint
	case token != "":
		base = dingTalkWebhookPrefix + token
	default:
		return "", fmt.Errorf("钉钉 Webhook 地址格式无效")
	}
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return base, nil
	}
	timestamp := time.Now().UnixMilli()
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(fmt.Sprintf("%d\n%s", timestamp, secret)))
	sign := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return base + "&timestamp=" + fmt.Sprint(timestamp) + "&sign=" + urlEscapeSign(sign), nil
}

func urlEscapeSign(s string) string {
	r := strings.NewReplacer("+", "%2B", "/", "%2F", "=", "%3D", "\n", "%0A")
	return r.Replace(s)
}

// SendDingTalk sends a text message via the dingtalk robot.
func SendDingTalk(endpoint, token, secret, title, content string) error {
	url, err := BuildDingTalkWebhook(endpoint, token, secret)
	if err != nil {
		return err
	}
	text := strings.TrimSpace(title) + "\n" + content
	if strings.TrimSpace(title) == "" {
		text = content
	}
	payload, _ := json.Marshal(map[string]any{
		"msgtype": "text",
		"text":    map[string]string{"content": text},
	})
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Post(url, "application/json", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("钉钉发送失败: %w", err)
	}
	defer resp.Body.Close()
	var body struct {
		ErrCode int64  `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("钉钉发送失败: HTTP %d", resp.StatusCode)
	}
	// 钉钉 200 也可能回 errcode != 0（如 IP 白名单 / 加签错误）
	if body.ErrCode != 0 {
		return fmt.Errorf("钉钉发送失败: errcode=%d %s", body.ErrCode, body.ErrMsg)
	}
	return nil
}
