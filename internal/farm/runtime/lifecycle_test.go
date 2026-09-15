package runtime

import (
	"sync"
	"testing"
	"time"
)

// addSessionLocked 直接入册一个会话（绕过网络启动），模拟 startAccountLocked
// 里“建会话 + 递增世代号 + 注册”这段可单测的核心。
func addSessionLocked(m *AccountManager, id string) *Session {
	s := newSession(SessionConfig{AccountID: id}, nil)
	m.mu.Lock()
	defer m.mu.Unlock()
	s.generation = m.nextGenerationLocked(id)
	m.sess[id] = s
	return s
}

// TestGenerationStaleDetection 世代号防过期（rust engine.rs is_stale_generation）：
// 重启替换后旧会话的退出必须判为过期；正常退出（注册表已摘）不算过期。
func TestGenerationStaleDetection(t *testing.T) {
	m := NewAccountManager(nil)

	// 注册表无记录：不算过期（正常停止 / 异常退出照常清理 + 排重连）。
	if m.isStaleGeneration("a1", 0) {
		t.Fatal("absent session must not be stale")
	}

	s1 := addSessionLocked(m, "a1")
	if s1.generation != 1 {
		t.Fatalf("first generation=%d, want 1", s1.generation)
	}
	if m.isStaleGeneration("a1", s1.generation) {
		t.Fatal("current generation must not be stale")
	}

	// 重启产生第 2 代：旧世代判定为过期，退出清理必须跳过。
	s2 := addSessionLocked(m, "a1")
	if s2.generation != 2 {
		t.Fatalf("second generation=%d, want 2", s2.generation)
	}
	if !m.isStaleGeneration("a1", s1.generation) {
		t.Fatal("old generation must be stale after restart")
	}
	if m.isStaleGeneration("a1", s2.generation) {
		t.Fatal("newest generation must not be stale")
	}

	// 会话退出、注册表摘除后：不算过期。
	m.mu.Lock()
	delete(m.sess, "a1")
	m.mu.Unlock()
	if m.isStaleGeneration("a1", s2.generation) {
		t.Fatal("removed session must not be stale")
	}
}

// TestLifecycleLockSerializesSameAccount 生命周期锁语义（rust lifecycle_locks）：
// 同一账号返回同一把锁，锁确实互斥；不同账号互不阻塞。
func TestLifecycleLockSerializesSameAccount(t *testing.T) {
	m := NewAccountManager(nil)
	if m.lifecycleLock("a1") != m.lifecycleLock("a1") {
		t.Fatal("same account must share one lifecycle lock")
	}
	if m.lifecycleLock("a1") == m.lifecycleLock("a2") {
		t.Fatal("different accounts must not share lifecycle locks")
	}

	l := m.lifecycleLock("a1")
	l.Lock()
	acquired := make(chan struct{})
	go func() {
		m.lifecycleLock("a1").Lock()
		close(acquired)
	}()
	select {
	case <-acquired:
		t.Fatal("locked account lifecycle lock must block")
	case <-time.After(100 * time.Millisecond):
	}
	l.Unlock()
	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatal("lock must be acquirable after unlock")
	}
}

// TestRunningWorkerCountExcludesRestartingAccount worker 计数：
// 重启替换自身不占用新名额（rust start_worker 先移除旧句柄再查上限）。
func TestRunningWorkerCountExcludesRestartingAccount(t *testing.T) {
	m := NewAccountManager(nil)
	s1 := addSessionLocked(m, "a1")
	s2 := addSessionLocked(m, "a2")
	s3 := addSessionLocked(m, "a3")
	// 直接置状态，绕开 setStatus 的广播/存活打点副作用。
	s1.status = StatusRunning
	s2.status = StatusStarting
	s3.status = StatusStopped

	if got := m.runningWorkerCount("zz"); got != 2 {
		t.Fatalf("running=%d, want 2 (starting/running only)", got)
	}
	if got := m.runningWorkerCount("a1"); got != 1 {
		t.Fatalf("excluding restarting a1: got %d, want 1", got)
	}
}

// TestCheckWorkerLimit worker 数上限（rust max_workers=16 超限报错）。
func TestCheckWorkerLimit(t *testing.T) {
	if maxWorkers != 16 {
		t.Fatalf("maxWorkers=%d, want 16 (rust EngineConfig::default)", maxWorkers)
	}
	m := NewAccountManager(nil)
	for i := 0; i < maxWorkers; i++ {
		s := addSessionLocked(m, "w"+string(rune('a'+i)))
		s.status = StatusRunning
	}
	if err := m.checkWorkerLimit("new-account"); err == nil {
		t.Fatal("starting beyond limit must fail")
	}
	// 替换已在跑的账号（重启）不受上限影响。
	if err := m.checkWorkerLimit("w" + string(rune('a'))); err != nil {
		t.Fatalf("restarting existing account must pass: %v", err)
	}
}

// TestLifecycleLocksAreLeafLocks 防死锁检查：isStaleGeneration / lifecycleLock
// 等 m.mu 使用方不会在持有 m.mu 时再等生命周期锁（顺序：生命周期锁 → m.mu）。
func TestLifecycleLocksAreLeafLocks(t *testing.T) {
	m := NewAccountManager(nil)
	done := make(chan struct{})
	go func() {
		defer close(done)
		lock := m.lifecycleLock("a1")
		lock.Lock()
		defer lock.Unlock()
		_ = m.isStaleGeneration("a1", 0)
		_ = m.generation("a1")
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("m.mu helpers must not deadlock against a held lifecycle lock")
	}
	// 并发压一下锁表扩容与世代递增（-race 下验证无数据竞争）。
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := "acc" + string(rune('0'+n%4))
			lock := m.lifecycleLock(id)
			lock.Lock()
			m.mu.Lock()
			_ = m.nextGenerationLocked(id)
			m.mu.Unlock()
			lock.Unlock()
		}(i)
	}
	wg.Wait()
}
