package tsdk_test

import (
	"context"
	"errors"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/farm/tsdk"
)

// 连续失败计数语义 1:1 对齐 rust crypto/tsdk.rs 的
// record_wasm_failure / record_wasm_success / is_reset_pending 测试。

func newStateRuntime(t *testing.T) *tsdk.Runtime {
	t.Helper()
	rt, err := tsdk.New(tsdk.Config{
		WASMPath:  "unused.wasm", // 纯状态测试不触发 wasm 加载
		AccountID: "test",
		DataRoot:  t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return rt
}

func TestConsecutiveFailureThreshold(t *testing.T) {
	rt := newStateRuntime(t)
	defer rt.Destroy()

	boom := errors.New("boom")
	rt.RecordCall(boom)
	rt.RecordCall(boom)
	if rt.ConsecutiveFailCount() != 2 {
		t.Fatalf("count = %d, want 2", rt.ConsecutiveFailCount())
	}
	if rt.IsResetPending() {
		t.Fatal("below threshold must not request reset")
	}

	// 第 3 次（WASMConsecutiveFailThreshold）失败：置 pending_reset
	rt.RecordCall(boom)
	if !rt.IsResetPending() {
		t.Fatal("threshold failure must request reset")
	}

	// 成功只清零计数，pending_reset 保持（对齐 rust：等待 worker 重建完成后
	// MarkResetCompleted 才清）
	rt.RecordCall(nil)
	if rt.ConsecutiveFailCount() != 0 {
		t.Fatalf("success must zero count, got %d", rt.ConsecutiveFailCount())
	}
	if !rt.IsResetPending() {
		t.Fatal("success must not clear pending reset")
	}

	rt.MarkResetCompleted()
	if rt.IsResetPending() || rt.ConsecutiveFailCount() != 0 {
		t.Fatal("MarkResetCompleted must clear reset state")
	}
}

func TestResetPendingShortCircuitsWasmCalls(t *testing.T) {
	rt := newStateRuntime(t)
	defer rt.Destroy()

	// pending_reset 短路优先于 not-ready（对齐 rust：host-side helper 在
	// 重置等待期一律报错，防止继续向坏死的 wasm 实例投递数据）
	rt.RequestReset()
	if _, err := rt.Encrypt([]byte("x")); err == nil || !strings.Contains(err.Error(), "重置") {
		t.Fatalf("Encrypt during pending reset = %v, want reset-pending error", err)
	}
	if err := rt.HeartbeatTick(); err == nil || !strings.Contains(err.Error(), "重置") {
		t.Fatalf("HeartbeatTick during pending reset = %v, want reset-pending error", err)
	}
	if err := rt.SendDataFromServer([]byte("x")); err == nil || !strings.Contains(err.Error(), "重置") {
		t.Fatalf("SendDataFromServer during pending reset = %v, want reset-pending error", err)
	}

	// 重建完成后恢复可用语义（此处仅验证短路解除：未 Init 退回 not-ready）
	rt.MarkResetCompleted()
	if _, err := rt.Encrypt([]byte("x")); err == nil || strings.Contains(err.Error(), "重置") {
		t.Fatalf("after reset completed the short-circuit must be gone, got %v", err)
	}
}

func TestRebuildWithoutInitFails(t *testing.T) {
	// 对齐 rust rebuild_without_init_returns_error：从未 init 过的 runtime
	// rebuild 必须返回错误（wasm 路径指向不存在的文件 → 加载失败）
	rt, err := tsdk.New(tsdk.Config{
		WASMPath:  filepath.Join(t.TempDir(), "missing.wasm"),
		AccountID: "test",
		DataRoot:  t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Destroy()
	if err := rt.Rebuild(context.Background()); err == nil {
		t.Fatal("rebuild without a loadable wasm must fail")
	}
}

func TestRebuildReinitializesAndRebinds(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	// internal/farm/tsdk → repo root
	root := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	wasmPath := filepath.Join(root, "resource", "farm", "tsdk.wasm")

	rt, err := tsdk.New(tsdk.Config{
		WASMPath:  wasmPath,
		AccountID: "test",
		DataRoot:  t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Destroy()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := rt.Init(ctx); err != nil {
		t.Fatalf("init: %v", err)
	}
	in := []byte("abc")
	enc, err := rt.Encrypt(in)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if len(enc) == 0 {
		t.Fatal("empty ciphertext")
	}

	// 模拟连续失败触发重建，重建后同一对象上的加密恢复可用
	for i := 0; i < tsdk.WASMConsecutiveFailThreshold; i++ {
		rt.RecordCall(errors.New("alloc failed"))
	}
	if !rt.IsResetPending() {
		t.Fatal("expected pending reset")
	}
	rt.RequestReset()
	if err := rt.Rebuild(ctx); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	if rt.IsResetPending() || rt.ConsecutiveFailCount() != 0 {
		t.Fatal("rebuild must clear reset state")
	}
	enc2, err := rt.Encrypt(in)
	if err != nil {
		t.Fatalf("encrypt after rebuild: %v", err)
	}
	if string(enc2) != string(enc) {
		t.Fatalf("encrypt after rebuild = %x, want %x (deterministic)", enc2, enc)
	}
}
