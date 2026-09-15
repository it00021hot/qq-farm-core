package protocol

import (
	"strings"
	"testing"
)

// stage/next/clear 语义 1:1 对齐 rust utils/random.rs 的 GatewayTokenProvider 测试。

func TestGatewayTokenProviderStagesAndConsumesOnce(t *testing.T) {
	p := NewGatewayTokenProvider()
	// 未 stage 时返回随机 token（64~127 字符 + '='）
	randomOne := p.Next()
	if len(randomOne) < 65 || len(randomOne) > 128 || !strings.HasSuffix(randomOne, "=") {
		t.Fatalf("random token malformed: len=%d", len(randomOne))
	}

	// 前后空白被 trim（对齐 rust stage_init_token("  abc123  ") → "abc123"）
	n, err := p.StageInitToken("  abc123  ")
	if err != nil {
		t.Fatalf("stage: %v", err)
	}
	if n != 6 {
		t.Fatalf("staged len = %d, want 6", n)
	}
	// 恰好消费一次：第一次返回 staged，第二次恢复随机
	if got := p.Next(); got != "abc123" {
		t.Fatalf("first next = %q, want staged %q", got, "abc123")
	}
	after := p.Next()
	if after == "abc123" || !strings.HasSuffix(after, "=") {
		t.Fatalf("second next should be a fresh random token, got %q", after)
	}
}

func TestGatewayTokenProviderNextMarkedReportsStaged(t *testing.T) {
	p := NewGatewayTokenProvider()
	if _, staged := p.NextMarked(); staged {
		t.Fatal("unstaged next must not be marked")
	}
	if _, err := p.StageInitToken("cred"); err != nil {
		t.Fatal(err)
	}
	token, staged := p.NextMarked()
	if !staged || token != "cred" {
		t.Fatalf("NextMarked = %q,%v want staged cred", token, staged)
	}
	if _, staged := p.NextMarked(); staged {
		t.Fatal("staged credential must be consumed exactly once")
	}
}

func TestGatewayTokenProviderEmptyIsIgnored(t *testing.T) {
	p := NewGatewayTokenProvider()
	n, err := p.StageInitToken("   ")
	if err != nil || n != 0 {
		t.Fatalf("empty stage = (%d,%v), want (0,nil)", n, err)
	}
	// 空 stage 不影响随机 token 流
	if got := p.Next(); len(got) < 65 {
		t.Fatalf("expected random token, got %q", got)
	}
}

func TestGatewayTokenProviderRejectsInvalidFormat(t *testing.T) {
	p := NewGatewayTokenProvider()
	if _, err := p.StageInitToken("has space"); err == nil {
		t.Fatal("space must be rejected")
	}
	if _, err := p.StageInitToken("中文凭据"); err == nil {
		t.Fatal("non-ASCII must be rejected")
	}
	long := strings.Repeat("x", 64*1024+1)
	if _, err := p.StageInitToken(long); err == nil {
		t.Fatal(">64KB must be rejected")
	}
	// 被拒绝后不影响随机 token 流
	if got := p.Next(); len(got) < 65 || !strings.HasSuffix(got, "=") {
		t.Fatalf("random stream broken after rejections: %q", got)
	}
}

func TestGatewayTokenProviderClearDropsStaged(t *testing.T) {
	p := NewGatewayTokenProvider()
	if _, err := p.StageInitToken("cred"); err != nil {
		t.Fatal(err)
	}
	p.Clear()
	if got := p.Next(); got == "cred" {
		t.Fatal("cleared credential must not be served")
	}
}

// Client 挂载点：登录后 stage → 发帧消费；会话结束（Close）丢弃未消费凭据。
func TestClientStageAndCloseClearsToken(t *testing.T) {
	client := NewClient(Options{})
	n, err := client.StageInitToken("cred")
	if err != nil || n != 4 {
		t.Fatalf("client stage = (%d,%v), want (4,nil)", n, err)
	}
	if token, staged := client.tokens.NextMarked(); !staged || token != "cred" {
		t.Fatalf("NextMarked = %q,%v", token, staged)
	}

	if _, err := client.StageInitToken("cred2"); err != nil {
		t.Fatal(err)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if token, staged := client.tokens.NextMarked(); staged || token == "cred2" {
		t.Fatalf("Close must drop staged token, got %q,%v", token, staged)
	}
}

func TestClientRebuildFlagsAndConnected(t *testing.T) {
	client := NewClient(Options{})
	if client.IsRebuilding() {
		t.Fatal("fresh client must not be rebuilding")
	}
	client.BeginRebuild()
	if !client.IsRebuilding() {
		t.Fatal("BeginRebuild must set flag")
	}
	client.EndRebuild()
	if client.IsRebuilding() {
		t.Fatal("EndRebuild must clear flag")
	}
	if client.Connected() {
		t.Fatal("unconnected client must not report connected")
	}
}
