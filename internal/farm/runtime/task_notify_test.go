package runtime

import (
	"context"
	"testing"

	"github.com/it00021hot/qq-farm-core/internal/farm/game"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/taskpb"
	"google.golang.org/protobuf/proto"
)

func notifyTestTaskInfo() *taskpb.TaskInfo {
	return &taskpb.TaskInfo{
		Tasks: []*taskpb.Task{
			{Id: 11, TaskType: 2, IsUnlocked: true, Progress: 5, TotalProgress: 5},                   // 每日，可领
			{Id: 12, TaskType: 1, IsUnlocked: true, Progress: 3, TotalProgress: 5},                   // 成长，未满
			{Id: 13, TaskType: 1, IsUnlocked: true, IsClaimed: true, Progress: 5, TotalProgress: 5},  // 已领
			{Id: 14, TaskType: 9, IsUnlocked: true, Progress: 2, TotalProgress: 2, ShareMultiple: 3}, // 主线，可分享翻倍
		},
		Actives: []*taskpb.Active{{
			Type: 1,
			Rewards: []*taskpb.ActiveReward{
				{PointId: 5, Status: 2},
				{PointId: 6, Status: 1},
			},
		}},
	}
}

// TestClaimTasksAndActivesClaimsOnlyClaimable 校验推送/轮询共用的领取核心：
// 只领可领任务（分享翻倍带 doShared），活跃度只领已达档位。
func TestClaimTasksAndActivesClaimsOnlyClaimable(t *testing.T) {
	sender := &farmOpSender{}
	api := &game.API{Sender: sender, GID: 1}

	claimTasksAndActives(context.Background(), api, 77, normalizeTaskInfo(notifyTestTaskInfo()))

	if got := sender.methodCount("ClaimTaskReward"); got != 2 {
		t.Fatalf("expected 2 ClaimTaskReward calls, got %d", got)
	}
	claimed := map[int64]bool{}
	doSharedByID := map[int64]bool{}
	for _, body := range sender.historyFor("ClaimTaskReward") {
		var req taskpb.ClaimTaskRewardRequest
		if err := proto.Unmarshal(body, &req); err != nil {
			t.Fatal(err)
		}
		claimed[req.Id] = true
		doSharedByID[req.Id] = req.DoShared
	}
	if !claimed[11] || !claimed[14] || claimed[12] || claimed[13] {
		t.Fatalf("unexpected claimed set: %v", claimed)
	}
	if doSharedByID[11] {
		t.Fatal("task 11 must not be shared")
	}
	if !doSharedByID[14] {
		t.Fatal("task 14 (ShareMultiple>1) must claim shared")
	}

	if got := sender.methodCount("ClaimDailyReward"); got != 1 {
		t.Fatalf("expected 1 ClaimDailyReward call, got %d", got)
	}
	var req taskpb.ClaimDailyRewardRequest
	if err := proto.Unmarshal(sender.body("ClaimDailyReward"), &req); err != nil {
		t.Fatal(err)
	}
	if req.Type != 1 || len(req.PointIds) != 1 || req.PointIds[0] != 5 {
		t.Fatalf("unexpected ClaimDailyReward request: %+v", req)
	}
}

// TestClaimTasksFromNotifyGates 校验推送领取入口：受 Automation.Task 开关约束、
// 单飞锁（taskChecking）期间跳过（降级到下一轮 tick）。
func TestClaimTasksFromNotifyGates(t *testing.T) {
	info := &taskpb.TaskInfo{Tasks: []*taskpb.Task{
		{Id: 21, TaskType: 2, IsUnlocked: true, Progress: 1, TotalProgress: 1},
	}}

	// Automation.Task 关闭 → 不领取。
	sender := &farmOpSender{}
	s := &Session{id: "77"}
	s.claimTasksFromNotify(context.Background(), &game.API{Sender: sender, GID: 1}, info)
	if sender.called("ClaimTaskReward") {
		t.Fatal("must not claim when automation.task is off")
	}

	// 单飞：已有领取在跑 → 跳过。
	senderFlying := &farmOpSender{}
	sFlying := &Session{id: "77"}
	sFlying.cfg.AccountConfig.Automation.Task = true
	sFlying.dailyState.taskChecking = true
	sFlying.claimTasksFromNotify(context.Background(), &game.API{Sender: senderFlying, GID: 1}, info)
	if senderFlying.called("ClaimTaskReward") {
		t.Fatal("must not claim while another task round is in flight")
	}
	if !sFlying.dailyState.taskChecking {
		t.Fatal("single-flight flag must stay held by the original round")
	}
}

// TestClaimTasksFromNotifyClaimsImmediately 校验推送到达即领取。
func TestClaimTasksFromNotifyClaimsImmediately(t *testing.T) {
	sender := &farmOpSender{}
	s := &Session{id: "77"}
	s.cfg.AccountConfig.Automation.Task = true
	s.claimTasksFromNotify(context.Background(), &game.API{Sender: sender, GID: 1}, notifyTestTaskInfo())
	if !sender.called("ClaimTaskReward") {
		t.Fatal("expected immediate claim on task push")
	}
	if s.dailyState.taskChecking {
		t.Fatal("single-flight flag must be released after the round")
	}
}
