package itempb

import (
	"fmt"

	"github.com/MQEnergy/go-skeleton/internal/farm/proto/corepb"
	"google.golang.org/protobuf/encoding/protowire"
)

func skipField(num protowire.Number, typ protowire.Type, data []byte) ([]byte, error) {
	n := protowire.ConsumeFieldValue(num, typ, data)
	if n < 0 {
		return nil, fmt.Errorf("itempb: bad field %d typ=%d", num, typ)
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

func appendItems(b []byte, field int, items []corepb.Item) []byte {
	for i := range items {
		raw := items[i].Marshal()
		b = appendMessage(b, field, raw)
	}
	return b
}

func consumeRepeatedInt64(num protowire.Number, typ protowire.Type, data []byte, dst *[]int64) ([]byte, error) {
	switch typ {
	case protowire.BytesType:
		raw, n := protowire.ConsumeBytes(data)
		if n < 0 {
			return nil, fmt.Errorf("itempb: bad packed int64 field %d", num)
		}
		for len(raw) > 0 {
			v, m := protowire.ConsumeVarint(raw)
			if m < 0 {
				return nil, fmt.Errorf("itempb: bad packed int64 field %d", num)
			}
			raw = raw[m:]
			*dst = append(*dst, int64(v))
		}
		return data[n:], nil
	case protowire.VarintType:
		v, n := protowire.ConsumeVarint(data)
		if n < 0 {
			return nil, fmt.Errorf("itempb: bad int64 field %d", num)
		}
		*dst = append(*dst, int64(v))
		return data[n:], nil
	default:
		return skipField(num, typ, data)
	}
}

func unmarshalItemField(data []byte, dst *[]*corepb.Item) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("itempb: bad items tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("itempb: bad item")
			}
			data = data[n:]
			item := &corepb.Item{}
			if err := item.Unmarshal(raw); err != nil {
				return err
			}
			*dst = append(*dst, item)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
