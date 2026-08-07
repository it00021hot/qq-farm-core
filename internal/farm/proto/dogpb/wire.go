package dogpb

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

func skipField(num protowire.Number, typ protowire.Type, data []byte) ([]byte, error) {
	n := protowire.ConsumeFieldValue(num, typ, data)
	if n < 0 {
		return nil, fmt.Errorf("dogpb: bad field %d typ=%d", num, typ)
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
