package wxlogin

import (
	"strings"
	"time"
)

const (
	DefaultExpiresIn         int64 = 7200
	WxKeepaliveInterval            = 30 * time.Minute
	WxKeepaliveAheadSecs     int64 = 45 * 60
	WxRefreshTokenRescanSecs int64 = 25 * 24 * 60 * 60
)

// WxAuthErrorKind classifies Yingyongbao refresh/mint failures.
type WxAuthErrorKind int

const (
	WxAuthErrorTransient WxAuthErrorKind = iota
	WxAuthErrorCredentialsDead
)

// WxAuthError is returned by OAuth refresh and mint paths.
type WxAuthError struct {
	Kind    WxAuthErrorKind
	Message string
}

func (e WxAuthError) Error() string {
	if e.Message == "" {
		return "Yingyongbao authorization error"
	}
	return e.Message
}

func wxAuthDead(msg string) WxAuthError {
	return WxAuthError{Kind: WxAuthErrorCredentialsDead, Message: msg}
}

func wxAuthTransient(msg string) WxAuthError {
	return WxAuthError{Kind: WxAuthErrorTransient, Message: msg}
}

// YybCredentials is the persisted Yingyongbao ticket bundle.
type YybCredentials struct {
	OpenID                 string
	AccessToken            string
	RefreshToken           string
	LoginBuffer            string
	ExpiresAt              int64
	ExpiresIn              int64
	RefreshTokenObservedAt int64
}

func (c YybCredentials) TokenDueForRefresh(aheadSecs int64) bool {
	if c.ExpiresAt <= 0 {
		return true
	}
	if aheadSecs < 0 {
		aheadSecs = 0
	}
	return time.Now().Unix()+aheadSecs >= c.ExpiresAt
}

func (c YybCredentials) ToWxAuth() WxAuth {
	return WxAuth{
		OpenID:       c.OpenID,
		LoginBuffer:  c.LoginBuffer,
		AccessToken:  c.AccessToken,
		RefreshToken: c.RefreshToken,
		ExpiresAt:    c.ExpiresAt,
	}
}

// ApplyNewRefreshToken keeps observed_at unless WeChat returns a different refresh_token.
func (c YybCredentials) ApplyNewRefreshToken(newRefresh string, now int64) YybCredentials {
	newRefresh = strings.TrimSpace(newRefresh)
	if newRefresh != "" && newRefresh != c.RefreshToken {
		c.RefreshToken = newRefresh
		c.RefreshTokenObservedAt = now
	}
	return c.EnsureObservedAt(now)
}

// EnsureObservedAt starts the refresh_token clock on first persist.
func (c YybCredentials) EnsureObservedAt(now int64) YybCredentials {
	if strings.TrimSpace(c.RefreshToken) != "" && c.RefreshTokenObservedAt <= 0 {
		c.RefreshTokenObservedAt = now
	}
	return c
}

// RescanRecommended is true after ~25 days on the same refresh_token.
func RescanRecommended(refreshToken string, observedAt, now int64) bool {
	if strings.TrimSpace(refreshToken) == "" || observedAt <= 0 {
		return false
	}
	return now-observedAt >= WxRefreshTokenRescanSecs
}

func classifyYybMessage(msg string) WxAuthErrorKind {
	m := strings.TrimSpace(msg)
	switch {
	case strings.Contains(m, "WeChat login buffer response is invalid"),
		strings.Contains(m, "Missing Yingyongbao authorization"),
		strings.Contains(m, "WeChat login session has not been confirmed"),
		strings.Contains(m, "refresh failed"),
		strings.Contains(m, "refresh response missing"),
		strings.Contains(m, "invalid quick authorization"),
		strings.Contains(m, "quick authorization code is missing"),
		strings.Contains(m, "missing refresh token"):
		return WxAuthErrorCredentialsDead
	default:
		return WxAuthErrorTransient
	}
}
