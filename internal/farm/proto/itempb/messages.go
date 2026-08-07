package itempb

import (
	"fmt"

	"github.com/MQEnergy/go-skeleton/internal/farm/proto/corepb"
	"google.golang.org/protobuf/encoding/protowire"
)

func (m *BagRequest) Marshal() []byte {
	return []byte{}
}

func (m *BagRequest) Unmarshal(data []byte) error {
	*m = BagRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("itempb.BagRequest: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("itempb.BagRequest: %w", err)
		}
	}
	return nil
}

func (m *BagReply) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	if m.ItemBag != nil {
		b = appendMessage(b, 1, m.ItemBag.Marshal())
	}
	return b
}

func (m *BagReply) Unmarshal(data []byte) error {
	*m = BagReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("itempb.BagReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("itempb.BagReply: bad item_bag")
			}
			data = data[n:]
			bag := &corepb.ItemBag{}
			if err := bag.Unmarshal(raw); err != nil {
				return err
			}
			m.ItemBag = bag
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("itempb.BagReply: %w", err)
			}
		}
	}
	return nil
}

func (m *SellRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	return appendItems(nil, 1, m.Items)
}

func (m *SellRequest) Unmarshal(data []byte) error {
	*m = SellRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("itempb.SellRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("itempb.SellRequest: bad item")
			}
			data = data[n:]
			item := corepb.Item{}
			if err := item.Unmarshal(raw); err != nil {
				return err
			}
			m.Items = append(m.Items, item)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("itempb.SellRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *SellReply) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	for _, item := range m.SellItems {
		if item != nil {
			b = appendMessage(b, 1, item.Marshal())
		}
	}
	for _, item := range m.GetItems {
		if item != nil {
			b = appendMessage(b, 2, item.Marshal())
		}
	}
	return b
}

func (m *SellReply) Unmarshal(data []byte) error {
	*m = SellReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("itempb.SellReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("itempb.SellReply: bad sell_item")
			}
			data = data[n:]
			item := &corepb.Item{}
			if err := item.Unmarshal(raw); err != nil {
				return err
			}
			m.SellItems = append(m.SellItems, item)
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("itempb.SellReply: bad get_item")
			}
			data = data[n:]
			item := &corepb.Item{}
			if err := item.Unmarshal(raw); err != nil {
				return err
			}
			m.GetItems = append(m.GetItems, item)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("itempb.SellReply: %w", err)
			}
		}
	}
	return nil
}

func (m *UseRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.ItemID)
	b = appendInt64Varint(b, 2, m.Count)
	b = appendPackedInt64s(b, 3, m.LandIDs)
	return b
}

func (m *UseRequest) Unmarshal(data []byte) error {
	*m = UseRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("itempb.UseRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("itempb.UseRequest: bad item_id")
			}
			data = data[n:]
			m.ItemID = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("itempb.UseRequest: bad count")
			}
			data = data[n:]
			m.Count = int64(v)
		case num == 3:
			var err error
			data, err = consumeRepeatedInt64(num, typ, data, &m.LandIDs)
			if err != nil {
				return fmt.Errorf("itempb.UseRequest: %w", err)
			}
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("itempb.UseRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *UseReply) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	for _, item := range m.Items {
		if item != nil {
			b = appendMessage(b, 1, item.Marshal())
		}
	}
	return b
}

func (m *UseReply) Unmarshal(data []byte) error {
	*m = UseReply{}
	return unmarshalItemField(data, &m.Items)
}

func (m *BatchUseRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	return appendItems(nil, 1, m.Items)
}

func (m *BatchUseRequest) Unmarshal(data []byte) error {
	*m = BatchUseRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("itempb.BatchUseRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("itempb.BatchUseRequest: bad item")
			}
			data = data[n:]
			item := corepb.Item{}
			if err := item.Unmarshal(raw); err != nil {
				return err
			}
			m.Items = append(m.Items, item)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("itempb.BatchUseRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *BatchUseReply) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	for _, item := range m.UsedItems {
		if item != nil {
			b = appendMessage(b, 1, item.Marshal())
		}
	}
	for _, item := range m.Items {
		if item != nil {
			b = appendMessage(b, 2, item.Marshal())
		}
	}
	return b
}

func (m *BatchUseReply) Unmarshal(data []byte) error {
	*m = BatchUseReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("itempb.BatchUseReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("itempb.BatchUseReply: bad used_item")
			}
			data = data[n:]
			item := &corepb.Item{}
			if err := item.Unmarshal(raw); err != nil {
				return err
			}
			m.UsedItems = append(m.UsedItems, item)
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("itempb.BatchUseReply: bad item")
			}
			data = data[n:]
			item := &corepb.Item{}
			if err := item.Unmarshal(raw); err != nil {
				return err
			}
			m.Items = append(m.Items, item)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("itempb.BatchUseReply: %w", err)
			}
		}
	}
	return nil
}

func (m *CannelNewRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	return appendItems(nil, 1, m.Items)
}

func (m *CannelNewRequest) Unmarshal(data []byte) error {
	*m = CannelNewRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("itempb.CannelNewRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("itempb.CannelNewRequest: bad item")
			}
			data = data[n:]
			item := corepb.Item{}
			if err := item.Unmarshal(raw); err != nil {
				return err
			}
			m.Items = append(m.Items, item)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("itempb.CannelNewRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *CannelNewReply) Marshal() []byte { return []byte{} }

func (m *CannelNewReply) Unmarshal(data []byte) error {
	*m = CannelNewReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("itempb.CannelNewReply: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("itempb.CannelNewReply: %w", err)
		}
	}
	return nil
}

func (m *ItemNotify) Unmarshal(data []byte) error {
	*m = ItemNotify{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("itempb.ItemNotify: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("itempb.ItemNotify: bad item")
			}
			data = data[n:]
			chg := &corepb.ItemChg{}
			if err := chg.Unmarshal(raw); err != nil {
				return err
			}
			m.Items = append(m.Items, chg)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("itempb.ItemNotify: %w", err)
			}
		}
	}
	return nil
}
