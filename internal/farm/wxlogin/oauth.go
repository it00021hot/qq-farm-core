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
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	OAUTHAppID          = "wxd44977328b36e647"
	TargetMiniProgramID = "wx5306c5978fdb76e4"
	OAuthScope          = "snsapi_login,snsapi_runtime_pcsdk"
	OAuthState          = "web"
	OAuthRedirectURI    = "https://yybadaccess.3g.qq.com/pc_yyb/pcyyb_oauth?login_type=WX"

	qrConnectURL         = "https://open.weixin.qq.com/connect/qrconnect"
	qrImageBase          = "https://open.weixin.qq.com/connect/qrcode/"
	qrPollURL            = "https://long.open.weixin.qq.com/connect/l/qrconnect"
	callbackURL          = "https://yybadaccess.3g.qq.com/pc_yyb/pcyyb_oauth"
	loginBufferURL       = "https://yybadaccess.3g.qq.com/pc_yyb_auth/pcyyb_get_wx_login_buffer_auth"
	refreshTokenURL      = "https://yybadaccess.3g.qq.com/pc_yyb_auth/pcyyb_refresh_token_auth"
	loginBufferAccessKey = "wgrdg373hy26ww2"
	userAgent            = "Mozilla/5.0"
)

var DesktopWechatPorts = []uint16{14013, 14014, 14015, 13013, 13014, 13015}

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
	client := &http.Client{}
	if jar != nil {
		client.Jar = jar
	}
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
		"scope":         {OAuthScope},
		"state":         {OAuthState},
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

func optionalCookie(session *Session, name string) string {
	u, _ := url.Parse(callbackURL)
	for _, cookie := range session.Cookies.Cookies(u) {
		if cookie.Name == name {
			return cookie.Value
		}
	}
	return ""
}

// Confirm exchanges the OAuth code for a native-protocol login buffer.
func (s *WxLoginService) Confirm(ctx context.Context, session *Session) (string, string, error) {
	if session.OAuthCode == "" {
		return "", "", fmt.Errorf("waiting for scan authorization")
	}
	creds, err := s.ExchangeOAuthCode(ctx, session.OAuthCode)
	if err != nil {
		return "", "", err
	}
	session.Cookies, _ = cookiejar.New(nil)
	session.OpenID = creds.OpenID
	session.AccessToken = creds.AccessToken
	session.RefreshToken = creds.RefreshToken
	session.LoginBuffer = creds.LoginBuffer
	session.ExpiresAt = creds.ExpiresAt
	session.ExpiresIn = creds.ExpiresIn
	return creds.OpenID, creds.LoginBuffer, nil
}

// ExchangeOAuthCode exchanges a WeChat OAuth code for Yingyongbao credentials.
func (s *WxLoginService) ExchangeOAuthCode(ctx context.Context, oauthCode string) (YybCredentials, error) {
	code := strings.TrimSpace(oauthCode)
	if code == "" {
		return YybCredentials{}, wxAuthDead("quick authorization code is missing")
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return YybCredentials{}, wxAuthTransient(err.Error())
	}
	params := url.Values{"login_type": {"WX"}, "code": {code}, "state": {OAuthState}}
	status, _, err := request(ctx, jar, callbackURL+"?"+params.Encode(), http.MethodGet, nil, nil, 35*time.Second)
	if err != nil {
		return YybCredentials{}, wxAuthTransient(err.Error())
	}
	if status < 200 || status >= 400 {
		return YybCredentials{}, wxAuthTransient(fmt.Sprintf("WeChat authorization callback failed (HTTP %d)", status))
	}
	openID, err := requiredCookie(&Session{Cookies: jar}, "openid")
	if err != nil {
		return YybCredentials{}, wxAuthDead(err.Error())
	}
	accessToken, err := requiredCookie(&Session{Cookies: jar}, "accesstoken")
	if err != nil {
		return YybCredentials{}, wxAuthDead(err.Error())
	}
	refreshToken := optionalCookie(&Session{Cookies: jar}, "refreshtoken")
	expiresIn := DefaultExpiresIn
	if raw := optionalCookie(&Session{Cookies: jar}, "expires_in"); raw != "" {
		if parsed, parseErr := strconv.ParseInt(raw, 10, 64); parseErr == nil && parsed > 0 {
			expiresIn = parsed
		}
	}
	creds := YybCredentials{
		OpenID:       openID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Unix() + expiresIn,
		ExpiresIn:    expiresIn,
	}
	creds = creds.EnsureObservedAt(time.Now().Unix())
	loginBuffer, err := s.postLoginBuffer(ctx, jar, &creds)
	if err != nil {
		return YybCredentials{}, err
	}
	creds.LoginBuffer = loginBuffer
	return creds, nil
}

// ParseQuickRedirectURL validates a desktop WeChat fast_login redirect and extracts the OAuth code.
func ParseQuickRedirectURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", wxAuthDead("invalid quick authorization redirect")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", wxAuthDead("invalid quick authorization redirect")
	}
	if parsed.Scheme != "https" {
		return "", wxAuthDead("invalid quick authorization redirect")
	}
	if parsed.Hostname() != "yybadaccess.3g.qq.com" {
		return "", wxAuthDead("invalid quick authorization redirect")
	}
	if parsed.Port() != "" || parsed.User != nil {
		return "", wxAuthDead("invalid quick authorization redirect")
	}
	if parsed.Path != "/pc_yyb/pcyyb_oauth" {
		return "", wxAuthDead("invalid quick authorization redirect")
	}
	q := parsed.Query()
	if q.Get("login_type") != "WX" || q.Get("state") != OAuthState {
		return "", wxAuthDead("invalid quick authorization state")
	}
	code := strings.TrimSpace(q.Get("code"))
	if code == "" || len(code) > 2048 {
		return "", wxAuthDead("quick authorization code is missing")
	}
	return code, nil
}

// RefreshLoginBuffer exchanges a stored Yingyongbao accesstoken for a new login_buffer.
func (s *WxLoginService) RefreshLoginBuffer(ctx context.Context, openID, accessToken string) (string, error) {
	openID = strings.TrimSpace(openID)
	accessToken = strings.TrimSpace(accessToken)
	if openID == "" || accessToken == "" {
		return "", wxAuthDead("Missing Yingyongbao authorization")
	}
	creds := YybCredentials{OpenID: openID, AccessToken: accessToken}
	return s.postLoginBuffer(ctx, nil, &creds)
}

// RefreshCredentials renews access/refresh tokens via pcyyb_refresh_token_auth.
func (s *WxLoginService) RefreshCredentials(ctx context.Context, creds YybCredentials) (YybCredentials, error) {
	if strings.TrimSpace(creds.RefreshToken) == "" {
		return YybCredentials{}, wxAuthDead("missing refresh token")
	}
	if strings.TrimSpace(creds.OpenID) == "" || strings.TrimSpace(creds.AccessToken) == "" {
		return YybCredentials{}, wxAuthDead("Missing Yingyongbao authorization")
	}
	payload, err := refreshTokenPayload(creds)
	if err != nil {
		return YybCredentials{}, wxAuthTransient(err.Error())
	}
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	nonce, err := loginBufferNonce()
	if err != nil {
		return YybCredentials{}, wxAuthTransient(err.Error())
	}
	headers := signedJSONHeaders(timestamp, nonce, loginBufferSignature(payload, timestamp, nonce))
	status, data, err := request(ctx, nil, refreshTokenURL, http.MethodPost, []byte(payload), headers, 35*time.Second)
	if err != nil {
		return YybCredentials{}, wxAuthTransient(err.Error())
	}
	if status < 200 || status >= 300 {
		return YybCredentials{}, wxAuthTransient(fmt.Sprintf("Unable to refresh Yingyongbao token (HTTP %d)", status))
	}
	return parseRefreshTokenJSON(data, creds)
}

// RefreshCredentialsAndBuffer always renews tokens when a refresh_token is present,
// then fetches a fresh login_buffer. Keepalive's 45-minute ahead window is the only gate.
func (s *WxLoginService) RefreshCredentialsAndBuffer(ctx context.Context, creds YybCredentials) (YybCredentials, error) {
	current := creds
	if strings.TrimSpace(current.RefreshToken) != "" {
		refreshed, err := s.RefreshCredentials(ctx, current)
		if err != nil {
			return YybCredentials{}, err
		}
		current = refreshed
	}
	loginBuffer, err := s.postLoginBuffer(ctx, nil, &current)
	if err != nil {
		return YybCredentials{}, err
	}
	current.LoginBuffer = loginBuffer
	return current, nil
}

// MintGatewayCode mints a one-shot wx.login code from login_buffer.
func (s *WxLoginService) MintGatewayCode(ctx context.Context, creds YybCredentials, appID string) (string, YybCredentials, error) {
	current := creds
	if strings.TrimSpace(current.LoginBuffer) == "" {
		return "", current, wxAuthDead("Missing Yingyongbao authorization")
	}
	if strings.TrimSpace(appID) == "" {
		appID = TargetMiniProgramID
	}
	code, err := getNativeWxLoginCode(ctx, current.LoginBuffer, appID)
	if err == nil {
		return code, current, nil
	}
	slog.Warn("login_buffer mint failed, refreshing via Yingyongbao", "err", err)
	if strings.TrimSpace(current.RefreshToken) != "" {
		refreshed, refreshErr := s.RefreshCredentialsAndBuffer(ctx, current)
		if refreshErr != nil {
			return "", current, refreshErr
		}
		current = refreshed
	} else if strings.TrimSpace(current.OpenID) != "" && strings.TrimSpace(current.AccessToken) != "" {
		buf, refreshErr := s.RefreshLoginBuffer(ctx, current.OpenID, current.AccessToken)
		if refreshErr != nil {
			return "", current, mapNativeMintErr(err, refreshErr)
		}
		current.LoginBuffer = buf
	} else {
		return "", current, mapNativeMintErr(err, nil)
	}
	code, err = getNativeWxLoginCode(ctx, current.LoginBuffer, appID)
	if err != nil {
		return "", current, mapNativeMintErr(err, nil)
	}
	return code, current, nil
}

func (s *WxLoginService) postLoginBuffer(ctx context.Context, jar *cookiejar.Jar, creds *YybCredentials) (string, error) {
	if creds == nil || strings.TrimSpace(creds.OpenID) == "" || strings.TrimSpace(creds.AccessToken) == "" {
		return "", wxAuthDead("Missing Yingyongbao authorization")
	}
	if jar == nil {
		var err error
		jar, err = cookiejar.New(nil)
		if err != nil {
			return "", wxAuthTransient(err.Error())
		}
	}
	payload, err := loginBufferPayload(creds.OpenID, creds.AccessToken)
	if err != nil {
		return "", wxAuthTransient(err.Error())
	}
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	nonce, err := loginBufferNonce()
	if err != nil {
		return "", wxAuthTransient(err.Error())
	}
	headers := signedJSONHeaders(timestamp, nonce, loginBufferSignature(payload, timestamp, nonce))
	if jar != nil && strings.TrimSpace(creds.RefreshToken) != "" {
		u, _ := url.Parse(callbackURL)
		jar.SetCookies(u, []*http.Cookie{
			{Name: "openid", Value: creds.OpenID},
			{Name: "accesstoken", Value: creds.AccessToken},
			{Name: "refreshtoken", Value: creds.RefreshToken},
		})
	}
	status, data, err := request(ctx, jar, loginBufferURL, http.MethodPost, []byte(payload), headers, 35*time.Second)
	if err != nil {
		return "", wxAuthTransient(err.Error())
	}
	if status < 200 || status >= 300 {
		return "", wxAuthTransient(fmt.Sprintf("unable to obtain WeChat login buffer (HTTP %d)", status))
	}
	return parseLoginBufferJSON(data)
}

func loginBufferPayload(openID, accessToken string) (string, error) {
	payload, err := json.Marshal(map[string]any{"extInfo": map[string]any{
		"listS": map[string]any{
			"unionid":      map[string]any{"value": []string{openID}},
			"user_id":      map[string]any{"value": []string{openID}},
			"access_token": map[string]any{"value": []string{accessToken}},
		},
		"listI": map[string]any{"user_type": map[string]any{"value": []int{0}}},
	}})
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func refreshTokenPayload(creds YybCredentials) (string, error) {
	payload, err := json.Marshal(map[string]any{"userInfo": map[string]any{
		"openId":       creds.OpenID,
		"refreshToken": creds.RefreshToken,
		"accessToken":  creds.AccessToken,
		"loginType":    "WX",
	}})
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func signedJSONHeaders(timestamp, nonce, signature string) http.Header {
	return http.Header{
		"Content-Type":          {"application/json"},
		"Ual-Access-Businessid": {"pc_yyb_auth"},
		"Ual-Access-Timestamp":  {timestamp},
		"Ual-Access-Nonce":      {nonce},
		"Ual-Access-Signature":  {signature},
	}
}

func loginBufferNonce() (string, error) {
	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	return strconv.Itoa(1000 + int(binary.BigEndian.Uint64(random[:])%9000)), nil
}

func loginBufferSignature(payload, timestamp, nonce string) string {
	sum := md5.Sum([]byte(payload + timestamp + loginBufferAccessKey + nonce))
	return hex.EncodeToString(sum[:])
}

func parseLoginBufferJSON(data []byte) (string, error) {
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
		return "", wxAuthTransient(fmt.Sprintf("JSON parse: %v", err))
	}
	if result.Code != 0 || len(result.ExtInfo.ListS.LoginBuffer.Value) == 0 || result.ExtInfo.ListS.LoginBuffer.Value[0] == "" {
		return "", wxAuthDead("WeChat login buffer response is invalid")
	}
	return result.ExtInfo.ListS.LoginBuffer.Value[0], nil
}

func parseRefreshTokenJSON(data []byte, base YybCredentials) (YybCredentials, error) {
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return YybCredentials{}, wxAuthTransient(fmt.Sprintf("JSON parse: %v", err))
	}
	code := int64FromAny(result["code"])
	if code != 0 {
		msg := stringFromAny(result["msg"])
		if msg == "" {
			msg = "refresh failed"
		}
		return YybCredentials{}, wxAuthDead(fmt.Sprintf("refresh failed: code=%d msg=%s", code, msg))
	}
	info, _ := result["user_info"].(map[string]any)
	if info == nil {
		info, _ = result["userInfo"].(map[string]any)
	}
	accessToken := firstString(info, "access_token", "accessToken")
	if accessToken == "" {
		return YybCredentials{}, wxAuthDead("refresh response missing access_token")
	}
	now := time.Now().Unix()
	updated := base.ApplyNewRefreshToken(firstString(info, "refresh_token", "refreshToken"), now)
	expiresIn := int64FromAny(info["expires_in"])
	if expiresIn <= 0 {
		expiresIn = int64FromAny(info["expiresIn"])
	}
	if expiresIn <= 0 {
		if base.ExpiresIn > 0 {
			expiresIn = base.ExpiresIn
		} else {
			expiresIn = DefaultExpiresIn
		}
	}
	updated.AccessToken = accessToken
	updated.LoginBuffer = base.LoginBuffer
	updated.ExpiresAt = now + expiresIn
	updated.ExpiresIn = expiresIn
	return updated, nil
}

func firstString(m map[string]any, keys ...string) string {
	if m == nil {
		return ""
	}
	for _, key := range keys {
		if s, ok := m[key].(string); ok && strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

func int64FromAny(v any) int64 {
	switch x := v.(type) {
	case int:
		return int64(x)
	case int64:
		return x
	case float64:
		return int64(x)
	case json.Number:
		n, _ := x.Int64()
		return n
	default:
		return 0
	}
}

func stringFromAny(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func mapNativeMintErr(first error, refreshErr error) WxAuthError {
	msg := ""
	if first != nil {
		msg = first.Error()
	}
	if refreshErr != nil {
		if msg != "" {
			msg += "; refresh login_buffer: " + refreshErr.Error()
		} else {
			msg = refreshErr.Error()
		}
	}
	if msg == "" {
		msg = "mint gateway code failed"
	}
	return WxAuthError{Kind: classifyYybMessage(msg), Message: msg}
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
	session.AccessToken = ""
	session.RefreshToken = ""
	session.LoginBuffer = ""
	session.ExpiresAt = 0
	session.ExpiresIn = 0
}
