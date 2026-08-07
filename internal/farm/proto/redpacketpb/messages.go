package redpacketpb

import (
	"fmt"

	"github.com/MQEnergy/go-skeleton/internal/farm/proto/corepb"
	"google.golang.org/protobuf/encoding/protowire"
)

func (m *RedPacketInfo) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt32(b, 1, m.ID)
	b = appendBool(b, 3, m.CanClaim)
	return b
}

func (m *RedPacketInfo) Unmarshal(data []byte) error {
	*m = RedPacketInfo{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("redpacketpb.RedPacketInfo: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("redpacketpb.RedPacketInfo: bad id")
			}
			data = data[n:]
			m.ID = int32(v)
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("redpacketpb.RedPacketInfo: bad can_claim")
			}
			data = data[n:]
			m.CanClaim = protowire.DecodeBool(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("redpacketpb.RedPacketInfo: %w", err)
			}
		}
	}
	return nil
}

func (m *GetTodayClaimStatusRequest) Marshal() []byte { return []byte{} }

func (m *GetTodayClaimStatusRequest) Unmarshal(data []byte) error {
	*m = GetTodayClaimStatusRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("redpacketpb.GetTodayClaimStatusRequest: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("redpacketpb.GetTodayClaimStatusRequest: %w", err)
		}
	}
	return nil
}

func (m *GetTodayClaimStatusReply) Unmarshal(data []byte) error {
	*m = GetTodayClaimStatusReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("redpacketpb.GetTodayClaimStatusReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("redpacketpb.GetTodayClaimStatusReply: bad info")
			}
			data = data[n:]
			info := &RedPacketInfo{}
			if err := info.Unmarshal(raw); err != nil {
				return err
			}
			m.Infos = append(m.Infos, info)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("redpacketpb.GetTodayClaimStatusReply: %w", err)
			}
		}
	}
	return nil
}

func (m *ClaimRedPacketRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	return appendInt32(nil, 1, m.ID)
}

func (m *ClaimRedPacketRequest) Unmarshal(data []byte) error {
	*m = ClaimRedPacketRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("redpacketpb.ClaimRedPacketRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("redpacketpb.ClaimRedPacketRequest: bad id")
			}
			data = data[n:]
			m.ID = int32(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("redpacketpb.ClaimRedPacketRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *ClaimRedPacketReply) Unmarshal(data []byte) error {
	*m = ClaimRedPacketReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("redpacketpb.ClaimRedPacketReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("redpacketpb.ClaimRedPacketReply: bad status")
			}
			data = data[n:]
			m.Status = int32(v)
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("redpacketpb.ClaimRedPacketReply: bad item")
			}
			data = data[n:]
			item := &corepb.Item{}
			if err := item.Unmarshal(raw); err != nil {
				return err
			}
			m.Item = item
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("redpacketpb.ClaimRedPacketReply: %w", err)
			}
		}
	}
	return nil
}
