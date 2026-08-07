package shoppb

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

func skipField(num protowire.Number, typ protowire.Type, data []byte) ([]byte, error) {
	n := protowire.ConsumeFieldValue(num, typ, data)
	if n < 0 {
		return nil, fmt.Errorf("shoppb: bad field %d typ=%d", num, typ)
	}
	return data[n:], nil
}

func appendInt64Varint(b []byte, field int, v int64) []byte {
	if v == 0 {
		return b
	}
	b = protowire.AppendTag(b, protowire.Number(field), protowire.VarintType)
	b = protowire.AppendVarint(b, uint64(v))
	return b
}

func appendInt32(b []byte, field int, v int32) []byte {
	if v == 0 {
		return b
	}
	b = protowire.AppendTag(b, protowire.Number(field), protowire.VarintType)
	b = protowire.AppendVarint(b, uint64(v))
	return b
}

func appendBool(b []byte, field int, v bool) []byte {
	if !v {
		return b
	}
	b = protowire.AppendTag(b, protowire.Number(field), protowire.VarintType)
	b = protowire.AppendVarint(b, 1)
	return b
}

func appendString(b []byte, field int, v string) []byte {
	if v == "" {
		return b
	}
	b = protowire.AppendTag(b, protowire.Number(field), protowire.BytesType)
	b = protowire.AppendString(b, v)
	return b
}

func appendMessage(b []byte, field int, raw []byte) []byte {
	if len(raw) == 0 {
		return b
	}
	b = protowire.AppendTag(b, protowire.Number(field), protowire.BytesType)
	b = protowire.AppendBytes(b, raw)
	return b
}

func marshalCond(c *Cond) []byte {
	if c == nil {
		return nil
	}
	var b []byte
	b = appendInt32(b, 1, c.Type)
	b = appendInt64Varint(b, 2, c.Param)
	return b
}

func unmarshalCond(m *Cond, data []byte) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("shoppb.Cond: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("shoppb.Cond: bad type")
			}
			data = data[n:]
			m.Type = int32(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("shoppb.Cond: bad param")
			}
			data = data[n:]
			m.Param = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("shoppb.Cond: %w", err)
			}
		}
	}
	return nil
}
