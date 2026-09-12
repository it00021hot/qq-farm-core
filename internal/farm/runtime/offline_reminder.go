package runtime

// 离线提醒（rust settings.rs test_offline_reminder / relogin_reminder 的 Go 侧实现）：
// 账号异常停止/掉线时按系统配置的渠道推送；未配置渠道时回落 farm.pushWebhook。

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/app/model"
	"github.com/it00021hot/qq-farm-core/internal/farm/push"
	"github.com/it00021hot/qq-farm-core/internal/vars"

	"gorm.io/gorm"
)

type offlineReminderConfig struct {
	Provider string `json:"provider"`
	Title    string `json:"title"`
	Msg      string `json:"msg"`
	Endpoint string `json:"endpoint"`
	Token    string `json:"token"`
	Secret   string `json:"secret"`
}

func loadOfflineReminderConfig() offlineReminderConfig {
	cfg := offlineReminderConfig{Provider: "none", Title: "农场账号离线", Msg: "账号已离线，请打开面板重新登录"}
	db := vars.DB
	if db == nil {
		return cfg
	}
	var row model.FarmSystemConfig
	if err := db.Where("config_key = ?", "offline_reminder").First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return cfg
		}
		return cfg
	}
	_ = json.Unmarshal([]byte(row.ConfigJSON), &cfg)
	if cfg.Provider == "" {
		cfg.Provider = "none"
	}
	if cfg.Title == "" {
		cfg.Title = "农场账号离线"
	}
	if cfg.Msg == "" {
		cfg.Msg = "账号已离线，请打开面板重新登录"
	}
	return cfg
}

// SendOfflineReminder pushes one offline notice via the configured provider.
func SendOfflineReminder(accountName, msg string) {
	cfg := loadOfflineReminderConfig()
	title := cfg.Title
	if accountName != "" {
		title = "农场账号 " + accountName + " 离线"
	}
	if msg == "" {
		msg = cfg.Msg
	}
	callCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = callCtx
	switch cfg.Provider {
	case "qqBot":
		if err := push.QqBotShared().SendText(title, msg); err != nil {
			slog.Warn("offline reminder qqbot send failed", "account", accountName, "err", err)
		}
	case "dingTalk":
		if strings.TrimSpace(cfg.Endpoint) == "" && strings.TrimSpace(cfg.Token) == "" {
			return
		}
		if err := push.SendDingTalk(cfg.Endpoint, cfg.Token, cfg.Secret, title, msg); err != nil {
			slog.Warn("offline reminder dingtalk send failed", "account", accountName, "err", err)
		}
	default:
		webhook := vars.Config.GetString("farm.pushWebhook")
		if webhook != "" {
			push.NotifyAll(webhook, title, msg)
		}
	}
}
