package runtime

// 账号通知（rust relogin_reminder 的 Go 侧实现）：下线 / 上线 / 应用宝授权失效
// 三种通知按系统配置的渠道推送；未配置渠道时回落 farm.pushWebhook。文案对齐
// rust notice_text（"账号 X 已下线 / 已上线 / 应用宝授权失效，请扫描二维码重新登录"），
// rust 设置页无分类型开关，渠道配置沿用现有 offline_reminder。

import (
	"encoding/json"
	"log/slog"
	"strconv"
	"strings"

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

// AccountNoticeKind mirrors rust relogin_reminder AccountNoticeKind.
type AccountNoticeKind string

const (
	// NoticeOffline 账号下线（默认）。
	NoticeOffline AccountNoticeKind = "offline"
	// NoticeOnline 账号上线（登录成功后触发，rust WorkerEvent::Started）。
	NoticeOnline AccountNoticeKind = "online"
	// NoticeYybQr 应用宝授权失效（rust YybQr）。
	NoticeYybQr AccountNoticeKind = "yyb_qr"
)

// accountNoticeLabel mirrors rust kind_label (下线/上线/应用宝授权二维码).
func accountNoticeLabel(kind AccountNoticeKind) string {
	switch kind {
	case NoticeOnline:
		return "上线"
	case NoticeYybQr:
		return "应用宝授权二维码"
	default:
		return "下线"
	}
}

// accountNoticeContent mirrors rust notice_text content per kind.
func accountNoticeContent(kind AccountNoticeKind, accountName, accountID string) string {
	accLabel := strings.TrimSpace(accountName)
	if accLabel == "" {
		accLabel = strings.TrimSpace(accountID)
	}
	if accLabel == "" {
		accLabel = "未知账号"
	}
	switch kind {
	case NoticeOnline:
		return "账号 " + accLabel + " 已上线"
	case NoticeYybQr:
		return "账号 " + accLabel + " 应用宝授权失效，请扫描二维码重新登录"
	default:
		return "账号 " + accLabel + " 已下线"
	}
}

// SendOfflineReminder pushes one offline notice via the configured provider
// (legacy entry: keeps the detailed failure reason as the push body).
func SendOfflineReminder(accountName, msg string) {
	cfg := loadOfflineReminderConfig()
	title := cfg.Title
	if accountName != "" {
		title = "农场账号 " + accountName + " 离线"
	}
	if msg == "" {
		msg = cfg.Msg
	}
	sendAccountNoticePush(cfg, title, msg, accountName, NoticeOffline, 0)
}

// SendAccountNotice pushes one account notice (offline/online/yyb_qr) via the
// configured provider with rust-aligned per-kind content.
func SendAccountNotice(kind AccountNoticeKind, accountID uint64, accountName string) {
	cfg := loadOfflineReminderConfig()
	display := strings.TrimSpace(accountName)
	if display == "" && accountID > 0 {
		display = strconv.FormatUint(accountID, 10)
	}
	sendAccountNoticePush(cfg, cfg.Title, accountNoticeContent(kind, accountName, strconv.FormatUint(accountID, 10)), display, kind, accountID)
}

// sendAccountNoticePush 按渠道推送（rust trigger_offline_reminder 的渠道分发）：
//   - qq_bot：YybQr 先申请登录码 + 启动扫码 watcher（闭环，relogin_watcher.go），
//     再发文本；文本成功后才推二维码图片（rust 同序，图片失败不回退重发文本）；
//   - ding_talk：仅文本（rust：钉钉文本消息不带二维码）；
//   - 默认：回落 farm.pushWebhook。
func sendAccountNoticePush(cfg offlineReminderConfig, title, msg, accountName string, kind AccountNoticeKind, accountID uint64) {
	switch cfg.Provider {
	case "qq_bot":
		if !push.QqBotShared().CurrentConfig().Complete() {
			// rust：「QQ 通知未绑定」。
			slog.Warn("account notice qqbot not bound", "account", accountName)
			return
		}
		// YybQr 闭环：先申请登录码并启动 watcher（推送失败也继续轮询，rust 同），
		// 拿到二维码图片 URL 后随文本推送一起发。
		var qrImageURL string
		if kind == NoticeYybQr {
			qrImageURL = StartYybQrRelogin(accountID, accountName)
		}
		if err := push.QqBotShared().SendText(title, msg); err != nil {
			slog.Warn("account notice qqbot send failed", "account", accountName, "err", err)
			return
		}
		slog.Info("账号通知发送成功", "account", accountName, "content", msg)
		if qrImageURL != "" {
			// 文本兜底已发出；图片失败仅记录（rust「应用宝授权二维码发送失败」）。
			if err := push.QqBotShared().SendQrImage(qrImageURL); err != nil {
				slog.Warn("应用宝授权二维码发送失败", "account", accountName, "err", err)
			} else {
				slog.Info("应用宝授权二维码发送成功", "account", accountName)
			}
		}
	case "ding_talk":
		if strings.TrimSpace(cfg.Endpoint) == "" && strings.TrimSpace(cfg.Token) == "" {
			return
		}
		if err := push.SendDingTalk(cfg.Endpoint, cfg.Token, cfg.Secret, title, msg); err != nil {
			slog.Warn("account notice dingtalk send failed", "account", accountName, "err", err)
		}
	default:
		webhook := vars.Config.GetString("farm.pushWebhook")
		if webhook != "" {
			push.NotifyAll(webhook, title, msg)
		}
	}
}
