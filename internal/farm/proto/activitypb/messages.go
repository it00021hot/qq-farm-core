package activitypb

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

func (m *QueryActivityRequest) Marshal() []byte {
	var b []byte
	b = appendInt64Varint(b, 1, m.ActivityID)
	b = appendInt64Varint(b, 2, m.OperateType)
	return b
}

func (m *ExchangeShopRequest) Marshal() []byte {
	var b []byte
	b = appendInt64Varint(b, 1, m.ActivityID)
	b = appendInt64Varint(b, 2, m.OperateType)
	if m.ExchangeShopOperate != nil {
		b = appendMessage(b, 101, marshalExchangeShopOperateParams(m.ExchangeShopOperate))
	}
	return b
}

func (m *OperateConstellationRequest) Marshal() []byte {
	var b []byte
	b = appendInt64Varint(b, 1, m.ActivityID)
	b = appendInt64Varint(b, 2, m.OperateType)
	// field_119 is an empty nested message
	b = appendMessage(b, 119, []byte{})
	return b
}

func (m *ActivityOperateReply) Unmarshal(data []byte) error {
	*m = ActivityOperateReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("activitypb.ActivityOperateReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("activitypb.ActivityOperateReply: bad activity_id")
			}
			data = data[n:]
			m.ActivityID = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("activitypb.ActivityOperateReply: bad operate_type")
			}
			data = data[n:]
			m.OperateType = int64(v)
		case num == 3 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("activitypb.ActivityOperateReply: bad data")
			}
			data = data[n:]
			d := &ActivityData{}
			if err := unmarshalActivityData(d, raw); err != nil {
				return err
			}
			m.Data = d
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("activitypb.ActivityOperateReply: %w", err)
			}
		}
	}
	return nil
}

func marshalExchangeShopOperateParams(m *ExchangeShopOperateParams) []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.GoodsID)
	b = appendInt64Varint(b, 2, m.Count)
	return b
}

func unmarshalActivityData(m *ActivityData, data []byte) error {
	*m = ActivityData{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("activitypb.ActivityData: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("activitypb.ActivityData: bad activity")
			}
			data = data[n:]
			act := &ActivityContent{}
			if err := unmarshalActivityContent(act, raw); err != nil {
				return err
			}
			m.Activity = act
		case num == 102 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("activitypb.ActivityData: bad catalog")
			}
			data = data[n:]
			catalog := &StarSandGoodsList{}
			if err := unmarshalStarSandGoodsList(catalog, raw); err != nil {
				return err
			}
			m.Catalog = catalog
		case num == 110 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("activitypb.ActivityData: bad constellation")
			}
			data = data[n:]
			c := &ConstellationData{}
			if err := unmarshalConstellationData(c, raw); err != nil {
				return err
			}
			m.Constellation = c
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("activitypb.ActivityData: %w", err)
			}
		}
	}
	return nil
}

func unmarshalActivityContent(m *ActivityContent, data []byte) error {
	*m = ActivityContent{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("activitypb.ActivityContent: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("activitypb.ActivityContent: bad activity_id")
			}
			data = data[n:]
			m.ActivityID = int64(v)
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("activitypb.ActivityContent: bad type")
			}
			data = data[n:]
			m.Type = int64(v)
		case num == 4 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("activitypb.ActivityContent: bad name")
			}
			data = data[n:]
			m.Name = append([]byte(nil), raw...)
		case num == 6 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("activitypb.ActivityContent: bad begin_time")
			}
			data = data[n:]
			m.BeginTime = int64(v)
		case num == 7 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("activitypb.ActivityContent: bad end_time")
			}
			data = data[n:]
			m.EndTime = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("activitypb.ActivityContent: %w", err)
			}
		}
	}
	return nil
}

func unmarshalStarSandGoodsList(m *StarSandGoodsList, data []byte) error {
	*m = StarSandGoodsList{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("activitypb.StarSandGoodsList: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("activitypb.StarSandGoodsList: bad goods")
			}
			data = data[n:]
			g := StarSandGoods{}
			if err := unmarshalStarSandGoods(&g, raw); err != nil {
				return err
			}
			m.Goods = append(m.Goods, g)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("activitypb.StarSandGoodsList: %w", err)
			}
		}
	}
	return nil
}

func unmarshalStarSandGoods(m *StarSandGoods, data []byte) error {
	*m = StarSandGoods{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("activitypb.StarSandGoods: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("activitypb.StarSandGoods: bad goods_id")
			}
			data = data[n:]
			m.GoodsID = int64(v)
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("activitypb.StarSandGoods: bad item")
			}
			data = data[n:]
			item := &ActivityItem{}
			if err := unmarshalActivityItem(item, raw); err != nil {
				return err
			}
			m.Item = item
		case num == 3 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("activitypb.StarSandGoods: bad cost")
			}
			data = data[n:]
			cost := &ActivityItem{}
			if err := unmarshalActivityItem(cost, raw); err != nil {
				return err
			}
			m.Cost = cost
		case num == 4 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("activitypb.StarSandGoods: bad status")
			}
			data = data[n:]
			m.Status = int64(v)
		case num == 5 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("activitypb.StarSandGoods: bad owned")
			}
			data = data[n:]
			m.Owned = protowire.DecodeBool(v)
		case num == 6 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("activitypb.StarSandGoods: bad sort_order")
			}
			data = data[n:]
			m.SortOrder = int64(v)
		case num == 7 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("activitypb.StarSandGoods: bad name")
			}
			data = data[n:]
			m.Name = append([]byte(nil), raw...)
		case num == 8 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("activitypb.StarSandGoods: bad resource_json")
			}
			data = data[n:]
			m.ResourceJSON = append([]byte(nil), raw...)
		case num == 10 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("activitypb.StarSandGoods: bad field_10")
			}
			data = data[n:]
			m.Field10 = int64(v)
		case num == 11 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("activitypb.StarSandGoods: bad field_11")
			}
			data = data[n:]
			m.Field11 = int64(v)
		case num == 12 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("activitypb.StarSandGoods: bad category")
			}
			data = data[n:]
			m.Category = append([]byte(nil), raw...)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("activitypb.StarSandGoods: %w", err)
			}
		}
	}
	return nil
}

func unmarshalActivityItem(m *ActivityItem, data []byte) error {
	*m = ActivityItem{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("activitypb.ActivityItem: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("activitypb.ActivityItem: bad item_id")
			}
			data = data[n:]
			m.ItemID = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("activitypb.ActivityItem: bad count")
			}
			data = data[n:]
			m.Count = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("activitypb.ActivityItem: %w", err)
			}
		}
	}
	return nil
}

func unmarshalConstellationData(m *ConstellationData, data []byte) error {
	*m = ConstellationData{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("activitypb.ConstellationData: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("activitypb.ConstellationData: bad field_1")
			}
			data = data[n:]
			m.Field1 = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("activitypb.ConstellationData: bad field_2")
			}
			data = data[n:]
			m.Field2 = int64(v)
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("activitypb.ConstellationData: bad field_3")
			}
			data = data[n:]
			m.Field3 = int64(v)
		case num == 4 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("activitypb.ConstellationData: bad node")
			}
			data = data[n:]
			node := ConstellationNode{}
			if err := unmarshalConstellationNode(&node, raw); err != nil {
				return err
			}
			m.Nodes = append(m.Nodes, node)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("activitypb.ConstellationData: %w", err)
			}
		}
	}
	return nil
}

func unmarshalConstellationNode(m *ConstellationNode, data []byte) error {
	*m = ConstellationNode{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("activitypb.ConstellationNode: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("activitypb.ConstellationNode: bad node_id")
			}
			data = data[n:]
			m.NodeID = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("activitypb.ConstellationNode: bad field_2")
			}
			data = data[n:]
			m.Field2 = protowire.DecodeBool(v)
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("activitypb.ConstellationNode: bad field_3")
			}
			data = data[n:]
			m.Field3 = protowire.DecodeBool(v)
		case num == 4 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("activitypb.ConstellationNode: bad field_4")
			}
			data = data[n:]
			m.Field4 = protowire.DecodeBool(v)
		case num == 5 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("activitypb.ConstellationNode: bad reward")
			}
			data = data[n:]
			item := ActivityItem{}
			if err := unmarshalActivityItem(&item, raw); err != nil {
				return err
			}
			m.Rewards = append(m.Rewards, item)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("activitypb.ConstellationNode: %w", err)
			}
		}
	}
	return nil
}

func (m *ActiviesChangeNotify) Unmarshal(data []byte) error {
	*m = ActiviesChangeNotify{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("activitypb.ActiviesChangeNotify: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("activitypb.ActiviesChangeNotify: bad activity")
			}
			data = data[n:]
			act := ActivityContent{}
			if err := unmarshalActivityContent(&act, raw); err != nil {
				return err
			}
			m.Activities = append(m.Activities, act)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("activitypb.ActiviesChangeNotify: %w", err)
			}
		}
	}
	return nil
}
