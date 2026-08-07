package careerpb

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

func (m *CareerInfo) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.CareerID)
	b = appendString(b, 2, m.Name)
	b = appendInt32(b, 3, m.Level)
	b = appendInt64Varint(b, 4, m.Exp)
	b = appendInt32(b, 5, m.Status)
	return b
}

func (m *CareerInfo) Unmarshal(data []byte) error {
	*m = CareerInfo{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("careerpb.CareerInfo: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("careerpb.CareerInfo: bad career_id")
			}
			data = data[n:]
			m.CareerID = int64(v)
		case num == 2 && typ == protowire.BytesType:
			v, n := protowire.ConsumeString(data)
			if n < 0 {
				return fmt.Errorf("careerpb.CareerInfo: bad name")
			}
			data = data[n:]
			m.Name = v
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("careerpb.CareerInfo: bad level")
			}
			data = data[n:]
			m.Level = int32(v)
		case num == 4 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("careerpb.CareerInfo: bad exp")
			}
			data = data[n:]
			m.Exp = int64(v)
		case num == 5 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("careerpb.CareerInfo: bad status")
			}
			data = data[n:]
			m.Status = int32(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("careerpb.CareerInfo: %w", err)
			}
		}
	}
	return nil
}

func (m *CareerInfoGetRequest) Marshal() []byte { return []byte{} }

func (m *CareerInfoGetRequest) Unmarshal(data []byte) error {
	*m = CareerInfoGetRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("careerpb.CareerInfoGetRequest: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("careerpb.CareerInfoGetRequest: %w", err)
		}
	}
	return nil
}

func (m *CareerInfoGetReply) Unmarshal(data []byte) error {
	*m = CareerInfoGetReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("careerpb.CareerInfoGetReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("careerpb.CareerInfoGetReply: bad career")
			}
			data = data[n:]
			c := &CareerInfo{}
			if err := c.Unmarshal(raw); err != nil {
				return err
			}
			m.Careers = append(m.Careers, c)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("careerpb.CareerInfoGetReply: %w", err)
			}
		}
	}
	return nil
}
