package shoppb

import (
	"fmt"

	"github.com/MQEnergy/go-skeleton/internal/farm/proto/corepb"
	"google.golang.org/protobuf/encoding/protowire"
)

func (m *GoodsInfo) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.ID)
	b = appendInt64Varint(b, 2, m.BoughtNum)
	b = appendInt64Varint(b, 3, m.Price)
	b = appendInt64Varint(b, 4, m.LimitCount)
	b = appendBool(b, 5, m.Unlocked)
	b = appendInt64Varint(b, 6, m.ItemID)
	b = appendInt64Varint(b, 7, m.ItemCount)
	for i := range m.Conds {
		b = appendMessage(b, 8, marshalCond(&m.Conds[i]))
	}
	return b
}

func (m *GoodsInfo) Unmarshal(data []byte) error {
	*m = GoodsInfo{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("shoppb.GoodsInfo: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("shoppb.GoodsInfo: bad id")
			}
			data = data[n:]
			m.ID = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("shoppb.GoodsInfo: bad bought_num")
			}
			data = data[n:]
			m.BoughtNum = int64(v)
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("shoppb.GoodsInfo: bad price")
			}
			data = data[n:]
			m.Price = int64(v)
		case num == 4 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("shoppb.GoodsInfo: bad limit_count")
			}
			data = data[n:]
			m.LimitCount = int64(v)
		case num == 5 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("shoppb.GoodsInfo: bad unlocked")
			}
			data = data[n:]
			m.Unlocked = protowire.DecodeBool(v)
		case num == 6 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("shoppb.GoodsInfo: bad item_id")
			}
			data = data[n:]
			m.ItemID = int64(v)
		case num == 7 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("shoppb.GoodsInfo: bad item_count")
			}
			data = data[n:]
			m.ItemCount = int64(v)
		case num == 8 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("shoppb.GoodsInfo: bad cond")
			}
			data = data[n:]
			cond := Cond{}
			if err := unmarshalCond(&cond, raw); err != nil {
				return err
			}
			m.Conds = append(m.Conds, cond)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("shoppb.GoodsInfo: %w", err)
			}
		}
	}
	return nil
}

func (m *ShopProfilesRequest) Marshal() []byte { return []byte{} }

func (m *ShopProfilesRequest) Unmarshal(data []byte) error {
	*m = ShopProfilesRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("shoppb.ShopProfilesRequest: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("shoppb.ShopProfilesRequest: %w", err)
		}
	}
	return nil
}

func (m *ShopProfile) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.ShopID)
	b = appendString(b, 2, m.ShopName)
	b = appendInt32(b, 3, m.ShopType)
	return b
}

func (m *ShopProfile) Unmarshal(data []byte) error {
	*m = ShopProfile{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("shoppb.ShopProfile: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("shoppb.ShopProfile: bad shop_id")
			}
			data = data[n:]
			m.ShopID = int64(v)
		case num == 2 && typ == protowire.BytesType:
			v, n := protowire.ConsumeString(data)
			if n < 0 {
				return fmt.Errorf("shoppb.ShopProfile: bad shop_name")
			}
			data = data[n:]
			m.ShopName = v
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("shoppb.ShopProfile: bad shop_type")
			}
			data = data[n:]
			m.ShopType = int32(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("shoppb.ShopProfile: %w", err)
			}
		}
	}
	return nil
}

func (m *ShopProfilesReply) Unmarshal(data []byte) error {
	*m = ShopProfilesReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("shoppb.ShopProfilesReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("shoppb.ShopProfilesReply: bad shop_profile")
			}
			data = data[n:]
			p := ShopProfile{}
			if err := p.Unmarshal(raw); err != nil {
				return err
			}
			m.ShopProfiles = append(m.ShopProfiles, p)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("shoppb.ShopProfilesReply: %w", err)
			}
		}
	}
	return nil
}

func (m *ShopInfoRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	return appendInt64Varint(nil, 1, m.ShopID)
}

func (m *ShopInfoRequest) Unmarshal(data []byte) error {
	*m = ShopInfoRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("shoppb.ShopInfoRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("shoppb.ShopInfoRequest: bad shop_id")
			}
			data = data[n:]
			m.ShopID = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("shoppb.ShopInfoRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *ShopInfoReply) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	for i := range m.GoodsList {
		b = appendMessage(b, 1, m.GoodsList[i].Marshal())
	}
	return b
}

func (m *ShopInfoReply) Unmarshal(data []byte) error {
	*m = ShopInfoReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("shoppb.ShopInfoReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("shoppb.ShopInfoReply: bad goods")
			}
			data = data[n:]
			goods := GoodsInfo{}
			if err := goods.Unmarshal(raw); err != nil {
				return err
			}
			m.GoodsList = append(m.GoodsList, goods)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("shoppb.ShopInfoReply: %w", err)
			}
		}
	}
	return nil
}

func (m *BuyGoodsRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.GoodsID)
	b = appendInt64Varint(b, 2, m.Num)
	b = appendInt64Varint(b, 3, m.Price)
	return b
}

func (m *BuyGoodsRequest) Unmarshal(data []byte) error {
	*m = BuyGoodsRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("shoppb.BuyGoodsRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("shoppb.BuyGoodsRequest: bad goods_id")
			}
			data = data[n:]
			m.GoodsID = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("shoppb.BuyGoodsRequest: bad num")
			}
			data = data[n:]
			m.Num = int64(v)
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("shoppb.BuyGoodsRequest: bad price")
			}
			data = data[n:]
			m.Price = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("shoppb.BuyGoodsRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *BuyGoodsReply) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	if m.Goods != nil {
		b = appendMessage(b, 1, m.Goods.Marshal())
	}
	for _, item := range m.GetItems {
		if item != nil {
			b = appendMessage(b, 2, item.Marshal())
		}
	}
	for _, item := range m.CostItems {
		if item != nil {
			b = appendMessage(b, 3, item.Marshal())
		}
	}
	return b
}

func (m *BuyGoodsReply) Unmarshal(data []byte) error {
	*m = BuyGoodsReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("shoppb.BuyGoodsReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("shoppb.BuyGoodsReply: bad goods")
			}
			data = data[n:]
			goods := &GoodsInfo{}
			if err := goods.Unmarshal(raw); err != nil {
				return err
			}
			m.Goods = goods
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("shoppb.BuyGoodsReply: bad get_item")
			}
			data = data[n:]
			item := &corepb.Item{}
			if err := item.Unmarshal(raw); err != nil {
				return err
			}
			m.GetItems = append(m.GetItems, item)
		case num == 3 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("shoppb.BuyGoodsReply: bad cost_item")
			}
			data = data[n:]
			item := &corepb.Item{}
			if err := item.Unmarshal(raw); err != nil {
				return err
			}
			m.CostItems = append(m.CostItems, item)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("shoppb.BuyGoodsReply: %w", err)
			}
		}
	}
	return nil
}

func (m *GoodsUnlockNotify) Unmarshal(data []byte) error {
	*m = GoodsUnlockNotify{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("shoppb.GoodsUnlockNotify: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("shoppb.GoodsUnlockNotify: bad goods")
			}
			data = data[n:]
			goods := GoodsInfo{}
			if err := goods.Unmarshal(raw); err != nil {
				return err
			}
			m.GoodsList = append(m.GoodsList, goods)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("shoppb.GoodsUnlockNotify: %w", err)
			}
		}
	}
	return nil
}
