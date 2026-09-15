package protocol

import (
	"fmt"
	"strings"
	"sync"
)

// GatewayTokenProvider 管理出站帧的 token 字段：登录成功后 bindUser 产出的
// 一次性 TSDK 加密初始化凭据 stage 进来，由下一条出站消息携带（恰好一次，
// 原子消费），之后恢复随机 token。
//
// 1:1 对齐 rust GatewayTokenProvider（qq-farm-rust utils/random.rs:65-116 /
// bot gateway-token.ts::GatewayTokenProvider）。缺这一步服务端 ACE 会话
// 不完整，会不定时静默丢弃连接。
type GatewayTokenProvider struct {
	mu               sync.Mutex
	pendingInitToken string
}

// NewGatewayTokenProvider creates an empty provider.
func NewGatewayTokenProvider() *GatewayTokenProvider {
	return &GatewayTokenProvider{}
}

// StageInitToken 暂存一次性初始化凭据，返回凭据长度（0 表示忽略）。
//
// 对齐 rust stage_init_token：空串（含纯空白）忽略；超长（>64KB）或含
// 非可打印 ASCII 视为格式无效，返回错误由调用方降级为告警，不影响
// 后续随机 token 流。
func (p *GatewayTokenProvider) StageInitToken(value string) (int, error) {
	token := strings.TrimSpace(value)
	if token == "" {
		return 0, nil
	}
	if len(token) > 64*1024 || !isPrintableASCII(token) {
		return 0, fmt.Errorf("protocol: TSDK 初始化凭据格式无效")
	}
	p.mu.Lock()
	p.pendingInitToken = token
	p.mu.Unlock()
	return len(token), nil
}

// Next 返回下一条消息的 token：有暂存凭据则原子消费返回一次，否则随机 token。
func (p *GatewayTokenProvider) Next() string {
	token, _ := p.NextMarked()
	return token
}

// NextMarked 同 Next，并返回该 token 是否为暂存的一次性凭据（诊断用）。
func (p *GatewayTokenProvider) NextMarked() (string, bool) {
	p.mu.Lock()
	staged := p.pendingInitToken
	p.pendingInitToken = ""
	p.mu.Unlock()
	if staged != "" {
		return staged, true
	}
	return CreateGatewayToken(), false
}

// Clear 清空暂存凭据（会话结束/重连时调用，对齐 rust end_session →
// token_provider.clear()，即 bot clearNetworkRuntime → gatewayTokens.clear()）。
func (p *GatewayTokenProvider) Clear() {
	p.mu.Lock()
	p.pendingInitToken = ""
	p.mu.Unlock()
}

// isPrintableASCII 对齐 rust 校验：每个字节都在 0x21..=0x7E（可见且非空格）。
func isPrintableASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < 0x21 || s[i] > 0x7E {
			return false
		}
	}
	return true
}
