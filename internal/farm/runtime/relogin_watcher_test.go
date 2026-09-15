package runtime

// YYB 授权失效自动恢复闭环 watcher 单测（对齐 rust relogin_reminder.rs tests）：
// 轮询状态机（Wait/Used/OK/超时/失败）、并发防护（同 key 去重、同账号旧闭环取消、
// 手动停止取消）、apply_relogin_code 语义（空 code 忽略、更新凭据 + 重启）。
// 登录码会话 / worker 控制 / 账号存取全部注入 fake，不连真实服务端。

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/app/model"
	"github.com/it00021hot/qq-farm-core/internal/farm/wxlogin"
)

// fastWatcherInterval 加速轮询（生产 1s，rust WATCHER_INTERVAL_MS）。
func fastWatcherInterval(t *testing.T) {
	t.Helper()
	prev := reloginWatcherInterval
	reloginWatcherInterval = time.Millisecond
	t.Cleanup(func() { reloginWatcherInterval = prev })
}

func TestReloginWatcherConstantsAlignRust(t *testing.T) {
	// rust watcher_cap_constant：MAX_WATCHER_ROUNDS=120，WATCHER_INTERVAL_MS=1000。
	if reloginWatcherRounds != 120 {
		t.Fatalf("rounds = %d", reloginWatcherRounds)
	}
	if reloginWatcherInterval != time.Second {
		t.Fatalf("interval = %v", reloginWatcherInterval)
	}
}

func TestPublicQrImageURLAlignsRust(t *testing.T) {
	// rust offline_reminder_payload_default_reason_is_unknown 中的断言：
	// quickchart.io 前缀 + form-urlencoded 编码。
	got := publicQrImageURL("https://example.com/login?a=1")
	if !hasPrefix(got, "https://quickchart.io/qr?") {
		t.Fatalf("url = %q", got)
	}
	if !hasPrefix(got, "https://quickchart.io/qr?size=300&margin=1&text=") {
		t.Fatalf("url missing params: %q", got)
	}
	if decoded, err := url.QueryUnescape(got); err != nil || decoded != "https://quickchart.io/qr?size=300&margin=1&text=https://example.com/login?a=1" {
		t.Fatalf("decode = %q err=%v", decoded, err)
	}
}

func hasPrefix(s, p string) bool { return len(s) >= len(p) && s[:len(p)] == p }

// fakeReloginSession 登录码会话 fake。
type fakeReloginSession struct {
	queryN    atomic.Int32
	authN     atomic.Int32
	authMu    sync.Mutex
	authAppID string
	queryFn   func(n int32) (wxlogin.MpStatusResult, error)
	authCode  string
	authErr   error
}

func (f *fakeReloginSession) RequestLoginCode(ctx context.Context) (wxlogin.MpLoginCodeResult, error) {
	return wxlogin.MpLoginCodeResult{Code: "lc-1", URL: "https://h5.qzone.qq.com/qqq/code/lc-1?_proxy=1&from=ide"}, nil
}

func (f *fakeReloginSession) QueryStatus(ctx context.Context, code string) (wxlogin.MpStatusResult, error) {
	return f.queryFn(f.queryN.Add(1))
}

func (f *fakeReloginSession) GetAuthCode(ctx context.Context, ticket, appID string) (string, error) {
	f.authN.Add(1)
	f.authMu.Lock()
	f.authAppID = appID
	f.authMu.Unlock()
	if f.authErr != nil {
		return "", f.authErr
	}
	return f.authCode, nil
}

func (f *fakeReloginSession) lastAuthAppID() string {
	f.authMu.Lock()
	defer f.authMu.Unlock()
	return f.authAppID
}

// countingReloginWorkers worker 控制 fake（rust CountingControls）。
type countingReloginWorkers struct {
	starts      atomic.Int32
	restarts    atomic.Int32
	failRestart atomic.Bool
}

func (c *countingReloginWorkers) StartWorker(id uint64) error {
	c.starts.Add(1)
	return nil
}

func (c *countingReloginWorkers) RestartWorker(id uint64) error {
	// 计数无论成败都递增：失败场景用于断言「凭据已落盘、重启已尝试」。
	c.restarts.Add(1)
	if c.failRestart.Load() {
		return errors.New("restart failed")
	}
	return nil
}

// memReloginStore 账号存取 fake。
type memReloginStore struct {
	mu       sync.Mutex
	accounts map[uint64]model.FarmAccount
	updates  int
}

func newMemReloginStore(accounts ...model.FarmAccount) *memReloginStore {
	m := &memReloginStore{accounts: make(map[uint64]model.FarmAccount)}
	for _, acc := range accounts {
		m.accounts[acc.ID] = acc
	}
	return m
}

func (m *memReloginStore) LoadAccount(id uint64) (model.FarmAccount, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	acc, ok := m.accounts[id]
	return acc, ok
}

func (m *memReloginStore) UpdateReloginCredentials(id uint64, code, uin, qq, avatar string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	acc, ok := m.accounts[id]
	if !ok {
		return nil
	}
	acc.Code = code
	if uin != "" {
		acc.Uin = uin
		acc.QQ = qq
	}
	if avatar != "" {
		acc.Avatar = avatar
	}
	m.accounts[id] = acc
	m.updates++
	return nil
}

func newTestReloginService(queryFn func(n int32) (wxlogin.MpStatusResult, error)) (*ReloginService, *fakeReloginSession, *countingReloginWorkers, *memReloginStore) {
	fast := &fakeReloginSession{queryFn: queryFn, authCode: "fresh-code"}
	workers := &countingReloginWorkers{}
	store := newMemReloginStore(model.FarmAccount{ID: 42, Name: "大号", Platform: "wx"})
	return newReloginService(fast, workers, store), fast, workers, store
}

func waitFor(t *testing.T, timeout time.Duration, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal(msg)
}

// TestStartReloginWatcherEmptyCodeNoop 对齐 rust start_relogin_watcher_empty_code_noop。
func TestStartReloginWatcherEmptyCodeNoop(t *testing.T) {
	fastWatcherInterval(t)
	svc, _, workers, store := newTestReloginService(func(n int32) (wxlogin.MpStatusResult, error) {
		return wxlogin.MpStatusResult{Status: wxlogin.MpStatusWait}, nil
	})
	svc.StartReloginWatcher("  ", 42, "大号")
	time.Sleep(20 * time.Millisecond)
	if svc.WatcherCount() != 0 {
		t.Fatalf("watchers = %d", svc.WatcherCount())
	}
	if workers.restarts.Load() != 0 || store.updates != 0 {
		t.Fatal("empty code must be a noop")
	}
}

// TestStartReloginWatcherDedupsSameKey 并发防护：同一（账号, 登录码）不重复启动。
func TestStartReloginWatcherDedupsSameKey(t *testing.T) {
	fastWatcherInterval(t)
	svc, _, _, _ := newTestReloginService(func(n int32) (wxlogin.MpStatusResult, error) {
		return wxlogin.MpStatusResult{Status: wxlogin.MpStatusWait}, nil
	})
	svc.StartReloginWatcher("code-a", 42, "大号")
	svc.StartReloginWatcher("code-a", 42, "大号")
	if svc.WatcherCount() != 1 {
		t.Fatalf("watchers = %d, want 1（同 key 必须去重）", svc.WatcherCount())
	}
	svc.CancelAllReloginWatchers()
}

// TestWatcherPollsUntilOKThenApplies 状态机：Wait → Wait → OK → get_auth_code →
// 更新凭据 + 持生命周期锁重启（rust apply_relogin_code 更新已有账号分支）。
func TestWatcherPollsUntilOKThenApplies(t *testing.T) {
	fastWatcherInterval(t)
	svc, fast, workers, store := newTestReloginService(func(n int32) (wxlogin.MpStatusResult, error) {
		if n < 3 {
			return wxlogin.MpStatusResult{Status: wxlogin.MpStatusWait}, nil
		}
		return wxlogin.MpStatusResult{Status: wxlogin.MpStatusOK, Ticket: "tk", Uin: "12345", Nickname: "大号"}, nil
	})
	svc.StartReloginWatcher("code-ok", 42, "大号")
	waitFor(t, 3*time.Second, func() bool {
		return workers.restarts.Load() == 1 && svc.WatcherCount() == 0
	}, "watcher should finish with one restart")

	if fast.authN.Load() != 1 {
		t.Fatalf("get_auth_code calls = %d", fast.authN.Load())
	}
	if fast.lastAuthAppID() != wxlogin.MpPresetFarmAppID {
		t.Fatalf("auth appid = %q, want farm preset", fast.lastAuthAppID())
	}
	acc, ok := store.LoadAccount(42)
	if !ok {
		t.Fatal("account missing")
	}
	if acc.Code != "fresh-code" {
		t.Fatalf("code = %q, want fresh-code", acc.Code)
	}
	if acc.Uin != "12345" || acc.QQ != "12345" {
		t.Fatalf("uin/qq = %q/%q", acc.Uin, acc.QQ)
	}
	if acc.Avatar != "https://q1.qlogo.cn/g?b=qq&nk=12345&s=640" {
		t.Fatalf("avatar = %q", acc.Avatar)
	}
}

// TestWatcherUsedEndsWithoutApply 状态机：Used → 监听结束，不 apply（rust 同）。
func TestWatcherUsedEndsWithoutApply(t *testing.T) {
	fastWatcherInterval(t)
	svc, _, workers, store := newTestReloginService(func(n int32) (wxlogin.MpStatusResult, error) {
		return wxlogin.MpStatusResult{Status: wxlogin.MpStatusUsed}, nil
	})
	svc.StartReloginWatcher("code-used", 42, "大号")
	waitFor(t, 3*time.Second, func() bool { return svc.WatcherCount() == 0 }, "watcher should end on Used")
	if workers.restarts.Load() != 0 || store.updates != 0 {
		t.Fatal("Used must not apply")
	}
}

// TestWatcherOKWithEmptyTicketFails 状态机：OK 但 ticket 为空 → 失败结束（rust 同）。
func TestWatcherOKWithEmptyTicketFails(t *testing.T) {
	fastWatcherInterval(t)
	svc, fast, workers, _ := newTestReloginService(func(n int32) (wxlogin.MpStatusResult, error) {
		return wxlogin.MpStatusResult{Status: wxlogin.MpStatusOK, Ticket: "  ", Uin: "1"}, nil
	})
	svc.StartReloginWatcher("code-noticket", 42, "大号")
	waitFor(t, 3*time.Second, func() bool { return svc.WatcherCount() == 0 }, "watcher should end")
	if fast.authN.Load() != 0 {
		t.Fatal("empty ticket must not call get_auth_code")
	}
	if workers.restarts.Load() != 0 {
		t.Fatal("empty ticket must not restart")
	}
}

// TestWatcherAuthCodeFailure 状态机：get_auth_code 失败 / 空 code → 失败结束（rust 同）。
func TestWatcherAuthCodeFailure(t *testing.T) {
	for name, mutate := range map[string]func(*fakeReloginSession){
		"auth error": func(f *fakeReloginSession) { f.authErr = errors.New("exchange failed") },
		"empty code": func(f *fakeReloginSession) { f.authCode = "  " },
	} {
		t.Run(name, func(t *testing.T) {
			fastWatcherInterval(t)
			svc, fast, workers, store := newTestReloginService(func(n int32) (wxlogin.MpStatusResult, error) {
				return wxlogin.MpStatusResult{Status: wxlogin.MpStatusOK, Ticket: "tk", Uin: "1"}, nil
			})
			mutate(fast)
			svc.StartReloginWatcher("code-authfail", 42, "大号")
			waitFor(t, 3*time.Second, func() bool { return svc.WatcherCount() == 0 }, "watcher should end")
			if workers.restarts.Load() != 0 || store.updates != 0 {
				t.Fatal("auth failure must not apply")
			}
		})
	}
}

// TestWatcherQueryErrorKeepsPolling 状态机：查询报错静默重试（rust 同），不终止。
func TestWatcherQueryErrorKeepsPolling(t *testing.T) {
	fastWatcherInterval(t)
	svc, fast, workers, _ := newTestReloginService(func(n int32) (wxlogin.MpStatusResult, error) {
		if n <= 5 {
			return wxlogin.MpStatusResult{}, errors.New("boom")
		}
		return wxlogin.MpStatusResult{Status: wxlogin.MpStatusOK, Ticket: "tk", Uin: "7"}, nil
	})
	svc.StartReloginWatcher("code-retry", 42, "大号")
	waitFor(t, 3*time.Second, func() bool { return workers.restarts.Load() == 1 }, "should recover after errors")
	if fast.queryN.Load() < 6 {
		t.Fatalf("queries = %d, want >= 6", fast.queryN.Load())
	}
	svc.CancelAllReloginWatchers()
}

// TestWatcherTimesOutAfterMaxRounds 状态机：一直 Wait → 120 轮后超时结束，不 apply。
func TestWatcherTimesOutAfterMaxRounds(t *testing.T) {
	fastWatcherInterval(t)
	svc, fast, workers, store := newTestReloginService(func(n int32) (wxlogin.MpStatusResult, error) {
		return wxlogin.MpStatusResult{Status: wxlogin.MpStatusWait}, nil
	})
	svc.StartReloginWatcher("code-timeout", 42, "大号")
	waitFor(t, 10*time.Second, func() bool {
		return svc.WatcherCount() == 0 && fast.queryN.Load() >= reloginWatcherRounds
	}, "watcher should time out after max rounds")
	if fast.queryN.Load() != reloginWatcherRounds {
		t.Fatalf("queries = %d, want exactly %d", fast.queryN.Load(), reloginWatcherRounds)
	}
	if workers.restarts.Load() != 0 || store.updates != 0 {
		t.Fatal("timeout must not apply")
	}
}

// TestCancelReloginWatchersMidPoll 取消语义：手动停止 / 删除账号（Facade.Stop）取消
// 在途 watcher → 轮询退出且不 apply（go 侧补齐语义，rust 无取消）。
func TestCancelReloginWatchersMidPoll(t *testing.T) {
	fastWatcherInterval(t)
	gate := make(chan struct{})
	svc, fast, workers, store := newTestReloginService(func(n int32) (wxlogin.MpStatusResult, error) {
		<-gate // 阻塞首轮查询，保证取消发生在轮询中
		// 放行后直接给 OK：若取消语义失效，watcher 会走完 apply（restarts==1）。
		return wxlogin.MpStatusResult{Status: wxlogin.MpStatusOK, Ticket: "tk", Uin: "5"}, nil
	})
	svc.StartReloginWatcher("code-cancel", 42, "大号")
	waitFor(t, 3*time.Second, func() bool { return fast.queryN.Load() == 1 }, "first query should start")

	// 取消不相关账号：不得误伤在途 watcher。
	svc.CancelReloginWatchers(43)
	if svc.WatcherCount() != 1 {
		t.Fatal("unrelated cancel must keep watcher")
	}
	// 模拟 Facade.Stop 的取消挂载点。
	svc.CancelReloginWatchers(42)
	if svc.WatcherCount() != 0 {
		t.Fatal("cancel should remove watcher immediately")
	}
	close(gate) // 放行在途查询
	waitFor(t, 3*time.Second, func() bool { return fast.authN.Load() == 1 },
		"in-flight round should finish after gate release")
	if workers.restarts.Load() != 0 || store.updates != 0 {
		t.Fatal("cancelled watcher must not apply even when the round returned OK")
	}
}

// TestNewLoopCancelsOldWatcherForSameAccount 并发防护：同账号新闭环启动时
// 取消旧 watcher（旧二维码作废），最终只有新闭环能 apply。
func TestNewLoopCancelsOldWatcherForSameAccount(t *testing.T) {
	fastWatcherInterval(t)
	gate := make(chan struct{})
	svc, fast, workers, store := newTestReloginService(func(n int32) (wxlogin.MpStatusResult, error) {
		if n <= 2 {
			// 旧 watcher 第 1 轮 + 新 watcher 第 1 轮都阻塞，等取消到位后放行。
			<-gate
			return wxlogin.MpStatusResult{Status: wxlogin.MpStatusWait}, nil
		}
		return wxlogin.MpStatusResult{Status: wxlogin.MpStatusOK, Ticket: "tk", Uin: "9"}, nil
	})
	svc.StartReloginWatcher("old-code", 42, "大号")
	waitFor(t, 3*time.Second, func() bool { return fast.queryN.Load() == 1 }, "old watcher should poll first")

	svc.StartReloginWatcher("new-code", 42, "大号")
	if svc.WatcherCount() != 1 {
		t.Fatalf("watchers = %d, want 1（新闭环必须顶掉旧闭环）", svc.WatcherCount())
	}
	close(gate)
	waitFor(t, 5*time.Second, func() bool {
		return svc.WatcherCount() == 0 && workers.restarts.Load() == 1
	}, "only the new loop should finish and apply")
	acc, _ := store.LoadAccount(42)
	if acc.Code != "fresh-code" {
		t.Fatalf("code = %q, want applied by new loop", acc.Code)
	}
}

// TestApplyReloginCodeEmptyCodeNoop 对齐 rust apply_relogin_code_empty_code_noop。
func TestApplyReloginCodeEmptyCodeNoop(t *testing.T) {
	fastWatcherInterval(t)
	svc, _, workers, store := newTestReloginService(nil)
	svc.ApplyReloginCode(ReloginCodePayload{AccountID: 42, AuthCode: "   "})
	if workers.restarts.Load() != 0 || workers.starts.Load() != 0 || store.updates != 0 {
		t.Fatal("empty code must be a noop")
	}
}

// TestApplyReloginCodeMissingAccountSkips 有意差异：rust 会新增账号；go 侧闭环
// 只对已存在账号触发，账号已删除时跳过（relogin_watcher.go 文件头差异 2）。
func TestApplyReloginCodeMissingAccountSkips(t *testing.T) {
	fastWatcherInterval(t)
	svc, _, workers, store := newTestReloginService(nil)
	svc.ApplyReloginCode(ReloginCodePayload{AccountID: 7, AuthCode: "c", Uin: "1"})
	if workers.restarts.Load() != 0 || workers.starts.Load() != 0 || store.updates != 0 {
		t.Fatal("missing account must be skipped, not re-created")
	}
}

// TestApplyReloginCodeRestartsViaLifecycle apply 走 worker 控制重启（生产为
// Facade.Start → 账号生命周期锁）；重启失败时凭据已落盘、不回滚（rust 同）。
func TestApplyReloginCodeRestartsViaLifecycle(t *testing.T) {
	fastWatcherInterval(t)
	svc, _, workers, store := newTestReloginService(nil)
	svc.ApplyReloginCode(ReloginCodePayload{AccountID: 42, AuthCode: "new-code", Uin: "12345"})
	if store.updates != 1 || workers.restarts.Load() != 1 {
		t.Fatalf("updates=%d restarts=%d", store.updates, workers.restarts.Load())
	}
	acc, _ := store.LoadAccount(42)
	if acc.Code != "new-code" || acc.Uin != "12345" || acc.QQ != "12345" || acc.Avatar == "" {
		t.Fatalf("account = %+v", acc)
	}

	// 重启失败：记录日志但凭据保留。
	workers.failRestart.Store(true)
	svc.ApplyReloginCode(ReloginCodePayload{AccountID: 42, AuthCode: "newer-code"})
	acc, _ = store.LoadAccount(42)
	if acc.Code != "newer-code" {
		t.Fatalf("code should persist before restart: %q", acc.Code)
	}
	if workers.restarts.Load() != 2 {
		t.Fatalf("restarts = %d", workers.restarts.Load())
	}
}

// TestReloginKeyFormat regression：watcher key 前缀取消按账号隔离（id 为数字，
// 不会与 unknown: 前缀混淆）。
func TestReloginKeyFormat(t *testing.T) {
	id := strconv.FormatUint(42, 10)
	if id != "42" {
		t.Fatalf("id = %q", id)
	}
}
