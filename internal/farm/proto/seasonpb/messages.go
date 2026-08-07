package seasonpb

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

func (m *GetSeasonInfoRequest) Marshal() []byte { return []byte{} }

func (m *GetSeasonInfoRequest) Unmarshal(data []byte) error {
	*m = GetSeasonInfoRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("seasonpb.GetSeasonInfoRequest: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *GetSeasonInfoReply) Marshal() []byte {
	if m == nil || m.SeasonInfo == nil {
		return nil
	}
	return appendMessage(nil, 1, marshalSeasonInfo(m.SeasonInfo))
}

func (m *GetSeasonInfoReply) Unmarshal(data []byte) error {
	*m = GetSeasonInfoReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("seasonpb.GetSeasonInfoReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("seasonpb.GetSeasonInfoReply: bad season_info")
			}
			data = data[n:]
			info := &SeasonInfo{}
			if err := unmarshalSeasonInfo(info, raw); err != nil {
				return err
			}
			m.SeasonInfo = info
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("seasonpb.GetSeasonInfoReply: %w", err)
			}
		}
	}
	return nil
}

func (m *ClaimBattlePassRewardsRequest) Marshal() []byte { return []byte{} }

func (m *ClaimBattlePassRewardsRequest) Unmarshal(data []byte) error {
	*m = ClaimBattlePassRewardsRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("seasonpb.ClaimBattlePassRewardsRequest: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *ClaimBattlePassRewardsReply) Unmarshal(data []byte) error {
	*m = ClaimBattlePassRewardsReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("seasonpb.ClaimBattlePassRewardsReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("seasonpb.ClaimBattlePassRewardsReply: bad reward")
			}
			data = data[n:]
			item := SeasonItem{}
			if err := unmarshalSeasonItem(&item, raw); err != nil {
				return err
			}
			m.Rewards = append(m.Rewards, item)
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("seasonpb.ClaimBattlePassRewardsReply: bad field_2")
			}
			data = data[n:]
			vals, err := decodePackedInt64(raw)
			if err != nil {
				return err
			}
			m.Field2 = append(m.Field2, vals...)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("seasonpb.ClaimBattlePassRewardsReply: bad field_2")
			}
			data = data[n:]
			m.Field2 = append(m.Field2, int64(v))
		case num == 3 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("seasonpb.ClaimBattlePassRewardsReply: bad pass")
			}
			data = data[n:]
			pass := &SeasonPass{}
			if err := unmarshalSeasonPass(pass, raw); err != nil {
				return err
			}
			m.Pass = pass
		case num == 4 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("seasonpb.ClaimBattlePassRewardsReply: bad field_4")
			}
			data = data[n:]
			m.Field4 = protowire.DecodeBool(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("seasonpb.ClaimBattlePassRewardsReply: %w", err)
			}
		}
	}
	return nil
}

func (m *BattlePassChangeNotify) Unmarshal(data []byte) error {
	*m = BattlePassChangeNotify{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("seasonpb.BattlePassChangeNotify: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("seasonpb.BattlePassChangeNotify: bad pass")
			}
			data = data[n:]
			pass := &SeasonPass{}
			if err := unmarshalSeasonPass(pass, raw); err != nil {
				return err
			}
			m.Pass = pass
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("seasonpb.BattlePassChangeNotify: %w", err)
			}
		}
	}
	return nil
}

func marshalSeasonInfo(m *SeasonInfo) []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.SeasonID)
	b = appendBytes(b, 2, m.Name)
	b = appendInt64Varint(b, 3, m.Status)
	b = appendInt64Varint(b, 5, m.BeginTime)
	b = appendInt64Varint(b, 6, m.EndTime)
	b = appendInt64Varint(b, 7, m.ServerTime)
	for i := range m.Activities {
		b = appendMessage(b, 8, marshalSeasonActivity(&m.Activities[i]))
	}
	if m.Pass != nil {
		b = appendMessage(b, 10, marshalSeasonPass(m.Pass))
	}
	return b
}

func unmarshalSeasonInfo(m *SeasonInfo, data []byte) error {
	*m = SeasonInfo{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("seasonpb.SeasonInfo: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("seasonpb.SeasonInfo: bad season_id")
			}
			data = data[n:]
			m.SeasonID = int64(v)
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("seasonpb.SeasonInfo: bad name")
			}
			data = data[n:]
			m.Name = append([]byte(nil), raw...)
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("seasonpb.SeasonInfo: bad status")
			}
			data = data[n:]
			m.Status = int64(v)
		case num == 5 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("seasonpb.SeasonInfo: bad begin_time")
			}
			data = data[n:]
			m.BeginTime = int64(v)
		case num == 6 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("seasonpb.SeasonInfo: bad end_time")
			}
			data = data[n:]
			m.EndTime = int64(v)
		case num == 7 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("seasonpb.SeasonInfo: bad server_time")
			}
			data = data[n:]
			m.ServerTime = int64(v)
		case num == 8 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("seasonpb.SeasonInfo: bad activity")
			}
			data = data[n:]
			act := SeasonActivity{}
			if err := unmarshalSeasonActivity(&act, raw); err != nil {
				return err
			}
			m.Activities = append(m.Activities, act)
		case num == 10 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("seasonpb.SeasonInfo: bad pass")
			}
			data = data[n:]
			pass := &SeasonPass{}
			if err := unmarshalSeasonPass(pass, raw); err != nil {
				return err
			}
			m.Pass = pass
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("seasonpb.SeasonInfo: %w", err)
			}
		}
	}
	return nil
}

func marshalSeasonActivity(m *SeasonActivity) []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.ActivityID)
	b = appendInt64Varint(b, 2, m.Type)
	b = appendBytes(b, 3, m.Name)
	b = appendInt64Varint(b, 4, m.BeginTime)
	b = appendInt64Varint(b, 5, m.EndTime)
	return b
}

func unmarshalSeasonActivity(m *SeasonActivity, data []byte) error {
	*m = SeasonActivity{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("seasonpb.SeasonActivity: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("seasonpb.SeasonActivity: bad activity_id")
			}
			data = data[n:]
			m.ActivityID = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("seasonpb.SeasonActivity: bad type")
			}
			data = data[n:]
			m.Type = int64(v)
		case num == 3 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("seasonpb.SeasonActivity: bad name")
			}
			data = data[n:]
			m.Name = append([]byte(nil), raw...)
		case num == 4 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("seasonpb.SeasonActivity: bad begin_time")
			}
			data = data[n:]
			m.BeginTime = int64(v)
		case num == 5 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("seasonpb.SeasonActivity: bad end_time")
			}
			data = data[n:]
			m.EndTime = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("seasonpb.SeasonActivity: %w", err)
			}
		}
	}
	return nil
}

func marshalSeasonPass(m *SeasonPass) []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.ActivityID)
	b = appendInt64Varint(b, 2, m.CurrentLevel)
	b = appendInt64Varint(b, 4, m.CurrentProgress)
	b = appendInt64Varint(b, 5, m.ProgressTarget)
	b = appendInt64Varint(b, 6, m.NodeCount)
	for i := range m.Nodes {
		b = appendMessage(b, 8, marshalSeasonRewardNode(&m.Nodes[i]))
	}
	b = appendInt64Varint(b, 9, m.ClaimedThroughLevel)
	b = appendBytes(b, 16, m.Title)
	b = appendBytes(b, 17, m.RulesJSON)
	return b
}

func unmarshalSeasonPass(m *SeasonPass, data []byte) error {
	*m = SeasonPass{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("seasonpb.SeasonPass: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("seasonpb.SeasonPass: bad activity_id")
			}
			data = data[n:]
			m.ActivityID = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("seasonpb.SeasonPass: bad current_level")
			}
			data = data[n:]
			m.CurrentLevel = int64(v)
		case num == 4 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("seasonpb.SeasonPass: bad current_progress")
			}
			data = data[n:]
			m.CurrentProgress = int64(v)
		case num == 5 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("seasonpb.SeasonPass: bad progress_target")
			}
			data = data[n:]
			m.ProgressTarget = int64(v)
		case num == 6 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("seasonpb.SeasonPass: bad node_count")
			}
			data = data[n:]
			m.NodeCount = int64(v)
		case num == 8 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("seasonpb.SeasonPass: bad node")
			}
			data = data[n:]
			node := SeasonRewardNode{}
			if err := unmarshalSeasonRewardNode(&node, raw); err != nil {
				return err
			}
			m.Nodes = append(m.Nodes, node)
		case num == 9 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("seasonpb.SeasonPass: bad claimed_through_level")
			}
			data = data[n:]
			m.ClaimedThroughLevel = int64(v)
		case num == 16 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("seasonpb.SeasonPass: bad title")
			}
			data = data[n:]
			m.Title = append([]byte(nil), raw...)
		case num == 17 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("seasonpb.SeasonPass: bad rules_json")
			}
			data = data[n:]
			m.RulesJSON = append([]byte(nil), raw...)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("seasonpb.SeasonPass: %w", err)
			}
		}
	}
	return nil
}

func marshalSeasonRewardNode(m *SeasonRewardNode) []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.NodeID)
	for i := range m.Rewards {
		b = appendMessage(b, 2, marshalSeasonItem(&m.Rewards[i]))
	}
	b = appendBool(b, 4, m.IsKeyLevel)
	return b
}

func unmarshalSeasonRewardNode(m *SeasonRewardNode, data []byte) error {
	*m = SeasonRewardNode{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("seasonpb.SeasonRewardNode: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("seasonpb.SeasonRewardNode: bad node_id")
			}
			data = data[n:]
			m.NodeID = int64(v)
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("seasonpb.SeasonRewardNode: bad reward")
			}
			data = data[n:]
			item := SeasonItem{}
			if err := unmarshalSeasonItem(&item, raw); err != nil {
				return err
			}
			m.Rewards = append(m.Rewards, item)
		case num == 4 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("seasonpb.SeasonRewardNode: bad is_key_level")
			}
			data = data[n:]
			m.IsKeyLevel = protowire.DecodeBool(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("seasonpb.SeasonRewardNode: %w", err)
			}
		}
	}
	return nil
}

func marshalSeasonItem(m *SeasonItem) []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.ItemID)
	b = appendInt64Varint(b, 2, m.Count)
	return b
}

func unmarshalSeasonItem(m *SeasonItem, data []byte) error {
	*m = SeasonItem{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("seasonpb.SeasonItem: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("seasonpb.SeasonItem: bad item_id")
			}
			data = data[n:]
			m.ItemID = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("seasonpb.SeasonItem: bad count")
			}
			data = data[n:]
			m.Count = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("seasonpb.SeasonItem: %w", err)
			}
		}
	}
	return nil
}

func decodePackedInt64(data []byte) ([]int64, error) {
	var vals []int64
	for len(data) > 0 {
		v, n := protowire.ConsumeVarint(data)
		if n < 0 {
			return nil, fmt.Errorf("seasonpb: bad packed int64")
		}
		data = data[n:]
		vals = append(vals, int64(v))
	}
	return vals, nil
}
