// Package deviceprofile maps login OS hints to gateway User-Agent / device fields
// (aligned with qq-farm-bot DEVICE_PRESETS).
package deviceprofile

import "strings"

// Profile is the client fingerprint used for WS dial + LoginRequest.
type Profile struct {
	OS          string
	SysSoftware string
	Network     string
	Memory      string
	DeviceID    string
	UserAgent   string
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
