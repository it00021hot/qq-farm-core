package wxlogin

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	localWechatHost             = "localhost.weixin.qq.com"
	localWechatCheckPath        = "/api/check-login"
	localWechatAuthorizePath    = "/api/authorize"
	localWechatDetectTimeout    = 3 * time.Second
	localWechatAuthorizeTimeout = 120 * time.Second
	localWechatUA               = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
)

// LocalWechatClient talks to desktop WeChat's loopback HTTPS API.
type LocalWechatClient struct {
	Scheme             string
	Host               string
	DialIP             string
	InsecureSkipVerify bool
}

// LocalWechatProfile is a successful /api/check-login result.
type LocalWechatProfile struct {
	Port          uint16
	AuthorizeUUID string
	Nickname      string
	Headimgurl    string
}

type localWechatPayload struct {
	ErrCode int64          `json:"errcode"`
	JSData  map[string]any `json:"jsdata"`
}

// DefaultLocalWechatClient binds 127.0.0.1 and accepts WeChat's local cert.
func DefaultLocalWechatClient() LocalWechatClient {
	return LocalWechatClient{
		Scheme:             "https",
		Host:               localWechatHost,
		DialIP:             "127.0.0.1",
		InsecureSkipVerify: true,
	}
}

func (c LocalWechatClient) scheme() string {
	if c.Scheme == "" {
		return "https"
	}
	return c.Scheme
}

func (c LocalWechatClient) host() string {
	if c.Host == "" {
		return localWechatHost
	}
	return c.Host
}

func (c LocalWechatClient) httpClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{Timeout: timeout}
	dialIP := c.DialIP
	host := c.host()
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: c.InsecureSkipVerify,
			ServerName:         host,
			MinVersion:         tls.VersionTLS12,
		},
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			if dialIP != "" {
				_, port, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, err
				}
				addr = net.JoinHostPort(dialIP, port)
			}
			return dialer.DialContext(ctx, network, addr)
		},
	}
	return &http.Client{Timeout: timeout, Transport: transport}
}

func (c LocalWechatClient) Request(ctx context.Context, port uint16, path string, body any, timeout time.Duration) (localWechatPayload, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return localWechatPayload{}, err
	}
	url := fmt.Sprintf("%s://%s:%d%s", c.scheme(), c.host(), port, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return localWechatPayload{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cache-Control", "no-store")
	req.Header.Set("User-Agent", localWechatUA)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	resp, err := c.httpClient(timeout).Do(req)
	if err != nil {
		return localWechatPayload{}, fmt.Errorf("连接本机微信失败（端口 %d）: %w", port, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return localWechatPayload{}, fmt.Errorf("读取本机微信响应失败: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return localWechatPayload{}, fmt.Errorf("本机微信 HTTP %d（端口 %d）", resp.StatusCode, port)
	}
	return parseLocalWechatResponse(data)
}

func parseLocalWechatResponse(data []byte) (localWechatPayload, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return localWechatPayload{}, fmt.Errorf("本机微信返回空响应")
	}
	var value any
	if err := json.Unmarshal(trimmed, &value); err != nil {
		return localWechatPayload{}, fmt.Errorf("本机微信返回了无法解析的响应")
	}
	if s, ok := value.(string); ok {
		if err := json.Unmarshal([]byte(s), &value); err != nil {
			return localWechatPayload{}, fmt.Errorf("本机微信返回了无法解析的响应")
		}
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return localWechatPayload{}, fmt.Errorf("本机微信返回了无法解析的响应")
	}
	var payload localWechatPayload
	if err := json.Unmarshal(encoded, &payload); err != nil {
		return localWechatPayload{}, fmt.Errorf("本机微信返回了无法解析的响应")
	}
	if payload.JSData == nil {
		payload.JSData = map[string]any{}
	}
	return payload, nil
}

func checkLoginBody() map[string]any {
	return map[string]any{
		"apiname": "qrconnectchecklogin",
		"jsdata": map[string]any{
			"appid":        OAUTHAppID,
			"scope":        OAuthScope,
			"redirect_uri": OAuthRedirectURI,
			"state":        OAuthState,
		},
	}
}

func authorizeBody(authorizeUUID string, x, y int) map[string]any {
	pos, _ := json.Marshal(map[string]int{"x": x, "y": y})
	return map[string]any{
		"apiname": "qrconnectfastauthorize",
		"jsdata": map[string]any{
			"data":           string(pos),
			"appid":          OAUTHAppID,
			"scope":          OAuthScope,
			"redirect_uri":   OAuthRedirectURI,
			"state":          OAuthState,
			"authorize_uuid": authorizeUUID,
		},
	}
}

func profileFromPayload(port uint16, payload localWechatPayload) *LocalWechatProfile {
	if payload.ErrCode != 0 {
		return nil
	}
	uuid, _ := payload.JSData["authorize_uuid"].(string)
	uuid = strings.TrimSpace(uuid)
	if uuid == "" {
		return nil
	}
	nickname, _ := payload.JSData["nickname"].(string)
	head, _ := payload.JSData["headimgurl"].(string)
	return &LocalWechatProfile{
		Port:          port,
		AuthorizeUUID: uuid,
		Nickname:      nickname,
		Headimgurl:    head,
	}
}

// Detect probes desktop WeChat ports in parallel.
func (c LocalWechatClient) Detect(ctx context.Context, ports []uint16) (*LocalWechatProfile, error) {
	if len(ports) == 0 {
		return nil, fmt.Errorf("未检测到可用的桌面微信")
	}
	type result struct {
		profile *LocalWechatProfile
		err     error
		port    uint16
	}
	out := make(chan result, len(ports))
	var wg sync.WaitGroup
	for _, port := range ports {
		wg.Add(1)
		go func(port uint16) {
			defer wg.Done()
			payload, err := c.Request(ctx, port, localWechatCheckPath, checkLoginBody(), localWechatDetectTimeout)
			if err != nil {
				out <- result{err: err, port: port}
				return
			}
			if profile := profileFromPayload(port, payload); profile != nil {
				out <- result{profile: profile, port: port}
				return
			}
			out <- result{err: fmt.Errorf("本机微信端口 %d 返回 errcode=%d", port, payload.ErrCode), port: port}
		}(port)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	var lastErr error
	var handshakeErr error
	for item := range out {
		if item.profile != nil {
			return item.profile, nil
		}
		lastErr = item.err
		if item.err != nil && !isLocalWechatUnreachable(item.err) {
			handshakeErr = item.err
		}
	}
	if handshakeErr != nil {
		return nil, handshakeErr
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("未检测到可用的桌面微信（请确认 Windows 桌面微信已登录且未锁定）")
}

func isLocalWechatUnreachable(err error) bool {
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "connection refused") ||
		strings.Contains(s, "connectex") ||
		strings.Contains(s, "no connection could be made") ||
		strings.Contains(s, "i/o timeout") ||
		strings.Contains(s, "timeout")
}

// Authorize waits for the desktop WeChat confirm dialog and returns redirect_url.
func (c LocalWechatClient) Authorize(ctx context.Context, port uint16, authorizeUUID string, x, y int) (string, error) {
	uuid := strings.TrimSpace(authorizeUUID)
	if uuid == "" {
		return "", fmt.Errorf("请先检测本机微信")
	}
	payload, err := c.Request(ctx, port, localWechatAuthorizePath, authorizeBody(uuid, x, y), localWechatAuthorizeTimeout)
	if err != nil {
		return "", err
	}
	switch payload.ErrCode {
	case 0:
	case 10050:
		return "", fmt.Errorf("已在微信中拒绝授权")
	case 10046:
		return "", fmt.Errorf("授权已超时，请重新检测")
	case 10057:
		return "", fmt.Errorf("当前应用仅支持扫码授权")
	default:
		return "", fmt.Errorf("桌面微信未返回有效授权结果（errcode=%d）", payload.ErrCode)
	}
	redirect, _ := payload.JSData["redirect_url"].(string)
	redirect = strings.TrimSpace(redirect)
	if redirect == "" {
		return "", fmt.Errorf("桌面微信未返回有效授权结果")
	}
	return redirect, nil
}

// DetectDesktopWechat probes the default desktop WeChat ports.
func (s *WxLoginService) DetectDesktopWechat(ctx context.Context) (*LocalWechatProfile, error) {
	return DefaultLocalWechatClient().Detect(ctx, DesktopWechatPorts)
}

// AuthorizeDesktopWechat calls /api/authorize on a previously detected port.
func (s *WxLoginService) AuthorizeDesktopWechat(ctx context.Context, port uint16, authorizeUUID string, x, y int) (string, error) {
	return DefaultLocalWechatClient().Authorize(ctx, port, authorizeUUID, x, y)
}
