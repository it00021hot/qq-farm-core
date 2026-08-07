package acepb

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

// AntiDataRequest is gamepb.acepb.AntiDataRequest.
type AntiDataRequest struct {
	Data []byte
}

// AntiDataReply is gamepb.acepb.AntiDataReply.
type AntiDataReply struct {
	Result []byte
}

func (m *AntiDataRequest) Marshal() []byte {
	if m == nil || len(m.Data) == 0 {
		return nil
	}
	var b []byte
	b = protowire.AppendTag(b, 1, protowire.BytesType)
	b = protowire.AppendBytes(b, m.Data)
	return b
}

func (m *AntiDataReply) Unmarshal(data []byte) error {
	*m = AntiDataReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("acepb.AntiDataReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("acepb.AntiDataReply: bad result")
			}
			data = data[n:]
			m.Result = append([]byte(nil), raw...)
		default:
			n := protowire.ConsumeFieldValue(num, typ, data)
			if n < 0 {
				return fmt.Errorf("acepb.AntiDataReply: bad field %d", num)
			}
			data = data[n:]
		}
	}
	return nil
}
