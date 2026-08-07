package guidepb

import (
	"fmt"

	"github.com/MQEnergy/go-skeleton/internal/farm/proto/corepb"
	"google.golang.org/protobuf/encoding/protowire"
)

func (m *GuideNode) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.NodeID)
	b = appendString(b, 2, m.Name)
	b = appendBool(b, 3, m.Completed)
	b = appendBool(b, 4, m.RewardClaimed)
	for _, item := range m.Rewards {
		if item == nil {
			continue
		}
		b = appendMessage(b, 5, item.Marshal())
	}
	return b
}

func (m *GuideNode) Unmarshal(data []byte) error {
	*m = GuideNode{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("guidepb.GuideNode: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("guidepb.GuideNode: bad node_id")
			}
			data = data[n:]
			m.NodeID = int64(v)
		case num == 2 && typ == protowire.BytesType:
			v, n := protowire.ConsumeString(data)
			if n < 0 {
				return fmt.Errorf("guidepb.GuideNode: bad name")
			}
			data = data[n:]
			m.Name = v
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("guidepb.GuideNode: bad completed")
			}
			data = data[n:]
			m.Completed = protowire.DecodeBool(v)
		case num == 4 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("guidepb.GuideNode: bad reward_claimed")
			}
			data = data[n:]
			m.RewardClaimed = protowire.DecodeBool(v)
		case num == 5 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("guidepb.GuideNode: bad reward")
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
				return fmt.Errorf("guidepb.GuideNode: %w", err)
			}
		}
	}
	return nil
}

func (m *SetWeakGuideNodeCompleteRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	return appendInt64Varint(nil, 1, m.NodeID)
}

func (m *SetWeakGuideNodeCompleteRequest) Unmarshal(data []byte) error {
	*m = SetWeakGuideNodeCompleteRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("guidepb.SetWeakGuideNodeCompleteRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("guidepb.SetWeakGuideNodeCompleteRequest: bad node_id")
			}
			data = data[n:]
			m.NodeID = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("guidepb.SetWeakGuideNodeCompleteRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *SetWeakGuideNodeCompleteReply) Unmarshal(data []byte) error {
	*m = SetWeakGuideNodeCompleteReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("guidepb.SetWeakGuideNodeCompleteReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("guidepb.SetWeakGuideNodeCompleteReply: bad success")
			}
			data = data[n:]
			m.Success = protowire.DecodeBool(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("guidepb.SetWeakGuideNodeCompleteReply: %w", err)
			}
		}
	}
	return nil
}

func (m *ClaimWeakGuideRewardRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	return appendInt64Varint(nil, 1, m.NodeID)
}

func (m *ClaimWeakGuideRewardRequest) Unmarshal(data []byte) error {
	*m = ClaimWeakGuideRewardRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("guidepb.ClaimWeakGuideRewardRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("guidepb.ClaimWeakGuideRewardRequest: bad node_id")
			}
			data = data[n:]
			m.NodeID = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("guidepb.ClaimWeakGuideRewardRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *ClaimWeakGuideRewardReply) Unmarshal(data []byte) error {
	*m = ClaimWeakGuideRewardReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("guidepb.ClaimWeakGuideRewardReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("guidepb.ClaimWeakGuideRewardReply: bad item")
			}
			data = data[n:]
			item := &corepb.Item{}
			if err := item.Unmarshal(raw); err != nil {
				return err
			}
			m.Items = append(m.Items, item)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("guidepb.ClaimWeakGuideRewardReply: bad success")
			}
			data = data[n:]
			m.Success = protowire.DecodeBool(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("guidepb.ClaimWeakGuideRewardReply: %w", err)
			}
		}
	}
	return nil
}
