package runtime

// 好友护主犬缓存与每日同步（对齐 bot pet-cache.ts + pet-sync.ts）。
//
// 数据只有一个来源：VisitService.Enter 回包的 brief_dog_info.dog_id。
// 所有进入好友农场的调用都顺手写入这里（偷菜、帮忙、捣乱、面板手动操作），
// 真正额外花 RPC 的只有每日同步的批量补齐。新鲜度按系统日期判定，跨日一律视为未知。

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/farm/game"
	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
	"github.com/it00021hot/qq-farm-core/internal/farm/protocol"
)

// logicInQuietHours applies the account's friend quiet-hours window.
func logicInQuietHours(cfg logic.AccountConfig) bool {
	return logic.InQuietHours(cfg.FriendQuietHours.Enabled, cfg.FriendQuietHours.Start, cfg.FriendQuietHours.End, time.Now().Format("15:04"))
}

// ProtectDogID mirrors bot PROTECT_DOG_ID (护主犬).
const ProtectDogID int64 = 90021

// pet-cache pacing constants (bot FRIEND_PET_SYNC_TUNING).
const (
	petSyncBatchSize        = 5
	petSyncGap              = 2 * time.Second
	petSyncBatchGap         = 3 * time.Second
	petSyncQuotaBase        = 10
	petSyncQuotaStep        = 5
	petSyncQuotaCap         = 25
	petSyncBusyCooldown     = 30 * time.Minute
	petSyncCheckInterval    = 10 * time.Minute
	petSyncFastInterval     = 3 * time.Minute
	petSyncContentionRetry  = time.Minute
	petSyncStartupDelay     = 90 * time.Second
	petSyncIdleWaitMax      = 8 * time.Second
	petSyncIdlePoll         = 250 * time.Millisecond
	petSyncFriendTaskWait   = 10 * time.Second
	petCacheFlushDebounce   = 2 * time.Second
)

// friendDogEntry is one day-scoped conclusion.
type friendDogEntry struct {
	DogID     int64 `json:"dogId"`
	Date      string `json:"date"`
	CheckedAt int64 `json:"checkedAt"`
}

type friendPetCacheFile struct {
	Version         int                        `json:"version"`
	LastFullSyncDate string                     `json:"lastFullSyncDate"`
	Entries         map[string]friendDogEntry   `json:"entries"`
}

// friendPetCache is the per-session dog cache with file persistence.
type friendPetCache struct {
	mu       sync.Mutex
	path     string
	entries  map[int64]friendDogEntry
	fullSyncDate string
	loaded   bool
	flushTimer *time.Timer
}

func newFriendPetCache(dataRoot, accountID string) *friendPetCache {
	sum := sha256.Sum256([]byte(accountID))
	token := hex.EncodeToString(sum[:])
	return &friendPetCache{
		path:    filepath.Join(dataRoot, "friend-pet-"+token+".json"),
		entries: make(map[int64]friendDogEntry),
	}
}

func todayKey() string { return time.Now().Format("2006-01-02") }

func (c *friendPetCache) loadLocked() {
	if c.loaded {
		return
	}
	c.loaded = true
	raw, err := os.ReadFile(c.path)
	if err != nil {
		return
	}
	var file friendPetCacheFile
	if json.Unmarshal(raw, &file) != nil || file.Version != 1 {
		return
	}
	c.fullSyncDate = file.LastFullSyncDate
	today := todayKey()
	if c.fullSyncDate != "" && c.fullSyncDate != today {
		c.fullSyncDate = ""
	}
	for key, entry := range file.Entries {
		gid, _ := strconv.ParseInt(key, 10, 64)
		if gid <= 0 || entry.Date != today {
			continue // 跨日记录没有价值，加载时直接丢掉
		}
		c.entries[gid] = entry
	}
}

func (c *friendPetCache) flushLocked() {
	today := todayKey()
	file := friendPetCacheFile{Version: 1, LastFullSyncDate: c.fullSyncDate, Entries: make(map[string]friendDogEntry, len(c.entries))}
	for gid, entry := range c.entries {
		if entry.Date != today {
			continue
		}
		file.Entries[strconv.FormatInt(gid, 10)] = entry
	}
	raw, err := json.Marshal(file)
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(c.path), 0o755)
	if err := os.WriteFile(c.path, raw, 0o644); err != nil {
		slog.Warn("保存好友宠物缓存失败", "err", err)
	}
}


func (c *friendPetCache) scheduleFlushLocked() {
	if c.flushTimer != nil {
		c.flushTimer.Stop()
	}
	c.flushTimer = time.AfterFunc(petCacheFlushDebounce, func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.flushLocked()
	})
}

func (c *friendPetCache) dropStaleLocked() {
	today := todayKey()
	changed := false
	for gid, entry := range c.entries {
		if entry.Date != today {
			delete(c.entries, gid)
			changed = true
		}
	}
	if c.fullSyncDate != "" && c.fullSyncDate != today {
		c.fullSyncDate = ""
		changed = true
	}
	if changed {
		c.scheduleFlushLocked()
	}
}

// record stores one friend's deployed dog; dogID 0 (no dog) is a valid conclusion.
func (c *friendPetCache) record(gid, dogID int64) {
	if gid <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.loadLocked()
	c.dropStaleLocked()
	if dogID < 0 {
		dogID = 0
	}
	today := todayKey()
	previous, existed := c.entries[gid]
	c.entries[gid] = friendDogEntry{DogID: dogID, Date: today, CheckedAt: time.Now().UnixMilli()}
	// 同一天内狗没变就不必反复落盘，只有结论变化或首次确认才写文件
	if !existed || previous.DogID != dogID {
		c.scheduleFlushLocked()
	}
}

// state returns the three-state conclusion for the friend.
func (c *friendPetCache) state(gid int64) friendDogState {
	if gid <= 0 {
		return dogStateUnknown
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.loadLocked()
	c.dropStaleLocked()
	entry, ok := c.entries[gid]
	if !ok {
		return dogStateUnknown
	}
	if entry.DogID == ProtectDogID {
		return dogStateProtect
	}
	return dogStateNone
}

func (c *friendPetCache) knownToday(gid int64) bool { return c.state(gid) != dogStateUnknown }

// dogID 返回该好友当前宠物的道具 ID（0 表示无宠物/未知）。
func (c *friendPetCache) dogID(gid int64) int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.loadLocked()
	entry, ok := c.entries[gid]
	if !ok {
		return 0
	}
	return entry.DogID
}

// FriendPetBadge 返回好友宠物状态徽标数据（对齐 rust scheduler
// apply_pet_state_overlays：protect/other/unknown 三态 + 宠物道具 ID）。
func (s *Session) FriendPetBadge(gid int64) (state string, dogID int64) {
	if s.petCache == nil {
		return "unknown", 0
	}
	switch s.petCache.state(gid) {
	case dogStateProtect:
		return "protect", ProtectDogID
	case dogStateNone:
		return "other", s.petCache.dogID(gid)
	default:
		return "unknown", 0
	}
}

func (c *friendPetCache) forget(gid int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.loadLocked()
	if _, ok := c.entries[gid]; ok {
		delete(c.entries, gid)
		c.scheduleFlushLocked()
	}
}

func (c *friendPetCache) fullSyncDoneToday() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.loadLocked()
	c.dropStaleLocked()
	return c.fullSyncDate == todayKey()
}

func (c *friendPetCache) markFullSyncDone() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.loadLocked()
	c.fullSyncDate = todayKey()
	c.flushLocked()
}

func (c *friendPetCache) stats() (known, protect int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.loadLocked()
	c.dropStaleLocked()
	for _, entry := range c.entries {
		known++
		if entry.DogID == ProtectDogID {
			protect++
		}
	}
	return known, protect
}

// FlushNow persists pending writes (called on shutdown).
func (c *friendPetCache) FlushNow() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.loadLocked()
	c.flushLocked()
}

// --- Session integration ---

// ensurePetCache lazily builds the session's dog cache.
func (s *Session) ensurePetCache() *friendPetCache {
	s.mu.Lock()
	if s.petCache == nil {
		s.petCache = newFriendPetCache(s.cfg.DataRoot, s.id)
	}
	cache := s.petCache
	s.mu.Unlock()
	return cache
}

// getFriendDogState returns the daily three-state dog conclusion.
func (s *Session) getFriendDogState(gid int64) friendDogState {
	if s == nil {
		return dogStateUnknown
	}
	return s.ensurePetCache().state(gid)
}

// recordFriendDogFromEnter stores the dog seen in an Enter reply (zero extra RPC).
func (s *Session) recordFriendDogFromEnter(gid int64, detail *game.EnterReplyDetail) {
	if s == nil || detail == nil {
		return
	}
	s.ensurePetCache().record(gid, detail.DogID)
}

// --- daily background sync (bot pet-sync.ts) ---

// petSyncPacing is the adaptive round pacing state.
type petSyncPacing struct {
	quota      int
	rampLocked bool
}

// friendPetSyncLoop runs the self-renewing sync round chain.
func (s *Session) friendPetSyncLoop(ctx context.Context) {
	delay := petSyncStartupDelay
	pacing := petSyncPacing{quota: petSyncQuotaBase}
	for {
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		result := s.runFriendPetSyncRound(ctx)
		delay, pacing = planNextPetSyncPacing(result, pacing)
	}
}

// petSyncResult mirrors bot FriendPetSyncResult (subset).
type petSyncResult struct {
	Outcome string // skipped | fresh | synced | deferred | error
	Reason  string
}

// planNextPetSyncPacing mirrors bot planNextSyncPacing.
func planNextPetSyncPacing(result petSyncResult, current petSyncPacing) (time.Duration, petSyncPacing) {
	switch result.Reason {
	case "gateway_busy":
		return petSyncCheckInterval, petSyncPacing{quota: petSyncQuotaBase, rampLocked: true}
	case "gateway_contention", "friend_task_busy":
		return petSyncContentionRetry, petSyncPacing{quota: petSyncQuotaBase, rampLocked: true}
	}
	if result.Outcome == "deferred" && result.Reason == "round_quota" {
		quota := current.quota
		if !current.rampLocked {
			quota = current.quota + petSyncQuotaStep
			if quota > petSyncQuotaCap {
				quota = petSyncQuotaCap
			}
		}
		return petSyncFastInterval, petSyncPacing{quota: quota, rampLocked: current.rampLocked}
	}
	return petSyncCheckInterval, current
}

// petSyncGate mirrors bot isSyncEnabled.
func (s *Session) petSyncGate() (bool, string) {
	cfg := s.Config()
	if !cfg.Automation.Friend {
		return false, "friend_off"
	}
	if !cfg.Automation.FriendHelp {
		return false, "friend_help_off"
	}
	// 护主犬开关关闭时这份数据没有消费方，一个额外 RPC 都不应该花。
	if !cfg.Automation.FriendHelpProtectDogIgnoreExpLimit {
		return false, "protect_dog_bypass_off"
	}
	return true, ""
}

// runFriendPetSyncRound probes unknown friends' dogs in background class with
// the bot pacing model (batches of 5, 2s gaps, adaptive quota).
func (s *Session) runFriendPetSyncRound(ctx context.Context) petSyncResult {
	api := s.GameAPI()
	if api == nil {
		return petSyncResult{Outcome: "skipped", Reason: "not_logged_in"}
	}
	if ok, reason := s.petSyncGate(); !ok {
		return petSyncResult{Outcome: "skipped", Reason: reason}
	}
	cache := s.ensurePetCache()
	if cache.fullSyncDoneToday() {
		return petSyncResult{Outcome: "fresh", Reason: "done_today"}
	}
	cfg := s.Config()
	if logicInQuietHours(cfg) {
		return petSyncResult{Outcome: "skipped", Reason: "quiet_hours"}
	}
	myGID := s.GID()
	if myGID == 0 {
		return petSyncResult{Outcome: "skipped", Reason: "not_logged_in"}
	}

	bgCtx := protocol.WithRequestClass(ctx, protocol.ClassBackground)
	friends, err := getFriendsList(bgCtx, s, api, cfg, false)
	if err != nil {
		return petSyncResult{Outcome: "error", Reason: err.Error()}
	}
	blacklist := makeIDSet(cfg.FriendBlacklist)
	pending := make([]friendVisitTarget, 0)
	seen := make(map[int64]struct{}, len(friends))
	for i := range friends {
		gid := friends[i].Gid
		if gid <= 0 || gid == myGID {
			continue
		}
		if _, dup := seen[gid]; dup {
			continue
		}
		seen[gid] = struct{}{}
		if _, blocked := blacklist[gid]; blocked {
			continue
		}
		if cache.knownToday(gid) {
			continue
		}
		name := friends[i].Remark
		if name == "" {
			name = friends[i].Name
		}
		pending = append(pending, friendVisitTarget{GID: gid, Name: name, Level: friends[i].Level})
	}
	if len(pending) == 0 {
		cache.markFullSyncDone()
		return petSyncResult{Outcome: "fresh", Reason: "all_known"}
	}

	quota := petSyncQuotaBase
	targets := pending
	if len(targets) > quota {
		targets = targets[:quota]
	}
	slog.Info("开始同步好友宠物", "account", parseAccountID(s.id), "round", len(targets), "pending", len(pending))

	checked, failed := 0, 0
	deferReason := ""
	for index := 0; index < len(targets); index += petSyncBatchSize {
		end := index + petSyncBatchSize
		if end > len(targets) {
			end = len(targets)
		}
		yielded := false
		for _, target := range targets[index:end] {
			select {
			case <-ctx.Done():
				return petSyncResult{Outcome: "deferred", Reason: "canceled"}
			default:
			}
			if ok, _ := s.petSyncGate(); !ok {
				deferReason = "switch_off"
				yielded = true
				break
			}
			if s.FriendCheckRunning() {
				// 好友巡查正在占着进农场这个状态，把剩下的好友留给下一次定时检查
				deferReason = "friend_task_busy"
				yielded = true
				break
			}
			detail, enterErr := api.VisitEnterDetailed(bgCtx, target.GID, enterReasonFriend)
			if enterErr != nil {
				if protocol.IsGatewayYieldError(enterErr) {
					deferReason = classifyGatewayDefer(s)
					yielded = true
					break
				}
				handleFriendEnterError(s, target.GID, enterErr)
				failed++
			} else {
				s.recordFriendDogFromEnter(target.GID, detail)
				_ = api.VisitLeave(protocol.WithRequestClass(bgCtx, protocol.ClassFriend), target.GID)
				checked++
			}
			select {
			case <-ctx.Done():
			case <-time.After(petSyncGap):
			}
		}
		if yielded {
			break
		}
		if end < len(targets) {
			select {
			case <-ctx.Done():
			case <-time.After(petSyncBatchGap):
			}
		}
	}

	if deferReason == "" {
		cache.markFullSyncDone()
	}
	known, protect := cache.stats()
	slog.Info("好友宠物同步完成",
		"account", parseAccountID(s.id),
		"checked", checked, "failed", failed,
		"defer", len(pending)-checked-failed, "reason", deferReason,
		"known", known, "protect", protect)
	if deferReason != "" {
		return petSyncResult{Outcome: "deferred", Reason: deferReason}
	}
	return petSyncResult{Outcome: "synced"}
}

// classifyGatewayDefer mirrors bot: healthy-but-busy → contention, silent → 30min cooldown.
func classifyGatewayDefer(s *Session) string {
	client := s.Client()
	if client != nil && client.IsGatewayHealthyForBusiness() {
		return "gateway_contention"
	}
	return "gateway_busy"
}
