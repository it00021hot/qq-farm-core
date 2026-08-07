package solartermspb

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

func (m *GetSolarTermsRequest) Marshal() []byte { return []byte{} }

func (m *GetSolarTermsReply) Unmarshal(data []byte) error {
	*m = GetSolarTermsReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("solartermspb.GetSolarTermsReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("solartermspb.GetSolarTermsReply: bad term")
			}
			data = data[n:]
			term := SolarTermInfo{}
			if err := unmarshalSolarTermInfo(&term, raw); err != nil {
				return err
			}
			m.Terms = append(m.Terms, term)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("solartermspb.GetSolarTermsReply: bad server_time")
			}
			data = data[n:]
			m.ServerTime = int64(v)
		case num == 3 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("solartermspb.GetSolarTermsReply: bad current_config")
			}
			data = data[n:]
			cfg := &SolarTermsConfig{}
			if err := unmarshalSolarTermsConfig(cfg, raw); err != nil {
				return err
			}
			m.CurrentConfig = cfg
		case num == 4 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("solartermspb.GetSolarTermsReply: bad config")
			}
			data = data[n:]
			cfg := SolarTermsConfig{}
			if err := unmarshalSolarTermsConfig(&cfg, raw); err != nil {
				return err
			}
			m.Configs = append(m.Configs, cfg)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("solartermspb.GetSolarTermsReply: %w", err)
			}
		}
	}
	return nil
}

func (m *ClaimSolarTermsRequest) Marshal() []byte {
	return appendInt64Varint(nil, 1, m.TermID)
}

func (m *ClaimSolarTermsReply) Unmarshal(data []byte) error {
	*m = ClaimSolarTermsReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("solartermspb.ClaimSolarTermsReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("solartermspb.ClaimSolarTermsReply: bad reward")
			}
			data = data[n:]
			r := SolarTermReward{}
			if err := unmarshalSolarTermReward(&r, raw); err != nil {
				return err
			}
			m.Rewards = append(m.Rewards, r)
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("solartermspb.ClaimSolarTermsReply: bad term")
			}
			data = data[n:]
			term := &SolarTermInfo{}
			if err := unmarshalSolarTermInfo(term, raw); err != nil {
				return err
			}
			m.Term = term
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("solartermspb.ClaimSolarTermsReply: %w", err)
			}
		}
	}
	return nil
}

func unmarshalSolarTermInfo(m *SolarTermInfo, data []byte) error {
	*m = SolarTermInfo{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("solartermspb.SolarTermInfo: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("solartermspb.SolarTermInfo: bad term_id")
			}
			data = data[n:]
			m.TermID = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("solartermspb.SolarTermInfo: bad status")
			}
			data = data[n:]
			m.Status = int64(v)
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("solartermspb.SolarTermInfo: bad begin_time")
			}
			data = data[n:]
			m.BeginTime = int64(v)
		case num == 4 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("solartermspb.SolarTermInfo: bad end_time")
			}
			data = data[n:]
			m.EndTime = int64(v)
		case num == 5 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("solartermspb.SolarTermInfo: bad reward")
			}
			data = data[n:]
			r := SolarTermReward{}
			if err := unmarshalSolarTermReward(&r, raw); err != nil {
				return err
			}
			m.Rewards = append(m.Rewards, r)
		case num == 6 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("solartermspb.SolarTermInfo: bad name")
			}
			data = data[n:]
			m.Name = append([]byte(nil), raw...)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("solartermspb.SolarTermInfo: %w", err)
			}
		}
	}
	return nil
}

func unmarshalSolarTermReward(m *SolarTermReward, data []byte) error {
	*m = SolarTermReward{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("solartermspb.SolarTermReward: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("solartermspb.SolarTermReward: bad item_id")
			}
			data = data[n:]
			m.ItemID = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("solartermspb.SolarTermReward: bad count")
			}
			data = data[n:]
			m.Count = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("solartermspb.SolarTermReward: %w", err)
			}
		}
	}
	return nil
}

func unmarshalSolarTermsConfig(m *SolarTermsConfig, data []byte) error {
	*m = SolarTermsConfig{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("solartermspb.SolarTermsConfig: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("solartermspb.SolarTermsConfig: bad config_id")
			}
			data = data[n:]
			m.ConfigID = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("solartermspb.SolarTermsConfig: bad activity_id")
			}
			data = data[n:]
			m.ActivityID = int64(v)
		case num == 3 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("solartermspb.SolarTermsConfig: bad rules_json")
			}
			data = data[n:]
			m.RulesJSON = append([]byte(nil), raw...)
		case num == 4 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("solartermspb.SolarTermsConfig: bad field_4")
			}
			data = data[n:]
			m.Field4 = append([]byte(nil), raw...)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("solartermspb.SolarTermsConfig: %w", err)
			}
		}
	}
	return nil
}
