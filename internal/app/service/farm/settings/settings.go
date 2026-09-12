package settings

// 设置面板扩展：系统配置（协议版本/设备指纹/时区）与离线提醒。
// 对齐 rust 桌面端 admin.rs（system_config）与 settings.rs（offline_reminder），
// 存储落在 cn_farm_system_config（config_key: system / offline_reminder）。

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"

	"github.com/it00021hot/qq-farm-core/internal/app/model"
	"github.com/it00021hot/qq-farm-core/internal/farm/deviceprofile"
	"github.com/it00021hot/qq-farm-core/internal/farm/push"
	"github.com/it00021hot/qq-farm-core/internal/vars"
)

const (
	systemConfigKey  = "system"
	offlineRemindKey = "offline_reminder"
	defaultClientVer = "1.14.0.3_20260909"
	defaultTimeZone  = "Asia/Shanghai"
	defaultPlatform  = "qq"
	defaultOS        = "Windows"
)

// Service exposes settings-panel helpers.
type Service struct{}

// SystemConfigPayload mirrors rust SystemConfigPayload.
type SystemConfigPayload struct {
	ServerURL     string                `json:"serverUrl"`
	ClientVersion string                `json:"clientVersion"`
	Platform      string                `json:"platform"`
	OS            string                `json:"os"`
	TimeZone      string                `json:"timeZone"`
	DeviceInfo    deviceprofile.Profile `json:"deviceInfo"`
}

// OfflineReminder mirrors rust global_config.OfflineReminder.
type OfflineReminder struct {
	Provider     string `json:"provider"` // none | qqBot | wechatBot | dingTalk
	QQBotBinding *struct {
		UserOpenID string `json:"userOpenid"`
		Nickname   string `json:"nickname"`
		BoundAt    int64  `json:"boundAt"`
	} `json:"qqBotBinding,omitempty"`
	Title            string `json:"title"`
	Msg              string `json:"msg"`
	OfflineDeleteSec int64  `json:"offlineDeleteSec"`
	Endpoint         string `json:"endpoint"`
	Token            string `json:"token"`
	Secret           string `json:"secret"`
}

func defaultOfflineReminder() OfflineReminder {
	return OfflineReminder{
		Provider:         "none",
		Title:            "农场账号离线",
		Msg:              "账号已离线，请打开面板重新登录",
		OfflineDeleteSec: 300,
	}
}

// db returns the shared DB handle (nil-safe for unit contexts).
func db() *gorm.DB { return vars.DB }

// readKey reads one config key's JSON payload from cn_farm_system_config.
func readKey(key string, out any) error {
	db := db()
	if db == nil {
		return nil
	}
	var row model.FarmSystemConfig
	if err := db.Where("config_key = ?", key).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if strings.TrimSpace(row.ConfigJSON) == "" {
		return nil
	}
	return json.Unmarshal([]byte(row.ConfigJSON), out)
}

// writeKey upserts one config key's JSON payload.
func writeKey(key, remark string, payload any) error {
	db := db()
	if db == nil {
		return errors.New("数据库未初始化")
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	now := uint(time.Now().Unix())
	var row model.FarmSystemConfig
	err = db.Where("config_key = ?", key).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row = model.FarmSystemConfig{
			ConfigKey: key, ConfigJSON: string(raw), Remark: remark, CreatedAt: now, UpdatedAt: now,
		}
		return db.Create(&row).Error
	}
	if err != nil {
		return err
	}
	return db.Model(&row).Updates(map[string]any{"config_json": string(raw), "updated_at": now}).Error
}

// deleteKey removes one config key.
func deleteKey(key string) error {
	db := db()
	if db == nil {
		return nil
	}
	return db.Where("config_key = ?", key).Delete(&model.FarmSystemConfig{}).Error
}

// SystemConfig returns the stored system config merged over defaults.
func (Service) SystemConfig(ctx fiber.Ctx) (SystemConfigPayload, error) {
	stored := SystemConfigPayload{}
	_ = readKey(systemConfigKey, &stored)
	if stored.ClientVersion == "" {
		stored.ClientVersion = vars.Config.GetString("farm.clientVersion")
	}
	if stored.ClientVersion == "" {
		stored.ClientVersion = defaultClientVer
	}
	if stored.ServerURL == "" {
		stored.ServerURL = vars.Config.GetString("farm.gatewayURL")
	}
	if stored.Platform == "" {
		stored.Platform = defaultPlatform
	}
	if stored.OS == "" {
		stored.OS = defaultOS
	}
	if stored.TimeZone == "" {
		stored.TimeZone = defaultTimeZone
	}
	if stored.DeviceInfo == (deviceprofile.Profile{}) {
		stored.DeviceInfo = deviceprofile.Resolve(stored.OS)
	}
	return stored, nil
}

// SaveSystemConfig persists the payload and applies the live knobs.
func (Service) SaveSystemConfig(ctx fiber.Ctx, payload SystemConfigPayload) (SystemConfigPayload, error) {
	if strings.TrimSpace(payload.ClientVersion) == "" {
		return payload, errors.New("clientVersion 不能为空")
	}
	if err := writeKey(systemConfigKey, "系统配置", payload); err != nil {
		return payload, err
	}
	// 立即生效：登录版本走 vars.Config 兜底链。
	if strings.TrimSpace(payload.ClientVersion) != "" {
		vars.Config.Set("farm.clientVersion", strings.TrimSpace(payload.ClientVersion))
	}
	return payload, nil
}

// ResetSystemConfig clears overrides and returns defaults.
func (Service) ResetSystemConfig(ctx fiber.Ctx) (SystemConfigPayload, error) {
	if err := deleteKey(systemConfigKey); err != nil {
		return SystemConfigPayload{}, err
	}
	vars.Config.Set("farm.clientVersion", defaultClientVer)
	return Service{}.SystemConfig(ctx)
}

// DevicePresets lists the built-in device fingerprints.
func (Service) DevicePresets(ctx fiber.Ctx) []deviceprofile.Preset {
	return deviceprofile.Presets()
}

// OfflineReminder returns the stored reminder config (defaults when absent).
func (Service) OfflineReminder(ctx fiber.Ctx) (OfflineReminder, error) {
	stored := defaultOfflineReminder()
	if err := readKey(offlineRemindKey, &stored); err != nil {
		return defaultOfflineReminder(), nil
	}
	base := defaultOfflineReminder()
	if stored.Provider == "" {
		stored.Provider = base.Provider
	}
	if stored.Title == "" {
		stored.Title = base.Title
	}
	if stored.Msg == "" {
		stored.Msg = base.Msg
	}
	if stored.OfflineDeleteSec <= 0 {
		stored.OfflineDeleteSec = base.OfflineDeleteSec
	}
	// 绑定状态实时来自 qqbot-binding.json（与 QQ 机器人绑定页共用状态）。
	if binding := push.CurrentBinding(); binding != nil {
		stored.QQBotBinding = &struct {
			UserOpenID string `json:"userOpenid"`
			Nickname   string `json:"nickname"`
			BoundAt    int64  `json:"boundAt"`
		}{UserOpenID: binding.UserOpenID, Nickname: binding.Nickname, BoundAt: binding.BoundAt}
	}
	return stored, nil
}

// SaveOfflineReminder persists the reminder config.
func (Service) SaveOfflineReminder(ctx fiber.Ctx, cfg OfflineReminder) (OfflineReminder, error) {
	cfg.QQBotBinding = nil // 绑定状态由 QQ 机器人绑定流程维护
	if err := writeKey(offlineRemindKey, "离线提醒", cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// TestOfflineReminder sends a test notification via the configured provider
// (rust test_offline_reminder 语义：wechatBot 未实现；qqBot 需先绑定；钉钉 endpoint/token 二选一).
func (Service) TestOfflineReminder(ctx fiber.Ctx, cfg OfflineReminder) (map[string]any, error) {
	reminder, err := Service{}.OfflineReminder(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.Provider) != "" {
		reminder.Provider = cfg.Provider
	}
	reminder.Endpoint, reminder.Token, reminder.Secret = cfg.Endpoint, cfg.Token, cfg.Secret
	reminder.Title, reminder.Msg = cfg.Title, cfg.Msg
	switch reminder.Provider {
	case "wechat_bot":
		return map[string]any{"ok": false, "code": "not_implemented", "msg": "微信机器人暂未实现"}, nil
	case "ding_talk":
		if strings.TrimSpace(reminder.Endpoint) == "" && strings.TrimSpace(reminder.Token) == "" {
			return map[string]any{"ok": false, "code": "missing_endpoint", "msg": "请填写钉钉 Webhook 地址或 Access Token"}, nil
		}
		if err := push.SendDingTalk(reminder.Endpoint, reminder.Token, reminder.Secret, "测试通知", "这是一条来自 QQ Farm 面板的钉钉测试消息"); err != nil {
			return map[string]any{"ok": false, "code": "send_failed", "msg": err.Error()}, nil
		}
		return map[string]any{"ok": true, "msg": "钉钉测试消息已发送"}, nil
	case "qq_bot":
		if err := push.QqBotShared().SendText("测试通知", "这是一条来自 QQ Farm 面板的测试消息"); err != nil {
			return map[string]any{"ok": false, "code": "send_failed", "msg": err.Error()}, nil
		}
		return map[string]any{"ok": true, "msg": "QQ 机器人测试消息已发送"}, nil
	default:
		return map[string]any{"ok": false, "code": "not_configured", "msg": "未启用任何通知渠道"}, nil
	}
}
