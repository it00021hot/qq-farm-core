package sharepb

import (
	"fmt"

	"github.com/MQEnergy/go-skeleton/internal/farm/proto/corepb"
	"google.golang.org/protobuf/encoding/protowire"
)

func (m *CheckCanShareRequest) Marshal() []byte { return []byte{} }

func (m *CheckCanShareRequest) Unmarshal(data []byte) error {
	*m = CheckCanShareRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("sharepb.CheckCanShareRequest: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("sharepb.CheckCanShareRequest: %w", err)
		}
	}
	return nil
}

func (m *CheckCanShareReply) Unmarshal(data []byte) error {
	*m = CheckCanShareReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("sharepb.CheckCanShareReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("sharepb.CheckCanShareReply: bad can_share")
			}
			data = data[n:]
			m.CanShare = protowire.DecodeBool(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("sharepb.CheckCanShareReply: %w", err)
			}
		}
	}
	return nil
}

func (m *ReportShareRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendBool(b, 1, m.Field1)
	b = appendInt32(b, 4, m.Field4)
	return b
}

func (m *ReportShareRequest) Unmarshal(data []byte) error {
	*m = ReportShareRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("sharepb.ReportShareRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("sharepb.ReportShareRequest: bad field_1")
			}
			data = data[n:]
			m.Field1 = protowire.DecodeBool(v)
		case num == 4 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("sharepb.ReportShareRequest: bad field_4")
			}
			data = data[n:]
			m.Field4 = int32(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("sharepb.ReportShareRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *ReportShareReply) Unmarshal(data []byte) error {
	*m = ReportShareReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("sharepb.ReportShareReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("sharepb.ReportShareReply: bad result")
			}
			data = data[n:]
			m.Result = append([]byte(nil), raw...)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("sharepb.ReportShareReply: %w", err)
			}
		}
	}
	return nil
}

func (m *ClaimShareRewardRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	return appendBool(nil, 1, m.Claimed)
}

func (m *ClaimShareRewardReply) Unmarshal(data []byte) error {
	*m = ClaimShareRewardReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("sharepb.ClaimShareRewardReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("sharepb.ClaimShareRewardReply: bad item")
			}
			data = data[n:]
			item := &corepb.Item{}
			if err := item.Unmarshal(raw); err != nil {
				return err
			}
			m.Items = append(m.Items, item)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("sharepb.ClaimShareRewardReply: %w", err)
			}
		}
	}
	return nil
}

func marshalInviteUser(m *InviteUser) []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.GID)
	b = appendInt64Varint(b, 2, m.Field2)
	b = appendString(b, 3, m.Name)
	b = appendString(b, 4, m.Avatar)
	return b
}

func unmarshalInviteUser(m *InviteUser, data []byte) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("sharepb.InviteUser: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("sharepb.InviteUser: bad gid")
			}
			data = data[n:]
			m.GID = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("sharepb.InviteUser: bad field_2")
			}
			data = data[n:]
			m.Field2 = int64(v)
		case num == 3 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("sharepb.InviteUser: bad name")
			}
			data = data[n:]
			m.Name = string(raw)
		case num == 4 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("sharepb.InviteUser: bad avatar")
			}
			data = data[n:]
			m.Avatar = string(raw)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("sharepb.InviteUser: %w", err)
			}
		}
	}
	return nil
}

func marshalInviteRewardStage(m *InviteRewardStage) []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.Index)
	b = appendInt64Varint(b, 2, m.RewardIndex)
	if m.Item != nil {
		b = appendMessage(b, 3, m.Item.Marshal())
	}
	b = appendInt32(b, 4, m.Status)
	return b
}

func unmarshalInviteRewardStage(m *InviteRewardStage, data []byte) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("sharepb.InviteRewardStage: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("sharepb.InviteRewardStage: bad index")
			}
			data = data[n:]
			m.Index = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("sharepb.InviteRewardStage: bad reward_index")
			}
			data = data[n:]
			m.RewardIndex = int64(v)
		case num == 3 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("sharepb.InviteRewardStage: bad item")
			}
			data = data[n:]
			item := &corepb.Item{}
			if err := item.Unmarshal(raw); err != nil {
				return err
			}
			m.Item = item
		case num == 4 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("sharepb.InviteRewardStage: bad status")
			}
			data = data[n:]
			m.Status = int32(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("sharepb.InviteRewardStage: %w", err)
			}
		}
	}
	return nil
}

func marshalInviteInfo(m *InviteInfo) []byte {
	if m == nil {
		return nil
	}
	var b []byte
	if m.UserA != nil {
		b = appendMessage(b, 1, marshalInviteUser(m.UserA))
	}
	b = appendInt32(b, 2, m.Status)
	if m.UserB != nil {
		b = appendMessage(b, 3, marshalInviteUser(m.UserB))
	}
	for _, stage := range m.RewardStages {
		b = appendMessage(b, 4, marshalInviteRewardStage(stage))
	}
	return b
}

func unmarshalInviteInfo(m *InviteInfo, data []byte) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("sharepb.InviteInfo: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("sharepb.InviteInfo: bad user_a")
			}
			data = data[n:]
			user := &InviteUser{}
			if err := unmarshalInviteUser(user, raw); err != nil {
				return err
			}
			m.UserA = user
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("sharepb.InviteInfo: bad status")
			}
			data = data[n:]
			m.Status = int32(v)
		case num == 3 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("sharepb.InviteInfo: bad user_b")
			}
			data = data[n:]
			user := &InviteUser{}
			if err := unmarshalInviteUser(user, raw); err != nil {
				return err
			}
			m.UserB = user
		case num == 4 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("sharepb.InviteInfo: bad reward_stage")
			}
			data = data[n:]
			stage := &InviteRewardStage{}
			if err := unmarshalInviteRewardStage(stage, raw); err != nil {
				return err
			}
			m.RewardStages = append(m.RewardStages, stage)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("sharepb.InviteInfo: %w", err)
			}
		}
	}
	return nil
}

func (m *GetInviteInfoRequest) Marshal() []byte { return []byte{} }

func (m *GetInviteInfoRequest) Unmarshal(data []byte) error {
	*m = GetInviteInfoRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("sharepb.GetInviteInfoRequest: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("sharepb.GetInviteInfoRequest: %w", err)
		}
	}
	return nil
}

func (m *GetInviteInfoReply) Unmarshal(data []byte) error {
	*m = GetInviteInfoReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("sharepb.GetInviteInfoReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("sharepb.GetInviteInfoReply: bad info")
			}
			data = data[n:]
			info := &InviteInfo{}
			if err := unmarshalInviteInfo(info, raw); err != nil {
				return err
			}
			m.Infos = append(m.Infos, info)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("sharepb.GetInviteInfoReply: %w", err)
			}
		}
	}
	return nil
}

func (m *GetInviteAwardRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	return appendInt64Varint(nil, 1, m.ShareCfgID)
}

func (m *GetInviteAwardRequest) Unmarshal(data []byte) error {
	*m = GetInviteAwardRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("sharepb.GetInviteAwardRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("sharepb.GetInviteAwardRequest: bad share_cfg_id")
			}
			data = data[n:]
			m.ShareCfgID = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("sharepb.GetInviteAwardRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *GetInviteAwardReply) Unmarshal(data []byte) error {
	*m = GetInviteAwardReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("sharepb.GetInviteAwardReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("sharepb.GetInviteAwardReply: bad success")
			}
			data = data[n:]
			m.Success = protowire.DecodeBool(v)
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("sharepb.GetInviteAwardReply: bad item")
			}
			data = data[n:]
			item := &corepb.Item{}
			if err := item.Unmarshal(raw); err != nil {
				return err
			}
			m.Items = append(m.Items, item)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("sharepb.GetInviteAwardReply: %w", err)
			}
		}
	}
	return nil
}
