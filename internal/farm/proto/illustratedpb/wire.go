package illustratedpb

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

func skipField(num protowire.Number, typ protowire.Type, data []byte) ([]byte, error) {
	n := protowire.ConsumeFieldValue(num, typ, data)
	if n < 0 {
		return nil, fmt.Errorf("illustratedpb: bad field %d typ=%d", num, typ)
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

func appendBytes(b []byte, field int, v []byte) []byte {
	if len(v) == 0 {
		return b
	}
	b = protowire.AppendTag(b, protowire.Number(field), protowire.BytesType)
	b = protowire.AppendBytes(b, v)
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
