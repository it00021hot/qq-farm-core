package plantpb

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

func (m *PlantPhaseInfo) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	if m.Phase != 0 {
		b = protowire.AppendTag(b, 1, protowire.VarintType)
		b = protowire.AppendVarint(b, uint64(uint32(m.Phase)))
	}
	b = appendInt64Varint(b, 2, m.BeginTime)
	b = appendInt64Varint(b, 3, m.PhaseID)
	b = appendInt64Varint(b, 6, m.DryTime)
	b = appendInt64Varint(b, 7, m.WeedsTime)
	b = appendInt64Varint(b, 8, m.InsectTime)
	return b
}

func unmarshalPlantPhaseInfo(m *PlantPhaseInfo, data []byte) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("plantpb.PlantPhaseInfo: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantPhaseInfo: bad phase")
			}
			data = data[n:]
			m.Phase = int32(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantPhaseInfo: bad begin_time")
			}
			data = data[n:]
			m.BeginTime = int64(v)
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantPhaseInfo: bad phase_id")
			}
			data = data[n:]
			m.PhaseID = int64(v)
		case num == 6 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantPhaseInfo: bad dry_time")
			}
			data = data[n:]
			m.DryTime = int64(v)
		case num == 7 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantPhaseInfo: bad weeds_time")
			}
			data = data[n:]
			m.WeedsTime = int64(v)
		case num == 8 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantPhaseInfo: bad insect_time")
			}
			data = data[n:]
			m.InsectTime = int64(v)
		case num == 10 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantPhaseInfo: bad mutant")
			}
			data = data[n:]
			mi := MutantInfo{}
			if err := unmarshalMutantInfo(&mi, raw); err != nil {
				return err
			}
			m.Mutants = append(m.Mutants, mi)
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

func unmarshalMutantInfo(m *MutantInfo, data []byte) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("plantpb.MutantInfo: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.MutantInfo: bad mutant_time")
			}
			data = data[n:]
			m.MutantTime = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.MutantInfo: bad mutant_config_id")
			}
			data = data[n:]
			m.MutantConfigID = int64(v)
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.MutantInfo: bad weather_id")
			}
			data = data[n:]
			m.WeatherID = int64(v)
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

func (m *PlantInfo) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.ID)
	if m.Name != "" {
		b = protowire.AppendTag(b, 2, protowire.BytesType)
		b = protowire.AppendString(b, m.Name)
	}
	for i := range m.Phases {
		b = appendMessage(b, 4, m.Phases[i].Marshal())
	}
	b = appendInt64Varint(b, 5, m.Season)
	b = appendInt64Varint(b, 6, m.DryNum)
	b = appendInt64Varint(b, 9, m.StoleNum)
	b = appendInt64Varint(b, 10, m.FruitID)
	b = appendInt64Varint(b, 11, m.FruitNum)
	b = appendPackedInt64s(b, 12, m.WeedOwners)
	b = appendPackedInt64s(b, 13, m.InsectOwners)
	if len(m.Stealers) > 0 {
		b = protowire.AppendTag(b, 14, protowire.BytesType)
		b = protowire.AppendBytes(b, m.Stealers)
	}
	b = appendInt64Varint(b, 15, m.GrowSec)
	b = appendBool(b, 16, m.Stealable)
	if m.LeftInorcFertTimes != nil {
		b = appendInt64Varint(b, 17, *m.LeftInorcFertTimes)
	}
	b = appendInt64Varint(b, 18, m.LeftFruitNum)
	b = appendInt64Varint(b, 19, m.StealIntimacyLevel)
	b = appendPackedInt64s(b, 20, m.MutantConfigIDs)
	b = appendBool(b, 21, m.IsNudged)
	b = appendInt64Varint(b, 22, m.StealPlayer)
	if len(m.StealNum) > 0 {
		b = protowire.AppendTag(b, 23, protowire.BytesType)
		b = protowire.AppendBytes(b, m.StealNum)
	}
	b = appendInt64Varint(b, 24, m.Field24)
	b = appendInt64Varint(b, 25, m.Field25)
	b = appendInt64Varint(b, 26, m.Field26)
	b = appendInt64Varint(b, 27, m.Field27)
	if len(m.Field32) > 0 {
		b = protowire.AppendTag(b, 32, protowire.BytesType)
		b = protowire.AppendBytes(b, m.Field32)
	}
	if m.Limit != nil {
		b = appendMessage(b, 34, m.Limit.Marshal())
	}
	if m.Activity != nil {
		b = appendMessage(b, 36, m.Activity.Marshal())
	}
	b = appendInt64Varint(b, 37, m.Field37)
	return b
}

func (m *PlantLimitInfo) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.ConfigID)
	b = appendInt64Varint(b, 2, m.Limit)
	b = appendInt64Varint(b, 3, m.Value)
	return b
}

func unmarshalPlantLimitInfo(m *PlantLimitInfo, data []byte) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("plantpb.PlantLimitInfo: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantLimitInfo: bad config_id")
			}
			data = data[n:]
			m.ConfigID = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantLimitInfo: bad limit")
			}
			data = data[n:]
			m.Limit = int64(v)
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantLimitInfo: bad value")
			}
			data = data[n:]
			m.Value = int64(v)
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

func (m *PlantActivityInfo) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.ActivityID)
	b = appendInt64Varint(b, 2, m.Param1)
	b = appendInt64Varint(b, 3, m.Param2)
	b = appendInt64Varint(b, 4, m.Date)
	return b
}

func unmarshalPlantActivityInfo(m *PlantActivityInfo, data []byte) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("plantpb.PlantActivityInfo: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantActivityInfo: bad activity_id")
			}
			data = data[n:]
			m.ActivityID = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantActivityInfo: bad param1")
			}
			data = data[n:]
			m.Param1 = int64(v)
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantActivityInfo: bad param2")
			}
			data = data[n:]
			m.Param2 = int64(v)
		case num == 4 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantActivityInfo: bad date")
			}
			data = data[n:]
			m.Date = int64(v)
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

func unmarshalPlantInfo(m *PlantInfo, data []byte) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("plantpb.PlantInfo: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantInfo: bad id")
			}
			data = data[n:]
			m.ID = int64(v)
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantInfo: bad name")
			}
			data = data[n:]
			m.Name = string(raw)
		case num == 4 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantInfo: bad phase")
			}
			data = data[n:]
			phase := PlantPhaseInfo{}
			if err := unmarshalPlantPhaseInfo(&phase, raw); err != nil {
				return err
			}
			m.Phases = append(m.Phases, phase)
		case num == 5 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantInfo: bad season")
			}
			data = data[n:]
			m.Season = int64(v)
		case num == 6 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantInfo: bad dry_num")
			}
			data = data[n:]
			m.DryNum = int64(v)
		case num == 9 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantInfo: bad stole_num")
			}
			data = data[n:]
			m.StoleNum = int64(v)
		case num == 10 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantInfo: bad fruit_id")
			}
			data = data[n:]
			m.FruitID = int64(v)
		case num == 11 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantInfo: bad fruit_num")
			}
			data = data[n:]
			m.FruitNum = int64(v)
		case num == 12:
			var err error
			data, err = consumeRepeatedInt64(num, typ, data, &m.WeedOwners)
			if err != nil {
				return fmt.Errorf("plantpb.PlantInfo: bad weed_owners")
			}
		case num == 13:
			var err error
			data, err = consumeRepeatedInt64(num, typ, data, &m.InsectOwners)
			if err != nil {
				return fmt.Errorf("plantpb.PlantInfo: bad insect_owners")
			}
		case num == 14 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantInfo: bad stealers")
			}
			data = data[n:]
			m.Stealers = append([]byte(nil), raw...)
		case num == 15 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantInfo: bad grow_sec")
			}
			data = data[n:]
			m.GrowSec = int64(v)
		case num == 16 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantInfo: bad stealable")
			}
			data = data[n:]
			m.Stealable = protowire.DecodeBool(v)
		case num == 17 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantInfo: bad left_inorc_fert_times")
			}
			data = data[n:]
			fert := int64(v)
			m.LeftInorcFertTimes = &fert
		case num == 18 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantInfo: bad left_fruit_num")
			}
			data = data[n:]
			m.LeftFruitNum = int64(v)
		case num == 19 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantInfo: bad steal_intimacy_level")
			}
			data = data[n:]
			m.StealIntimacyLevel = int64(v)
		case num == 20:
			var err error
			data, err = consumeRepeatedInt64(num, typ, data, &m.MutantConfigIDs)
			if err != nil {
				return fmt.Errorf("plantpb.PlantInfo: bad mutant_config_ids")
			}
		case num == 21 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantInfo: bad is_nudged")
			}
			data = data[n:]
			m.IsNudged = protowire.DecodeBool(v)
		case num == 22 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantInfo: bad steal_player")
			}
			data = data[n:]
			m.StealPlayer = int64(v)
		case num == 23 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantInfo: bad steal_num")
			}
			data = data[n:]
			m.StealNum = append([]byte(nil), raw...)
		case num == 24 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantInfo: bad field_24")
			}
			data = data[n:]
			m.Field24 = int64(v)
		case num == 25 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantInfo: bad field_25")
			}
			data = data[n:]
			m.Field25 = int64(v)
		case num == 26 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantInfo: bad field_26")
			}
			data = data[n:]
			m.Field26 = int64(v)
		case num == 27 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantInfo: bad field_27")
			}
			data = data[n:]
			m.Field27 = int64(v)
		case num == 32 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantInfo: bad field_32")
			}
			data = data[n:]
			m.Field32 = append([]byte(nil), raw...)
		case num == 34 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantInfo: bad limit")
			}
			data = data[n:]
			lim := &PlantLimitInfo{}
			if err := unmarshalPlantLimitInfo(lim, raw); err != nil {
				return err
			}
			m.Limit = lim
		case num == 36 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantInfo: bad activity")
			}
			data = data[n:]
			act := &PlantActivityInfo{}
			if err := unmarshalPlantActivityInfo(act, raw); err != nil {
				return err
			}
			m.Activity = act
		case num == 37 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.PlantInfo: bad field_37")
			}
			data = data[n:]
			m.Field37 = int64(v)
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

func (m *LandInfo) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.ID)
	b = appendBool(b, 2, m.Unlocked)
	b = appendInt64Varint(b, 3, m.Level)
	b = appendInt64Varint(b, 4, m.MaxLevel)
	b = appendBool(b, 5, m.CouldUnlock)
	b = appendBool(b, 6, m.CouldUpgrade)
	if m.Plant != nil {
		b = appendMessage(b, 10, m.Plant.Marshal())
	}
	b = appendInt64Varint(b, 13, m.MasterLandID)
	b = appendPackedInt64s(b, 14, m.SlaveLandIDs)
	b = appendInt64Varint(b, 15, m.LandSize)
	b = appendInt64Varint(b, 16, m.LandsLevel)
	return b
}

// Unmarshal decodes wire bytes into LandInfo.
func (m *LandInfo) Unmarshal(data []byte) error {
	*m = LandInfo{}
	return unmarshalLandInfo(m, data)
}

func unmarshalLandInfo(m *LandInfo, data []byte) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("plantpb.LandInfo: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.LandInfo: bad id")
			}
			data = data[n:]
			m.ID = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.LandInfo: bad unlocked")
			}
			data = data[n:]
			m.Unlocked = protowire.DecodeBool(v)
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.LandInfo: bad level")
			}
			data = data[n:]
			m.Level = int64(v)
		case num == 4 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.LandInfo: bad max_level")
			}
			data = data[n:]
			m.MaxLevel = int64(v)
		case num == 5 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.LandInfo: bad could_unlock")
			}
			data = data[n:]
			m.CouldUnlock = protowire.DecodeBool(v)
		case num == 6 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.LandInfo: bad could_upgrade")
			}
			data = data[n:]
			m.CouldUpgrade = protowire.DecodeBool(v)
		case num == 10 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("plantpb.LandInfo: bad plant")
			}
			data = data[n:]
			plant := &PlantInfo{}
			if err := unmarshalPlantInfo(plant, raw); err != nil {
				return err
			}
			m.Plant = plant
		case num == 13 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.LandInfo: bad master_land_id")
			}
			data = data[n:]
			m.MasterLandID = int64(v)
		case num == 14:
			var err error
			data, err = consumeRepeatedInt64(num, typ, data, &m.SlaveLandIDs)
			if err != nil {
				return fmt.Errorf("plantpb.LandInfo: bad slave_land_ids")
			}
		case num == 15 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.LandInfo: bad land_size")
			}
			data = data[n:]
			m.LandSize = int64(v)
		case num == 16 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.LandInfo: bad lands_level")
			}
			data = data[n:]
			m.LandsLevel = int64(v)
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

func (m *OperationLimit) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.ID)
	b = appendInt64Varint(b, 2, m.DayTimes)
	b = appendInt64Varint(b, 3, m.DayTimesLt)
	b = appendInt64Varint(b, 4, m.DayShareID)
	b = appendInt64Varint(b, 5, m.DayExpTimes)
	b = appendInt64Varint(b, 6, m.DayExpTimesLt)
	b = appendInt64Varint(b, 7, m.DayExpShareID)
	return b
}

func unmarshalOperationLimit(m *OperationLimit, data []byte) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("plantpb.OperationLimit: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.OperationLimit: bad id")
			}
			data = data[n:]
			m.ID = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.OperationLimit: bad day_times")
			}
			data = data[n:]
			m.DayTimes = int64(v)
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.OperationLimit: bad day_times_lt")
			}
			data = data[n:]
			m.DayTimesLt = int64(v)
		case num == 4 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.OperationLimit: bad day_share_id")
			}
			data = data[n:]
			m.DayShareID = int64(v)
		case num == 5 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.OperationLimit: bad day_exp_times")
			}
			data = data[n:]
			m.DayExpTimes = int64(v)
		case num == 6 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.OperationLimit: bad day_ex_times_lt")
			}
			data = data[n:]
			m.DayExpTimesLt = int64(v)
		case num == 7 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("plantpb.OperationLimit: bad day_exp_share_id")
			}
			data = data[n:]
			m.DayExpShareID = int64(v)
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
