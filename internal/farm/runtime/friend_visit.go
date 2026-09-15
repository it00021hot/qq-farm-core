package runtime

// 好友单次访问巡逻（对齐 bot visit-plan.ts + scheduler.checkFriends + visitFriend）。
//
// 以前巡查分三段跑（偷菜/帮忙/捣乱各自 Enter/Leave），同一好友可能被进两次农场。
// 现在先按好友列表气泡算出「每位好友这一轮要做哪几件事」，再对每位好友只进一次农场，
// 在里面把 帮忙（除草/除虫/浇水）+ 偷菜 + 捣乱（放草/放虫）一次做完。

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/farm/game"
	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/friendpb"
	"github.com/it00021hot/qq-farm-core/internal/farm/stats"
)

// maxBadOnlyVisitsPerRound mirrors bot MAX_BAD_ONLY_VISITS_PER_ROUND.
const maxBadOnlyVisitsPerRound = 20

// friendDogState is the daily pet-cache conclusion (bot pet-cache three-state).
type friendDogState string

const (
	dogStateUnknown friendDogState = "unknown"
	dogStateProtect friendDogState = "protect"
	dogStateNone    friendDogState = "none"
)

// friendVisitTarget is one friend's planned actions for this round.
type friendVisitTarget struct {
	GID       int64
	Name      string
	Level     int64
	StealNum  int64
	HelpNum   int64
	DryNum    int64
	WeedNum   int64
	InsectNum int64
	WantSteal bool
	WantHelp  bool
	WantBad   bool
}

// friendVisitPlan aggregates the round's targets (bot FriendVisitPlan).
type friendVisitPlan struct {
	Visits           []friendVisitTarget
	StealCount       int
	HelpCount        int
	BadOnlyCount     int
	SkippedExpLimit  int
	SkippedUnknownDog int
}

// friendPlanOptions carries the round's toggles into the planner.
type friendPlanOptions struct {
	StealEnabled           bool
	HelpEnabled            bool
	BadEnabled             bool
	HelpAllowedForAll      bool
	ProtectDogBypassEnabled bool
	GetDogState            func(gid int64) friendDogState
	BadBudget              int64
	MaxBadOnlyVisits       int
}

// buildFriendVisitPlan is a pure port of bot visit-plan.ts buildFriendVisitPlan.
func buildFriendVisitPlan(friends []friendpb.GameFriend, myGID int64, blacklist map[int64]struct{}, opts friendPlanOptions) friendVisitPlan {
	getDogState := opts.GetDogState
	if getDogState == nil {
		getDogState = func(int64) friendDogState { return dogStateUnknown }
	}
	maxBadOnly := opts.MaxBadOnlyVisits
	if maxBadOnly < 0 {
		maxBadOnly = 0
	}
	badAllowed := opts.BadEnabled && opts.BadBudget > 0 && maxBadOnly > 0

	primary := make([]friendVisitTarget, 0, len(friends))
	badOnly := make([]friendVisitTarget, 0)
	seen := make(map[int64]struct{}, len(friends))
	plan := friendVisitPlan{}

	for i := range friends {
		friend := &friends[i]
		gid := friend.Gid
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

		name := strings.TrimSpace(friend.Remark)
		if name == "" {
			name = strings.TrimSpace(friend.Name)
		}
		if name == "" {
			name = fmt.Sprintf("GID:%d", gid)
		}
		var stealNum, dryNum, weedNum, insectNum int64
		if friend.Plant != nil {
			stealNum = friend.Plant.StealPlantNum
			dryNum = friend.Plant.DryNum
			weedNum = friend.Plant.WeedNum
			insectNum = friend.Plant.InsectNum
		}
		helpNum := dryNum + weedNum + insectNum

		wantSteal := opts.StealEnabled && stealNum > 0
		wantHelp := opts.HelpEnabled && helpNum > 0
		if wantHelp && !opts.HelpAllowedForAll {
			// 经验已满：只有「护主犬无视经验上限」开着、且当天缓存已确认是护主犬时才值得进农场。
			dog := getDogState(gid)
			bypass := opts.ProtectDogBypassEnabled && dog == dogStateProtect
			if !bypass {
				wantHelp = false
				plan.SkippedExpLimit++
				// 宠物没同步的好友这一轮不试探，等每日宠物同步给出结论
				if opts.ProtectDogBypassEnabled && dog == dogStateUnknown {
					plan.SkippedUnknownDog++
				}
			}
		}

		target := friendVisitTarget{
			GID: gid, Name: name, Level: friend.Level,
			StealNum: stealNum, HelpNum: helpNum,
			DryNum: dryNum, WeedNum: weedNum, InsectNum: insectNum,
			WantSteal: wantSteal, WantHelp: wantHelp,
		}

		if wantSteal || wantHelp {
			primary = append(primary, target)
			continue
		}
		// 既没可偷也没可帮的好友才是捣乱对象：不在偷/帮的访问里顺手放草放虫，
		// 免得每日捣乱额度被花在错误的好友身上。
		if badAllowed && stealNum == 0 && helpNum == 0 {
			badOnly = append(badOnly, target)
		}
	}

	// 偷得多的先走，其次是帮助需求大的，最后按等级
	sort.SliceStable(primary, func(i, j int) bool {
		a, b := &primary[i], &primary[j]
		if a.StealNum != b.StealNum {
			return a.StealNum > b.StealNum
		}
		if a.HelpNum != b.HelpNum {
			return a.HelpNum > b.HelpNum
		}
		return a.Level > b.Level
	})
	// 捣乱优先挑等级高的好友
	sort.SliceStable(badOnly, func(i, j int) bool { return badOnly[i].Level > badOnly[j].Level })

	if len(badOnly) > maxBadOnly {
		badOnly = badOnly[:maxBadOnly]
	}
	for i := range badOnly {
		badOnly[i].WantBad = true
	}

	for _, t := range primary {
		if t.WantSteal {
			plan.StealCount++
		}
		if t.WantHelp {
			plan.HelpCount++
		}
	}
	plan.BadOnlyCount = len(badOnly)
	plan.Visits = append(primary, badOnly...)
	return plan
}

// friendVisitTotals accumulates one round's action counts (bot totalActions).
type friendVisitTotals struct {
	Steal   int
	Farming int
	PutBug  int
	PutWeed int
}

// RunFriendCheckTick is the unified friend patrol round (bot checkFriends).
func RunFriendCheckTick(ctx context.Context, s *Session, opts RunFriendTickOptions) (bool, error) {
	if s == nil {
		return false, fmt.Errorf("farm session is unavailable")
	}
	api := s.GameAPI()
	cfg := s.Config()
	myGID := s.GID()
	accountID := parseAccountID(s.id)
	if api == nil {
		return false, fmt.Errorf("farm API is unavailable")
	}

	helpEnabled := cfg.Automation.Friend && cfg.Automation.FriendHelp
	stealEnabled := cfg.Automation.Friend && cfg.Automation.FriendSteal
	badEnabled := cfg.Automation.Friend && cfg.Automation.FriendBad
	if !opts.IgnoreToggles {
		if opts.OnlyHelp {
			helpEnabled, stealEnabled, badEnabled = true, false, false
		}
		if opts.OnlySteal {
			helpEnabled, stealEnabled, badEnabled = false, true, false
		}
		if opts.OnlyBad {
			helpEnabled, stealEnabled, badEnabled = false, false, true
		}
	}
	if !helpEnabled && !stealEnabled && !badEnabled {
		return false, nil
	}

	friends, err := getFriendsList(ctx, s, api, cfg, true)
	if err != nil {
		return false, err
	}
	SyncFriendsToDB(accountID, myGID, friends)
	friends = s.applyFriendStealOverrides(friends)
	friends = s.applyFriendPushHints(friends)

	helpState := s.ensureHelpState()
	stopWhenExpLimit := cfg.Automation.FriendHelpExpLimit && !opts.IgnoreExpLimit
	if !stopWhenExpLimit {
		helpState.setCanGetHelpExp(true)
	}
	protectDogBypassEnabled := cfg.Automation.FriendHelpProtectDogIgnoreExpLimit
	helpAllowedForAll := !stopWhenExpLimit || helpState.getCanGetHelpExp()

	blacklist := makeIDSet(cfg.FriendBlacklist)
	plan := buildFriendVisitPlan(friends, myGID, blacklist, friendPlanOptions{
		StealEnabled:            stealEnabled,
		HelpEnabled:             helpEnabled,
		BadEnabled:              badEnabled && !helpState.isBadOperationLimitReached(),
		HelpAllowedForAll:       helpAllowedForAll,
		ProtectDogBypassEnabled: protectDogBypassEnabled,
		GetDogState:             s.getFriendDogState,
		BadBudget:               int64(helpState.getRemainingBadOperationTimes()),
		MaxBadOnlyVisits:        maxBadOnlyVisitsPerRound,
	})

	if plan.SkippedExpLimit > 0 {
		slog.Info("经验已达上限，本轮跳过非护主犬好友（未进农场）",
			"account", accountID,
			"skipped", plan.SkippedExpLimit,
			"unknown_dog", plan.SkippedUnknownDog,
		)
	}
	if len(plan.Visits) == 0 {
		return false, nil
	}
	slog.Info("开始好友巡查",
		"account", accountID,
		"visits", len(plan.Visits),
		"steal", plan.StealCount,
		"help", plan.HelpCount,
		"bad", plan.BadOnlyCount,
	)

	var totals friendVisitTotals
	midRoundExpSkipped := 0

	for i := range plan.Visits {
		target := &plan.Visits[i]
		if shouldAbortFriendPatrol(ctx, s) {
			break
		}
		if target.WantBad {
			// 纯捣乱的好友都排在队尾，额度一用完这一轮就可以收工
			if helpState.isBadOperationLimitReached() || helpState.getRemainingBadOperationTimes() <= 0 {
				break
			}
		} else if target.WantHelp && !target.WantSteal && stopWhenExpLimit && !helpState.getCanGetHelpExp() {
			// 帮忙是这次进农场的唯一目的，但经验在本轮中途满了：不是护主犬就别进去了
			if !protectDogBypassEnabled || s.getFriendDogState(target.GID) != dogStateProtect {
				midRoundExpSkipped++
				continue
			}
		}

		outcome, visitErr := visitFriendTarget(ctx, s, api, cfg, myGID, target, visitTargetOptions{
			AllowSteal:    target.WantSteal,
			AllowHelp:     target.WantHelp,
			AllowBad:      target.WantBad,
			StopWhenExp:   stopWhenExpLimit,
			DogBypass:     protectDogBypassEnabled,
		})
		s.clearFriendPlantHint(target.GID)
		if visitErr != nil {
			if handleFriendEnterError(s, target.GID, visitErr) {
				continue
			}
			if isTransientNetworkError(visitErr) {
				abortFriendPatrol(s, accountID, "friend", visitErr)
				break
			}
			writeInteractLog(accountID, 0, target.GID, "visit", "error", map[string]any{
				"error": friendlyNetworkError(visitErr),
			})
			continue
		}
		recordVisitOutcome(accountID, target, outcome, &totals)

		// 捣乱访问之间放慢一些，其余保持原节奏
		delay := 800 * time.Millisecond
		if target.WantBad {
			delay = time.Duration(2000+rand.Intn(1500)) * time.Millisecond
		} else if !target.WantBad {
			delay = time.Duration(500+rand.Intn(300)) * time.Millisecond
		}
		if i < len(plan.Visits)-1 {
			_ = waitFarmDelay(ctx, delay)
		}
	}

	if midRoundExpSkipped > 0 {
		slog.Info("本轮帮助经验在中途达到上限，跳过剩余非护主犬好友",
			"account", accountID, "skipped", midRoundExpSkipped)
	}

	// 偷菜后自动出售
	if totals.Steal > 0 && cfg.Automation.Sell {
		sold, gold, names, sellErr := sellAllFruitsDetailed(ctx, api)
		if sellErr != nil {
			slog.Warn("steal sell failed", "account", accountID, "err", sellErr)
		} else if sold > 0 {
			stats.RecordOp(accountID, 0, "sell", 1)
			if gold > 0 {
				stats.RecordExpGold(accountID, 0, 0, gold)
			}
			logSellFruits(s, accountID, names, gold, sold)
		}
	}

	summary := formatVisitSummary(totals)
	if summary != "" {
		slog.Info("巡查完成", "account", accountID, "visited", len(plan.Visits), "summary", summary)
		// 对齐 rust friend scheduler：好友巡查汇总面板日志（非空才记）。
		if s.hub != nil {
			s.hub.PublishJSON("friend_interact", accountID, map[string]any{
				"tag":     "好友",
				"event":   "friend_cycle",
				"module":  "friend",
				"message": "巡查完成 → " + summary,
			})
		}
		return true, nil
	}
	return false, nil
}

// RunFriendTickOptions mirror bot CheckFriendsOptions (manual single-op triggers).
type RunFriendTickOptions struct {
	OnlyHelp       bool
	OnlySteal      bool
	OnlyBad        bool
	IgnoreExpLimit bool
	// IgnoreToggles keeps automation flags as-is (used by the loop).
	IgnoreToggles bool
}

func formatVisitSummary(totals friendVisitTotals) string {
	parts := make([]string, 0, 4)
	if totals.Steal > 0 {
		parts = append(parts, fmt.Sprintf("偷%d", totals.Steal))
	}
	if totals.Farming > 0 {
		parts = append(parts, fmt.Sprintf("一键务农%d", totals.Farming))
	}
	if totals.PutBug > 0 {
		parts = append(parts, fmt.Sprintf("放虫%d", totals.PutBug))
	}
	if totals.PutWeed > 0 {
		parts = append(parts, fmt.Sprintf("放草%d", totals.PutWeed))
	}
	return strings.Join(parts, "/")
}

// recordVisitOutcome writes interact logs and accumulates round totals.
func recordVisitOutcome(accountID uint64, target *friendVisitTarget, outcome friendVisitOutcome, totals *friendVisitTotals) {
	if outcome.Count > 0 && outcome.Mode == visitModeSteal {
		totals.Steal += outcome.Count
		stats.RecordOp(accountID, 0, "steal", outcome.Count)
		writeInteractLog(accountID, 0, target.GID, "steal", "ok", map[string]any{
			"count":   outcome.Count,
			"plants":  outcome.Plants,
			"summary": outcome.Summary,
			"score":   outcome.Score,
			"value":   outcome.Value,
		})
		if outcome.Score > 0 {
			writeInteractLog(accountID, 0, target.GID, "steal_score", "ok", map[string]any{
				"count":   int(outcome.Score),
				"summary": fmt.Sprintf("获得积分x%d", outcome.Score),
			})
		}
	}
	if outcome.HelpCount > 0 {
		totals.Farming += outcome.HelpCount
		stats.RecordOp(accountID, 0, "help", outcome.HelpCount)
		writeInteractLog(accountID, 0, target.GID, "help", "ok", map[string]any{
			"count":   outcome.HelpCount,
			"summary": outcome.HelpSummary,
			"weed":    outcome.Weed,
			"bug":     outcome.Bug,
			"water":   outcome.Water,
		})
	}
	if outcome.Mode == visitModeBad && outcome.Count > 0 {
		totals.PutWeed += outcome.PutWeed
		totals.PutBug += outcome.PutBug
		writeInteractLog(accountID, 0, target.GID, "bad", "ok", map[string]any{
			"count":   outcome.Count,
			"summary": outcome.Summary,
			"putBug":  outcome.PutBug,
			"putWeed": outcome.PutWeed,
		})
	}
}

// visitTargetOptions gate the three op groups inside one visit.
type visitTargetOptions struct {
	AllowSteal  bool
	AllowHelp   bool
	AllowBad    bool
	StopWhenExp bool
	DogBypass   bool
}

// canBypassHelpExpLimitForDog mirrors bot canBypassHelpExpLimitForProtectDog:
// the entered friend actually has the protect dog deployed.
func canBypassHelpExpLimitForDog(dogID int64) bool {
	return dogID == ProtectDogID
}

// visitFriendTarget enters a friend's farm once and runs help + steal + bad
// inside (bot visitFriend). Order: help → steal → bad.
func visitFriendTarget(ctx context.Context, s *Session, api *game.API, cfg logic.AccountConfig, myGID int64, target *friendVisitTarget, opts visitTargetOptions) (friendVisitOutcome, error) {
	var out friendVisitOutcome
	helpState := s.ensureHelpState()

	stealWanted := opts.AllowSteal
	badWanted := opts.AllowBad && !helpState.isBadOperationLimitReached()

	// 经验满之后唯一还值得帮忙的对象是挂着护主犬的好友（同气连枝礼包）。
	expLimitReached := opts.StopWhenExp && !helpState.getCanGetHelpExp()
	helpWanted := opts.AllowHelp
	if helpWanted && expLimitReached {
		if !opts.DogBypass || s.getFriendDogState(target.GID) != dogStateProtect {
			helpWanted = false
		}
	}
	if !stealWanted && !badWanted && !helpWanted {
		// 这一轮对这位好友无事可做：一个请求都不发
		return out, nil
	}

	detail, err := api.VisitEnterDetailed(ctx, target.GID, enterReasonFriend)
	if err != nil {
		if handleFriendEnterError(s, target.GID, err) {
			return out, nil
		}
		return out, err
	}
	// Enter 回包是护主犬信息的唯一来源：所有进好友农场的调用都顺手写缓存（零额外 RPC）。
	s.recordFriendDogFromEnter(target.GID, detail)
	defer func() { _ = api.VisitLeave(ctx, target.GID) }()

	lands := detail.Lands
	if len(lands) == 0 {
		return out, nil
	}

	// 1. 帮忙（除草/除虫/浇水）。经验满时只帮护主犬（开关打开且缓存确认）。
	if helpWanted {
		if !opts.DogBypass || !canBypassHelpExpLimitForDog(detail.DogID) {
			if expLimitReached {
				helpWanted = false
			}
		}
	}
	if helpWanted {
		helpOut := helpOnEnteredLands(ctx, s, api, cfg, target.GID, lands, opts.StopWhenExp)
		out.HelpCount = helpOut.Count
		out.HelpSummary = helpOut.Summary
		out.Weed, out.Bug, out.Water = helpOut.Weed, helpOut.Bug, helpOut.Water
		if helpOut.Count > 0 {
			_ = waitFarmDelay(ctx, time.Duration(500+rand.Intn(300))*time.Millisecond)
		}
	}

	// 2. 偷菜
	if stealWanted {
		out.Mode = visitModeSteal
		stealOut, stealErr := stealOnEnteredLands(ctx, s, api, cfg, target.GID, lands)
		if stealErr != nil {
			return out, stealErr
		}
		out.Count = stealOut.Count
		out.Plants = stealOut.Plants
		out.Score = stealOut.Score
		out.Value = stealOut.Value
		out.Summary = stealOut.Summary
		if out.Count > 0 {
			_ = waitFarmDelay(ctx, time.Duration(500+rand.Intn(300))*time.Millisecond)
		}
	}

	// 3. 捣乱（放草/放虫）
	if badWanted && !helpState.isBadOperationLimitReached() {
		out.Mode = visitModeBad
		badOut := badOnEnteredLands(ctx, s, api, myGID, target.GID, lands)
		out.PutWeed += badOut.PutWeed
		out.PutBug += badOut.PutBug
		out.Count += badOut.PutWeed + badOut.PutBug
		if out.Count > 0 {
			out.Summary = formatBadSummary(out.PutBug, out.PutWeed)
		}
	}
	return out, nil
}

// helpOutcome carries the help-phase result of one visit.
type helpOutcome struct {
	Count   int
	Weed    int
	Bug     int
	Water   int
	Summary string
}

// helpOnEnteredLands mirrors bot visitFriend's help phase (already inside the farm).
func helpOnEnteredLands(ctx context.Context, s *Session, api *game.API, cfg logic.AccountConfig, gid int64, lands []logic.LandInfo, stopWhenExpLimit bool) helpOutcome {
	var out helpOutcome
	helpState := s.ensureHelpState()
	needWater, needWeed, needBug := logic.AnalyzeFriendHelpLands(lands)
	out.Weed, out.Bug, out.Water = len(needWeed), len(needBug), len(needWater)
	allHelp := uniqueLandIDs(needWeed, needBug, needWater)
	if len(allHelp) == 0 {
		return out
	}
	allExpIDs := []int64{friendOpWeed, friendOpBug, friendOpWater}
	if stopWhenExpLimit && !(helpState.canGetExpByCandidates(allExpIDs) && helpState.getCanGetHelpExp()) {
		return out
	}
	beforeExp := s.playerExpSnapshot()
	reply, farmErr := api.FriendFarming(ctx, gid, allHelp)
	if farmErr != nil {
		if !isFarmingNoopError(farmErr) {
			slog.Warn("friend help failed", "account", sessionID(s), "friend_gid", gid, "err", farmErr)
		}
		return out
	}
	count := len(allHelp)
	if reply != nil {
		helpState.updateLimits(reply.OperationLimits)
		if landIDs := farmingResultLandIDs(reply.Results); len(landIDs) > 0 {
			count = len(landIDs)
		} else if len(reply.Results) > 0 {
			count = len(reply.Results)
		}
		if stopWhenExpLimit && count > 0 {
			_ = waitFarmDelay(ctx, 200*time.Millisecond)
			if afterExp := s.playerExpSnapshot(); afterExp <= beforeExp {
				helpState.autoDisableHelpByExpLimit()
				out.Count = count
				out.Summary = formatHelpSummary(count, out.Weed, out.Bug, out.Water)
				return out
			}
		}
	}
	out.Count = count
	if count > 0 {
		out.Summary = formatHelpSummary(count, out.Weed, out.Bug, out.Water)
	}
	return out
}

// stealOnEnteredLands mirrors bot visitFriend's steal phase (batch first, per-land fallback).
func stealOnEnteredLands(ctx context.Context, s *Session, api *game.API, cfg logic.AccountConfig, gid int64, lands []logic.LandInfo) (friendVisitOutcome, error) {
	var out friendVisitOutcome
	blacklist := makeIDSet(cfg.PlantBlacklist)
	landsMap := logic.BuildLandMap(lands)
	targets := make([]int64, 0, len(lands))
	plantByLand := make(map[int64]string, len(lands))
	for i := range lands {
		land := &lands[i]
		if logic.IsOccupiedSlaveLand(land, landsMap) {
			continue
		}
		if land.Plant == nil || !land.Plant.Stealable || !isMature(*land) {
			continue
		}
		if isPlantBlacklistedBySeed(blacklist, land.Plant.ID) {
			continue
		}
		targets = append(targets, land.ID)
		displayID, name := logic.ResolvePlantDisplayName(land.Plant)
		if name == "" || name == "未知" {
			name = strings.TrimSpace(land.Plant.Name)
		}
		if name != "" {
			plantByLand[land.ID] = name
		}
		_ = displayID
	}
	if len(targets) == 0 {
		return out, nil
	}
	// 微信 10008 无限：不调 CheckCanOperate。QQ 失败则 fail-open。
	canOperate, canSteal := true, int64(0)
	if !wxStealUnlimited(sessionPlatform(s)) {
		var checkErr error
		canOperate, canSteal, checkErr = api.CheckCanOperate(ctx, gid, friendOpSteal)
		if checkErr != nil {
			canOperate, canSteal = true, 0
		}
	}
	if !canOperate {
		return out, nil
	}
	if canSteal > 0 && int64(len(targets)) > canSteal {
		targets = targets[:canSteal]
	}
	stolenIDs, items, harvestErr := friendHarvestWithFallback(ctx, s, api, gid, targets)
	if harvestErr != nil && len(stolenIDs) == 0 {
		return out, harvestErr
	}
	out.Count = len(stolenIDs)
	if out.Count > 0 {
		s.markFriendStealCleared(gid)
		out.Plants = mergeUniqueNames(
			uniquePlantNames(plantByLand, stolenIDs),
			plantNamesFromHarvestItems(items),
		)
		out.Score, out.Value = summarizeHarvestRewards(items)
		out.Summary = formatStealSummary(out.Count, out.Plants, out.Score, out.Value)
	}
	return out, nil
}

// badOnEnteredLands mirrors bot visitFriend's bad phase: weeds first then insects,
// one land per request, shared 10003 quota.
func badOnEnteredLands(ctx context.Context, s *Session, api *game.API, myGID, gid int64, lands []logic.LandInfo) friendVisitOutcome {
	var out friendVisitOutcome
	helpState := s.ensureHelpState()
	weedTargets, bugTargets := collectBadLandTargets(lands, myGID)

	if len(weedTargets) > 0 && !helpState.isBadOperationLimitReached() {
		remaining := helpState.getRemainingBadOperationTimes()
		if remaining > 0 {
			if remaining < len(weedTargets) {
				weedTargets = weedTargets[:remaining]
			}
			ok, putErr := api.PutWeeds(ctx, gid, weedTargets, game.PutPlantHooks{
				OnLimits: helpState.updateLimits,
				Continue: func() bool {
					return !helpState.isBadOperationLimitReached() && helpState.getRemainingBadOperationTimes() > 0
				},
			})
			if putErr != nil && !isBadOpLimitError(putErr) {
				slog.Debug("put weeds failed", "account", sessionID(s), "friend_gid", gid, "err", putErr)
			}
			out.PutWeed = ok
		}
	}
	if !helpState.isBadOperationLimitReached() && len(bugTargets) > 0 {
		remaining := helpState.getRemainingBadOperationTimes()
		if remaining > 0 {
			if remaining < len(bugTargets) {
				bugTargets = bugTargets[:remaining]
			}
			ok, putErr := api.PutInsects(ctx, gid, bugTargets, game.PutPlantHooks{
				OnLimits: helpState.updateLimits,
				Continue: func() bool {
					return !helpState.isBadOperationLimitReached() && helpState.getRemainingBadOperationTimes() > 0
				},
			})
			if putErr != nil && !isBadOpLimitError(putErr) {
				slog.Debug("put insects failed", "account", sessionID(s), "friend_gid", gid, "err", putErr)
			}
			out.PutBug = ok
		}
	}
	return out
}
