package runtime

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/farm/game"
	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/emailpb"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/taskpb"
	"github.com/it00021hot/qq-farm-core/internal/farm/stats"
)

const (
	ticketItemID             int64 = 1002
	emailCheckCooldown             = 5 * time.Minute
	dailyCheckCooldown             = 10 * time.Minute
	illustratedMinTicketGain int64 = 200
)

// DailyState tracks per-session daily routine progress.
type DailyState struct {
	mu sync.Mutex

	lastDailyDateKey string

	emailDoneDateKey string
	emailLastCheck   time.Time

	shareDoneDateKey string
	shareLastCheck   time.Time

	monthCardDoneDateKey string
	monthCardLastCheck   time.Time

	freeGiftDoneDateKey string
	freeGiftLastCheck   time.Time

	vipDoneDateKey string
	vipLastCheck   time.Time

	greenPlumDoneDateKey string
	greenPlumLastCheck   time.Time

	greenPlumBrewDateKey string
	greenPlumBrewCheck   time.Time

	taskChecking bool
}

func localDateKey(t time.Time) string {
	return t.Format("2006-01-02")
}

// RunDailyRoutines executes email/share/monthcard/vip/free-gift routines.
// Aligned with qq-farm-bot worker.runDailyRoutines: always enabled (no per-flag toggles).
func RunDailyRoutines(ctx context.Context, api *game.API, cfg logic.AccountConfig, state *DailyState, force bool) {
	if api == nil || state == nil {
		return
	}
	_ = cfg
	now := time.Now()
	today := localDateKey(now)

	state.mu.Lock()
	if force || state.lastDailyDateKey != today {
		state.lastDailyDateKey = today
	}
	state.mu.Unlock()

	runEmailClaim(ctx, api, state, force, now)
	runShareClaim(ctx, api, state, force, now)
	runMonthCardClaim(ctx, api, state, force, now)
	runFreeGifts(ctx, api, state, force, now)
	runVipGift(ctx, api, state, force, now)
	runGreenPlumClaim(ctx, api, state, force, now)
	runGreenPlumBrew(ctx, api, state, force, now)
}

// RunTaskClaims auto-claims tasks, actives, and illustrated rewards when Automation.Task is enabled.
func RunTaskClaims(ctx context.Context, api *game.API, cfg logic.AccountConfig, accountID uint64, state *DailyState) {
	if api == nil || state == nil || !cfg.Automation.Task {
		return
	}

	state.mu.Lock()
	if state.taskChecking {
		state.mu.Unlock()
		return
	}
	state.taskChecking = true
	state.mu.Unlock()

	defer func() {
		state.mu.Lock()
		state.taskChecking = false
		state.mu.Unlock()
	}()

	reply, err := api.TaskInfo(ctx)
	if err != nil {
		slog.Warn("task claim: TaskInfo failed", "err", err)
		return
	}
	if reply == nil || reply.TaskInfo == nil {
		return
	}

	normalized := normalizeTaskInfo(reply.TaskInfo)
	claimable := append(
		append(
			analyzeTaskList(normalized.dailyTasks, "daily"),
			analyzeTaskList(normalized.growthTasks, "growth")...,
		),
		analyzeTaskList(normalized.otherTasks, "main")...,
	)

	for _, task := range claimable {
		doShared := task.ShareMultiple > 1
		if _, err := api.ClaimTaskReward(ctx, task.ID, doShared); err != nil {
			continue
		}
		stats.RecordOp(accountID, 0, "taskClaim", 1)
		time.Sleep(300 * time.Millisecond)
	}

	claimActives(ctx, api, normalized.actives)
	claimIllustratedIfWorthwhile(ctx, api, accountID)
}

type normalizedTaskInfo struct {
	growthTasks []*taskpb.Task
	dailyTasks  []*taskpb.Task
	otherTasks  []*taskpb.Task
	actives     []*taskpb.Active
}

type claimableTask struct {
	ID            int64
	Desc          string
	Category      string
	ShareMultiple int64
}

func normalizeTaskInfo(info *taskpb.TaskInfo) normalizedTaskInfo {
	out := normalizedTaskInfo{}
	if info == nil {
		return out
	}
	seen := make(map[int64]struct{})

	appendTask := func(task *taskpb.Task, target *[]*taskpb.Task) {
		if task == nil {
			return
		}
		if task.Id > 0 {
			if _, ok := seen[task.Id]; ok {
				return
			}
			seen[task.Id] = struct{}{}
		}
		*target = append(*target, task)
	}

	for _, task := range info.Tasks {
		switch task.TaskType {
		case 1:
			appendTask(task, &out.growthTasks)
		case 2:
			appendTask(task, &out.dailyTasks)
		default:
			appendTask(task, &out.otherTasks)
		}
	}
	for _, task := range info.GrowthTasks {
		appendTask(task, &out.growthTasks)
	}
	for _, task := range info.DailyTasks {
		appendTask(task, &out.dailyTasks)
	}
	out.actives = append(out.actives, info.Actives...)
	return out
}

func analyzeTaskList(tasks []*taskpb.Task, category string) []claimableTask {
	out := make([]claimableTask, 0)
	for _, task := range tasks {
		if task == nil || task.Id <= 0 {
			continue
		}
		if !task.IsUnlocked || task.IsClaimed {
			continue
		}
		if task.TotalProgress <= 0 || task.Progress < task.TotalProgress {
			continue
		}
		desc := task.Desc
		if desc == "" {
			desc = "任务"
		}
		out = append(out, claimableTask{
			ID:            task.Id,
			Desc:          desc,
			Category:      category,
			ShareMultiple: task.ShareMultiple,
		})
	}
	return out
}

func claimActives(ctx context.Context, api *game.API, actives []*taskpb.Active) {
	for _, active := range actives {
		if active == nil {
			continue
		}
		pointIDs := make([]int64, 0)
		for _, reward := range active.Rewards {
			if reward != nil && reward.Status == 2 && reward.PointId > 0 {
				pointIDs = append(pointIDs, reward.PointId)
			}
		}
		if len(pointIDs) == 0 {
			continue
		}
		if _, err := api.ClaimDailyReward(ctx, active.Type, pointIDs); err != nil {
			slog.Warn("task claim: active reward failed", "type", active.Type, "err", err)
			continue
		}
		time.Sleep(300 * time.Millisecond)
	}
}

func claimIllustratedIfWorthwhile(ctx context.Context, api *game.API, accountID uint64) {
	before := ticketBalance(ctx, api)
	reply, err := api.ClaimAllIllustratedRewards(ctx, true)
	if err != nil {
		return
	}
	after := ticketBalance(ctx, api)
	gain := after - before
	if gain < illustratedMinTicketGain {
		return
	}
	if reply != nil && (len(reply.Items) > 0 || len(reply.BonusItems) > 0) {
		stats.RecordOp(accountID, 0, "taskClaim", 1)
		slog.Info("task claim: illustrated rewards", "ticketGain", gain)
	}
}

func ticketBalance(ctx context.Context, api *game.API) int64 {
	bag, err := api.Bag(ctx)
	if err != nil {
		return 0
	}
	for _, item := range game.GetBagItems(bag) {
		if item.Id == ticketItemID {
			return item.Count
		}
	}
	return 0
}

func runEmailClaim(ctx context.Context, api *game.API, state *DailyState, force bool, now time.Time) {
	state.mu.Lock()
	today := localDateKey(now)
	if !force && state.emailDoneDateKey == today {
		state.mu.Unlock()
		return
	}
	if !force && now.Sub(state.emailLastCheck) < emailCheckCooldown {
		state.mu.Unlock()
		return
	}
	state.emailLastCheck = now
	state.mu.Unlock()

	box1, err1 := api.GetEmailList(ctx, 1)
	box2, err2 := api.GetEmailList(ctx, 2)
	if err1 != nil && err2 != nil {
		slog.Warn("daily: email list failed", "err1", err1, "err2", err2)
		return
	}

	claimable := mergeClaimableEmails(box1, box2)
	if len(claimable) == 0 {
		state.mu.Lock()
		state.emailDoneDateKey = today
		state.mu.Unlock()
		return
	}

	byBox := map[int32][]emailEntry{}
	for _, e := range claimable {
		byBox[e.boxType] = append(byBox[e.boxType], e)
	}
	for boxType, list := range byBox {
		if len(list) == 0 {
			continue
		}
		_, _ = api.BatchClaimEmail(ctx, boxType, list[0].id)
	}
	for _, e := range claimable {
		_, _ = api.ClaimEmail(ctx, e.boxType, e.id)
	}

	state.mu.Lock()
	state.emailDoneDateKey = today
	state.mu.Unlock()
}

// runGreenPlumClaim auto-claims the daily 青梅 seed once per day when the
// activity is known and active. The daily seed activity id is located by live
// query (it carries data.qingmei_daily_seed), with a discovery pass when
// nothing is registered yet. Unknown activities are skipped gracefully.
func runGreenPlumClaim(ctx context.Context, api *game.API, state *DailyState, force bool, now time.Time) {
	state.mu.Lock()
	today := localDateKey(now)
	if !force && state.greenPlumDoneDateKey == today {
		state.mu.Unlock()
		return
	}
	if !force && now.Sub(state.greenPlumLastCheck) < dailyCheckCooldown {
		state.mu.Unlock()
		return
	}
	state.greenPlumLastCheck = now
	state.mu.Unlock()

	if api == nil {
		return
	}
	activityID := api.FindGreenPlumDailyActivityID(ctx)
	if activityID <= 0 {
		slog.Debug("daily: green plum seed activity not recognized yet, skip claim")
		return
	}

	reply, err := api.ClaimGreenPlumSeed(ctx, activityID)
	if err != nil {
		if game.IsAlreadyClaimedGreenPlum(err) {
			api.RememberGreenPlumSeedClaimed()
			slog.Info("daily: green plum seed already claimed today", "activityId", activityID)
		} else {
			slog.Warn("daily: green plum seed claim failed", "activityId", activityID, "err", err)
			return
		}
	} else {
		slog.Info("daily: green plum seed claimed", "activityId", activityID, "operateType", reply.GetOperateType())
	}
	state.mu.Lock()
	state.greenPlumDoneDateKey = today
	state.mu.Unlock()
}

// runGreenPlumBrew auto-brews 青梅 through every round and settles via the
// boosted shared (1.5x) mode, once per day. Flow: start with all available
// 青梅 when nothing is brewing, continue rounds until the last one, then
// report the share and settle. Each stage is guarded by the live brew state so
// a partially completed run resumes naturally on the next pass.
func runGreenPlumBrew(ctx context.Context, api *game.API, state *DailyState, force bool, now time.Time) {
	state.mu.Lock()
	today := localDateKey(now)
	if !force && state.greenPlumBrewDateKey == today {
		state.mu.Unlock()
		return
	}
	if !force && now.Sub(state.greenPlumBrewCheck) < dailyCheckCooldown {
		state.mu.Unlock()
		return
	}
	state.greenPlumBrewCheck = now
	state.mu.Unlock()

	if api == nil {
		return
	}
	activityID := api.FindGreenPlumBrewActivityID(ctx)
	if activityID <= 0 {
		slog.Debug("daily: green plum brew activity not recognized yet, skip brew")
		return
	}

	reply, err := api.QueryGreenPlum(ctx, activityID)
	if err != nil || reply == nil || reply.Data == nil {
		slog.Warn("daily: green plum brew query failed", "err", err)
		return
	}
	brew := reply.Data.QingmeiBrew
	if brew == nil {
		slog.Debug("daily: green plum brew entry not ready, skip")
		return
	}

	if brew.Finished {
		slog.Info("daily: green plum brew already settled", "round", brew.CurrentRound)
		state.mu.Lock()
		state.greenPlumBrewDateKey = today
		state.mu.Unlock()
		return
	}

	if brew.BaseGold <= 0 {
		// Not started yet: invest the whole 青梅 balance and start brewing.
		bag, bagErr := api.Bag(ctx)
		if bagErr != nil {
			slog.Warn("daily: green plum brew bag failed", "err", bagErr)
			return
		}
		count := int64(0)
		for _, item := range game.GetBagItems(bag) {
			if item.Id == game.GreenPlumItemID {
				count += item.Count
			}
		}
		if count <= 0 {
			slog.Debug("daily: no green plum to brew, skip")
			return
		}
		if _, err := api.StartGreenPlumBrewAll(ctx, activityID); err != nil {
			slog.Warn("daily: green plum brew start failed", "err", err)
			return
		}
		slog.Info("daily: green plum brew started", "activityId", activityID, "count", count)
		return
	}

	maxRounds := brew.MaxRounds
	if maxRounds <= 0 {
		maxRounds = 3
	}
	if brew.CurrentRound < maxRounds {
		if _, err := api.ContinueGreenPlumBrew(ctx, activityID); err != nil {
			slog.Warn("daily: green plum brew continue failed", "round", brew.CurrentRound, "err", err)
			return
		}
		slog.Info("daily: green plum brew continued", "activityId", activityID, "round", brew.CurrentRound)
		return
	}

	if _, err := api.SettleGreenPlumBrew(ctx, activityID); err != nil {
		slog.Warn("daily: green plum brew settle failed", "round", brew.CurrentRound, "err", err)
		return
	}
	slog.Info("daily: green plum brew settled", "activityId", activityID, "round", brew.CurrentRound)
	state.mu.Lock()
	state.greenPlumBrewDateKey = today
	state.mu.Unlock()
}

type emailEntry struct {
	id      string
	boxType int32
}

func mergeClaimableEmails(box1, box2 *emailpb.GetEmailListReply) []emailEntry {
	type keyed struct {
		entry     emailEntry
		claimable bool
	}
	merged := map[string]keyed{}
	add := func(items []*emailpb.EmailItem, boxType int32) {
		for _, item := range items {
			if item == nil || item.Id == "" {
				continue
			}
			entry := emailEntry{id: item.Id, boxType: boxType}
			claimable := item.HasReward && !item.Claimed
			if !claimable {
				continue
			}
			if old, ok := merged[item.Id]; !ok {
				merged[item.Id] = keyed{entry: entry, claimable: true}
			} else if !old.claimable {
				merged[item.Id] = keyed{entry: entry, claimable: true}
			}
		}
	}
	if box1 != nil {
		add(box1.Emails, 1)
	}
	if box2 != nil {
		add(box2.Emails, 2)
	}
	out := make([]emailEntry, 0, len(merged))
	for _, v := range merged {
		if v.claimable {
			out = append(out, v.entry)
		}
	}
	return out
}

func runShareClaim(ctx context.Context, api *game.API, state *DailyState, force bool, now time.Time) {
	state.mu.Lock()
	today := localDateKey(now)
	if !force && state.shareDoneDateKey == today {
		state.mu.Unlock()
		return
	}
	if !force && now.Sub(state.shareLastCheck) < dailyCheckCooldown {
		state.mu.Unlock()
		return
	}
	state.shareLastCheck = now
	state.mu.Unlock()

	can, err := api.CheckCanShare(ctx)
	if err != nil {
		slog.Warn("daily: share check failed", "err", err)
		return
	}
	if can == nil || !can.CanShare {
		state.mu.Lock()
		state.shareDoneDateKey = today
		state.mu.Unlock()
		return
	}
	report, err := api.ReportShare(ctx)
	if err != nil || report == nil {
		return
	}
	reply, err := api.ClaimShareReward(ctx)
	if err != nil {
		if isAlreadyClaimedError(err) {
			state.mu.Lock()
			state.shareDoneDateKey = today
			state.mu.Unlock()
		}
		return
	}
	if reply == nil {
		return
	}
	state.mu.Lock()
	state.shareDoneDateKey = today
	state.mu.Unlock()
	slog.Info("daily: share reward claimed", "items", len(reply.Items))
}

func runMonthCardClaim(ctx context.Context, api *game.API, state *DailyState, force bool, now time.Time) {
	state.mu.Lock()
	today := localDateKey(now)
	if !force && state.monthCardDoneDateKey == today {
		state.mu.Unlock()
		return
	}
	if !force && now.Sub(state.monthCardLastCheck) < dailyCheckCooldown {
		state.mu.Unlock()
		return
	}
	state.monthCardLastCheck = now
	state.mu.Unlock()

	rep, err := api.GetMonthCardInfos(ctx)
	if err != nil {
		slog.Warn("daily: month card info failed", "err", err)
		return
	}
	if rep == nil || len(rep.Infos) == 0 {
		state.mu.Lock()
		state.monthCardDoneDateKey = today
		state.mu.Unlock()
		return
	}
	claimed := 0
	for _, info := range rep.Infos {
		if info == nil || !info.CanClaim || info.GoodsId <= 0 {
			continue
		}
		if _, err := api.ClaimMonthCardReward(ctx, info.GoodsId); err != nil {
			continue
		}
		claimed++
	}
	state.mu.Lock()
	state.monthCardDoneDateKey = today
	state.mu.Unlock()
	if claimed > 0 {
		slog.Info("daily: month card claimed", "count", claimed)
	}
}

func runFreeGifts(ctx context.Context, api *game.API, state *DailyState, force bool, now time.Time) {
	state.mu.Lock()
	today := localDateKey(now)
	if !force && state.freeGiftDoneDateKey == today {
		state.mu.Unlock()
		return
	}
	if !force && now.Sub(state.freeGiftLastCheck) < dailyCheckCooldown {
		state.mu.Unlock()
		return
	}
	state.freeGiftLastCheck = now
	state.mu.Unlock()

	bought, err := api.BuyFreeGifts(ctx)
	if err != nil {
		slog.Warn("daily: free gifts failed", "err", err)
		return
	}
	state.mu.Lock()
	state.freeGiftDoneDateKey = today
	state.mu.Unlock()
	if bought > 0 {
		slog.Info("daily: free gifts purchased", "count", bought)
	}
}

func runVipGift(ctx context.Context, api *game.API, state *DailyState, force bool, now time.Time) {
	state.mu.Lock()
	today := localDateKey(now)
	if !force && state.vipDoneDateKey == today {
		state.mu.Unlock()
		return
	}
	if !force && now.Sub(state.vipLastCheck) < dailyCheckCooldown {
		state.mu.Unlock()
		return
	}
	state.vipLastCheck = now
	state.mu.Unlock()

	status, err := api.GetDailyGiftStatus(ctx)
	if err != nil {
		slog.Warn("daily: vip status failed", "err", err)
		return
	}
	if status == nil || !status.CanClaim {
		state.mu.Lock()
		state.vipDoneDateKey = today
		state.mu.Unlock()
		return
	}
	if _, err := api.ClaimDailyGift(ctx); err != nil {
		if isAlreadyClaimedError(err) {
			state.mu.Lock()
			state.vipDoneDateKey = today
			state.mu.Unlock()
		}
		return
	}
	state.mu.Lock()
	state.vipDoneDateKey = today
	state.mu.Unlock()
	slog.Info("daily: vip gift claimed")
}

func isAlreadyClaimedError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "code=1021002") ||
		strings.Contains(msg, "今日已领取") ||
		strings.Contains(msg, "已领取")
}
