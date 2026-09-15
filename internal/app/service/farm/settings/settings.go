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
	defaultClientVer = "1.14.0.4_20260911"
	// defaultClientVerUpdatedAtMs: 默认客户端版本的发布时间（毫秒）。
	// 保存的版本只有在其时间戳**更新**时才沿用，防止旧存档把升级链锁死在
	// 过期版本上（rust DEFAULT_CLIENT_VERSION_UPDATED_AT / bot resolveClientVersion）。
	defaultClientVerUpdatedAtMs int64 = 1_789_352_998_016
	defaultTimeZone                   = "Asia/Shanghai"
	defaultPlatform                   = "qq"
	defaultOS                         = "Windows"
)

// DefaultClientVerUpdatedAt exposes the default version publish timestamp.
func DefaultClientVerUpdatedAt() int64 { return defaultClientVerUpdatedAtMs }

// ResolveClientVersion mirrors rust resolve_client_version: 保存值只有在其
// updatedAt 比默认值新（严格大于）时才沿用，否则回默认版本与默认时间戳。
func ResolveClientVersion(savedVersion string, savedUpdatedAt int64) (string, int64) {
	version := strings.TrimSpace(savedVersion)
	if version != "" && savedUpdatedAt > defaultClientVerUpdatedAtMs {
		return version, savedUpdatedAt
	}
	return defaultClientVer, defaultClientVerUpdatedAtMs
}

// ResolveClientVersionUpdatedAt mirrors rust resolve_client_version_updated_at:
// 显式传入的时间戳优先；版本发生变化记 now；否则保留当前值（缺省回默认）。
func ResolveClientVersionUpdatedAt(clientVersion, currentVersion string, currentUpdatedAt, requestedUpdatedAt, nowMs int64) int64 {
	if requestedUpdatedAt > 0 {
		return requestedUpdatedAt
	}
	if strings.TrimSpace(clientVersion) != strings.TrimSpace(currentVersion) {
		return nowMs
	}
	if currentUpdatedAt > 0 {
		return currentUpdatedAt
	}
	return defaultClientVerUpdatedAtMs
}

// Service exposes settings-panel helpers.
type Service struct{}

// SystemConfigPayload mirrors rust SystemConfigPayload.
type SystemConfigPayload struct {
	ServerURL     string `json:"serverUrl"`
	ClientVersion string `json:"clientVersion"`
	// ClientVersionUpdatedAt 客户端版本的保存时间（毫秒）；决定保存版本能否
	// 覆盖更新的默认值（rust clientVersionUpdatedAt）。
	ClientVersionUpdatedAt int64                 `json:"clientVersionUpdatedAt"`
	Platform               string                `json:"platform"`
	OS                     string                `json:"os"`
	TimeZone               string                `json:"timeZone"`
	DeviceInfo             deviceprofile.Profile `json:"deviceInfo"`
}

// systemDeviceView 是 deviceInfo 的对外视图：小驼峰键 + 镜像顶层 clientVersion
// （rust 桌面端契约，web 设置页的输入框按此结构渲染/回填）。
type systemDeviceView struct {
	OS            string `json:"os"`
	ClientVersion string `json:"clientVersion"`
	SysSoftware   string `json:"sysSoftware"`
	Network       string `json:"network"`
	Memory        string `json:"memory"`
	DeviceID      string `json:"deviceId"`
	UserAgent     string `json:"userAgent"`
}

// systemPayloadView 是 SystemConfigPayload 的对外视图。
type systemPayloadView struct {
	ServerURL              string           `json:"serverUrl"`
	ClientVersion          string           `json:"clientVersion"`
	ClientVersionUpdatedAt int64            `json:"clientVersionUpdatedAt"`
	Platform               string           `json:"platform"`
	OS                     string           `json:"os"`
	TimeZone               string           `json:"timeZone"`
	DeviceInfo             systemDeviceView `json:"deviceInfo"`
}

// SystemConfigPanel 是系统配置 GET/SAVE/RESET 的统一返回（rust
// get_settings_panel 的 {default, saved} 契约；前端取 saved 回显、default 供重置）。
type SystemConfigPanel struct {
	Default systemPayloadView `json:"default"`
	Saved   systemPayloadView `json:"saved"`
}

func systemPayloadViewOf(p SystemConfigPayload) systemPayloadView {
	return systemPayloadView{
		ServerURL:              p.ServerURL,
		ClientVersion:          p.ClientVersion,
		ClientVersionUpdatedAt: p.ClientVersionUpdatedAt,
		Platform:               p.Platform,
		OS:                     p.OS,
		TimeZone:               p.TimeZone,
		DeviceInfo: systemDeviceView{
			OS:            p.DeviceInfo.OS,
			ClientVersion: p.ClientVersion,
			SysSoftware:   p.DeviceInfo.SysSoftware,
			Network:       p.DeviceInfo.Network,
			Memory:        p.DeviceInfo.Memory,
			DeviceID:      p.DeviceInfo.DeviceID,
			UserAgent:     p.DeviceInfo.UserAgent,
		},
	}
}

// OfflineReminder mirrors rust global_config.OfflineReminder.
type OfflineReminder struct {
	Provider     string `json:"provider"` // none | qqBot | wechatBot | dingTalk
	QQBotBinding *struct {
		UserOpenID string `json:"userOpenid"`
		Nickname   string `json:"nickname"`
		BoundAt    int64  `json:"boundAt"`
	} `json:"qqBotBinding,omitempty"`
	Title    string `json:"title"`
	Msg      string `json:"msg"`
	Endpoint string `json:"endpoint"`
	Token    string `json:"token"`
	Secret   string `json:"secret"`
}

func defaultOfflineReminder() OfflineReminder {
	return OfflineReminder{
		Provider: "none",
		Title:    "农场账号离线",
		Msg:      "账号已离线，请打开面板重新登录",
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

// resolveSystemConfig merges stored overrides over defaults（serverUrl/版本守卫/
// 设备指纹等兜底链）。
func resolveSystemConfig(stored SystemConfigPayload) SystemConfigPayload {
	if stored.ClientVersion == "" {
		stored.ClientVersion = vars.Config.GetString("farm.clientVersion")
	}
	// 版本守卫（rust bootstrap resolve_client_version）：保存的版本只有在其
	// updatedAt 比默认值新时才沿用，否则回默认。
	stored.ClientVersion, stored.ClientVersionUpdatedAt =
		ResolveClientVersion(stored.ClientVersion, stored.ClientVersionUpdatedAt)
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
	return stored
}

// SystemConfig returns the system config panel: pure defaults + stored-merged view.
func (Service) SystemConfig(ctx fiber.Ctx) (SystemConfigPanel, error) {
	stored := SystemConfigPayload{}
	_ = readKey(systemConfigKey, &stored)
	saved := resolveSystemConfig(stored)
	def := resolveSystemConfig(SystemConfigPayload{})
	return SystemConfigPanel{Default: systemPayloadViewOf(def), Saved: systemPayloadViewOf(saved)}, nil
}

// SaveSystemConfig persists the payload and applies the live knobs.
func (Service) SaveSystemConfig(ctx fiber.Ctx, payload SystemConfigPayload) (SystemConfigPanel, error) {
	// rust set_system_config 语义：顶层版本为空不拦截，沿用手头已存版本
	//（仍为空则读取侧 resolveSystemConfig 回默认版本）。
	current := SystemConfigPayload{}
	_ = readKey(systemConfigKey, &current)
	if strings.TrimSpace(payload.ClientVersion) == "" {
		payload.ClientVersion = current.ClientVersion
	}
	// 版本时间戳规则（rust set_system_config resolve_client_version_updated_at）：
	// 显式传入优先；版本变化记 now；未变化保留当前值。
	payload.ClientVersionUpdatedAt = ResolveClientVersionUpdatedAt(
		payload.ClientVersion,
		current.ClientVersion,
		current.ClientVersionUpdatedAt,
		payload.ClientVersionUpdatedAt,
		time.Now().UnixMilli(),
	)
	if err := writeKey(systemConfigKey, "系统配置", payload); err != nil {
		return SystemConfigPanel{}, err
	}
	// 立即生效：登录版本走 vars.Config 兜底链。
	if strings.TrimSpace(payload.ClientVersion) != "" {
		vars.Config.Set("farm.clientVersion", strings.TrimSpace(payload.ClientVersion))
	}
	saved := resolveSystemConfig(payload)
	def := resolveSystemConfig(SystemConfigPayload{})
	return SystemConfigPanel{Default: systemPayloadViewOf(def), Saved: systemPayloadViewOf(saved)}, nil
}

// ResetSystemConfig clears overrides and returns defaults.
func (Service) ResetSystemConfig(ctx fiber.Ctx) (SystemConfigPanel, error) {
	if err := deleteKey(systemConfigKey); err != nil {
		return SystemConfigPanel{}, err
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
