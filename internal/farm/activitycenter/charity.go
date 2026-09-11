package activitycenter

// 公益小红花（对齐 bot services/activity-center/charity.ts）。
//
// 四个操作：领取种子(35) / 捐赠爱心(36) / 领取进度奖励(37) / 每日公益礼包(38)。
// 进度奖励的服务端快照只报解锁状态（领取前后 status 都为 1），本地状态是
// 领取历史的权威来源；未初始化时按顺序领取序恢复历史前缀。

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/farm/game"
	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/activitypb"
)

const (
	CharityGroupID          int64 = 2026090900
	CharityActivityID       int64 = 2026090901
	operateCharitySeed      int64 = 35
	operateCharityDonate    int64 = 36
	operateCharityProgress  int64 = 37
	operateCharityDailyGift int64 = 38

	charityProgressAlreadyClaimedCode = 1034087
	charityFlowHarvested              = "2"
	charityFlowDailyGiftClaimed       = "3"
)

// charityProgressState is the local claim history for progress rewards.
type charityProgressState struct {
	claimed map[string]struct{}
	pending map[string]struct{}
}

var charityMu sync.Mutex
var charityStates = map[int64]*charityProgressState{}

func charityStateFor(activityID int64) *charityProgressState {
	if st, ok := charityStates[activityID]; ok {
		return st
	}
	st := &charityProgressState{claimed: map[string]struct{}{}, pending: map[string]struct{}{}}
	charityStates[activityID] = st
	return st
}

// reconcileCharityProgress recovers the claim prefix from the snapshot when
// the local state is uninitialized (bot reconcileCharityProgressState).
func reconcileCharityProgress(activityID int64, state *activitypb.CharityRedFlowerData) *charityProgressReconcile {
	st := charityStateFor(activityID)
	if state == nil || len(state.ProgressRewards) == 0 {
		return &charityProgressReconcile{st}
	}
	var reached []int64
	for _, reward := range state.ProgressRewards {
		if reward == nil || reward.Status != 1 || state.DonatedLove < reward.Target {
			continue
		}
		if reward.Target > 0 {
			reached = append(reached, reward.Target)
		}
	}
	sort.Slice(reached, func(i, j int) bool { return reached[i] < reached[j] })
	uninitialized := len(st.claimed) == 0 && len(st.pending) == 0
	if uninitialized && len(reached) > 0 {
		for _, target := range reached[:len(reached)-1] {
			st.claimed[strconv.FormatInt(target, 10)] = struct{}{}
		}
		st.pending[strconv.FormatInt(reached[len(reached)-1], 10)] = struct{}{}
	} else {
		for _, target := range reached {
			key := strconv.FormatInt(target, 10)
			if _, c := st.claimed[key]; !c {
				if _, p := st.pending[key]; !p {
					st.pending[key] = struct{}{}
				}
			}
		}
	}
	for target := range st.claimed {
		delete(st.pending, target)
	}
	return &charityProgressReconcile{st}
}

type charityProgressReconcile struct{ state *charityProgressState }

func findActivityEntry(activities []*activitypb.ActivityData, activityID int64) *activitypb.ActivityData {
	queue := make([]*activitypb.ActivityData, 0, len(activities))
	queue = append(queue, activities...)
	for len(queue) > 0 {
		entry := queue[0]
		queue = queue[1:]
		if entry == nil {
			continue
		}
		if entry.Activity != nil && entry.Activity.ActivityId == activityID {
			return entry
		}
		queue = append(queue, entry.Children...)
	}
	return nil
}

func charityWindowFor(windows []*activitypb.ActivityWindow, activityID int64) *activitypb.ActivityWindow {
	for i := range windows {
		if windows[i].Id == activityID {
			return windows[i]
		}
	}
	for i := range windows {
		if windows[i].Id == CharityGroupID {
			return windows[i]
		}
	}
	return nil
}

// BuildCharity builds the 公益小红花 panel DTO (bot charityRedFlowerDto).
func BuildCharity(ctx context.Context, api *game.API) (map[string]any, error) {
	reply, err := api.ListActivityWindows(ctx)
	if err != nil {
		return nil, err
	}
	entry := findActivityEntry(reply.Activities, CharityActivityID)
	if entry == nil || entry.CharityRedFlower == nil || entry.Activity == nil {
		return nil, charityErr("CHARITY_RED_FLOWER_UNAVAILABLE", "服务端未发现公益小红花活动状态")
	}
	var window *activitypb.ActivityWindow
	if len(reply.ActivityWindows) > 0 {
		window = charityWindowFor(reply.ActivityWindows, CharityActivityID)
	}
	return charityDTO(entry, window), nil
}

func charityDTO(entry *activitypb.ActivityData, window *activitypb.ActivityWindow) map[string]any {
	activity := entry.Activity
	state := entry.CharityRedFlower
	activityID := CharityActivityID
	if activity.ActivityId > 0 {
		activityID = activity.ActivityId
	}

	serverTime := logic.GetServerTimeSec()
	startTime := activity.BeginTime
	endTime := activity.EndTime
	if window != nil {
		if window.BeginTime > 0 {
			startTime = window.BeginTime
		}
		if window.EndTime > 0 {
			endTime = window.EndTime
		}
	}
	if state.EndTime > 0 {
		endTime = state.EndTime
	}
	active := serverTime >= startTime && serverTime <= endTime

	progress := reconcileCharityProgress(activityID, state)
	flowStatus := strconv.FormatInt(state.FlowStatus, 10)
	currentDateKey := time.Now().Format("20060102")
	publicFundDate := strconv.FormatInt(0, 10)
	if state.PublicFund != nil {
		publicFundDate = strconv.FormatInt(state.PublicFund.Date, 10)
	}
	dailyGiftClaimed := flowStatus == charityFlowDailyGiftClaimed || (publicFundDate != "0" && publicFundDate == currentDateKey)
	dailyGiftHarvested := flowStatus == charityFlowHarvested || flowStatus == charityFlowDailyGiftClaimed

	progressRewards := make([]map[string]any, 0, len(state.ProgressRewards))
	for _, reward := range state.ProgressRewards {
		if reward == nil {
			continue
		}
		key := strconv.FormatInt(reward.Target, 10)
		_, claimed := progress.state.claimed[key]
		_, pending := progress.state.pending[key]
		reached := state.DonatedLove >= reward.Target
		progressRewards = append(progressRewards, map[string]any{
			"target":    key,
			"reward":    activityItemDTO(reward.Reward.ItemId, reward.Reward.Count),
			"statusCode": strconv.FormatInt(reward.Status, 10),
			"reached":   reached,
			"claimed":   claimed,
			"claimable": active && reached && reward.Status == 1 && !claimed && pending,
			"claimSupported": true,
		})
	}

	globalRewardTarget := state.GlobalReward.GetTarget()
	if globalRewardTarget == 0 {
		globalRewardTarget = state.GlobalTargetLove
	}
	settlementGlobalReached := globalRewardTarget != 0 && state.GlobalDonatedLove >= globalRewardTarget
	settlementPersonalReached := state.DonatedLove >= state.SettlementRequiredLove

	loveCount := state.LoveBalance
	return map[string]any{
		"groupId":          strconv.FormatInt(CharityGroupID, 10),
		"activityId":       strconv.FormatInt(activityID, 10),
		"name":             stringOr(activity.Name, "公益小红花"),
		"title":            stringOr(activity.Name, "公益小红花"),
		"startTime":        strconv.FormatInt(startTime, 10),
		"endTime":          strconv.FormatInt(endTime, 10),
		"serverTime":       strconv.FormatInt(serverTime, 10),
		"active":           active,
		"rules":            activityRulesDTO(activity.Extra),
		"love":             activityItemDTO(state.LoveItemId, loveCount),
		"loveBalance":      strconv.FormatInt(loveCount, 10),
		"donatedLove":      strconv.FormatInt(state.DonatedLove, 10),
		"flowStatus":       flowStatus,
		"agreementStatus":  strconv.FormatInt(state.AgreementStatus, 10),
		"seedReward": map[string]any{
			"statusCode": strconv.FormatInt(state.SeedRewardStatus, 10),
			"claimable":  active && state.SeedRewardStatus == 2,
			"claimed":    state.SeedRewardStatus == 3,
			"reward":     activityItemDTO(state.SeedReward.ItemId, state.SeedReward.Count),
		},
		"dailyGift": map[string]any{
			"statusCode":      strconv.FormatInt(state.DailyRewardStatus, 10),
			"claimed":         dailyGiftClaimed,
			"harvestedToday":  dailyGiftHarvested,
			"reward":          activityItemDTO(state.DailyReward.ItemId, state.DailyReward.Count),
		},
		"progressRewards": progressRewards,
		"globalProgress": map[string]any{
			"donated":     strconv.FormatInt(state.GlobalDonatedLove, 10),
			"target":      strconv.FormatInt(state.GlobalTargetLove, 10),
			"reached":     state.GlobalDonatedLove >= state.GlobalTargetLove,
			"rewardTarget": strconv.FormatInt(globalRewardTarget, 10),
			"reward":      activityItemDTO(state.GlobalReward.Reward.ItemId, state.GlobalReward.Reward.Count),
		},
		"settlement": map[string]any{
			"requiredLove":   strconv.FormatInt(state.SettlementRequiredLove, 10),
			"eligible":       settlementGlobalReached && settlementPersonalReached,
			"globalReached":  settlementGlobalReached,
			"personalReached": settlementPersonalReached,
			"reward":         activityItemDTO(state.SettlementReward.ItemId, state.SettlementReward.Count),
		},
		"actions": map[string]any{
			"claimSeeds": map[string]any{
				"enabled": active && state.SeedRewardStatus == 2, "available": active && state.SeedRewardStatus == 2, "availabilityKnown": true,
			},
			"donateLove": map[string]any{
				"enabled": active && loveCount > 0, "available": active && loveCount > 0, "availabilityKnown": true, "count": loveCount,
			},
			"claimDailyGift": map[string]any{
				"enabled": active && dailyGiftHarvested && !dailyGiftClaimed,
				"available": active && dailyGiftHarvested && !dailyGiftClaimed,
				"attemptable": active && dailyGiftHarvested && !dailyGiftClaimed,
				"availabilityKnown": true,
			},
		},
	}
}

func charityErr(code, message string) error {
	return activityError{Code: code, Message: message}
}

// operateCharity sends one charity operate and validates the echo (bot operateCharityRedFlower).
func operateCharity(ctx context.Context, api *game.API, operateType int64, selector *activitypb.CharityRedFlowerOperateRequest) (*activitypb.ActivityOperateReply, error) {
	req := selector
	req.ActivityId = CharityActivityID
	req.OperateType = operateType
	reply, err := api.OperateRaw(ctx, req, CharityActivityID, operateType)
	if err != nil {
		return nil, err
	}
	return reply, nil
}

func charitySnapshotFromReply(reply *activitypb.ActivityOperateReply) map[string]any {
	if reply == nil || reply.Data == nil || reply.Data.CharityRedFlower == nil {
		return nil
	}
	return charityDTO(reply.Data, nil)
}

// ClaimCharitySeeds claims the 小红花种子 (operate 35).
func ClaimCharitySeeds(ctx context.Context, api *game.API) (map[string]any, error) {
	reply, err := operateCharity(ctx, api, operateCharitySeed, &activitypb.CharityRedFlowerOperateRequest{ClaimSeed: &activitypb.CharityRedFlowerOperateRequest_Empty{}})
	if err != nil {
		return nil, err
	}
	rewards := []map[string]any{}
	if reply.CharitySeedResult != nil {
		rewards = append(rewards, activityItemDTO(reply.CharitySeedResult.Reward.Id, reply.CharitySeedResult.Reward.Count))
	}
	return map[string]any{
		"rewards":  rewards,
		"message":  "小红花种子领取成功",
		"snapshot": charitySnapshotFromReply(reply),
	}, nil
}

// DonateCharityLove donates the whole love balance (operate 36).
func DonateCharityLove(ctx context.Context, api *game.API) (map[string]any, error) {
	reply, err := operateCharity(ctx, api, operateCharityDonate, &activitypb.CharityRedFlowerOperateRequest{DonateLove: &activitypb.CharityRedFlowerOperateRequest_Empty{}})
	if err != nil {
		return nil, err
	}
	donated := int64(0)
	globalDonated := int64(0)
	if reply.CharityDonateResult != nil {
		if reply.CharityDonateResult.Donated > 0 {
			donated = reply.CharityDonateResult.Donated
		} else {
			donated = reply.CharityDonateResult.Count
		}
		globalDonated = reply.CharityDonateResult.GlobalDonated
	}
	return map[string]any{
		"donated":       strconv.FormatInt(donated, 10),
		"globalDonated": strconv.FormatInt(globalDonated, 10),
		"message":       fmt.Sprintf("已捐赠全部 %d 份爱心", donated),
		"snapshot":      charitySnapshotFromReply(reply),
	}, nil
}

// ClaimCharityDailyGift claims today's 公益礼包 (operate 38).
func ClaimCharityDailyGift(ctx context.Context, api *game.API) (map[string]any, error) {
	reply, err := operateCharity(ctx, api, operateCharityDailyGift, &activitypb.CharityRedFlowerOperateRequest{SendPublicFund: &activitypb.CharityRedFlowerOperateRequest_Empty{}})
	if err != nil {
		return nil, err
	}
	rewards := []map[string]any{}
	status := int64(0)
	if reply.CharityPublicFundResult != nil {
		rewards = append(rewards, activityItemDTO(reply.CharityPublicFundResult.Reward.Id, reply.CharityPublicFundResult.Reward.Count))
		status = reply.CharityPublicFundResult.Status
	}
	return map[string]any{
		"rewards": rewards,
		"publicFund": map[string]any{
			"statusCode": strconv.FormatInt(status, 10),
		},
		"message":  "今日公益礼包领取成功",
		"snapshot": charitySnapshotFromReply(reply),
	}, nil
}

// ClaimCharityProgressReward claims one progress milestone (operate 37).
func ClaimCharityProgressReward(ctx context.Context, api *game.API, target int64) (map[string]any, error) {
	if target <= 0 {
		return nil, charityErr("INVALID_CHARITY_PROGRESS_TARGET", "target 必须是正十进制整数")
	}
	reply, err := operateCharity(ctx, api, operateCharityProgress, &activitypb.CharityRedFlowerOperateRequest{
		ClaimProgressReward: &activitypb.CharityRedFlowerOperateRequest_ProgressRewardParams{Target: target},
	})
	alreadyClaimed := false
	if err != nil {
		if parseActivityErrorCode(err) != charityProgressAlreadyClaimedCode {
			return nil, err
		}
		alreadyClaimed = true
		reply = nil
	}
	// 本地状态是领取历史的权威来源。
	st := charityStateFor(CharityActivityID)
	key := strconv.FormatInt(target, 10)
	st.claimed[key] = struct{}{}
	delete(st.pending, key)

	rewards := []map[string]any{}
	if reply != nil && reply.CharityProgressRewardResult != nil {
		rewards = append(rewards, activityItemDTO(reply.CharityProgressRewardResult.Reward.Id, reply.CharityProgressRewardResult.Reward.Count))
	}
	message := fmt.Sprintf("公益进度奖励领取成功（%d 份爱心）", target)
	if alreadyClaimed {
		message = fmt.Sprintf("公益进度奖励已领取（%d 份爱心）", target)
	}
	return map[string]any{
		"target":         key,
		"rewards":        rewards,
		"claimed":        true,
		"alreadyClaimed": alreadyClaimed,
		"message":        message,
		"snapshot":       charitySnapshotFromReply(reply),
	}, nil
}

func parseActivityErrorCode(err error) int64 {
	msg := err.Error()
	for i := 0; i+len("code=") <= len(msg); i++ {
		if msg[i:i+len("code=")] == "code=" {
			start := i + len("code=")
			end := start
			for end < len(msg) && msg[end] >= '0' && msg[end] <= '9' {
				end++
			}
			code, _ := strconv.ParseInt(msg[start:end], 10, 64)
			return code
		}
	}
	return 0
}
