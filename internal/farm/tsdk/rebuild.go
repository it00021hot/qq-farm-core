package tsdk

import (
	"context"
	"fmt"
	"log/slog"
)

// WASMConsecutiveFailThreshold 连续 wasm 失败达到该次数即请求重建
// （对齐 rust WASM_CONSECUTIVE_FAIL_THRESHOLD = 3）。
const WASMConsecutiveFailThreshold = 3

// 本文件实现 TSDK wasm 连续失败检测与重建闭环（对齐 rust TsdkRuntime 的
// record_wasm_failure / request_reset / rebuild）：
//
//  1. 任一 wasm 调用失败 → consecFails+1（RecordCall），成功清零；
//  2. 达到 WASMConsecutiveFailThreshold → pendingReset=true（RequestReset），
//     之后所有 host 侧调用在 assertReadyLocked 短路，不再向坏死的实例投递数据；
//  3. Session 的重建循环观察 pendingReset（限频）→ BeginRebuild →
//     Rebuild（销毁旧实例 + 重新编译实例化 + 重绑用户）→ 原子替换 encryptor →
//     EndRebuild → MarkResetCompleted。
//
// 缺这一步时 wasm 内存增长 / 状态坏死只会表现为加密持续失败 → 心跳全部
// 超时 → 静默掉线，无法自愈。

// RecordWasmSuccess 记录一次 wasm 调用成功（清零连续失败计数，
// 对齐 rust record_wasm_success；不影响 pendingReset——重建完成后由
// Rebuild 统一 MarkResetCompleted）。
func (r *Runtime) RecordWasmSuccess() {
	r.mu.Lock()
	r.consecFails = 0
	r.mu.Unlock()
}

// RecordCall 按一次 wasm 调用的结果记账：成功清零连续失败，失败 +1；
// 首次达到 WASMConsecutiveFailThreshold 时置 pendingReset 并打点
// （对齐 rust record_wasm_failure + request_reset）。
func (r *Runtime) RecordCall(err error) {
	if err == nil {
		r.RecordWasmSuccess()
		return
	}
	r.mu.Lock()
	r.consecFails++
	reached := r.consecFails >= WASMConsecutiveFailThreshold
	firstReached := reached && !r.pendingReset
	if reached {
		r.pendingReset = true
	}
	count := r.consecFails
	r.mu.Unlock()
	if firstReached {
		// rust 在 alloc failed 路径打 error 日志；这里统一限首次达到阈值时打
		slog.Error("TSDK wasm 连续失败达到阈值，请求重建",
			"component", "tsdk",
			"consecutiveFails", count,
			"threshold", WASMConsecutiveFailThreshold,
			"lastErr", err.Error(),
		)
	}
}

// ConsecutiveFailCount 当前连续失败计数（诊断用，对齐 rust consecutive_fail_count）。
func (r *Runtime) ConsecutiveFailCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.consecFails
}

// IsResetPending 报告是否已请求重建（对齐 rust is_reset_pending）。
func (r *Runtime) IsResetPending() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.pendingReset
}

// RequestReset 置重建请求标记。此后 host 侧调用全部短路，等待重建。
func (r *Runtime) RequestReset() {
	r.mu.Lock()
	r.pendingReset = true
	r.mu.Unlock()
}

// MarkResetCompleted 重建完成后清零重置标记与失败计数
// （对齐 rust mark_reset_completed；Rebuild 成功路径内部已调用）。
func (r *Runtime) MarkResetCompleted() {
	r.mu.Lock()
	r.consecFails = 0
	r.pendingReset = false
	r.mu.Unlock()
}

// Rebuild 同步销毁并重建 wasm 实例（对齐 rust TsdkRuntime::rebuild）：
// 销毁旧实例（wasm 内存随模块关闭释放）→ 用同一 wasmPath 重新编译实例化 →
// 用上次 bindUser 的 openID 重绑用户 → 清零重置状态和失败计数。
//
// 调用方约定（对齐 rust rebuild 注释）：
//  1. 调用前先让网关进入重建期（protocol.Client.BeginRebuild，心跳静默
//     阈值放宽 90s）；
//  2. 成功后替换 encryptor（protocol.Client.ReplaceEncryptor）并 EndRebuild。
//
// 本方法是原地重建：Runtime 对象不变、仅换内部 wasm 实例，因此 Session 的
// aceLoop（每 tick 重新读取 s.tsdk）无需像 rust 一样显式重启 ACE。
func (r *Runtime) Rebuild(parent context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cfg.WASMPath == "" {
		return fmt.Errorf("tsdk: rebuild 失败：从未 init 过 TSDK，没有 wasmPath")
	}
	openID := r.lastOpenID
	// 1. 销毁旧实例：旧 wasm 内存（heap + stack）随模块关闭析构
	r.destroyLocked()
	r.destroyed = false
	// 2. 重新编译 wasm、重新解密 merged 数据、重新创建实例
	if err := r.initLocked(parent); err != nil {
		return err
	}
	// 3. 重新 bindUser：把 wasm 内部 state 重新绑定到当前账号
	//    （走内部方法，此刻 pendingReset 已随销毁清理，无短路）
	if openID != "" {
		if err := r.bindUserLocked(openID); err != nil {
			return err
		}
	}
	// 4. 清零重置状态和失败计数（rust mark_reset_completed）
	r.consecFails = 0
	r.pendingReset = false
	return nil
}
