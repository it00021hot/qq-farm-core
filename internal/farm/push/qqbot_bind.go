package push

// QQ 机器人绑定流程（对齐 rust bind.rs + handle_c2c_message 语义）：
// 面板发起绑定 → 用户在 QQ 里私聊机器人任意消息完成绑定 → 回执确认并持久化；
// 发送「解绑」清除绑定。绑定落在 data/qqbot-binding.json，启动时合并进运行配置。

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// QqBotBinding is one persisted binding.
type QqBotBinding struct {
	UserOpenID string `json:"userOpenid"`
	Nickname   string `json:"nickname"`
	BoundAt    int64  `json:"boundAt"`
}

var (
	bindMu       sync.Mutex
	bindSession  *bindSessionInfo             // panel-initiated pending session
	bindings     = map[string]*QqBotBinding{} // username → binding
	openidUser   = map[string]string{}        // openid → username
	bindFilePath string
)

type bindSessionInfo struct {
	Username  string
	SessionID string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// InitQqBotBinding loads persisted bindings from dataDir.
func InitQqBotBinding(dataDir string) {
	bindFilePath = filepath.Join(dataDir, "qqbot-binding.json")
	raw, err := os.ReadFile(bindFilePath)
	if err != nil {
		return
	}
	var persisted map[string]QqBotBinding
	if json.Unmarshal(raw, &persisted) != nil {
		return
	}
	bindMu.Lock()
	defer bindMu.Unlock()
	for username, binding := range persisted {
		b := binding
		bindings[username] = &b
		if b.UserOpenID != "" {
			openidUser[b.UserOpenID] = username
		}
	}
}

// StartBindSession opens a pending bind for username (bot BindSessionManager::start).
func StartBindSession(username string) string {
	bindMu.Lock()
	defer bindMu.Unlock()
	sessionID := "bind-" + username + "-" + time.Now().Format("150405.000000000")
	bindSession = &bindSessionInfo{
		Username: username, SessionID: sessionID,
		CreatedAt: time.Now(), ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	return sessionID
}

// PendingBindUser returns the username with a live bind session ("" if none).
func PendingBindUser() string {
	bindMu.Lock()
	defer bindMu.Unlock()
	if bindSession == nil || time.Now().After(bindSession.ExpiresAt) {
		return ""
	}
	return bindSession.Username
}

// CurrentBinding returns the admin binding (single-admin deployment).
func CurrentBinding() *QqBotBinding {
	bindMu.Lock()
	defer bindMu.Unlock()
	for _, binding := range bindings {
		if binding.UserOpenID != "" {
			out := *binding
			return &out
		}
	}
	return nil
}

// PollBindSession reports a panel-initiated bind session status (rust poll_qq_bot_bind):
// pending / bound (with the fresh binding) / expired.
func PollBindSession(sessionID string) (string, *QqBotBinding) {
	bindMu.Lock()
	defer bindMu.Unlock()
	if bindSession == nil || bindSession.SessionID != sessionID {
		return "expired", nil
	}
	if time.Now().After(bindSession.ExpiresAt) {
		return "expired", nil
	}
	if b := bindings[bindSession.Username]; b != nil && b.UserOpenID != "" {
		out := *b
		return "bound", &out
	}
	return "pending", nil
}

// Unbind clears the stored binding (面板解绑，等价于机器人收到「解绑」).
func Unbind() bool {
	bindMu.Lock()
	defer bindMu.Unlock()
	removed := false
	for username, binding := range bindings {
		if binding.UserOpenID != "" {
			delete(bindings, username)
			delete(openidUser, binding.UserOpenID)
			removed = true
		}
	}
	if removed {
		persistBindings()
		applyBindingToService(&QqBotBinding{})
	}
	return removed
}

// CredentialsConfigured reports whether appId+clientSecret are configured.
func CredentialsConfigured() bool {
	return QqBotShared().CurrentConfig().complete()
}

// BotInviteURL returns the configured bot invite URL (farm.qqBot.inviteUrl).
func BotInviteURL() string {
	return botInviteURL
}

var botInviteURL string

// SetBotInviteURL wires the configured invite URL (called at boot).
func SetBotInviteURL(u string) { botInviteURL = strings.TrimSpace(u) }

// HandleC2CMessage applies the bind/unbind protocol to one private message
// (rust complete_from_message). Returns (repliedText, handled).
func HandleC2CMessage(userOpenID, nickname, content string) (string, bool) {
	openid := strings.TrimSpace(userOpenID)
	if openid == "" {
		return "", false
	}
	if strings.TrimSpace(content) == "解绑" {
		bindMu.Lock()
		username, had := openidUser[openid]
		if had {
			delete(openidUser, openid)
			delete(bindings, username)
		}
		session := bindSession
		if session != nil && session.Username == username {
			bindSession = nil
		}
		bindMu.Unlock()
		if had {
			persistBindings()
			return "已解绑 QQ 下线通知。", true
		}
		return "", false
	}

	now := time.Now()
	bindMu.Lock()
	// 完成挂起的绑定会话（面板发起后，用户任意一条消息即完成）。
	if bindSession != nil && bindSession.ExpiresAt.After(now) {
		username := bindSession.Username
		binding := &QqBotBinding{UserOpenID: openid, Nickname: strings.TrimSpace(nickname), BoundAt: now.Unix()}
		bindings[username] = binding
		openidUser[openid] = username
		bindSession = nil
		bindMu.Unlock()
		persistBindings()
		applyBindingToService(binding)
		slog.Info("QQ Bot 绑定成功", "username", username, "openid", openid)
		return "绑定成功，后续账号下线会通知到这里。", true
	}
	bindMu.Unlock()
	return "", false
}

// applyBindingToService reconfigures the shared service with the new target.
func applyBindingToService(binding *QqBotBinding) {
	cfg := QqBotShared().CurrentConfig()
	if cfg.AppID == "" {
		return
	}
	QqBotShared().Configure(QqBotConfig{
		AppID: cfg.AppID, ClientSecret: cfg.ClientSecret, UserOpenID: binding.UserOpenID,
	})
}

// persistBindings writes bindings to the JSON file.
func persistBindings() {
	bindMu.Lock()
	out := make(map[string]QqBotBinding, len(bindings))
	for username, binding := range bindings {
		out[username] = *binding
	}
	path := bindFilePath
	bindMu.Unlock()
	if path == "" {
		return
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		slog.Warn("QQ Bot 绑定落盘失败", "err", err)
	}
}
