package mutantpb

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

func skipField(num protowire.Number, typ protowire.Type, data []byte) ([]byte, error) {
	n := protowire.ConsumeFieldValue(num, typ, data)
	if n < 0 {
		return nil, fmt.Errorf("mutantpb: bad field %d typ=%d", num, typ)
	}
	return data[n:], nil
}

func appendMessage(b []byte, field int, raw []byte) []byte {
	if len(raw) == 0 {
		return b
	}
	b = protowire.AppendTag(b, protowire.Number(field), protowire.BytesType)
	b = protowire.AppendBytes(b, raw)
	return b
}
