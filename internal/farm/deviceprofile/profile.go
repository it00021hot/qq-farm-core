// Package deviceprofile maps login OS hints to gateway User-Agent / device fields
// (aligned with qq-farm-bot DEVICE_PRESETS).
package deviceprofile

import "strings"

// Profile is the client fingerprint used for WS dial + LoginRequest.
// JSON 标签小驼峰（系统设置面板 deviceInfo 契约，对齐 rust DeviceInfo serde camelCase）。
type Profile struct {
	OS          string `json:"os"`
	SysSoftware string `json:"sysSoftware"`
	Network     string `json:"network"`
	Memory      string `json:"memory"`
	DeviceID    string `json:"deviceId"`
	UserAgent   string `json:"userAgent"`
}

var presets = []struct {
	aliases []string
	profile Profile
}{
	{
		aliases: []string{"windows", "win"},
		profile: Profile{
			OS:          "Windows",
			SysSoftware: "Windows",
			Network:     "wifi",
			Memory:      "16384",
			DeviceID:    "DESKTOP-PC<WPC>",
			UserAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36 MicroMessenger/7.0.20.1781(0x6700143B) NetType/WIFI MiniProgramEnv/Windows WindowsWechat/WMPF WindowsWechat(0x63090a13)",
		},
	},
	{
		aliases: []string{"os x", "osx", "mac", "macos", "mac os", "mac os x"},
		profile: Profile{
			OS:          "OS X",
			SysSoftware: "macOS 14.4",
			Network:     "wifi",
			Memory:      "16384",
			DeviceID:    "MacBookPro18,1",
			UserAgent:   "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) MiniProgramEnv/Mac MacWechat",
		},
	},
	{
		aliases: []string{"ios", "iphone", "ipad"},
		profile: Profile{
			OS:          "iOS",
			SysSoftware: "iPadOS 17.4",
			Network:     "wifi",
			Memory:      "16384",
			DeviceID:    "iPad14,6",
			UserAgent:   "Mozilla/5.0 (iPad; CPU OS 17_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 MicroMessenger/8.0.47(0x18002f2c) NetType/WIFI Language/zh_CN",
		},
	},
	{
		aliases: []string{"android"},
		profile: Profile{
			OS:          "Android",
			SysSoftware: "Android 14",
			Network:     "wifi",
			Memory:      "8192",
			DeviceID:    "Xiaomi 14",
			UserAgent:   "Mozilla/5.0 (Linux; Android 14; 23127PN0CC Build/UKQ1.231003.002) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/116.0.0.0 Mobile Safari/537.36 XWEB/1165009 MMWEBSDK/20240407 MiniProgramEnv/android MicroMessenger/8.0.49.2680(0x28003137) NetType/WIFI Language/zh_CN ABI/arm64",
		},
	},
}

// Resolve picks a device profile from login-url os hint (literal os kept when present).
// Preset is one named device fingerprint option for the settings UI
// （对齐 rust config::system_config::DevicePreset：id/name/description/deviceInfo）。
type Preset struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Profile     Profile `json:"deviceInfo"`
}

// devicePresets 与 rust device_presets() 顺序与数据一致（clientVersion 由
// 设置面板视图镜像顶层版本，预设本身不带）。
var devicePresets = []Preset{
	{
		ID: "windows_pc", Name: "Windows PC", Description: "Windows 微信PC客户端",
		Profile: Profile{
			OS: "Windows", SysSoftware: "Windows", Network: "wifi", Memory: "16384",
			DeviceID:  "DESKTOP-PC<WPC>",
			UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36 MicroMessenger/7.0.20.1781(0x6700143B) NetType/WIFI MiniProgramEnv/Windows WindowsWechat/WMPF WindowsWechat(0x63090a13)",
		},
	},
	{
		ID: "iphone_15_pro", Name: "iPhone 15 Pro", Description: "iPhone 15 Pro (iOS 17)",
		Profile: Profile{
			OS: "iOS", SysSoftware: "iOS 17.4.1", Network: "wifi", Memory: "7672",
			DeviceID:  "iPhone15,2",
			UserAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 17_4_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 MicroMessenger/8.0.47(0x18002f2c) NetType/WIFI Language/zh_CN",
		},
	},
	{
		ID: "iphone_16_pro", Name: "iPhone 16 Pro", Description: "iPhone 16 Pro (iOS 18)",
		Profile: Profile{
			OS: "iOS", SysSoftware: "iOS 18.2.1", Network: "wifi", Memory: "8192",
			DeviceID:  "iPhone17,1",
			UserAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 18_2_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 MicroMessenger/8.0.54(0x28003137) NetType/WIFI Language/zh_CN",
		},
	},
	{
		ID: "android_xiaomi", Name: "小米手机", Description: "小米/Redmi (Android 14)",
		Profile: Profile{
			OS: "Android", SysSoftware: "Android 14", Network: "wifi", Memory: "8192",
			DeviceID:  "Xiaomi 14",
			UserAgent: "Mozilla/5.0 (Linux; Android 14; 23127PN0CC Build/UKQ1.231003.002) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/116.0.0.0 Mobile Safari/537.36 XWEB/1165009 MMWEBSDK/20240407 MiniProgramEnv/android MicroMessenger/8.0.49.2680(0x28003137) NetType/WIFI Language/zh_CN ABI/arm64",
		},
	},
	{
		ID: "android_huawei", Name: "华为手机", Description: "华为 (Android 14)",
		Profile: Profile{
			OS: "Android", SysSoftware: "Android 14", Network: "wifi", Memory: "12288",
			DeviceID:  "HUAWEI Mate 60",
			UserAgent: "Mozilla/5.0 (Linux; Android 14; ALN-AL10 Build/HUAWEIALN-AL10) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/116.0.0.0 Mobile Safari/537.36 XWEB/1165009 MMWEBSDK/20240407 MiniProgramEnv/android MicroMessenger/8.0.49.2680(0x28003137) NetType/WIFI Language/zh_CN ABI/arm64",
		},
	},
	{
		ID: "ipad_pro", Name: "iPad Pro", Description: "iPad Pro 12.9 (iPadOS 17)",
		Profile: Profile{
			OS: "iOS", SysSoftware: "iPadOS 17.4", Network: "wifi", Memory: "16384",
			DeviceID:  "iPad14,6",
			UserAgent: "Mozilla/5.0 (iPad; CPU OS 17_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 MicroMessenger/8.0.47(0x18002f2c) NetType/WIFI Language/zh_CN",
		},
	},
}

// Presets exposes the built-in device fingerprint presets (settings 系统设置).
func Presets() []Preset {
	out := make([]Preset, len(devicePresets))
	copy(out, devicePresets)
	return out
}

func Resolve(osHint string) Profile {
	needle := strings.ToLower(strings.TrimSpace(osHint))
	base := presets[0].profile // default Windows like bot
	if needle == "" {
		return base
	}
	for _, p := range presets {
		for _, a := range p.aliases {
			if a == needle {
				out := p.profile
				// Keep game-side literal (e.g. "OS X") from URL when provided
				if osHint != "" {
					out.OS = strings.TrimSpace(osHint)
				}
				return out
			}
		}
	}
	out := base
	out.OS = strings.TrimSpace(osHint)
	out.SysSoftware = out.OS
	return out
}

// DefaultGatewayOrigin is required by the farm WSS gateway.
const DefaultGatewayOrigin = "https://gate-obt.nqf.qq.com"
