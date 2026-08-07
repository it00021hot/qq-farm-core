package visitpb

import (
	"fmt"

	"github.com/MQEnergy/go-skeleton/internal/farm/proto/plantpb"
	"github.com/MQEnergy/go-skeleton/internal/farm/proto/userpb"
	"google.golang.org/protobuf/encoding/protowire"
)

func (m *EnterRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.HostGID)
	b = appendInt32(b, 2, m.Reason)
	return b
}

func (m *EnterRequest) Unmarshal(data []byte) error {
	*m = EnterRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("visitpb.EnterRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("visitpb.EnterRequest: bad host_gid")
			}
			data = data[n:]
			m.HostGID = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("visitpb.EnterRequest: bad reason")
			}
			data = data[n:]
			m.Reason = int32(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("visitpb.EnterRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *EnterReply) Unmarshal(data []byte) error {
	*m = EnterReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("visitpb.EnterReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("visitpb.EnterReply: bad basic")
			}
			data = data[n:]
			basic := &userpb.BasicInfo{}
			if err := basic.Unmarshal(raw); err != nil {
				return err
			}
			m.Basic = basic
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("visitpb.EnterReply: bad land")
			}
			data = data[n:]
			land := &plantpb.LandInfo{}
			if err := land.Unmarshal(raw); err != nil {
				return err
			}
			m.Lands = append(m.Lands, land)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("visitpb.EnterReply: %w", err)
			}
		}
	}
	return nil
}

func (m *LeaveRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	return appendInt64Varint(nil, 1, m.HostGID)
}

func (m *LeaveRequest) Unmarshal(data []byte) error {
	*m = LeaveRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("visitpb.LeaveRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("visitpb.LeaveRequest: bad host_gid")
			}
			data = data[n:]
			m.HostGID = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("visitpb.LeaveRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *LeaveReply) Unmarshal(data []byte) error {
	*m = LeaveReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("visitpb.LeaveReply: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("visitpb.LeaveReply: %w", err)
		}
	}
	return nil
}
