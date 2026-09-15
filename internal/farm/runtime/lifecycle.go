// 账号生命周期串行化与世代号（移植 rust runtime/engine.rs 语义）：
//   - lifecycle_locks（engine.rs:112-114）：每账号一把锁，在最外层启动入口获取，
//     串行化手动启动 / 定时重连 / 启动重连 / 重启，防止同账号并发拉起双会话互踢；
//   - generation（engine.rs:110-116、:292-318）：每次启动/重启递增世代号，
//     后台回调（worker 退出清理）按世代校验，过期 worker 的退出不得摘掉
//     新 worker 的注册、不得排重连，否则幽灵重连会再开一个会话与新 worker 互踢。
package runtime

import (
	"fmt"
	"sync"
)

// maxWorkers 同时运行的账号 worker 上限（rust EngineConfig::default 的
// max_workers = 16，start_worker 超限报错 engine.rs:872-874）。
const maxWorkers = 16

// lifecycleLock 取（或创建）某账号的生命周期锁（rust engine.rs lifecycle_lock）。
// 语义对齐 tokio::sync::Mutex：不可重入，只在这些入口最外层获取，
// 持锁期间不得再次调用会取同一把锁的路径。
func (m *AccountManager) lifecycleLock(accountID string) *sync.Mutex {
	m.mu.Lock()
	defer m.mu.Unlock()
	if l, ok := m.lifecycleLocks[accountID]; ok {
		return l
	}
	l := &sync.Mutex{}
	m.lifecycleLocks[accountID] = l
	return l
}

// nextGenerationLocked 递增并返回账号世代号（rust start_worker 的
// generation.fetch_add）：每次启动/重启都会使上一代的后台回调过期。
// 调用方必须已持有 m.mu。
func (m *AccountManager) nextGenerationLocked(accountID string) uint64 {
	if m.generations == nil {
		m.generations = make(map[string]uint64)
	}
	m.generations[accountID]++
	return m.generations[accountID]
}

// generation 返回账号当前世代号（无记录时为 0）。
func (m *AccountManager) generation(accountID string) uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.generations[accountID]
}

// isStaleGeneration 判断某会话世代是否已过期（rust engine.rs is_stale_generation：
// 注册表里已是更新的世代）。过期会话的退出清理（摘注册 / 排重连 / 清存活打点）
// 必须整体跳过，否则旧 worker 迟到的退出会波及新 worker。
// 会话已不在注册表（正常停止 / 异常退出）时不视为过期，退出照常处理。
func (m *AccountManager) isStaleGeneration(accountID string, generation uint64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	cur, ok := m.sess[accountID]
	if !ok {
		return false
	}
	return cur.generation != generation
}

// runningWorkerCount 统计当前在跑（starting/running/stopping）的 worker 数。
// excludeAccountID 用于重启场景：rust start_worker 先移除旧句柄再查上限，
// 替换自身不占用新名额。
func (m *AccountManager) runningWorkerCount(excludeAccountID string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for id, s := range m.sess {
		if id == excludeAccountID {
			continue
		}
		switch s.Status() {
		case StatusStarting, StatusRunning, StatusStopping:
			n++
		}
	}
	return n
}

// checkWorkerLimit worker 数上限校验（rust engine.rs:872-874 超限报错）。
func (m *AccountManager) checkWorkerLimit(excludeAccountID string) error {
	if n := m.runningWorkerCount(excludeAccountID); n >= maxWorkers {
		return fmt.Errorf("运行账号数已达上限（%d），请先停止部分账号再启动", maxWorkers)
	}
	return nil
}
