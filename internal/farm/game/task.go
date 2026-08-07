package game

import (
	"context"
	"fmt"

	"github.com/it00021hot/qq-farm-core/internal/farm/proto/illustratedpb"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/taskpb"
)

const taskService = "gamepb.taskpb.TaskService"
const illustratedService = "gamepb.illustratedpb.IllustratedService"

func (a *API) sendTask(ctx context.Context, method string, body []byte) ([]byte, error) {
	if err := a.requireSender(); err != nil {
		return nil, err
	}
	raw, _, err := a.Sender.Send(ctx, taskService, method, nonNilBody(body))
	return raw, err
}

func (a *API) sendIllustrated(ctx context.Context, method string, body []byte) ([]byte, error) {
	if err := a.requireSender(); err != nil {
		return nil, err
	}
	raw, _, err := a.Sender.Send(ctx, illustratedService, method, nonNilBody(body))
	return raw, err
}

// TaskInfo fetches task lists and active rewards.
func (a *API) TaskInfo(ctx context.Context) (*taskpb.TaskInfoReply, error) {
	raw, err := a.sendTask(ctx, "TaskInfo", marshalMessage(&taskpb.TaskInfoRequest{}))
	if err != nil {
		return nil, err
	}
	reply := &taskpb.TaskInfoReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// ClaimTaskReward claims one task reward.
func (a *API) ClaimTaskReward(ctx context.Context, taskID int64, doShared bool) (*taskpb.ClaimTaskRewardReply, error) {
	req := &taskpb.ClaimTaskRewardRequest{Id: taskID, DoShared: doShared}
	raw, err := a.sendTask(ctx, "ClaimTaskReward", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &taskpb.ClaimTaskRewardReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// BatchClaimTaskReward claims multiple task rewards.
func (a *API) BatchClaimTaskReward(ctx context.Context, ids []int64) (*taskpb.BatchClaimTaskRewardReply, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("no task ids")
	}
	req := &taskpb.BatchClaimTaskRewardRequest{Ids: ids}
	raw, err := a.sendTask(ctx, "BatchClaimTaskReward", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &taskpb.BatchClaimTaskRewardReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// ClaimDailyReward claims active point rewards.
func (a *API) ClaimDailyReward(ctx context.Context, activeType int32, pointIDs []int64) (*taskpb.ClaimDailyRewardReply, error) {
	if len(pointIDs) == 0 {
		return nil, fmt.Errorf("no point ids")
	}
	req := &taskpb.ClaimDailyRewardRequest{Type: activeType, PointIds: pointIDs}
	raw, err := a.sendTask(ctx, "ClaimDailyReward", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &taskpb.ClaimDailyRewardReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// ClaimAllIllustratedRewards claims all illustrated rewards (V2).
func (a *API) ClaimAllIllustratedRewards(ctx context.Context, onlyClaimable bool) (*illustratedpb.ClaimAllRewardsV2Reply, error) {
	req := &illustratedpb.ClaimAllRewardsV2Request{OnlyClaimable: onlyClaimable}
	raw, err := a.sendIllustrated(ctx, "ClaimAllRewardsV2", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &illustratedpb.ClaimAllRewardsV2Reply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// ClaimAllTasks claims all currently claimable tasks and active rewards.
func (a *API) ClaimAllTasks(ctx context.Context) (claimedTasks int, claimedActives int, err error) {
	reply, err := a.TaskInfo(ctx)
	if err != nil {
		return 0, 0, err
	}
	if reply.TaskInfo == nil {
		return 0, 0, nil
	}
	info := reply.TaskInfo
	var ids []int64
	for _, t := range append(append(info.GrowthTasks, info.DailyTasks...), info.Tasks...) {
		if t != nil && t.IsUnlocked && !t.IsClaimed && t.Progress >= t.TotalProgress && t.TotalProgress > 0 {
			ids = append(ids, t.Id)
		}
	}
	if len(ids) > 0 {
		if _, err := a.BatchClaimTaskReward(ctx, ids); err != nil {
			for _, id := range ids {
				if _, claimErr := a.ClaimTaskReward(ctx, id, false); claimErr == nil {
					claimedTasks++
				}
			}
		} else {
			claimedTasks = len(ids)
		}
	}
	for _, active := range info.Actives {
		if active == nil {
			continue
		}
		var pointIDs []int64
		for _, point := range active.Rewards {
			if point != nil && point.Status == 2 {
				pointIDs = append(pointIDs, point.PointId)
			}
		}
		if len(pointIDs) == 0 {
			continue
		}
		if _, err := a.ClaimDailyReward(ctx, active.Type, pointIDs); err == nil {
			claimedActives += len(pointIDs)
		}
	}
	return claimedTasks, claimedActives, nil
}
