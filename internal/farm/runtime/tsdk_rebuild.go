// tsdk_rebuild.go 实现 TSDK wasm 连续失败检测 → 重建闭环（对齐 rust
// worker.rs WasmReset handler + services/ace.rs pending_reset 短路 +
// worker_loop 重建期心跳静默阈值放宽），以及 ACE 离线短路门。
package runtime

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/farm/protocol"
	"github.com/it00021hot/qq-farm-core/internal/farm/tsdk"
)

// trackedEncryptor 包装 *tsdk.Runtime 实现 protocol.Encryptor，把 wasm
// 调用成败喂入 runtime 的连续失败计数。对齐 rust：TsdkRuntime::transform
// 内部 record_wasm_failure/success——alloc failed（wasm 内存增长的首要信号）
// 高发于加密路径，必须计入统计。
type trackedEncryptor struct{ rt *tsdk.Runtime }

func (e *trackedEncryptor) Encrypt(buf []byte) ([]byte, error) {
	out, err := e.rt.Encrypt(buf)
	e.rt.RecordCall(err)
	return out, err
}

func (e *trackedEncryptor) Decrypt(buf []byte) ([]byte, error) {
	out, err := e.rt.Decrypt(buf)
	e.rt.RecordCall(err)
	return out, err
}

// transportOnline 对齐 rust AceSender::online（gateway.phase() == Online）：
// 会话处于 Online 且底层连接存活。离线时 ACE 停止产出/消费，防 wasm 数据
// 只产不消导致队列持续增长、最终 memory.grow 失败 → alloc failed。
func (s *Session) transportOnline() bool {
	if s.Status() != StatusRunning {
		return false
	}
	c := s.Client()
	return c != nil && c.Connected()
}

// currentTsdk returns the live tsdk runtime (nil when torn down).
func (s *Session) currentTsdk() *tsdk.Runtime {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.tsdk
}

// recordTsdkResult 把一次 wasm 调用结果记入连续失败计数（对齐 rust
// record_wasm_failure/success：失败 +1、成功清零，达阈值置 pending_reset）。
func (s *Session) recordTsdkResult(rt *tsdk.Runtime, err error) {
	if rt == nil {
		return
	}
	rt.RecordCall(err)
}

// stageInitCredential 取 TSDK 加密初始化凭据并暂存到网关 token provider。
// 对齐 rust gateway.rs login 第 6 步：bindUser 成功 → get_encrypted_init_info
// → token_provider.stage_init_token；空凭据忽略，失败降级为告警。
// 缺这一步服务端 ACE 会话不完整，会不定时静默丢弃连接。
func (s *Session) stageInitCredential(client *protocol.Client, rt *tsdk.Runtime) {
	info, err := rt.GetEncryptedInitInfo()
	if err != nil {
		slog.Warn("TSDK get_encrypted_init_info 失败", "account", s.id, "err", err)
		return
	}
	n, err := client.StageInitToken(info)
	if err != nil {
		slog.Warn("TSDK 初始化凭据暂存失败", "account", s.id, "err", err)
		return
	}
	if n > 0 {
		slog.Info("TSDK 初始化凭据已就绪，将随下一条请求发送", "account", s.id, "len", n)
	}
}

// tsdkRebuildLoop 观察 TSDK 连续失败 → 重建闭环，语义对齐 rust：
//   - ace.rs：AntiData 任务发现 pending_reset 时限频 30s 发出 WasmReset；
//   - worker.rs WasmReset handler：begin_rebuild（心跳静默阈值放宽 90s）→
//     同步重建 TSDK runtime → gateway.replace_encryptor → end_rebuild →
//     仅当 Online 才重启 ACE；
//   - worker_loop：重建期心跳判死阈值放宽到 90s（在 gameHeartbeatLoop 实现）。
//
// 与 rust 的有意差异：tsdk.Runtime 是原地重建（同一对象换新 wasm 实例，
// rust rebuild 同语义），aceLoop 每 tick 重新读取 s.tsdk、encryptor 指向
// 同一 runtime，因此无需显式重启 ACE / 换对象。
func (s *Session) tsdkRebuildLoop(ctx context.Context) {
	const (
		// 观察周期：对齐 rust ACE anti_data 5s tick（pending_reset 由它观察发出）
		checkEvery = 5 * time.Second
		// 重置信号限频：对齐 rust last_reset_emit_ms 的 30s
		resetEmitCooldown = 30 * time.Second
	)
	ticker := time.NewTicker(checkEvery)
	defer ticker.Stop()

	var lastEmit time.Time
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		rt := s.currentTsdk()
		client := s.Client()
		if rt == nil || client == nil || !rt.IsResetPending() {
			continue
		}
		// 在线门（rust：WasmReset 事件由 ACE 任务发出，而 ACE 任务都有
		// sender_online 门，离线不会触发重建）。
		if !s.transportOnline() {
			continue
		}
		now := time.Now()
		if now.Sub(lastEmit) < resetEmitCooldown {
			continue
		}
		lastEmit = now

		fails := rt.ConsecutiveFailCount()
		slog.Error("TSDK wasm 连续失败达到阈值，开始重建",
			"account", s.id,
			"consecutiveFails", fails,
		)
		s.publishTsdkLog(fmt.Sprintf("TSDK 重建中（连续失败 %d 次）", fails), true)

		// 1. 进入重建期：心跳判死静默阈值放宽 90s
		client.BeginRebuild()
		// 2. 同步重建（wasm 编译 + 实例化，rust 走 spawn_blocking；本循环
		// 独立 goroutine，短暂阻塞只影响重建本身）
		err := rt.Rebuild(ctx)
		if err == nil {
			// 3. 原子替换 encryptor（同一 runtime 对象，新 wasm 实例生效；
			// 对齐 rust replace_encryptor）
			client.ReplaceEncryptor(&trackedEncryptor{rt: rt})
			// 4. 退出重建期
			client.EndRebuild()
			slog.Info("TSDK 重建完成", "account", s.id)
			s.publishTsdkLog(fmt.Sprintf("TSDK 重建完成（连续失败 %d 次）", fails), false)
		} else {
			// 重建失败：清 rebuilding，pending_reset 保持，下个观察周期再试
			// （对齐 rust Ok(Err(e)) 分支）
			client.EndRebuild()
			slog.Error("TSDK 重建失败，下次 reset 时再试", "account", s.id, "err", err)
			s.publishTsdkLog("TSDK 重建失败: "+err.Error(), true)
		}
	}
}

// publishTsdkLog 把 TSDK 重建进展推到面板 runtime_log（对齐 rust WorkerEvent::Log）。
func (s *Session) publishTsdkLog(message string, isWarn bool) {
	if s.hub == nil {
		return
	}
	s.hub.PublishJSON("runtime_log", parseAccountID(s.id), map[string]any{
		"tag":       "tsdk",
		"event":     "TSDK 重建",
		"message":   message,
		"isWarn":    isWarn,
		"accountId": parseAccountID(s.id),
	})
}
