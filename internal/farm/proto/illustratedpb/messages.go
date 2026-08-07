package illustratedpb

import (
	"fmt"

	"github.com/MQEnergy/go-skeleton/internal/farm/proto/corepb"
	"google.golang.org/protobuf/encoding/protowire"
)

func (m *GetIllustratedListV2Request) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendBool(b, 1, m.Refresh)
	b = appendBool(b, 2, m.Full)
	return b
}

func (m *GetIllustratedListV2Request) Unmarshal(data []byte) error {
	*m = GetIllustratedListV2Request{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("illustratedpb.GetIllustratedListV2Request: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("illustratedpb.GetIllustratedListV2Request: bad refresh")
			}
			data = data[n:]
			m.Refresh = protowire.DecodeBool(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("illustratedpb.GetIllustratedListV2Request: bad full")
			}
			data = data[n:]
			m.Full = protowire.DecodeBool(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("illustratedpb.GetIllustratedListV2Request: %w", err)
			}
		}
	}
	return nil
}

func (m *IllustratedItem) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.SeedID)
	b = appendBool(b, 2, m.Unlocked)
	b = appendBool(b, 3, m.Planted)
	b = appendInt32(b, 4, m.PlantedCount)
	b = appendInt32(b, 5, m.HarvestCount)
	b = appendBytes(b, 6, m.RewardDetail)
	b = appendBool(b, 7, m.HasReward)
	return b
}

func (m *IllustratedItem) Unmarshal(data []byte) error {
	*m = IllustratedItem{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("illustratedpb.IllustratedItem: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("illustratedpb.IllustratedItem: bad seed_id")
			}
			data = data[n:]
			m.SeedID = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("illustratedpb.IllustratedItem: bad unlocked")
			}
			data = data[n:]
			m.Unlocked = protowire.DecodeBool(v)
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("illustratedpb.IllustratedItem: bad planted")
			}
			data = data[n:]
			m.Planted = protowire.DecodeBool(v)
		case num == 4 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("illustratedpb.IllustratedItem: bad planted_count")
			}
			data = data[n:]
			m.PlantedCount = int32(v)
		case num == 5 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("illustratedpb.IllustratedItem: bad harvest_count")
			}
			data = data[n:]
			m.HarvestCount = int32(v)
		case num == 6 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("illustratedpb.IllustratedItem: bad reward_detail")
			}
			data = data[n:]
			m.RewardDetail = append([]byte(nil), raw...)
		case num == 7 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("illustratedpb.IllustratedItem: bad has_reward")
			}
			data = data[n:]
			m.HasReward = protowire.DecodeBool(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("illustratedpb.IllustratedItem: %w", err)
			}
		}
	}
	return nil
}

func (m *GetIllustratedListV2Reply) Unmarshal(data []byte) error {
	*m = GetIllustratedListV2Reply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("illustratedpb.GetIllustratedListV2Reply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("illustratedpb.GetIllustratedListV2Reply: bad item")
			}
			data = data[n:]
			item := IllustratedItem{}
			if err := item.Unmarshal(raw); err != nil {
				return err
			}
			m.Items = append(m.Items, item)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("illustratedpb.GetIllustratedListV2Reply: %w", err)
			}
		}
	}
	return nil
}

func (m *ClaimAllRewardsV2Request) Marshal() []byte {
	if m == nil {
		return nil
	}
	return appendBool(nil, 1, m.OnlyClaimable)
}

func (m *ClaimAllRewardsV2Request) Unmarshal(data []byte) error {
	*m = ClaimAllRewardsV2Request{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("illustratedpb.ClaimAllRewardsV2Request: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("illustratedpb.ClaimAllRewardsV2Request: bad only_claimable")
			}
			data = data[n:]
			m.OnlyClaimable = protowire.DecodeBool(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("illustratedpb.ClaimAllRewardsV2Request: %w", err)
			}
		}
	}
	return nil
}

func (m *ClaimAllRewardsV2Reply) Unmarshal(data []byte) error {
	*m = ClaimAllRewardsV2Reply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("illustratedpb.ClaimAllRewardsV2Reply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("illustratedpb.ClaimAllRewardsV2Reply: bad item")
			}
			data = data[n:]
			item := &corepb.Item{}
			if err := item.Unmarshal(raw); err != nil {
				return err
			}
			m.Items = append(m.Items, item)
		case num == 4 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("illustratedpb.ClaimAllRewardsV2Reply: bad bonus_item")
			}
			data = data[n:]
			item := &corepb.Item{}
			if err := item.Unmarshal(raw); err != nil {
				return err
			}
			m.BonusItems = append(m.BonusItems, item)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("illustratedpb.ClaimAllRewardsV2Reply: %w", err)
			}
		}
	}
	return nil
}

func (m *IllustratedRewardRedDotNotifyV2) Unmarshal(data []byte) error {
	*m = IllustratedRewardRedDotNotifyV2{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("illustratedpb.IllustratedRewardRedDotNotifyV2: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("illustratedpb.IllustratedRewardRedDotNotifyV2: %w", err)
		}
	}
	return nil
}

func (m *ClearNewUnlockedFruitsV2Request) Marshal() []byte {
	if m == nil {
		return nil
	}
	return appendInt64Varint(nil, 1, m.SeedID)
}

func (m *ClearNewUnlockedFruitsV2Request) Unmarshal(data []byte) error {
	*m = ClearNewUnlockedFruitsV2Request{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("illustratedpb.ClearNewUnlockedFruitsV2Request: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("illustratedpb.ClearNewUnlockedFruitsV2Request: bad seed_id")
			}
			data = data[n:]
			m.SeedID = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("illustratedpb.ClearNewUnlockedFruitsV2Request: %w", err)
			}
		}
	}
	return nil
}

func (m *ClearNewUnlockedFruitsV2Reply) Marshal() []byte { return []byte{} }

func (m *ClearNewUnlockedFruitsV2Reply) Unmarshal(data []byte) error {
	*m = ClearNewUnlockedFruitsV2Reply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("illustratedpb.ClearNewUnlockedFruitsV2Reply: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("illustratedpb.ClearNewUnlockedFruitsV2Reply: %w", err)
		}
	}
	return nil
}
