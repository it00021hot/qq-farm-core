package wxlogin

// 扫码登录 — QQ 小程序开发者工具登录态获取（移植 rust services/qrlogin.rs，
// 原 TS qrlogin.ts 的 1:1 对齐层）。用于「YYB 授权失效自动恢复闭环」：
//
//  1. GET  https://q.qq.com/ide/devtoolAuth/GetLoginCode 拿一次性登录码
//  2. 用户用手机 QQ 扫 https://h5.qzone.qq.com/qqq/code/{code}?_proxy=1&from=ide
//  3. 轮询 https://q.qq.com/ide/devtoolAuth/syncScanSateGetTicket?code={code} 拿 ticket
//  4. POST https://q.qq.com/ide/login 用 ticket 换游戏登录 auth code（换码重启账号用）
//
// 与 rust 的差异（有意）：rust request_login_code 会用 qrcode crate 把扫码 URL
// 渲染成本地 PNG data URL（image 字段），该字段在闭环中未被消费（QQ Bot 富媒体
// 接口要求公网 http(s) URL，实际走 quickchart.io 在线渲染，见 runtime/relogin_watcher.go
// publicQrImageURL），go 侧省略本地渲染、Image 恒为空，避免引入二维码渲染依赖。
//
// HTTP 执行层抽成 MpHTTPClient 接口、端点字段化，便于单测注入 fake / httptest
// （对齐 rust reqwest::Client 组装点）。

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	// MpChromeUA Chrome 浏览器 UA（模拟 IDE 请求，rust CHROME_UA）。
	MpChromeUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	// MpQUA QQ 平台版本标识（rust QUA）。
	MpQUA = "V1_HT5_QDT_0.70.2209190_x64_0_DEV_D"

	mpHost             = "q.qq.com"
	mpLoginCodeURL     = "https://q.qq.com/ide/devtoolAuth/GetLoginCode"
	mpSyncStatusURL    = "https://q.qq.com/ide/devtoolAuth/syncScanSateGetTicket"
	mpLoginExchangeURL = "https://q.qq.com/ide/login"

	// mpRequestTimeout 单请求超时（rust reqwest::Client timeout 10s）。
	mpRequestTimeout = 10 * time.Second

	// MpPresetFarmAppID 内置预设 farm：QQ经典农场小程序（rust PRESETS，与
	// tsdk.QQMiniProgramAppID 一致）；扫码成功后用该 appid 换游戏登录 code。
	MpPresetFarmAppID = "1112386029"
)

// MpLoginCodeResult 登录码响应（rust MpLoginCodeResult）。
type MpLoginCodeResult struct {
	Code  string // 一次性登录码
	URL   string // 用户扫码地址（BuildMpLoginURL(code)）
	Image string // 二维码图片地址；go 侧恒为空（文件头注释「与 rust 的差异」）
}

// MpStatus 扫码状态（rust MpStatus，PascalCase 取值）。
type MpStatus string

const (
	// MpStatusWait 等待用户扫码 / 确认。
	MpStatusWait MpStatus = "Wait"
	// MpStatusOK 扫码成功（携带 ticket）。
	MpStatusOK MpStatus = "OK"
	// MpStatusUsed 二维码已使用 / 失效。
	MpStatusUsed MpStatus = "Used"
	// MpStatusError 错误（业务码非 0 / 非 200）。
	MpStatusError MpStatus = "Error"
)

// MpStatusResult 扫码状态查询结果（rust MpStatusResult；可缺省字段用空串表示）。
type MpStatusResult struct {
	Status   MpStatus
	Ticket   string
	Uin      string
	Nickname string
	Msg      string
}

// MpHTTPClient 抽象 HTTP 执行层（生产为带超时的 *http.Client，测试注入 fake）。
type MpHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// MiniProgramLoginSession QQ 小程序开发者工具登录态会话（rust MiniProgramLoginSession）。
type MiniProgramLoginSession struct {
	client MpHTTPClient
	// 三个端点默认指向生产地址；包内单测可替换为 httptest 服务地址。
	loginCodeURL     string
	syncStatusURL    string
	loginExchangeURL string
}

// NewMiniProgramLoginSession 创建会话（rust new：10s 超时 client）。
func NewMiniProgramLoginSession() *MiniProgramLoginSession {
	return &MiniProgramLoginSession{
		client:           &http.Client{Timeout: mpRequestTimeout},
		loginCodeURL:     mpLoginCodeURL,
		syncStatusURL:    mpSyncStatusURL,
		loginExchangeURL: mpLoginExchangeURL,
	}
}

// mpHeaders 构造请求头（rust get_headers：qua/host/accept/content-type/user-agent）。
func mpHeaders() http.Header {
	return http.Header{
		"qua":          {MpQUA},
		"host":         {mpHost},
		"accept":       {"application/json"},
		"content-type": {"application/json"},
		"user-agent":   {MpChromeUA},
	}
}

// BuildMpLoginURL 构造用户扫码地址（rust build_login_url）。
func BuildMpLoginURL(code string) string {
	return "https://h5.qzone.qq.com/qqq/code/" + code + "?_proxy=1&from=ide"
}

// do 发请求，返回（HTTP 状态码, 原始响应体, err）；err 仅表示网络错误或组装错误。
// host 头跳过——Host 行由传输层按请求 URL 生成（rust reqwest 同样按 URL 派生），
// 手写会造成重复 Host 被服务端拒 400；user-agent 用规范键写入（否则传输层会补
// Go 默认 UA）；qua 等自定义头按原始小写写入（rust from_static("qua") 同）。
func (s *MiniProgramLoginSession) do(ctx context.Context, method, rawURL string, body []byte) (int, []byte, error) {
	ctx, cancel := context.WithTimeout(ctx, mpRequestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, rawURL, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	for key, values := range mpHeaders() {
		switch key {
		case "host":
			continue
		case "user-agent":
			req.Header["User-Agent"] = append([]string(nil), values...)
		default:
			req.Header[key] = append([]string(nil), values...)
		}
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	return resp.StatusCode, data, err
}

// parseJSON 把响应体解析为 JSON 对象（rust resp.json()：解析失败 → Err）。
func parseJSON(data []byte) (map[string]any, error) {
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, errors.New("响应不是合法 JSON: " + string(data))
	}
	return out, nil
}

// RequestLoginCode 调 q.qq.com 拿登录码（rust request_login_code）：
// 响应体必须为 JSON；业务码非 0 报「获取登录码失败」；成功返回登录码与扫码 URL。
func (s *MiniProgramLoginSession) RequestLoginCode(ctx context.Context) (MpLoginCodeResult, error) {
	_, data, err := s.do(ctx, http.MethodGet, s.loginCodeURL, nil)
	if err != nil {
		return MpLoginCodeResult{}, err
	}
	body, err := parseJSON(data)
	if err != nil {
		return MpLoginCodeResult{}, err
	}
	// rust：body.code.as_i64().unwrap_or(-1)，字段缺失视为失败。
	resCode := int64(-1)
	if v, ok := body["code"]; ok && v != nil {
		resCode = int64FromAny(v)
	}
	if resCode != 0 {
		return MpLoginCodeResult{}, errors.New("获取登录码失败")
	}
	loginCode := ""
	if d, ok := body["data"].(map[string]any); ok {
		loginCode = stringFromAny(d["code"])
	}
	return MpLoginCodeResult{Code: loginCode, URL: BuildMpLoginURL(loginCode)}, nil
}

// QueryStatus 查询扫码状态（rust query_status）：
//   - 非 200 → Error("non-200 status")（rust 先查状态码、不解析 body，同序）；
//   - 业务码 0 且 data.ok==1 → OK(ticket/uin/nick)，否则 Wait；
//   - 业务码 -10003 → Used；其它非 0 → Error("Code: N")；
//   - 网络错误 / 响应解析失败原样返回 err（调用方等一轮再试）。
func (s *MiniProgramLoginSession) QueryStatus(ctx context.Context, code string) (MpStatusResult, error) {
	status, data, err := s.do(ctx, http.MethodGet, s.syncStatusURL+"?code="+url.QueryEscape(code), nil)
	if err != nil {
		return MpStatusResult{}, err
	}
	if status != http.StatusOK {
		return MpStatusResult{Status: MpStatusError, Msg: "non-200 status"}, nil
	}
	body, err := parseJSON(data)
	if err != nil {
		return MpStatusResult{}, err
	}
	resCode := int64(-1) // rust：unwrap_or(-1)，字段缺失按非 0 处理
	if v, ok := body["code"]; ok && v != nil {
		resCode = int64FromAny(v)
	}
	switch resCode {
	case 0:
		data, _ := body["data"].(map[string]any)
		if data == nil || int64FromAny(data["ok"]) != 1 {
			return MpStatusResult{Status: MpStatusWait}, nil
		}
		return MpStatusResult{
			Status:   MpStatusOK,
			Ticket:   stringFromAny(data["ticket"]),
			Uin:      stringFromAny(data["uin"]),
			Nickname: stringFromAny(data["nick"]),
		}, nil
	case -10003:
		return MpStatusResult{Status: MpStatusUsed}, nil
	default:
		return MpStatusResult{Status: MpStatusError, Msg: "Code: " + strconv.FormatInt(resCode, 10)}, nil
	}
}

// GetAuthCode 用 ticket 换游戏登录 code（rust get_auth_code，POST JSON {appid, ticket}）：
// 非 200 返回空 code（rust 先查状态码、不解析 body，同序，调用方按失败处理）；
// 成功解析 body.code 字符串。
func (s *MiniProgramLoginSession) GetAuthCode(ctx context.Context, ticket, appID string) (string, error) {
	payload, err := json.Marshal(map[string]string{"appid": appID, "ticket": ticket})
	if err != nil {
		return "", err
	}
	status, data, err := s.do(ctx, http.MethodPost, s.loginExchangeURL, payload)
	if err != nil {
		return "", err
	}
	if status != http.StatusOK {
		return "", nil
	}
	body, err := parseJSON(data)
	if err != nil {
		return "", err
	}
	return stringFromAny(body["code"]), nil
}
