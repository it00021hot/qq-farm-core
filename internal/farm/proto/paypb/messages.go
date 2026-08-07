package paypb

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

func (m *GetRechargeInfoRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.Platform)
	b = appendInt64Varint(b, 2, m.Version)
	return b
}

func (m *GetRechargeInfoRequest) Unmarshal(data []byte) error {
	*m = GetRechargeInfoRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("paypb.GetRechargeInfoRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("paypb.GetRechargeInfoRequest: bad platform")
			}
			data = data[n:]
			m.Platform = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("paypb.GetRechargeInfoRequest: bad version")
			}
			data = data[n:]
			m.Version = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("paypb.GetRechargeInfoRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *GetRechargeInfoReply) Unmarshal(data []byte) error {
	*m = GetRechargeInfoReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("paypb.GetRechargeInfoReply: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("paypb.GetRechargeInfoReply: %w", err)
		}
	}
	return nil
}

func (m *RechargeInfoNotify) Unmarshal(data []byte) error {
	*m = RechargeInfoNotify{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("paypb.RechargeInfoNotify: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("paypb.RechargeInfoNotify: bad status")
			}
			data = data[n:]
			m.Status = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("paypb.RechargeInfoNotify: %w", err)
			}
		}
	}
	return nil
}
