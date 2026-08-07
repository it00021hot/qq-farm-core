// Package corepb provides minimal hand-written corepb types (protoc unavailable).
// Source: qq-farm-bot/core/src/proto/corepb.proto
package corepb

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

// Item is corepb.Item (subset used by plant replies).
type Item struct {
	ID          int64
	Count       int64
	ExpireTime  int64
	UID         int64
	IsNew       bool
	MutantTypes []int64
}

func (m *Item) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	if m.ID != 0 {
		b = protowire.AppendTag(b, 1, protowire.VarintType)
		b = protowire.AppendVarint(b, uint64(m.ID))
	}
	if m.Count != 0 {
		b = protowire.AppendTag(b, 2, protowire.VarintType)
		b = protowire.AppendVarint(b, uint64(m.Count))
	}
	if m.ExpireTime != 0 {
		b = protowire.AppendTag(b, 3, protowire.VarintType)
		b = protowire.AppendVarint(b, uint64(m.ExpireTime))
	}
	if m.UID != 0 {
		b = protowire.AppendTag(b, 6, protowire.VarintType)
		b = protowire.AppendVarint(b, uint64(m.UID))
	}
	if m.IsNew {
		b = protowire.AppendTag(b, 7, protowire.VarintType)
		b = protowire.AppendVarint(b, protowire.EncodeBool(m.IsNew))
	}
	if len(m.MutantTypes) > 0 {
		var packed []byte
		for _, v := range m.MutantTypes {
			packed = protowire.AppendVarint(packed, uint64(v))
		}
		b = protowire.AppendTag(b, 8, protowire.BytesType)
		b = protowire.AppendBytes(b, packed)
	}
	return b
}

func (m *Item) Unmarshal(data []byte) error {
	*m = Item{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("corepb.Item: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("corepb.Item: bad id")
			}
			data = data[n:]
			m.ID = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("corepb.Item: bad count")
			}
			data = data[n:]
			m.Count = int64(v)
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("corepb.Item: bad expire_time")
			}
			data = data[n:]
			m.ExpireTime = int64(v)
		case num == 6 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("corepb.Item: bad uid")
			}
			data = data[n:]
			m.UID = int64(v)
		case num == 7 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("corepb.Item: bad is_new")
			}
			data = data[n:]
			m.IsNew = protowire.DecodeBool(v)
		case num == 8:
			switch typ {
			case protowire.BytesType:
				raw, n := protowire.ConsumeBytes(data)
				if n < 0 {
					return fmt.Errorf("corepb.Item: bad mutant_types")
				}
				data = data[n:]
				for len(raw) > 0 {
					v, mlen := protowire.ConsumeVarint(raw)
					if mlen < 0 {
						return fmt.Errorf("corepb.Item: bad mutant_types packed")
					}
					raw = raw[mlen:]
					m.MutantTypes = append(m.MutantTypes, int64(v))
				}
			case protowire.VarintType:
				v, n := protowire.ConsumeVarint(data)
				if n < 0 {
					return fmt.Errorf("corepb.Item: bad mutant_types")
				}
				data = data[n:]
				m.MutantTypes = append(m.MutantTypes, int64(v))
			default:
				var err error
				data, err = skipField(num, typ, data)
				if err != nil {
					return fmt.Errorf("corepb.Item: %w", err)
				}
			}
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("corepb.Item: %w", err)
			}
		}
	}
	return nil
}

func skipField(num protowire.Number, typ protowire.Type, data []byte) ([]byte, error) {
	n := protowire.ConsumeFieldValue(num, typ, data)
	if n < 0 {
		return nil, fmt.Errorf("bad field %d typ=%d", num, typ)
	}
	return data[n:], nil
}

// ItemBag is corepb.ItemBag (repeated items).
type ItemBag struct {
	Items []*Item
}

func (m *ItemBag) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	for _, item := range m.Items {
		if item == nil {
			continue
		}
		raw := item.Marshal()
		b = protowire.AppendTag(b, 1, protowire.BytesType)
		b = protowire.AppendBytes(b, raw)
	}
	return b
}

func (m *ItemBag) Unmarshal(data []byte) error {
	*m = ItemBag{}
	items, err := unmarshalItems(data)
	if err != nil {
		return err
	}
	m.Items = items
	return nil
}

// ItemChg is corepb.ItemChg (item + delta).
type ItemChg struct {
	Item  *Item
	Delta int64
}

func (m *ItemChg) Unmarshal(data []byte) error {
	*m = ItemChg{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("corepb.ItemChg: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("corepb.ItemChg: bad item")
			}
			data = data[n:]
			item := &Item{}
			if err := item.Unmarshal(raw); err != nil {
				return err
			}
			m.Item = item
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("corepb.ItemChg: bad delta")
			}
			data = data[n:]
			m.Delta = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("corepb.ItemChg: %w", err)
			}
		}
	}
	return nil
}

func unmarshalItems(data []byte) ([]*Item, error) {
	var items []*Item
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return nil, fmt.Errorf("corepb: bad items tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return nil, fmt.Errorf("corepb: bad item")
			}
			data = data[n:]
			item := &Item{}
			if err := item.Unmarshal(raw); err != nil {
				return nil, err
			}
			items = append(items, item)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return nil, err
			}
		}
	}
	return items, nil
}
