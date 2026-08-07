package randomdroppb

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

func (m *DropReward) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.ItemID)
	b = appendInt64Varint(b, 2, m.Count)
	b = appendInt32(b, 3, m.Probability)
	b = appendBool(b, 4, m.Claimed)
	return b
}

func (m *DropReward) Unmarshal(data []byte) error {
	*m = DropReward{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("randomdroppb.DropReward: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("randomdroppb.DropReward: bad item_id")
			}
			data = data[n:]
			m.ItemID = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("randomdroppb.DropReward: bad count")
			}
			data = data[n:]
			m.Count = int64(v)
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("randomdroppb.DropReward: bad probability")
			}
			data = data[n:]
			m.Probability = int32(v)
		case num == 4 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("randomdroppb.DropReward: bad claimed")
			}
			data = data[n:]
			m.Claimed = protowire.DecodeBool(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("randomdroppb.DropReward: %w", err)
			}
		}
	}
	return nil
}

func (m *DropActivityInfo) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.ActivityID)
	b = appendString(b, 2, m.Name)
	b = appendInt32(b, 3, m.Status)
	b = appendInt64Varint(b, 4, m.BeginTime)
	b = appendInt64Varint(b, 5, m.EndTime)
	for _, r := range m.Rewards {
		if r == nil {
			continue
		}
		b = appendMessage(b, 6, r.Marshal())
	}
	b = appendInt32(b, 7, m.DropCount)
	b = appendInt32(b, 8, m.MaxDropCount)
	return b
}

func (m *DropActivityInfo) Unmarshal(data []byte) error {
	*m = DropActivityInfo{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("randomdroppb.DropActivityInfo: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("randomdroppb.DropActivityInfo: bad activity_id")
			}
			data = data[n:]
			m.ActivityID = int64(v)
		case num == 2 && typ == protowire.BytesType:
			v, n := protowire.ConsumeString(data)
			if n < 0 {
				return fmt.Errorf("randomdroppb.DropActivityInfo: bad name")
			}
			data = data[n:]
			m.Name = v
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("randomdroppb.DropActivityInfo: bad status")
			}
			data = data[n:]
			m.Status = int32(v)
		case num == 4 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("randomdroppb.DropActivityInfo: bad begin_time")
			}
			data = data[n:]
			m.BeginTime = int64(v)
		case num == 5 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("randomdroppb.DropActivityInfo: bad end_time")
			}
			data = data[n:]
			m.EndTime = int64(v)
		case num == 6 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("randomdroppb.DropActivityInfo: bad reward")
			}
			data = data[n:]
			r := &DropReward{}
			if err := r.Unmarshal(raw); err != nil {
				return err
			}
			m.Rewards = append(m.Rewards, r)
		case num == 7 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("randomdroppb.DropActivityInfo: bad drop_count")
			}
			data = data[n:]
			m.DropCount = int32(v)
		case num == 8 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("randomdroppb.DropActivityInfo: bad max_drop_count")
			}
			data = data[n:]
			m.MaxDropCount = int32(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("randomdroppb.DropActivityInfo: %w", err)
			}
		}
	}
	return nil
}

func (m *GetActivityInfoRequest) Marshal() []byte { return []byte{} }

func (m *GetActivityInfoRequest) Unmarshal(data []byte) error {
	*m = GetActivityInfoRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("randomdroppb.GetActivityInfoRequest: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("randomdroppb.GetActivityInfoRequest: %w", err)
		}
	}
	return nil
}

func (m *GetActivityInfoReply) Unmarshal(data []byte) error {
	*m = GetActivityInfoReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("randomdroppb.GetActivityInfoReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("randomdroppb.GetActivityInfoReply: bad activity")
			}
			data = data[n:]
			a := &DropActivityInfo{}
			if err := a.Unmarshal(raw); err != nil {
				return err
			}
			m.Activities = append(m.Activities, a)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("randomdroppb.GetActivityInfoReply: %w", err)
			}
		}
	}
	return nil
}
