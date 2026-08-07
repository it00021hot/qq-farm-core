package qqvippb

import (
	"fmt"

	"github.com/MQEnergy/go-skeleton/internal/farm/proto/corepb"
	"google.golang.org/protobuf/encoding/protowire"
)

func (m *GetDailyGiftStatusRequest) Marshal() []byte { return []byte{} }

func (m *GetDailyGiftStatusRequest) Unmarshal(data []byte) error {
	*m = GetDailyGiftStatusRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("qqvippb.GetDailyGiftStatusRequest: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("qqvippb.GetDailyGiftStatusRequest: %w", err)
		}
	}
	return nil
}

func (m *GetDailyGiftStatusReply) Unmarshal(data []byte) error {
	*m = GetDailyGiftStatusReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("qqvippb.GetDailyGiftStatusReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("qqvippb.GetDailyGiftStatusReply: bad can_claim")
			}
			data = data[n:]
			m.CanClaim = protowire.DecodeBool(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("qqvippb.GetDailyGiftStatusReply: bad has_gift")
			}
			data = data[n:]
			m.HasGift = protowire.DecodeBool(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("qqvippb.GetDailyGiftStatusReply: %w", err)
			}
		}
	}
	return nil
}

func (m *ClaimDailyGiftRequest) Marshal() []byte { return []byte{} }

func (m *ClaimDailyGiftRequest) Unmarshal(data []byte) error {
	*m = ClaimDailyGiftRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("qqvippb.ClaimDailyGiftRequest: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("qqvippb.ClaimDailyGiftRequest: %w", err)
		}
	}
	return nil
}

func (m *ClaimDailyGiftReply) Unmarshal(data []byte) error {
	*m = ClaimDailyGiftReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("qqvippb.ClaimDailyGiftReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("qqvippb.ClaimDailyGiftReply: bad item")
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
				return fmt.Errorf("qqvippb.ClaimDailyGiftReply: %w", err)
			}
		}
	}
	return nil
}

func (m *VipInfoUpdatedNTF) Unmarshal(data []byte) error {
	*m = VipInfoUpdatedNTF{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("qqvippb.VipInfoUpdatedNTF: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("qqvippb.VipInfoUpdatedNTF: bad vip_level")
			}
			data = data[n:]
			m.VipLevel = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("qqvippb.VipInfoUpdatedNTF: bad can_claim")
			}
			data = data[n:]
			m.CanClaim = protowire.DecodeBool(v)
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("qqvippb.VipInfoUpdatedNTF: bad has_gift")
			}
			data = data[n:]
			m.HasGift = protowire.DecodeBool(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("qqvippb.VipInfoUpdatedNTF: %w", err)
			}
		}
	}
	return nil
}
