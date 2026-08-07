package mallpb

import (
	"fmt"

	"github.com/MQEnergy/go-skeleton/internal/farm/proto/corepb"
	"google.golang.org/protobuf/encoding/protowire"
)

func (m *GetMallListBySlotTypeRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	return appendInt32(nil, 1, m.SlotType)
}

func (m *GetMallListBySlotTypeRequest) Unmarshal(data []byte) error {
	*m = GetMallListBySlotTypeRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("mallpb.GetMallListBySlotTypeRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("mallpb.GetMallListBySlotTypeRequest: bad slot_type")
			}
			data = data[n:]
			m.SlotType = int32(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("mallpb.GetMallListBySlotTypeRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *GetMallListBySlotTypeResponse) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	for _, raw := range m.GoodsList {
		b = appendBytes(b, 1, raw)
	}
	b = appendInt64Varint(b, 2, m.Timestamp)
	return b
}

func (m *GetMallListBySlotTypeResponse) Unmarshal(data []byte) error {
	*m = GetMallListBySlotTypeResponse{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("mallpb.GetMallListBySlotTypeResponse: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("mallpb.GetMallListBySlotTypeResponse: bad goods")
			}
			data = data[n:]
			m.GoodsList = append(m.GoodsList, append([]byte(nil), raw...))
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("mallpb.GetMallListBySlotTypeResponse: bad timestamp")
			}
			data = data[n:]
			m.Timestamp = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("mallpb.GetMallListBySlotTypeResponse: %w", err)
			}
		}
	}
	return nil
}

func (m *MallGoods) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt32(b, 1, m.GoodsID)
	b = appendString(b, 2, m.Name)
	b = appendInt32(b, 3, m.Type)
	b = appendBytes(b, 4, m.ItemIDs)
	b = appendBytes(b, 5, m.Price)
	b = appendBool(b, 6, m.IsFree)
	b = appendBytes(b, 7, m.Limit)
	b = appendBool(b, 8, m.IsLimited)
	b = appendString(b, 10, m.Discount)
	return b
}

func (m *MallGoods) Unmarshal(data []byte) error {
	*m = MallGoods{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("mallpb.MallGoods: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("mallpb.MallGoods: bad goods_id")
			}
			data = data[n:]
			m.GoodsID = int32(v)
		case num == 2 && typ == protowire.BytesType:
			v, n := protowire.ConsumeString(data)
			if n < 0 {
				return fmt.Errorf("mallpb.MallGoods: bad name")
			}
			data = data[n:]
			m.Name = v
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("mallpb.MallGoods: bad type")
			}
			data = data[n:]
			m.Type = int32(v)
		case num == 4 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("mallpb.MallGoods: bad item_ids")
			}
			data = data[n:]
			m.ItemIDs = append([]byte(nil), raw...)
		case num == 5 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("mallpb.MallGoods: bad price")
			}
			data = data[n:]
			m.Price = append([]byte(nil), raw...)
		case num == 6 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("mallpb.MallGoods: bad is_free")
			}
			data = data[n:]
			m.IsFree = protowire.DecodeBool(v)
		case num == 7 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("mallpb.MallGoods: bad limit")
			}
			data = data[n:]
			m.Limit = append([]byte(nil), raw...)
		case num == 8 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("mallpb.MallGoods: bad is_limited")
			}
			data = data[n:]
			m.IsLimited = protowire.DecodeBool(v)
		case num == 10 && typ == protowire.BytesType:
			v, n := protowire.ConsumeString(data)
			if n < 0 {
				return fmt.Errorf("mallpb.MallGoods: bad discount")
			}
			data = data[n:]
			m.Discount = v
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("mallpb.MallGoods: %w", err)
			}
		}
	}
	return nil
}

// DecodeMallGoodsList decodes each serialized MallGoods blob.
func DecodeMallGoodsList(rawList [][]byte) ([]MallGoods, error) {
	out := make([]MallGoods, 0, len(rawList))
	for _, raw := range rawList {
		if len(raw) == 0 {
			continue
		}
		g := MallGoods{}
		if err := g.Unmarshal(raw); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, nil
}

func (m *PurchaseRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt32(b, 1, m.GoodsID)
	b = appendInt32(b, 2, m.Count)
	return b
}

func (m *PurchaseRequest) Unmarshal(data []byte) error {
	*m = PurchaseRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("mallpb.PurchaseRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("mallpb.PurchaseRequest: bad goods_id")
			}
			data = data[n:]
			m.GoodsID = int32(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("mallpb.PurchaseRequest: bad count")
			}
			data = data[n:]
			m.Count = int32(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("mallpb.PurchaseRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *PurchaseResponse) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt32(b, 1, m.GoodsID)
	b = appendInt32(b, 2, m.Count)
	b = appendBytes(b, 3, m.RewardInfo)
	b = appendBytes(b, 5, m.Result)
	return b
}

func (m *PurchaseResponse) Unmarshal(data []byte) error {
	*m = PurchaseResponse{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("mallpb.PurchaseResponse: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("mallpb.PurchaseResponse: bad goods_id")
			}
			data = data[n:]
			m.GoodsID = int32(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("mallpb.PurchaseResponse: bad count")
			}
			data = data[n:]
			m.Count = int32(v)
		case num == 3 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("mallpb.PurchaseResponse: bad reward_info")
			}
			data = data[n:]
			m.RewardInfo = append([]byte(nil), raw...)
		case num == 5 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("mallpb.PurchaseResponse: bad result")
			}
			data = data[n:]
			m.Result = append([]byte(nil), raw...)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("mallpb.PurchaseResponse: %w", err)
			}
		}
	}
	return nil
}

func (m *GetMonthCardInfosRequest) Marshal() []byte { return []byte{} }

func (m *GetMonthCardInfosRequest) Unmarshal(data []byte) error {
	*m = GetMonthCardInfosRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("mallpb.GetMonthCardInfosRequest: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("mallpb.GetMonthCardInfosRequest: %w", err)
		}
	}
	return nil
}

func (m *MonthCardInfo) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt32(b, 1, m.GoodsID)
	if m.Reward != nil {
		b = appendMessage(b, 2, m.Reward.Marshal())
	}
	b = appendBool(b, 3, m.CanClaim)
	return b
}

func (m *MonthCardInfo) Unmarshal(data []byte) error {
	*m = MonthCardInfo{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("mallpb.MonthCardInfo: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("mallpb.MonthCardInfo: bad goods_id")
			}
			data = data[n:]
			m.GoodsID = int32(v)
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("mallpb.MonthCardInfo: bad reward")
			}
			data = data[n:]
			item := &corepb.Item{}
			if err := item.Unmarshal(raw); err != nil {
				return err
			}
			m.Reward = item
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("mallpb.MonthCardInfo: bad can_claim")
			}
			data = data[n:]
			m.CanClaim = protowire.DecodeBool(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("mallpb.MonthCardInfo: %w", err)
			}
		}
	}
	return nil
}

func (m *GetMonthCardInfosReply) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	for i := range m.Infos {
		b = appendMessage(b, 1, m.Infos[i].Marshal())
	}
	return b
}

func (m *GetMonthCardInfosReply) Unmarshal(data []byte) error {
	*m = GetMonthCardInfosReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("mallpb.GetMonthCardInfosReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("mallpb.GetMonthCardInfosReply: bad info")
			}
			data = data[n:]
			info := MonthCardInfo{}
			if err := info.Unmarshal(raw); err != nil {
				return err
			}
			m.Infos = append(m.Infos, info)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("mallpb.GetMonthCardInfosReply: %w", err)
			}
		}
	}
	return nil
}

func (m *ClaimMonthCardRewardRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	return appendInt32(nil, 1, m.GoodsID)
}

func (m *ClaimMonthCardRewardRequest) Unmarshal(data []byte) error {
	*m = ClaimMonthCardRewardRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("mallpb.ClaimMonthCardRewardRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("mallpb.ClaimMonthCardRewardRequest: bad goods_id")
			}
			data = data[n:]
			m.GoodsID = int32(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("mallpb.ClaimMonthCardRewardRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *ClaimMonthCardRewardReply) Marshal() []byte {
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

func (m *ClaimMonthCardRewardReply) Unmarshal(data []byte) error {
	*m = ClaimMonthCardRewardReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("mallpb.ClaimMonthCardRewardReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("mallpb.ClaimMonthCardRewardReply: bad item")
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
				return fmt.Errorf("mallpb.ClaimMonthCardRewardReply: %w", err)
			}
		}
	}
	return nil
}

func (m *ProductsHasChangedNotify) Unmarshal(data []byte) error {
	*m = ProductsHasChangedNotify{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("mallpb.ProductsHasChangedNotify: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("mallpb.ProductsHasChangedNotify: bad slot_type")
			}
			data = data[n:]
			m.SlotType = int32(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("mallpb.ProductsHasChangedNotify: %w", err)
			}
		}
	}
	return nil
}

func (m *NeedNotify) Unmarshal(data []byte) error {
	*m = NeedNotify{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("mallpb.NeedNotify: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("mallpb.NeedNotify: bad need_type")
			}
			data = data[n:]
			m.NeedType = int32(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("mallpb.NeedNotify: %w", err)
			}
		}
	}
	return nil
}
