package runtime

// YYB 授权失效自动恢复闭环 — 移植 rust runtime/relogin_reminder.rs 的
// public_qr_image_url / start_relogin_watcher / poll_relogin_status /
// apply_relogin_code，及 trigger_offline_reminder 的 YybQr 闭环分支：
//
//	授权失效 → SendAccountNotice(NoticeYybQr)（offline_reminder.go）
//	→ 申请一次性登录码（wxlogin.MiniProgramLoginSession，rust MiniProgramLoginSession）
//	→ quickchart.io 把扫码 URL 渲染成二维码图片 URL，推给用户 QQ 机器人（先文本后图片）
//	→ 后台 watcher 轮询扫码状态（最多 120 次 × 1s，rust MAX_WATCHER_ROUNDS/WATCHER_INTERVAL_MS）
//	→ 扫码成功 get_auth_code 换新游戏 code → 落盘 → 持账号生命周期锁重启账号
//	全程无需人工去面板操作。
//
// 与 rust 的有意差异：
//  1. 取消语义：rust 的 watcher 无取消（轮询到超时自然结束）；go 侧补齐——手动停止 /
//     删除账号（Facade.Stop / StopAll）与同账号新闭环启动时中止在途 watcher，
//     防止已删除账号被重新拉起、旧二维码覆盖用户刚扫的新凭据。
//  2. apply_relogin_code 对不存在的账号：rust 会新增账号（平台 qq）；go 侧闭环只对
//     已存在账号触发（三个触发点都带真实账号 ID），账号消失视为用户已删除，跳过并记录日志。
//  3. 申请登录码时 rust 同步生成 PNG data URL（qrcode crate）；该字段在闭环中未被
//     消费（QQ Bot 富媒体要求公网 http(s) URL，实际用 quickchart.io 渲染），go 侧省略，
//     避免引入二维码渲染依赖（见 wxlogin/qrlogin.go 文件头）。

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/app/model"
	"github.com/it00021hot/qq-farm-core/internal/farm/wxlogin"
	"github.com/it00021hot/qq-farm-core/internal/vars"
)

// reloginWatcherRounds watcher 轮询上限（rust MAX_WATCHER_ROUNDS = 120）。
const reloginWatcherRounds = 120

// reloginWatcherInterval watcher 单轮间隔（rust WATCHER_INTERVAL_MS = 1000）。
// var 仅为单测可加速轮询，生产保持 1s。
var reloginWatcherInterval = time.Second

// logEventRelogin runtime_log 事件名（rust logger.log 的「重登录监听」系列文案）。
const logEventRelogin = "重登录"

// publicQrImageURL 把扫码 URL 交给 quickchart.io 渲染成二维码图片 URL
// （rust public_qr_image_url：form-urlencoded 编码 + size=300&margin=1）。
func publicQrImageURL(content string) string {
	return "https://quickchart.io/qr?size=300&margin=1&text=" + url.QueryEscape(content)
}

// ReloginCodePayload 重登录码载荷（rust ReloginCodePayload）。
type ReloginCodePayload struct {
	AccountID   uint64
	AccountName string
	AuthCode    string
	Uin         string
}

// reloginSession 登录码会话接口（生产为 *wxlogin.MiniProgramLoginSession，
// 单测注入 fake；对齐 rust Arc<MiniProgramLoginSession> 注入点）。
type reloginSession interface {
	RequestLoginCode(ctx context.Context) (wxlogin.MpLoginCodeResult, error)
	QueryStatus(ctx context.Context, code string) (wxlogin.MpStatusResult, error)
	GetAuthCode(ctx context.Context, ticket, appID string) (string, error)
}

// reloginWorkerControls worker 控制接口（rust WorkerControls）：
// 重登录成功后按生命周期锁重启（或启动）账号 worker。
type reloginWorkerControls interface {
	StartWorker(accountID uint64) error
	RestartWorker(accountID uint64) error
}

// reloginStore 账号存取接口（生产走 vars.DB，单测注入内存实现）。
type reloginStore interface {
	LoadAccount(accountID uint64) (model.FarmAccount, bool)
	UpdateReloginCredentials(accountID uint64, code, uin, qq, avatar string) error
}

// facadeWorkerControls 默认 worker 控制：走 Facade.Start（内部持账号生命周期锁，
// 对齐 rust EngineWorkerControls.restart_worker → engine.restart_worker 的
// lifecycle_lock + 旧会话退出等待 + 宽限语义）。
type facadeWorkerControls struct{}

func (facadeWorkerControls) StartWorker(accountID uint64) error   { return Default.Start(accountID) }
func (facadeWorkerControls) RestartWorker(accountID uint64) error { return Default.Start(accountID) }

// dbReloginStore 默认账号存取（vars.DB）。
type dbReloginStore struct{}

func (dbReloginStore) LoadAccount(accountID uint64) (model.FarmAccount, bool) {
	return loadFarmAccount(accountID)
}

// UpdateReloginCredentials 落盘重登录凭据（rust apply_relogin_code 更新已有账号：
// code 必更；uin 存在时同步 qq/uin/头像，否则沿用原值）。
func (dbReloginStore) UpdateReloginCredentials(accountID uint64, code, uin, qq, avatar string) error {
	if vars.DB == nil || accountID == 0 {
		return nil
	}
	updates := map[string]any{
		"code":       code,
		"updated_at": uint(time.Now().Unix()),
	}
	if uin != "" {
		updates["uin"] = uin
		updates["qq"] = qq
	}
	if avatar != "" {
		updates["avatar"] = avatar
	}
	return vars.DB.Model(&model.FarmAccount{}).Where("id = ?", accountID).Updates(updates).Error
}

// reloginWatchEntry 单个在途 watcher：启动时间 + 取消信号。
type reloginWatchEntry struct {
	startedAt time.Time
	cancel    chan struct{}
	once      sync.Once
}

// ReloginService 重登录提醒服务的 watcher/apply 部分（rust ReloginReminderService；
// 通知推送分发在 offline_reminder.go）。
type ReloginService struct {
	mp      reloginSession
	workers reloginWorkerControls
	store   reloginStore

	mu       sync.Mutex
	watchers map[string]*reloginWatchEntry
}

func newReloginService(mp reloginSession, workers reloginWorkerControls, store reloginStore) *ReloginService {
	return &ReloginService{
		mp:       mp,
		workers:  workers,
		store:    store,
		watchers: make(map[string]*reloginWatchEntry),
	}
}

// defaultRelogin 进程级单例（rust engine 装配的 ReloginReminderService）。
var defaultRelogin = newReloginService(
	wxlogin.NewMiniProgramLoginSession(),
	facadeWorkerControls{},
	dbReloginStore{},
)

// StartYybQrRelogin 申请一次性登录码并启动扫码 watcher（rust trigger_offline_reminder
// 的 YybQr 分支）。返回二维码图片 URL；申请失败返回空串（调用方仍发文本，rust 同）。
// 仅在 QQ Bot 渠道 + 绑定完整时由 SendAccountNotice 调用（rust 同门控）。
func StartYybQrRelogin(accountID uint64, accountName string) string {
	qr, err := defaultRelogin.mp.RequestLoginCode(context.Background())
	if err != nil {
		appendRuntimeLog(accountID, logEventRelogin, "获取重登录链接失败: "+err.Error(), true)
		return ""
	}
	qrImage := ""
	if u := strings.TrimSpace(qr.URL); u != "" {
		qrImage = publicQrImageURL(u)
	}
	// rust：登录码非空才启动 watcher（启动时机在文本发送之前，推送失败也继续轮询）。
	if code := strings.TrimSpace(qr.Code); code != "" {
		defaultRelogin.StartReloginWatcher(code, accountID, accountName)
	}
	return qrImage
}

// StartReloginWatcher 启动扫码状态轮询（rust start_relogin_watcher）：
//   - 相同 key（账号:登录码）已在途则不重复启动（rust watchers 去重）；
//   - go 侧补齐：同账号新闭环启动时先取消旧 watcher（旧二维码作废）。
func (s *ReloginService) StartReloginWatcher(loginCode string, accountID uint64, accountName string) {
	code := strings.TrimSpace(loginCode)
	if code == "" {
		return
	}
	idStr := strconv.FormatUint(accountID, 10)
	key := idStr + ":" + code
	if accountID == 0 {
		key = "unknown:" + code
	}

	s.mu.Lock()
	if _, exists := s.watchers[key]; exists {
		// rust：同 key 已在途，直接返回（防同一二维码重复 watcher）。
		s.mu.Unlock()
		return
	}
	if accountID != 0 {
		for k, e := range s.watchers {
			if strings.HasPrefix(k, idStr+":") {
				s.cancelLocked(k, e)
			}
		}
	}
	entry := &reloginWatchEntry{startedAt: time.Now(), cancel: make(chan struct{})}
	s.watchers[key] = entry
	s.mu.Unlock()

	accLabel := strings.TrimSpace(accountName)
	if accLabel == "" {
		accLabel = idStr
	}
	if accLabel == "" {
		accLabel = "未知账号"
	}
	appendRuntimeLog(accountID, logEventRelogin, "已启动重登录监听: "+accLabel, false)
	go s.watchRelogin(key, entry, code, accountID, accLabel, strings.TrimSpace(accountName))
}

// watchRelogin 轮询协程：轮询到结果后摘除 watcher 并按需 apply（rust spawn 的
// 轮询协程：不论结果如何都清理 watcher）。
func (s *ReloginService) watchRelogin(key string, entry *reloginWatchEntry, code string, accountID uint64, accLabel, accountName string) {
	payload := s.pollReloginStatus(entry, code, accountID, accLabel)
	cancelled := entry.isCancelled()

	// 收尾：摘除 watcher（若未被新闭环替换/取消摘除）并记录结束。
	s.mu.Lock()
	if cur, ok := s.watchers[key]; ok && cur == entry {
		delete(s.watchers, key)
	}
	s.mu.Unlock()
	elapsed := time.Since(entry.startedAt).Milliseconds()
	if cancelled {
		appendRuntimeLog(accountID, logEventRelogin,
			fmt.Sprintf("重登录监听已中止: %s (%dms)", accLabel, elapsed), false)
		return
	}
	appendRuntimeLog(accountID, logEventRelogin,
		fmt.Sprintf("重登录监听结束: %s (%dms)", accLabel, elapsed), false)

	if payload != nil {
		payload.AccountID = accountID // rust：由 caller 注入 account_id / account_name
		payload.AccountName = accountName
		s.ApplyReloginCode(*payload)
	}
}

// pollReloginStatus 轮询登录码状态直到 OK / Used / 失败 / 超时 / 取消
// （rust poll_relogin_status）。成功返回换好的 ReloginCodePayload。
func (s *ReloginService) pollReloginStatus(entry *reloginWatchEntry, code string, accountID uint64, accLabel string) *ReloginCodePayload {
	ctx := context.Background()
	for round := 0; round < reloginWatcherRounds; round++ {
		// go 侧取消语义：手动停止 / 删除账号 / 新闭环启动时立即退出。
		if entry.isCancelled() {
			return nil
		}
		status, err := s.mp.QueryStatus(ctx, code)
		if err != nil {
			// rust：查询出错静默等一轮再试。
			if !entry.sleepRound() {
				return nil
			}
			continue
		}
		switch status.Status {
		case wxlogin.MpStatusWait:
			if !entry.sleepRound() {
				return nil
			}
		case wxlogin.MpStatusUsed:
			appendRuntimeLog(accountID, logEventRelogin, "重登录二维码已失效: "+accLabel, false)
			return nil
		case wxlogin.MpStatusOK:
			ticket := strings.TrimSpace(status.Ticket)
			if ticket == "" {
				appendRuntimeLog(accountID, logEventRelogin, "重登录监听失败: ticket 为空", true)
				return nil
			}
			// rust：get_auth_code(ticket, "1112386029")（farm 预设 appid）。
			authCode, err := s.mp.GetAuthCode(ctx, ticket, wxlogin.MpPresetFarmAppID)
			if err != nil || strings.TrimSpace(authCode) == "" {
				appendRuntimeLog(accountID, logEventRelogin, "重登录监听失败: 未获取到新 code", true)
				return nil
			}
			return &ReloginCodePayload{AuthCode: strings.TrimSpace(authCode), Uin: status.Uin}
		default:
			// 其它状态（Error 等）继续轮询（rust 同）。
			if !entry.sleepRound() {
				return nil
			}
		}
	}
	appendRuntimeLog(accountID, logEventRelogin, "重登录监听超时: "+accLabel, false)
	return nil
}

// isCancelled 报告 watcher 是否已被取消。
func (e *reloginWatchEntry) isCancelled() bool {
	select {
	case <-e.cancel:
		return true
	default:
		return false
	}
}

// sleepRound 等一轮；被取消时返回 false（rust sleep(WATCHER_INTERVAL_MS)）。
func (e *reloginWatchEntry) sleepRound() bool {
	if e.isCancelled() {
		return false
	}
	select {
	case <-e.cancel:
		return false
	case <-time.After(reloginWatcherInterval):
		return true
	}
}

// cancelLocked 关闭取消信号并从注册表摘除（调用方必须持有 s.mu）。
func (s *ReloginService) cancelLocked(key string, e *reloginWatchEntry) {
	e.once.Do(func() { close(e.cancel) })
	if cur, ok := s.watchers[key]; ok && cur == e {
		delete(s.watchers, key)
	}
}

// CancelReloginWatchers 取消某账号全部在途 watcher（go 侧取消语义）。
func (s *ReloginService) CancelReloginWatchers(accountID uint64) {
	idStr := strconv.FormatUint(accountID, 10)
	s.mu.Lock()
	for k, e := range s.watchers {
		if strings.HasPrefix(k, idStr+":") {
			s.cancelLocked(k, e)
		}
	}
	s.mu.Unlock()
}

// CancelAllReloginWatchers 取消全部在途 watcher（停所有账号时调用）。
func (s *ReloginService) CancelAllReloginWatchers() {
	s.mu.Lock()
	for k, e := range s.watchers {
		s.cancelLocked(k, e)
	}
	s.mu.Unlock()
}

// WatcherCount 当前在途 watcher 数（测试用，对齐 rust watcher_count）。
func (s *ReloginService) WatcherCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.watchers)
}

// ApplyReloginCode 应用新 code（rust apply_relogin_code）：更新账号网关 code /
// uin / qq / 头像并持生命周期锁重启 worker。空 code 直接忽略。
func (s *ReloginService) ApplyReloginCode(payload ReloginCodePayload) {
	code := strings.TrimSpace(payload.AuthCode)
	if code == "" {
		return
	}
	uin := strings.TrimSpace(payload.Uin)
	avatar := ""
	if uin != "" {
		// rust：QQ 头像地址按 uin 拼 qlogo。
		avatar = "https://q1.qlogo.cn/g?b=qq&nk=" + uin + "&s=640"
	}

	if payload.AccountID == 0 {
		// rust：无账号 ID 时新增账号（平台 qq）；go 侧闭环只对已存在账号触发，
		// 新增需要归属用户等面板信息，这里按跳过处理（文件头「有意差异」2）。
		slog.Warn("relogin payload missing account id, skip apply", "uin", uin)
		return
	}
	found, ok := s.store.LoadAccount(payload.AccountID)
	if !ok {
		appendRuntimeLog(payload.AccountID, logEventRelogin, "重登录成功但账号已不存在，跳过应用", true)
		return
	}
	display := wxAccountDisplay(found, strconv.FormatUint(payload.AccountID, 10))

	if err := s.store.UpdateReloginCredentials(payload.AccountID, code, uin, uin, avatar); err != nil {
		appendRuntimeLog(payload.AccountID, logEventRelogin, "重登录凭据落盘失败: "+err.Error(), true)
		return
	}
	// 持账号生命周期锁重启（Facade.Start → StartAccount → lifecycle lock + 旧会话
	// 退出等待 + 宽限），对齐 rust restart_worker 语义。
	if err := s.workers.RestartWorker(payload.AccountID); err != nil {
		appendRuntimeLog(payload.AccountID, logEventRelogin, "重登录重启账号失败: "+err.Error(), true)
		return
	}
	// rust：账号日志「重登录成功，已更新账号」+ 全局日志「重登录成功，账号已更新并重启」。
	appendRuntimeLog(payload.AccountID, logEventRelogin, "重登录成功，已更新账号: "+display, false)
	appendRuntimeLog(0, logEventRelogin, "重登录成功，账号已更新并重启: "+display, false)
}

// CancelReloginWatchersFor 挂载点：手动停止 / 删除账号时中止该账号在途
// 重登录 watcher（Facade.Stop 调用）。
func CancelReloginWatchersFor(accountID uint64) {
	defaultRelogin.CancelReloginWatchers(accountID)
}

// CancelAllReloginWatchers 挂载点：停所有账号时取消全部在途 watcher（Facade.StopAll）。
func CancelAllReloginWatchers() {
	defaultRelogin.CancelAllReloginWatchers()
}
