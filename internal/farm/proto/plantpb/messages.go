package plantpb

import (
	"fmt"

	"github.com/MQEnergy/go-skeleton/internal/farm/proto/corepb"
	"google.golang.org/protobuf/encoding/protowire"
)

func (m *AllLandsRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.HostGID)
	return b
}

func (m *AllLandsRequest) Unmarshal(data []byte) error {
	*m = AllLandsRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("plantpb.AllLandsRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.AllLandsRequest: bad host_gid")
			}
			data = data[n:]
			m.HostGID = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("plantpb.AllLandsRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *HarvestRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendPackedInt64s(b, 1, m.LandIDs)
	b = appendInt64Varint(b, 2, m.HostGID)
	b = appendBool(b, 3, m.IsAll)
	return b
}

func (m *HarvestRequest) Unmarshal(data []byte) error {
	*m = HarvestRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("plantpb.HarvestRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1:
			var err error
			data, err = consumeRepeatedInt64(num, typ, data, &m.LandIDs)
			if err != nil {
				return fmt.Errorf("plantpb.HarvestRequest: %w", err)
			}
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.HarvestRequest: bad host_gid")
			}
			data = data[n:]
			m.HostGID = int64(v)
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.HarvestRequest: bad is_all")
			}
			data = data[n:]
			m.IsAll = protowire.DecodeBool(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("plantpb.HarvestRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *WaterLandRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendPackedInt64s(b, 1, m.LandIDs)
	b = appendInt64Varint(b, 2, m.HostGID)
	return b
}

func (m *WaterLandRequest) Unmarshal(data []byte) error {
	*m = WaterLandRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("plantpb.WaterLandRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1:
			var err error
			data, err = consumeRepeatedInt64(num, typ, data, &m.LandIDs)
			if err != nil {
				return fmt.Errorf("plantpb.WaterLandRequest: %w", err)
			}
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.WaterLandRequest: bad host_gid")
			}
			data = data[n:]
			m.HostGID = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("plantpb.WaterLandRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *FarmingRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendPackedInt64s(b, 1, m.LandIDs)
	b = appendInt64Varint(b, 2, m.HostGID)
	if m.Field3 != 0 {
		b = protowire.AppendTag(b, 3, protowire.VarintType)
		b = protowire.AppendVarint(b, uint64(uint32(m.Field3)))
	}
	if m.Field4 != 0 {
		b = protowire.AppendTag(b, 4, protowire.VarintType)
		b = protowire.AppendVarint(b, uint64(uint32(m.Field4)))
	}
	return b
}

func (m *FarmingRequest) Unmarshal(data []byte) error {
	*m = FarmingRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("plantpb.FarmingRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1:
			var err error
			data, err = consumeRepeatedInt64(num, typ, data, &m.LandIDs)
			if err != nil {
				return fmt.Errorf("plantpb.FarmingRequest: %w", err)
			}
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.FarmingRequest: bad host_gid")
			}
			data = data[n:]
			m.HostGID = int64(v)
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.FarmingRequest: bad field_3")
			}
			data = data[n:]
			m.Field3 = int32(v)
		case num == 4 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.FarmingRequest: bad field_4")
			}
			data = data[n:]
			m.Field4 = int32(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("plantpb.FarmingRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *FertilizeRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendPackedInt64s(b, 1, m.LandIDs)
	b = appendInt64Varint(b, 2, m.FertilizerID)
	return b
}

func (m *FertilizeRequest) Unmarshal(data []byte) error {
	*m = FertilizeRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("plantpb.FertilizeRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1:
			var err error
			data, err = consumeRepeatedInt64(num, typ, data, &m.LandIDs)
			if err != nil {
				return fmt.Errorf("plantpb.FertilizeRequest: %w", err)
			}
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.FertilizeRequest: bad fertilizer_id")
			}
			data = data[n:]
			m.FertilizerID = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("plantpb.FertilizeRequest: %w", err)
			}
		}
	}
	return nil
}

func marshalPlantItem(item *PlantItem) []byte {
	if item == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, item.SeedID)
	b = appendPackedInt64s(b, 2, item.LandIDs)
	b = appendBool(b, 3, item.AutoSlave)
	return b
}

func unmarshalPlantItem(m *PlantItem, data []byte) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("plantpb.PlantItem: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantItem: bad seed_id")
			}
			data = data[n:]
			m.SeedID = int64(v)
		case num == 2:
			var err error
			data, err = consumeRepeatedInt64(num, typ, data, &m.LandIDs)
			if err != nil {
				return fmt.Errorf("plantpb.PlantItem: %w", err)
			}
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantItem: bad auto_slave")
			}
			data = data[n:]
			m.AutoSlave = protowire.DecodeBool(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("plantpb.PlantItem: %w", err)
			}
		}
	}
	return nil
}

func (m *PlantRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	for i := range m.Items {
		raw := marshalPlantItem(&m.Items[i])
		b = appendMessage(b, 2, raw)
	}
	return b
}

func (m *PlantRequest) Unmarshal(data []byte) error {
	*m = PlantRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("plantpb.PlantRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantRequest: bad item")
			}
			data = data[n:]
			item := PlantItem{}
			if err := unmarshalPlantItem(&item, raw); err != nil {
				return err
			}
			m.Items = append(m.Items, item)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("plantpb.PlantRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *RemovePlantRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendPackedInt64s(b, 1, m.LandIDs)
	return b
}

func (m *RemovePlantRequest) Unmarshal(data []byte) error {
	*m = RemovePlantRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("plantpb.RemovePlantRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1:
			var err error
			data, err = consumeRepeatedInt64(num, typ, data, &m.LandIDs)
			if err != nil {
				return fmt.Errorf("plantpb.RemovePlantRequest: %w", err)
			}
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("plantpb.RemovePlantRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *UnlockLandRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.LandID)
	b = appendBool(b, 2, m.DoShared)
	return b
}

func (m *UnlockLandRequest) Unmarshal(data []byte) error {
	*m = UnlockLandRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("plantpb.UnlockLandRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.UnlockLandRequest: bad land_id")
			}
			data = data[n:]
			m.LandID = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.UnlockLandRequest: bad do_shared")
			}
			data = data[n:]
			m.DoShared = protowire.DecodeBool(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("plantpb.UnlockLandRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *UpgradeLandRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.LandID)
	return b
}

func (m *UpgradeLandRequest) Unmarshal(data []byte) error {
	*m = UpgradeLandRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("plantpb.UpgradeLandRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.UpgradeLandRequest: bad land_id")
			}
			data = data[n:]
			m.LandID = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("plantpb.UpgradeLandRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *AllLandsReply) Unmarshal(data []byte) error {
	*m = AllLandsReply{}
	if len(data) == 0 {
		return fmt.Errorf("plantpb.AllLandsReply: empty body")
	}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("plantpb.AllLandsReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("plantpb.AllLandsReply: bad land")
			}
			data = data[n:]
			land := &LandInfo{}
			if err := unmarshalLandInfo(land, raw); err != nil {
				return err
			}
			m.Lands = append(m.Lands, land)
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("plantpb.AllLandsReply: bad operation_limit")
			}
			data = data[n:]
			limit := &OperationLimit{}
			if err := unmarshalOperationLimit(limit, raw); err != nil {
				return err
			}
			m.OperationLimits = append(m.OperationLimits, limit)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("plantpb.AllLandsReply: %w", err)
			}
		}
	}
	return nil
}

func (m *AllLandsReply) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	for _, land := range m.Lands {
		b = appendMessage(b, 1, land.Marshal())
	}
	for _, limit := range m.OperationLimits {
		b = appendMessage(b, 2, limit.Marshal())
	}
	return b
}

func (m *HarvestReply) Unmarshal(data []byte) error {
	*m = HarvestReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("plantpb.HarvestReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("plantpb.HarvestReply: bad land")
			}
			data = data[n:]
			land := &LandInfo{}
			if err := unmarshalLandInfo(land, raw); err != nil {
				return err
			}
			m.Land = append(m.Land, land)
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("plantpb.HarvestReply: bad item")
			}
			data = data[n:]
			item := &corepb.Item{}
			if err := item.Unmarshal(raw); err != nil {
				return err
			}
			m.Items = append(m.Items, item)
		case num == 3 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("plantpb.HarvestReply: bad lost_item")
			}
			data = data[n:]
			item := &corepb.Item{}
			if err := item.Unmarshal(raw); err != nil {
				return err
			}
			m.LostItems = append(m.LostItems, item)
		case num == 4 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("plantpb.HarvestReply: bad operation_limit")
			}
			data = data[n:]
			limit := &OperationLimit{}
			if err := unmarshalOperationLimit(limit, raw); err != nil {
				return err
			}
			m.OperationLimits = append(m.OperationLimits, limit)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("plantpb.HarvestReply: %w", err)
			}
		}
	}
	return nil
}

func (m *WaterLandReply) Unmarshal(data []byte) error {
	*m = WaterLandReply{}
	return unmarshalLandOperationReply(data, &m.Land, &m.OperationLimits, "plantpb.WaterLandReply")
}

func (m *FarmingReply) Unmarshal(data []byte) error {
	*m = FarmingReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("plantpb.FarmingReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("plantpb.FarmingReply: bad land")
			}
			data = data[n:]
			land := &LandInfo{}
			if err := unmarshalLandInfo(land, raw); err != nil {
				return err
			}
			m.Land = append(m.Land, land)
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("plantpb.FarmingReply: bad operation_limit")
			}
			data = data[n:]
			limit := &OperationLimit{}
			if err := unmarshalOperationLimit(limit, raw); err != nil {
				return err
			}
			m.OperationLimits = append(m.OperationLimits, limit)
		case num == 3 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("plantpb.FarmingReply: bad result")
			}
			data = data[n:]
			result := &FarmingResult{}
			if err := unmarshalFarmingResult(result, raw); err != nil {
				return err
			}
			m.Results = append(m.Results, result)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("plantpb.FarmingReply: %w", err)
			}
		}
	}
	return nil
}

func unmarshalFarmingResult(m *FarmingResult, data []byte) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("plantpb.FarmingResult: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.FarmingResult: bad land_id")
			}
			data = data[n:]
			m.LandID = int64(v)
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("plantpb.FarmingResult: bad reward")
			}
			data = data[n:]
			item := &corepb.Item{}
			if err := item.Unmarshal(raw); err != nil {
				return err
			}
			m.Reward = item
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("plantpb.FarmingResult: %w", err)
			}
		}
	}
	return nil
}

func (m *FertilizeReply) Unmarshal(data []byte) error {
	*m = FertilizeReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("plantpb.FertilizeReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("plantpb.FertilizeReply: bad land")
			}
			data = data[n:]
			land := &LandInfo{}
			if err := unmarshalLandInfo(land, raw); err != nil {
				return err
			}
			m.Land = append(m.Land, land)
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("plantpb.FertilizeReply: bad operation_limit")
			}
			data = data[n:]
			limit := &OperationLimit{}
			if err := unmarshalOperationLimit(limit, raw); err != nil {
				return err
			}
			m.OperationLimits = append(m.OperationLimits, limit)
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.FertilizeReply: bad fertilizer")
			}
			data = data[n:]
			m.Fertilizer = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("plantpb.FertilizeReply: %w", err)
			}
		}
	}
	return nil
}

func (m *PlantReply) Unmarshal(data []byte) error {
	*m = PlantReply{}
	return unmarshalLandOperationReply(data, &m.Land, &m.OperationLimits, "plantpb.PlantReply")
}

func (m *RemovePlantReply) Unmarshal(data []byte) error {
	*m = RemovePlantReply{}
	return unmarshalLandOperationReply(data, &m.Land, &m.OperationLimits, "plantpb.RemovePlantReply")
}

func unmarshalLandOperationReply(data []byte, lands *[]*LandInfo, limits *[]*OperationLimit, msg string) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("%s: bad tag", msg)
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("%s: bad land", msg)
			}
			data = data[n:]
			land := &LandInfo{}
			if err := unmarshalLandInfo(land, raw); err != nil {
				return err
			}
			*lands = append(*lands, land)
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("%s: bad operation_limit", msg)
			}
			data = data[n:]
			limit := &OperationLimit{}
			if err := unmarshalOperationLimit(limit, raw); err != nil {
				return err
			}
			*limits = append(*limits, limit)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("%s: %w", msg, err)
			}
		}
	}
	return nil
}

func (m *UnlockLandReply) Unmarshal(data []byte) error {
	*m = UnlockLandReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("plantpb.UnlockLandReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("plantpb.UnlockLandReply: bad land")
			}
			data = data[n:]
			land := &LandInfo{}
			if err := unmarshalLandInfo(land, raw); err != nil {
				return err
			}
			m.Land = land
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("plantpb.UnlockLandReply: %w", err)
			}
		}
	}
	return nil
}

func (m *UpgradeLandReply) Unmarshal(data []byte) error {
	*m = UpgradeLandReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("plantpb.UpgradeLandReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("plantpb.UpgradeLandReply: bad land")
			}
			data = data[n:]
			land := &LandInfo{}
			if err := unmarshalLandInfo(land, raw); err != nil {
				return err
			}
			m.Land = land
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("plantpb.UpgradeLandReply: %w", err)
			}
		}
	}
	return nil
}

func (m *LandsNotify) Unmarshal(data []byte) error {
	*m = LandsNotify{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("plantpb.LandsNotify: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("plantpb.LandsNotify: bad land")
			}
			data = data[n:]
			land := &LandInfo{}
			if err := unmarshalLandInfo(land, raw); err != nil {
				return err
			}
			m.Lands = append(m.Lands, land)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.LandsNotify: bad host_gid")
			}
			data = data[n:]
			m.HostGID = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("plantpb.LandsNotify: %w", err)
			}
		}
	}
	return nil
}

func (m *LandsNotify) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	for _, land := range m.Lands {
		b = appendMessage(b, 1, land.Marshal())
	}
	b = appendInt64Varint(b, 2, m.HostGID)
	return b
}

func (m *PutInsectsRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.HostGID)
	b = appendPackedInt64s(b, 2, m.LandIDs)
	return b
}

func (m *PutInsectsRequest) Unmarshal(data []byte) error {
	*m = PutInsectsRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("plantpb.PutInsectsRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PutInsectsRequest: bad host_gid")
			}
			data = data[n:]
			m.HostGID = int64(v)
		case num == 2:
			var err error
			data, err = consumeRepeatedInt64(num, typ, data, &m.LandIDs)
			if err != nil {
				return fmt.Errorf("plantpb.PutInsectsRequest: %w", err)
			}
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("plantpb.PutInsectsRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *PutInsectsReply) Unmarshal(data []byte) error {
	*m = PutInsectsReply{}
	return unmarshalLandOperationReply(data, &m.Land, &m.OperationLimits, "plantpb.PutInsectsReply")
}

func (m *PutWeedsRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.HostGID)
	b = appendPackedInt64s(b, 2, m.LandIDs)
	return b
}

func (m *PutWeedsRequest) Unmarshal(data []byte) error {
	*m = PutWeedsRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("plantpb.PutWeedsRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PutWeedsRequest: bad host_gid")
			}
			data = data[n:]
			m.HostGID = int64(v)
		case num == 2:
			var err error
			data, err = consumeRepeatedInt64(num, typ, data, &m.LandIDs)
			if err != nil {
				return fmt.Errorf("plantpb.PutWeedsRequest: %w", err)
			}
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("plantpb.PutWeedsRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *PutWeedsReply) Unmarshal(data []byte) error {
	*m = PutWeedsReply{}
	return unmarshalLandOperationReply(data, &m.Land, &m.OperationLimits, "plantpb.PutWeedsReply")
}

func (m *CheckCanOperateRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.HostGID)
	b = appendInt64Varint(b, 2, m.OperationID)
	return b
}

func (m *CheckCanOperateRequest) Unmarshal(data []byte) error {
	*m = CheckCanOperateRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("plantpb.CheckCanOperateRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.CheckCanOperateRequest: bad host_gid")
			}
			data = data[n:]
			m.HostGID = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.CheckCanOperateRequest: bad operation_id")
			}
			data = data[n:]
			m.OperationID = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("plantpb.CheckCanOperateRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *CheckCanOperateReply) Unmarshal(data []byte) error {
	*m = CheckCanOperateReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("plantpb.CheckCanOperateReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.CheckCanOperateReply: bad can_operate")
			}
			data = data[n:]
			m.CanOperate = protowire.DecodeBool(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.CheckCanOperateReply: bad can_steal_num")
			}
			data = data[n:]
			m.CanStealNum = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("plantpb.CheckCanOperateReply: %w", err)
			}
		}
	}
	return nil
}

func (m *PutSocialItemRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.HostGID)
	b = appendInt64Varint(b, 2, m.LandID)
	b = appendInt64Varint(b, 3, m.ItemID)
	return b
}

func (m *PutSocialItemRequest) Unmarshal(data []byte) error {
	*m = PutSocialItemRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("plantpb.PutSocialItemRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PutSocialItemRequest: bad host_gid")
			}
			data = data[n:]
			m.HostGID = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PutSocialItemRequest: bad land_id")
			}
			data = data[n:]
			m.LandID = int64(v)
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PutSocialItemRequest: bad item_id")
			}
			data = data[n:]
			m.ItemID = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("plantpb.PutSocialItemRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *ItemChange) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.LandID)
	b = appendInt64Varint(b, 2, m.ItemID)
	b = appendInt64Varint(b, 3, m.Count)
	return b
}

func (m *ItemChange) Unmarshal(data []byte) error {
	*m = ItemChange{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("plantpb.ItemChange: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.ItemChange: bad land_id")
			}
			data = data[n:]
			m.LandID = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.ItemChange: bad item_id")
			}
			data = data[n:]
			m.ItemID = int64(v)
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.ItemChange: bad count")
			}
			data = data[n:]
			m.Count = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("plantpb.ItemChange: %w", err)
			}
		}
	}
	return nil
}

func (m *PutSocialItemReply) Unmarshal(data []byte) error {
	*m = PutSocialItemReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("plantpb.PutSocialItemReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PutSocialItemReply: bad land")
			}
			data = data[n:]
			land := &LandInfo{}
			if err := unmarshalLandInfo(land, raw); err != nil {
				return err
			}
			m.Land = land
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PutSocialItemReply: bad operation_limit")
			}
			data = data[n:]
			limit := &OperationLimit{}
			if err := unmarshalOperationLimit(limit, raw); err != nil {
				return err
			}
			m.OperationLimit = limit
		case num == 3 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PutSocialItemReply: bad reward")
			}
			data = data[n:]
			chg := &ItemChange{}
			if err := chg.Unmarshal(raw); err != nil {
				return err
			}
			m.Rewards = append(m.Rewards, chg)
		case num == 4 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PutSocialItemReply: bad consumed")
			}
			data = data[n:]
			chg := &ItemChange{}
			if err := chg.Unmarshal(raw); err != nil {
				return err
			}
			m.Consumed = append(m.Consumed, chg)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("plantpb.PutSocialItemReply: %w", err)
			}
		}
	}
	return nil
}
