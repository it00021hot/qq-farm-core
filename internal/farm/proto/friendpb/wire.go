package friendpb

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

func skipField(num protowire.Number, typ protowire.Type, data []byte) ([]byte, error) {
	n := protowire.ConsumeFieldValue(num, typ, data)
	if n < 0 {
		return nil, fmt.Errorf("friendpb: bad field %d typ=%d", num, typ)
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
	b = protowire.AppendVarint(b, uint64(uint32(v)))
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

func appendPackedInt64s(b []byte, field int, vals []int64) []byte {
	if len(vals) == 0 {
		return b
	}
	var packed []byte
	for _, v := range vals {
		packed = protowire.AppendVarint(packed, uint64(v))
	}
	b = protowire.AppendTag(b, protowire.Number(field), protowire.BytesType)
	b = protowire.AppendBytes(b, packed)
	return b
}

func consumeRepeatedInt64(num protowire.Number, typ protowire.Type, data []byte, dst *[]int64) ([]byte, error) {
	switch typ {
	case protowire.BytesType:
		raw, n := protowire.ConsumeBytes(data)
		if n < 0 {
			return nil, fmt.Errorf("friendpb: bad packed int64 field %d", num)
		}
		for len(raw) > 0 {
			v, m := protowire.ConsumeVarint(raw)
			if m < 0 {
				return nil, fmt.Errorf("friendpb: bad packed int64 field %d", num)
			}
			raw = raw[m:]
			*dst = append(*dst, int64(v))
		}
		return data[n:], nil
	case protowire.VarintType:
		v, n := protowire.ConsumeVarint(data)
		if n < 0 {
			return nil, fmt.Errorf("friendpb: bad int64 field %d", num)
		}
		*dst = append(*dst, int64(v))
		return data[n:], nil
	default:
		return skipField(num, typ, data)
	}
}
