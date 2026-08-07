package wxlogin

import "net/http/cookiejar"

// ScanStatus describes the current state of a WeChat QR login.
type ScanStatus string

const (
	ScanWaiting    ScanStatus = "waiting"
	ScanScanned    ScanStatus = "scanned"
	ScanAuthorized ScanStatus = "authorized"
	ScanCancelled  ScanStatus = "cancelled"
	ScanExpired    ScanStatus = "expired"
)

// Session contains all state associated with one WeChat QR login attempt.
type Session struct {
	Cookies     *cookiejar.Jar
	UUID        string
	OAuthCode   string
	OpenID      string
	LoginBuffer string
}
