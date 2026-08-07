package wxlogin

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

const (
	OAUTHAppID          = "wxd44977328b36e647"
	TargetMiniProgramID = "wx5306c5978fdb76e4"

	qrConnectURL         = "https://open.weixin.qq.com/connect/qrconnect"
	qrImageBase          = "https://open.weixin.qq.com/connect/qrcode/"
	qrPollURL            = "https://long.open.weixin.qq.com/connect/l/qrconnect"
	callbackURL          = "https://yybadaccess.3g.qq.com/pc_yyb/pcyyb_oauth"
	loginBufferURL       = "https://yybadaccess.3g.qq.com/pc_yyb_auth/pcyyb_get_wx_login_buffer_auth"
	loginBufferAccessKey = "wgrdg373hy26ww2"
	userAgent            = "Mozilla/5.0"
)

var (
	uuidPattern    = regexp.MustCompile(`/connect/qrcode/([^"'>\s]+)`)
	errCodePattern = regexp.MustCompile(`wx_errcode\s*=\s*(\d+)`)
	codePattern    = regexp.MustCompile(`wx_code\s*=\s*'([^']+)'`)
)

// WxLoginService performs the browser OAuth phase and native protocol phase.
type WxLoginService struct{}

func NewWxLoginService() *WxLoginService { return &WxLoginService{} }

func request(ctx context.Context, jar *cookiejar.Jar, rawURL, method string, body []byte, headers http.Header, timeout time.Duration) (int, []byte, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, rawURL, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	req.Header.Set("User-Agent", userAgent)
	client := &http.Client{Jar: jar}
	response, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	return response.StatusCode, data, err
}

// CreateQrSession starts an OAuth login and returns the QR image bytes.
func (s *WxLoginService) CreateQrSession(ctx context.Context) (*Session, []byte, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, nil, err
	}
	params := url.Values{
		"appid":         {OAUTHAppID},
		"redirect_uri":  {callbackURL + "?login_type=WX"},
		"response_type": {"code"},
		"scope":         {"snsapi_login,snsapi_runtime_pcsdk"},
		"state":         {"web"},
		"fast_login":    {"1"},
		"self_redirect": {"true"},
	}
	status, page, err := request(ctx, jar, qrConnectURL+"?"+params.Encode(), http.MethodGet, nil, nil, 35*time.Second)
	if err != nil {
		return nil, nil, err
	}
	if status < 200 || status >= 300 {
		return nil, nil, fmt.Errorf("unable to create WeChat QR session (HTTP %d)", status)
	}
	match := uuidPattern.FindSubmatch(page)
	if len(match) < 2 {
		return nil, nil, fmt.Errorf("unable to parse the WeChat QR session")
	}
	session := &Session{Cookies: jar, UUID: string(match[1])}
	status, qr, err := request(ctx, jar, qrImageBase+url.PathEscape(session.UUID), http.MethodGet, nil, nil, 35*time.Second)
	if err != nil {
		return nil, nil, err
	}
	if status < 200 || status >= 300 {
		return nil, nil, fmt.Errorf("unable to download WeChat QR image (HTTP %d)", status)
	}
	return session, qr, nil
}

// Poll waits for and interprets the next QR scanner state.
func (s *WxLoginService) Poll(ctx context.Context, session *Session) (ScanStatus, error) {
	if session.OAuthCode != "" {
		return ScanAuthorized, nil
	}
	params := url.Values{"uuid": {session.UUID}, "_": {strconv.FormatInt(time.Now().UnixMilli(), 10)}}
	status, data, err := request(ctx, session.Cookies, qrPollURL+"?"+params.Encode(), http.MethodGet, nil, nil, 35*time.Second)
	if err != nil {
		return "", err
	}
	if status < 200 || status >= 300 {
		return "", fmt.Errorf("WeChat QR polling failed (HTTP %d)", status)
	}
	match := errCodePattern.FindSubmatch(data)
	if len(match) < 2 {
		return "", fmt.Errorf("unrecognized WeChat QR polling response")
	}
	switch string(match[1]) {
	case "408":
		return ScanWaiting, nil
	case "404":
		return ScanScanned, nil
	case "403":
		return ScanCancelled, nil
	case "402":
		return ScanExpired, nil
	case "405":
		code := codePattern.FindSubmatch(data)
		if len(code) < 2 {
			return "", fmt.Errorf("WeChat authorization response did not include a code")
		}
		session.OAuthCode = string(code[1])
		return ScanAuthorized, nil
	default:
		return "", fmt.Errorf("unrecognized WeChat QR polling response")
	}
}

func requiredCookie(session *Session, name string) (string, error) {
	u, _ := url.Parse(callbackURL)
	for _, cookie := range session.Cookies.Cookies(u) {
		if cookie.Name == name && cookie.Value != "" {
			return cookie.Value, nil
		}
	}
	return "", fmt.Errorf("WeChat OAuth callback did not provide %s", name)
}

// Confirm exchanges the OAuth code for a native-protocol login buffer.
func (s *WxLoginService) Confirm(ctx context.Context, session *Session) (string, string, error) {
	if session.OAuthCode == "" {
		return "", "", fmt.Errorf("waiting for scan authorization")
	}
	params := url.Values{"login_type": {"WX"}, "code": {session.OAuthCode}, "state": {"web"}}
	status, _, err := request(ctx, session.Cookies, callbackURL+"?"+params.Encode(), http.MethodGet, nil, nil, 35*time.Second)
	if err != nil {
		return "", "", err
	}
	if status < 200 || status >= 400 {
		return "", "", fmt.Errorf("WeChat authorization callback failed (HTTP %d)", status)
	}
	openID, err := requiredCookie(session, "openid")
	if err != nil {
		return "", "", err
	}
	accessToken, err := requiredCookie(session, "accesstoken")
	if err != nil {
		return "", "", err
	}
	payload, err := json.Marshal(map[string]any{"extInfo": map[string]any{
		"listS": map[string]any{
			"unionid":      map[string]any{"value": []string{openID}},
			"user_id":      map[string]any{"value": []string{openID}},
			"access_token": map[string]any{"value": []string{accessToken}},
		},
		"listI": map[string]any{"user_type": map[string]any{"value": []int{0}}},
	}})
	if err != nil {
		return "", "", err
	}
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	var random [8]byte
	if _, err = rand.Read(random[:]); err != nil {
		return "", "", err
	}
	nonce := strconv.Itoa(1000 + int(binary.BigEndian.Uint64(random[:])%9000))
	sum := md5.Sum([]byte(string(payload) + timestamp + loginBufferAccessKey + nonce))
	headers := http.Header{
		"Content-Type":          {"application/json"},
		"Ual-Access-Businessid": {"pc_yyb_auth"},
		"Ual-Access-Timestamp":  {timestamp},
		"Ual-Access-Nonce":      {nonce},
		"Ual-Access-Signature":  {hex.EncodeToString(sum[:])},
	}
	status, data, err := request(ctx, session.Cookies, loginBufferURL, http.MethodPost, payload, headers, 35*time.Second)
	if err != nil {
		return "", "", err
	}
	if status < 200 || status >= 300 {
		return "", "", fmt.Errorf("unable to obtain WeChat login buffer (HTTP %d)", status)
	}
	var result struct {
		Code    int `json:"code"`
		ExtInfo struct {
			ListS struct {
				LoginBuffer struct {
					Value []string `json:"value"`
				} `json:"login_buffer"`
			} `json:"list_s"`
		} `json:"ext_info"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", "", err
	}
	if result.Code != 0 || len(result.ExtInfo.ListS.LoginBuffer.Value) == 0 || result.ExtInfo.ListS.LoginBuffer.Value[0] == "" {
		return "", "", fmt.Errorf("WeChat login buffer response is invalid")
	}
	loginBuffer := result.ExtInfo.ListS.LoginBuffer.Value[0]
	session.Cookies, _ = cookiejar.New(nil)
	session.OpenID = openID
	session.LoginBuffer = loginBuffer
	return openID, loginBuffer, nil
}

// IssueCode requests a wx.login code for appID over WeChat's native protocol.
func (s *WxLoginService) IssueCode(ctx context.Context, session *Session, appID string) (string, error) {
	if session.LoginBuffer == "" {
		return "", fmt.Errorf("WeChat login session has not been confirmed")
	}
	return getNativeWxLoginCode(ctx, session.LoginBuffer, appID)
}

// Destroy clears all secrets and cookies held by a login session.
func (s *WxLoginService) Destroy(session *Session) {
	if session == nil {
		return
	}
	session.Cookies, _ = cookiejar.New(nil)
	session.OAuthCode = ""
	session.OpenID = ""
	session.LoginBuffer = ""
}
