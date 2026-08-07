package taskpb

import (
	"fmt"

	"github.com/MQEnergy/go-skeleton/internal/farm/proto/corepb"
	"google.golang.org/protobuf/encoding/protowire"
)

func (m *TaskInfoRequest) Marshal() []byte { return []byte{} }

func (m *TaskInfoRequest) Unmarshal(data []byte) error {
	*m = TaskInfoRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("taskpb.TaskInfoRequest: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("taskpb.TaskInfoRequest: %w", err)
		}
	}
	return nil
}

func (m *TaskInfoReply) Unmarshal(data []byte) error {
	*m = TaskInfoReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("taskpb.TaskInfoReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("taskpb.TaskInfoReply: bad task_info")
			}
			data = data[n:]
			info := &TaskInfo{}
			if err := unmarshalTaskInfo(info, raw); err != nil {
				return err
			}
			m.TaskInfo = info
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("taskpb.TaskInfoReply: %w", err)
			}
		}
	}
	return nil
}

func (m *ClaimTaskRewardRequest) Marshal() []byte {
	var b []byte
	b = appendInt64Varint(b, 1, m.ID)
	b = appendBool(b, 2, m.DoShared)
	return b
}

func (m *ClaimTaskRewardRequest) Unmarshal(data []byte) error {
	*m = ClaimTaskRewardRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("taskpb.ClaimTaskRewardRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("taskpb.ClaimTaskRewardRequest: bad id")
			}
			data = data[n:]
			m.ID = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("taskpb.ClaimTaskRewardRequest: bad do_shared")
			}
			data = data[n:]
			m.DoShared = protowire.DecodeBool(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("taskpb.ClaimTaskRewardRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *ClaimTaskRewardReply) Unmarshal(data []byte) error {
	*m = ClaimTaskRewardReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("taskpb.ClaimTaskRewardReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("taskpb.ClaimTaskRewardReply: bad item")
			}
			data = data[n:]
			item := &corepb.Item{}
			if err := item.Unmarshal(raw); err != nil {
				return err
			}
			m.Items = append(m.Items, item)
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("taskpb.ClaimTaskRewardReply: bad task_info")
			}
			data = data[n:]
			info := &TaskInfo{}
			if err := unmarshalTaskInfo(info, raw); err != nil {
				return err
			}
			m.TaskInfo = info
		case num == 3 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("taskpb.ClaimTaskRewardReply: bad compensated_item")
			}
			data = data[n:]
			item := &corepb.Item{}
			if err := item.Unmarshal(raw); err != nil {
				return err
			}
			m.CompensatedItems = append(m.CompensatedItems, item)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("taskpb.ClaimTaskRewardReply: %w", err)
			}
		}
	}
	return nil
}

func (m *BatchClaimTaskRewardRequest) Marshal() []byte {
	var b []byte
	b = appendPackedInt64(b, 1, m.IDs)
	b = appendBool(b, 2, m.DoShared)
	return b
}

func (m *BatchClaimTaskRewardReply) Unmarshal(data []byte) error {
	*m = BatchClaimTaskRewardReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("taskpb.BatchClaimTaskRewardReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("taskpb.BatchClaimTaskRewardReply: bad item")
			}
			data = data[n:]
			item := &corepb.Item{}
			if err := item.Unmarshal(raw); err != nil {
				return err
			}
			m.Items = append(m.Items, item)
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("taskpb.BatchClaimTaskRewardReply: bad task_info")
			}
			data = data[n:]
			info := &TaskInfo{}
			if err := unmarshalTaskInfo(info, raw); err != nil {
				return err
			}
			m.TaskInfo = info
		case num == 3 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("taskpb.BatchClaimTaskRewardReply: bad compensated_item")
			}
			data = data[n:]
			item := &corepb.Item{}
			if err := item.Unmarshal(raw); err != nil {
				return err
			}
			m.CompensatedItems = append(m.CompensatedItems, item)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("taskpb.BatchClaimTaskRewardReply: %w", err)
			}
		}
	}
	return nil
}

func (m *ClaimDailyRewardRequest) Marshal() []byte {
	var b []byte
	b = appendInt32(b, 1, m.Type)
	for _, id := range m.PointIDs {
		b = appendInt64Varint(b, 2, id)
	}
	return b
}

func (m *ClaimDailyRewardRequest) Unmarshal(data []byte) error {
	*m = ClaimDailyRewardRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("taskpb.ClaimDailyRewardRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("taskpb.ClaimDailyRewardRequest: bad type")
			}
			data = data[n:]
			m.Type = int32(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("taskpb.ClaimDailyRewardRequest: bad point_id")
			}
			data = data[n:]
			m.PointIDs = append(m.PointIDs, int64(v))
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("taskpb.ClaimDailyRewardRequest: bad packed point_ids")
			}
			data = data[n:]
			for len(raw) > 0 {
				v, consumed := protowire.ConsumeVarint(raw)
				if consumed < 0 {
					break
				}
				raw = raw[consumed:]
				m.PointIDs = append(m.PointIDs, int64(v))
			}
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("taskpb.ClaimDailyRewardRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *ClaimDailyRewardReply) Unmarshal(data []byte) error {
	*m = ClaimDailyRewardReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("taskpb.ClaimDailyRewardReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("taskpb.ClaimDailyRewardReply: bad item")
			}
			data = data[n:]
			item := &corepb.Item{}
			if err := item.Unmarshal(raw); err != nil {
				return err
			}
			m.Items = append(m.Items, item)
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("taskpb.ClaimDailyRewardReply: bad task_info")
			}
			data = data[n:]
			info := &TaskInfo{}
			if err := unmarshalTaskInfo(info, raw); err != nil {
				return err
			}
			m.TaskInfo = info
		case num == 3 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("taskpb.ClaimDailyRewardReply: bad compensated_item")
			}
			data = data[n:]
			item := &corepb.Item{}
			if err := item.Unmarshal(raw); err != nil {
				return err
			}
			m.CompensatedItems = append(m.CompensatedItems, item)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("taskpb.ClaimDailyRewardReply: %w", err)
			}
		}
	}
	return nil
}

func (m *ClientReportProgressRequest) Marshal() []byte {
	var b []byte
	b = appendInt64Varint(b, 1, m.TaskID)
	b = appendInt64Varint(b, 2, m.Progress)
	return b
}

func (m *ClientReportProgressRequest) Unmarshal(data []byte) error {
	*m = ClientReportProgressRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("taskpb.ClientReportProgressRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("taskpb.ClientReportProgressRequest: bad task_id")
			}
			data = data[n:]
			m.TaskID = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("taskpb.ClientReportProgressRequest: bad progress")
			}
			data = data[n:]
			m.Progress = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("taskpb.ClientReportProgressRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *ClientReportProgressReply) Unmarshal(data []byte) error {
	*m = ClientReportProgressReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("taskpb.ClientReportProgressReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("taskpb.ClientReportProgressReply: bad task_info")
			}
			data = data[n:]
			info := &TaskInfo{}
			if err := unmarshalTaskInfo(info, raw); err != nil {
				return err
			}
			m.TaskInfo = info
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("taskpb.ClientReportProgressReply: %w", err)
			}
		}
	}
	return nil
}

func (m *TaskInfoNotify) Unmarshal(data []byte) error {
	*m = TaskInfoNotify{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("taskpb.TaskInfoNotify: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("taskpb.TaskInfoNotify: bad task_info")
			}
			data = data[n:]
			info := &TaskInfo{}
			if err := unmarshalTaskInfo(info, raw); err != nil {
				return err
			}
			m.TaskInfo = info
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("taskpb.TaskInfoNotify: %w", err)
			}
		}
	}
	return nil
}

func unmarshalTaskInfo(m *TaskInfo, data []byte) error {
	*m = TaskInfo{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("taskpb.TaskInfo: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("taskpb.TaskInfo: bad growth_task")
			}
			data = data[n:]
			t := Task{}
			if err := unmarshalTask(&t, raw); err != nil {
				return err
			}
			m.GrowthTasks = append(m.GrowthTasks, t)
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("taskpb.TaskInfo: bad daily_task")
			}
			data = data[n:]
			t := Task{}
			if err := unmarshalTask(&t, raw); err != nil {
				return err
			}
			m.DailyTasks = append(m.DailyTasks, t)
		case num == 3 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("taskpb.TaskInfo: bad task")
			}
			data = data[n:]
			t := Task{}
			if err := unmarshalTask(&t, raw); err != nil {
				return err
			}
			m.Tasks = append(m.Tasks, t)
		case num == 4 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("taskpb.TaskInfo: bad active")
			}
			data = data[n:]
			a := Active{}
			if err := unmarshalActive(&a, raw); err != nil {
				return err
			}
			m.Actives = append(m.Actives, a)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("taskpb.TaskInfo: %w", err)
			}
		}
	}
	return nil
}

func unmarshalTask(m *Task, data []byte) error {
	*m = Task{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("taskpb.Task: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("taskpb.Task: bad id")
			}
			data = data[n:]
			m.ID = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("taskpb.Task: bad progress")
			}
			data = data[n:]
			m.Progress = int64(v)
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("taskpb.Task: bad is_claimed")
			}
			data = data[n:]
			m.IsClaimed = protowire.DecodeBool(v)
		case num == 4 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("taskpb.Task: bad is_unlocked")
			}
			data = data[n:]
			m.IsUnlocked = protowire.DecodeBool(v)
		case num == 5 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("taskpb.Task: bad reward")
			}
			data = data[n:]
			item := &corepb.Item{}
			if err := item.Unmarshal(raw); err != nil {
				return err
			}
			m.Rewards = append(m.Rewards, item)
		case num == 6 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("taskpb.Task: bad total_progress")
			}
			data = data[n:]
			m.TotalProgress = int64(v)
		case num == 7 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("taskpb.Task: bad share_multiple")
			}
			data = data[n:]
			m.ShareMultiple = int64(v)
		case num == 9 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("taskpb.Task: bad desc")
			}
			data = data[n:]
			m.Desc = string(raw)
		case num == 10 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("taskpb.Task: bad task_type")
			}
			data = data[n:]
			m.TaskType = int32(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("taskpb.Task: %w", err)
			}
		}
	}
	return nil
}

func unmarshalActive(m *Active, data []byte) error {
	*m = Active{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("taskpb.Active: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("taskpb.Active: bad type")
			}
			data = data[n:]
			m.Type = int32(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("taskpb.Active: bad progress")
			}
			data = data[n:]
			m.Progress = int64(v)
		case num == 3 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("taskpb.Active: bad reward")
			}
			data = data[n:]
			r := ActiveReward{}
			if err := unmarshalActiveReward(&r, raw); err != nil {
				return err
			}
			m.Rewards = append(m.Rewards, r)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("taskpb.Active: %w", err)
			}
		}
	}
	return nil
}

func unmarshalActiveReward(m *ActiveReward, data []byte) error {
	*m = ActiveReward{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("taskpb.ActiveReward: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("taskpb.ActiveReward: bad point_id")
			}
			data = data[n:]
			m.PointID = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("taskpb.ActiveReward: bad need_progress")
			}
			data = data[n:]
			m.NeedProgress = int64(v)
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("taskpb.ActiveReward: bad status")
			}
			data = data[n:]
			m.Status = int32(v)
		case num == 4 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("taskpb.ActiveReward: bad reward")
			}
			data = data[n:]
			item := &corepb.Item{}
			if err := item.Unmarshal(raw); err != nil {
				return err
			}
			m.Rewards = append(m.Rewards, item)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("taskpb.ActiveReward: %w", err)
			}
		}
	}
	return nil
}

// Claimable returns whether the task can be claimed.
func (t *Task) Claimable() bool {
	return t.IsUnlocked && !t.IsClaimed && t.Progress >= t.TotalProgress && t.TotalProgress > 0
}

// ClaimableActivePointIDs returns point IDs with status DONE (2).
func (a *Active) ClaimablePointIDs() []int64 {
	var ids []int64
	for _, r := range a.Rewards {
		if r.Status == 2 {
			ids = append(ids, r.PointID)
		}
	}
	return ids
}
