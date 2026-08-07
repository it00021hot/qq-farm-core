// Package taskpb provides hand-written task service protobuf types.
package taskpb

import "github.com/MQEnergy/go-skeleton/internal/farm/proto/corepb"

type Task struct {
	ID            int64
	Progress      int64
	IsClaimed     bool
	IsUnlocked    bool
	Rewards       []*corepb.Item
	TotalProgress int64
	ShareMultiple int64
	Desc          string
	TaskType      int32
}

type ActiveReward struct {
	PointID      int64
	NeedProgress int64
	Status       int32
	Rewards      []*corepb.Item
}

type Active struct {
	Type     int32
	Progress int64
	Rewards  []ActiveReward
}

type TaskInfo struct {
	GrowthTasks []Task
	DailyTasks  []Task
	Tasks       []Task
	Actives     []Active
}

type TaskInfoRequest struct{}

type TaskInfoReply struct {
	TaskInfo *TaskInfo
}

type ClaimTaskRewardRequest struct {
	ID       int64
	DoShared bool
}

type ClaimTaskRewardReply struct {
	Items            []*corepb.Item
	TaskInfo         *TaskInfo
	CompensatedItems []*corepb.Item
}

type BatchClaimTaskRewardRequest struct {
	IDs      []int64
	DoShared bool
}

type BatchClaimTaskRewardReply struct {
	Items            []*corepb.Item
	TaskInfo         *TaskInfo
	CompensatedItems []*corepb.Item
}

type ClaimDailyRewardRequest struct {
	Type     int32
	PointIDs []int64
}

type ClaimDailyRewardReply struct {
	Items            []*corepb.Item
	TaskInfo         *TaskInfo
	CompensatedItems []*corepb.Item
}

type ClientReportProgressRequest struct {
	TaskID   int64
	Progress int64
}

type ClientReportProgressReply struct {
	TaskInfo *TaskInfo
}

type TaskInfoNotify struct {
	TaskInfo *TaskInfo
}
