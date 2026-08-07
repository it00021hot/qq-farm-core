// Package loginurl extracts farm gateway auth code and client hints from login input.
package loginurl

import (
	"net/url"
	"regexp"
	"strings"
)

const fallbackGateHost = "gate-obt.nqf.qq.com"

// ClientHints are optional platform/os/ver parsed from a full login URL.
type ClientHints struct {
	Platform string
	OS       string
	Ver      string
}

var (
	reCode     = regexp.MustCompile(`(?i)[?&]code=([^&\s#]+)`)
	rePlatform = regexp.MustCompile(`(?i)[?&]platform=([^&\s#]+)`)
	reOS       = regexp.MustCompile(`(?i)[?&]os=([^&\s#]+)`)
	reVer      = regexp.MustCompile(`(?i)[?&]ver=([^&\s#]+)`)
	reHintKey  = regexp.MustCompile(`(?i)[?&](?:platform|os|ver|code)=`)
)

func decodeParam(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if out, err := url.QueryUnescape(raw); err == nil {
		return out
	}
	return raw
}

func toHref(rawInput string) string {
	raw := strings.TrimSpace(rawInput)
	if raw == "" {
		return ""
	}
	if matched, _ := regexp.MatchString(`(?i)^[a-z][a-z0-9+.-]*:`, raw); matched {
		return raw
	}
	if strings.HasPrefix(raw, "/") || strings.Contains(raw, "?") {
		if strings.HasPrefix(raw, "/") {
			return "wss://" + fallbackGateHost + raw
		}
		return "wss://" + fallbackGateHost + "/" + strings.TrimPrefix(raw, "/")
	}
	if reHintKey.MatchString(raw) {
		return "wss://" + fallbackGateHost + "/prod/ws?" + strings.TrimPrefix(raw, "?")
	}
	return ""
}

// ExtractCode returns the bare gateway auth code from a raw code or full login URL.
func ExtractCode(rawInput string) string {
	raw := strings.TrimSpace(rawInput)
	if raw == "" {
		return ""
	}

	if href := toHref(raw); href != "" {
		if u, err := url.Parse(href); err == nil {
			if code := decodeParam(u.Query().Get("code")); code != "" {
				return code
			}
		}
	}

	if m := reCode.FindStringSubmatch(raw); len(m) > 1 {
		return decodeParam(m[1])
	}

	// bare code: no whitespace / URL separators
	if !strings.ContainsAny(raw, " \t\r\n/?&=") {
		return raw
	}
	return ""
}

// ExtractClientHints parses platform/os/ver from a login URL (empty if bare code).
func ExtractClientHints(rawInput string) ClientHints {
	raw := strings.TrimSpace(rawInput)
	hints := ClientHints{}
	if raw == "" {
		return hints
	}

	if href := toHref(raw); href != "" {
		if u, err := url.Parse(href); err == nil {
			q := u.Query()
			hints.Platform = strings.ToLower(decodeParam(q.Get("platform")))
			hints.OS = decodeParam(q.Get("os"))
			hints.Ver = decodeParam(q.Get("ver"))
			return hints
		}
	}

	if m := rePlatform.FindStringSubmatch(raw); len(m) > 1 {
		hints.Platform = strings.ToLower(decodeParam(m[1]))
	}
	if m := reOS.FindStringSubmatch(raw); len(m) > 1 {
		hints.OS = decodeParam(m[1])
	}
	if m := reVer.FindStringSubmatch(raw); len(m) > 1 {
		hints.Ver = decodeParam(m[1])
	}
	return hints
}

// NormalizePlatform returns qq/wx or empty.
func NormalizePlatform(platform string) string {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "qq", "wx":
		return strings.ToLower(strings.TrimSpace(platform))
	default:
		return ""
	}
}
