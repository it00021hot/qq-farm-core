package runtime

// WS 400 状态码特化（对齐 rust runtime/worker.rs:377-399 + engine.rs:466-507）：
// WS 握手被网关以 HTTP 400 拒绝 = 一次性登录码失效。rust 侧按 parse_ws_http_code
// 从错误串解析状态码，命中 400 时账号日志以独立分类 ws_400 记录专用文案：
//   - 持有应用宝授权：「账号 X 登录码失效，稍后将用应用宝授权重连」
//   - 无应用宝授权：「账号 X 登录失效，请更新 Code」
// 其余拨号失败仍走通用「网关连接失败」提示（go 侧 ready 通道错误保持不变）。

import (
	"strconv"
	"strings"
)

// logEventWs400 rust add_account_log 的 action 分类名（go 侧沿用 appendRuntimeLog
// 的 event 字段承载）。
const logEventWs400 = "ws_400"

// parseWsHTTPCode mirrors rust engine.rs parse_ws_http_code：从错误串中
// 「unexpected server response:」之后读取十进制状态码；无则返回 0。
func parseWsHTTPCode(msg string) int64 {
	lower := strings.ToLower(msg)
	needle := "unexpected server response:"
	idx := strings.Index(lower, needle)
	if idx < 0 {
		return 0
	}
	rest := strings.TrimLeft(msg[idx+len(needle):], " \t")
	end := 0
	for end < len(rest) && rest[end] >= '0' && rest[end] <= '9' {
		end++
	}
	if end == 0 {
		return 0
	}
	code, err := strconv.ParseInt(rest[:end], 10, 64)
	if err != nil || code <= 0 {
		return 0
	}
	return code
}

// logWs400DialFailure 网关拨号失败时识别 HTTP 400 并按 ws_400 分类记录账号日志；
// 非 400 失败静默返回（通用错误文案由调用方原样上抛）。
func (s *Session) logWs400DialFailure(dialErr error) {
	if dialErr == nil || parseWsHTTPCode(dialErr.Error()) != 400 {
		return
	}
	accountID := parseAccountID(s.id)
	display := s.id
	if acc, ok := loadFarmAccount(accountID); ok {
		display = wxAccountDisplay(acc, s.id)
	}
	if s.cfg.HasWxAuth {
		appendRuntimeLog(accountID, logEventWs400, "账号 "+display+" 登录码失效，稍后将用应用宝授权重连", true)
		return
	}
	appendRuntimeLog(accountID, logEventWs400, "账号 "+display+" 登录失效，请更新 Code", true)
}
