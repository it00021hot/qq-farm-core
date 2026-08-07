package interactpb

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

func marshalInteractRecordExtra(m *InteractRecordExtra) []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt32(b, 1, m.LandID)
	b = appendInt32(b, 2, m.Flag1)
	b = appendInt32(b, 3, m.Flag2)
	return b
}

func unmarshalInteractRecordExtra(m *InteractRecordExtra, data []byte) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("interactpb.InteractRecordExtra: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("interactpb.InteractRecordExtra: bad land_id")
			}
			data = data[n:]
			m.LandID = int32(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("interactpb.InteractRecordExtra: bad flag1")
			}
			data = data[n:]
			m.Flag1 = int32(v)
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("interactpb.InteractRecordExtra: bad flag2")
			}
			data = data[n:]
			m.Flag2 = int32(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("interactpb.InteractRecordExtra: %w", err)
			}
		}
	}
	return nil
}

func (m *InteractRecord) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.ServerTime)
	b = appendInt32(b, 2, m.ActionType)
	b = appendInt64Varint(b, 3, m.VisitorGID)
	b = appendString(b, 4, m.Nick)
	b = appendString(b, 5, m.AvatarURL)
	b = appendInt32(b, 6, m.CropID)
	b = appendInt32(b, 7, m.CropCount)
	b = appendInt32(b, 8, m.Times)
	b = appendInt32(b, 9, m.FromType)
	b = appendInt32(b, 10, m.Level)
	if m.Extra != nil {
		b = appendMessage(b, 11, marshalInteractRecordExtra(m.Extra))
	}
	return b
}

func (m *InteractRecord) Unmarshal(data []byte) error {
	*m = InteractRecord{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("interactpb.InteractRecord: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("interactpb.InteractRecord: bad server_time")
			}
			data = data[n:]
			m.ServerTime = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("interactpb.InteractRecord: bad action_type")
			}
			data = data[n:]
			m.ActionType = int32(v)
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("interactpb.InteractRecord: bad visitor_gid")
			}
			data = data[n:]
			m.VisitorGID = int64(v)
		case num == 4 && typ == protowire.BytesType:
			v, n := protowire.ConsumeString(data)
			if n < 0 {
				return fmt.Errorf("interactpb.InteractRecord: bad nick")
			}
			data = data[n:]
			m.Nick = v
		case num == 5 && typ == protowire.BytesType:
			v, n := protowire.ConsumeString(data)
			if n < 0 {
				return fmt.Errorf("interactpb.InteractRecord: bad avatar_url")
			}
			data = data[n:]
			m.AvatarURL = v
		case num == 6 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("interactpb.InteractRecord: bad crop_id")
			}
			data = data[n:]
			m.CropID = int32(v)
		case num == 7 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("interactpb.InteractRecord: bad crop_count")
			}
			data = data[n:]
			m.CropCount = int32(v)
		case num == 8 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("interactpb.InteractRecord: bad times")
			}
			data = data[n:]
			m.Times = int32(v)
		case num == 9 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("interactpb.InteractRecord: bad from_type")
			}
			data = data[n:]
			m.FromType = int32(v)
		case num == 10 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("interactpb.InteractRecord: bad level")
			}
			data = data[n:]
			m.Level = int32(v)
		case num == 11 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("interactpb.InteractRecord: bad extra")
			}
			data = data[n:]
			extra := &InteractRecordExtra{}
			if err := unmarshalInteractRecordExtra(extra, raw); err != nil {
				return err
			}
			m.Extra = extra
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("interactpb.InteractRecord: %w", err)
			}
		}
	}
	return nil
}

func (m *InteractRecordsRequest) Marshal() []byte { return []byte{} }

func (m *InteractRecordsRequest) Unmarshal(data []byte) error {
	*m = InteractRecordsRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("interactpb.InteractRecordsRequest: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("interactpb.InteractRecordsRequest: %w", err)
		}
	}
	return nil
}

func (m *InteractRecordsReply) Unmarshal(data []byte) error {
	*m = InteractRecordsReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("interactpb.InteractRecordsReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("interactpb.InteractRecordsReply: bad record")
			}
			data = data[n:]
			rec := &InteractRecord{}
			if err := rec.Unmarshal(raw); err != nil {
				return err
			}
			m.Records = append(m.Records, rec)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("interactpb.InteractRecordsReply: %w", err)
			}
		}
	}
	return nil
}

func (m *InteractInfo) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.VisitorGID)
	b = appendString(b, 2, m.Nick)
	b = appendString(b, 3, m.AvatarURL)
	b = appendInt32(b, 4, m.ActionType)
	b = appendInt64Varint(b, 5, m.ServerTime)
	b = appendInt32(b, 6, m.CropID)
	b = appendInt32(b, 7, m.CropCount)
	b = appendInt32(b, 8, m.Level)
	return b
}

func (m *InteractInfo) Unmarshal(data []byte) error {
	*m = InteractInfo{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("interactpb.InteractInfo: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("interactpb.InteractInfo: bad visitor_gid")
			}
			data = data[n:]
			m.VisitorGID = int64(v)
		case num == 2 && typ == protowire.BytesType:
			v, n := protowire.ConsumeString(data)
			if n < 0 {
				return fmt.Errorf("interactpb.InteractInfo: bad nick")
			}
			data = data[n:]
			m.Nick = v
		case num == 3 && typ == protowire.BytesType:
			v, n := protowire.ConsumeString(data)
			if n < 0 {
				return fmt.Errorf("interactpb.InteractInfo: bad avatar_url")
			}
			data = data[n:]
			m.AvatarURL = v
		case num == 4 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("interactpb.InteractInfo: bad action_type")
			}
			data = data[n:]
			m.ActionType = int32(v)
		case num == 5 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("interactpb.InteractInfo: bad server_time")
			}
			data = data[n:]
			m.ServerTime = int64(v)
		case num == 6 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("interactpb.InteractInfo: bad crop_id")
			}
			data = data[n:]
			m.CropID = int32(v)
		case num == 7 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("interactpb.InteractInfo: bad crop_count")
			}
			data = data[n:]
			m.CropCount = int32(v)
		case num == 8 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("interactpb.InteractInfo: bad level")
			}
			data = data[n:]
			m.Level = int32(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("interactpb.InteractInfo: %w", err)
			}
		}
	}
	return nil
}

func (m *GetInteractInfoRequest) Marshal() []byte { return []byte{} }

func (m *GetInteractInfoRequest) Unmarshal(data []byte) error {
	*m = GetInteractInfoRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("interactpb.GetInteractInfoRequest: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("interactpb.GetInteractInfoRequest: %w", err)
		}
	}
	return nil
}

func (m *GetInteractInfoReply) Unmarshal(data []byte) error {
	*m = GetInteractInfoReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("interactpb.GetInteractInfoReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("interactpb.GetInteractInfoReply: bad info")
			}
			data = data[n:]
			info := &InteractInfo{}
			if err := info.Unmarshal(raw); err != nil {
				return err
			}
			m.Infos = append(m.Infos, info)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("interactpb.GetInteractInfoReply: %w", err)
			}
		}
	}
	return nil
}

func (m *InteractSummary) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.TotalWater)
	b = appendInt64Varint(b, 2, m.TotalInsecticide)
	b = appendInt64Varint(b, 3, m.TotalWeed)
	b = appendInt64Varint(b, 4, m.TotalSteal)
	b = appendInt64Varint(b, 5, m.TotalStolen)
	b = appendInt64Varint(b, 6, m.VisitorCount)
	return b
}

func (m *InteractSummary) Unmarshal(data []byte) error {
	*m = InteractSummary{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("interactpb.InteractSummary: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("interactpb.InteractSummary: bad total_water")
			}
			data = data[n:]
			m.TotalWater = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("interactpb.InteractSummary: bad total_insecticide")
			}
			data = data[n:]
			m.TotalInsecticide = int64(v)
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("interactpb.InteractSummary: bad total_weed")
			}
			data = data[n:]
			m.TotalWeed = int64(v)
		case num == 4 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("interactpb.InteractSummary: bad total_steal")
			}
			data = data[n:]
			m.TotalSteal = int64(v)
		case num == 5 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("interactpb.InteractSummary: bad total_stolen")
			}
			data = data[n:]
			m.TotalStolen = int64(v)
		case num == 6 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("interactpb.InteractSummary: bad visitor_count")
			}
			data = data[n:]
			m.VisitorCount = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("interactpb.InteractSummary: %w", err)
			}
		}
	}
	return nil
}

func (m *GetInteractSummaryRequest) Marshal() []byte { return []byte{} }

func (m *GetInteractSummaryRequest) Unmarshal(data []byte) error {
	*m = GetInteractSummaryRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("interactpb.GetInteractSummaryRequest: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("interactpb.GetInteractSummaryRequest: %w", err)
		}
	}
	return nil
}

func (m *GetInteractSummaryReply) Unmarshal(data []byte) error {
	*m = GetInteractSummaryReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("interactpb.GetInteractSummaryReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("interactpb.GetInteractSummaryReply: bad summary")
			}
			data = data[n:]
			s := &InteractSummary{}
			if err := s.Unmarshal(raw); err != nil {
				return err
			}
			m.Summary = s
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("interactpb.GetInteractSummaryReply: %w", err)
			}
		}
	}
	return nil
}

func (m *DismissInteractPopupRequest) Marshal() []byte { return []byte{} }

func (m *DismissInteractPopupRequest) Unmarshal(data []byte) error {
	*m = DismissInteractPopupRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("interactpb.DismissInteractPopupRequest: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("interactpb.DismissInteractPopupRequest: %w", err)
		}
	}
	return nil
}

func (m *DismissInteractPopupReply) Unmarshal(data []byte) error {
	*m = DismissInteractPopupReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("interactpb.DismissInteractPopupReply: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("interactpb.DismissInteractPopupReply: %w", err)
		}
	}
	return nil
}
