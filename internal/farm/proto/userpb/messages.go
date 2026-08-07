package userpb

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

func skipField(num protowire.Number, typ protowire.Type, data []byte) ([]byte, error) {
	n := protowire.ConsumeFieldValue(num, typ, data)
	if n < 0 {
		return nil, fmt.Errorf("userpb: bad field %d typ=%d", num, typ)
	}
	return data[n:], nil
}

func appendInt64Varint(b []byte, field int, v int64) []byte {
	if v == 0 {
		return b
	}
	b = protowire.AppendTag(b, protowire.Number(field), protowire.VarintType)
	b = protowire.AppendVarint(b, uint64(v))
	return b
}

func appendInt32(b []byte, field int, v int32) []byte {
	if v == 0 {
		return b
	}
	b = protowire.AppendTag(b, protowire.Number(field), protowire.VarintType)
	b = protowire.AppendVarint(b, uint64(uint32(v)))
	return b
}

func appendBool(b []byte, field int, v bool) []byte {
	if !v {
		return b
	}
	b = protowire.AppendTag(b, protowire.Number(field), protowire.VarintType)
	b = protowire.AppendVarint(b, 1)
	return b
}

func appendString(b []byte, field int, v string) []byte {
	if v == "" {
		return b
	}
	b = protowire.AppendTag(b, protowire.Number(field), protowire.BytesType)
	b = protowire.AppendString(b, v)
	return b
}

func appendMessage(b []byte, field int, raw []byte) []byte {
	if len(raw) == 0 {
		return b
	}
	b = protowire.AppendTag(b, protowire.Number(field), protowire.BytesType)
	b = protowire.AppendBytes(b, raw)
	return b
}

func appendPackedInt64s(b []byte, field int, vals []int64) []byte {
	if len(vals) == 0 {
		return b
	}
	var packed []byte
	for _, v := range vals {
		packed = protowire.AppendVarint(packed, uint64(v))
	}
	b = protowire.AppendTag(b, protowire.Number(field), protowire.BytesType)
	b = protowire.AppendBytes(b, packed)
	return b
}

func consumeRepeatedInt64(num protowire.Number, typ protowire.Type, data []byte, dst *[]int64) ([]byte, error) {
	switch typ {
	case protowire.BytesType:
		raw, n := protowire.ConsumeBytes(data)
		if n < 0 {
			return nil, fmt.Errorf("userpb: bad packed int64 field %d", num)
		}
		for len(raw) > 0 {
			v, m := protowire.ConsumeVarint(raw)
			if m < 0 {
				return nil, fmt.Errorf("userpb: bad packed int64 field %d", num)
			}
			raw = raw[m:]
			*dst = append(*dst, int64(v))
		}
		return data[n:], nil
	case protowire.VarintType:
		v, n := protowire.ConsumeVarint(data)
		if n < 0 {
			return nil, fmt.Errorf("userpb: bad int64 field %d", num)
		}
		*dst = append(*dst, int64(v))
		return data[n:], nil
	default:
		return skipField(num, typ, data)
	}
}

func marshalQQGroupInfo(m *QQGroupInfo) []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendString(b, 1, m.QQGroupID)
	b = appendString(b, 2, m.QQGroupName)
	return b
}

func unmarshalQQGroupInfo(m *QQGroupInfo, data []byte) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("userpb.QQGroupInfo: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("userpb.QQGroupInfo: bad qq_group_id")
			}
			data = data[n:]
			m.QQGroupID = string(raw)
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("userpb.QQGroupInfo: bad qq_group_name")
			}
			data = data[n:]
			m.QQGroupName = string(raw)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("userpb.QQGroupInfo: %w", err)
			}
		}
	}
	return nil
}

func marshalVersionInfo(m *VersionInfo) []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt32(b, 1, m.Status)
	b = appendString(b, 2, m.VersionRecommend)
	b = appendString(b, 3, m.VersionForce)
	b = appendString(b, 4, m.ResVersion)
	return b
}

func unmarshalVersionInfo(m *VersionInfo, data []byte) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("userpb.VersionInfo: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("userpb.VersionInfo: bad status")
			}
			data = data[n:]
			m.Status = int32(v)
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("userpb.VersionInfo: bad version_recommend")
			}
			data = data[n:]
			m.VersionRecommend = string(raw)
		case num == 3 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("userpb.VersionInfo: bad version_force")
			}
			data = data[n:]
			m.VersionForce = string(raw)
		case num == 4 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("userpb.VersionInfo: bad res_version")
			}
			data = data[n:]
			m.ResVersion = string(raw)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("userpb.VersionInfo: %w", err)
			}
		}
	}
	return nil
}

func (m *HeartbeatReply) Unmarshal(data []byte) error {
	*m = HeartbeatReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("userpb.HeartbeatReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("userpb.HeartbeatReply: bad server_time")
			}
			data = data[n:]
			m.ServerTime = int64(v)
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("userpb.HeartbeatReply: bad version_info")
			}
			data = data[n:]
			vi := &VersionInfo{}
			if err := unmarshalVersionInfo(vi, raw); err != nil {
				return err
			}
			m.VersionInfo = vi
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("userpb.HeartbeatReply: %w", err)
			}
		}
	}
	return nil
}

func (m *ReportArkClickRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.SharerID)
	b = appendString(b, 2, m.SharerOpenID)
	b = appendInt64Varint(b, 3, m.ShareCfgID)
	b = appendString(b, 4, m.SceneID)
	return b
}

func (m *ReportArkClickRequest) Unmarshal(data []byte) error {
	*m = ReportArkClickRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("userpb.ReportArkClickRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("userpb.ReportArkClickRequest: bad sharer_id")
			}
			data = data[n:]
			m.SharerID = int64(v)
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("userpb.ReportArkClickRequest: bad sharer_open_id")
			}
			data = data[n:]
			m.SharerOpenID = string(raw)
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("userpb.ReportArkClickRequest: bad share_cfg_id")
			}
			data = data[n:]
			m.ShareCfgID = int64(v)
		case num == 4 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("userpb.ReportArkClickRequest: bad scene_id")
			}
			data = data[n:]
			m.SceneID = string(raw)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("userpb.ReportArkClickRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *ReportArkClickReply) Marshal() []byte { return []byte{} }

func (m *ReportArkClickReply) Unmarshal(data []byte) error {
	*m = ReportArkClickReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("userpb.ReportArkClickReply: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("userpb.ReportArkClickReply: %w", err)
		}
	}
	return nil
}

func marshalClientFlowItem(m *ClientFlowItem) []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendString(b, 1, m.EventName)
	b = appendInt64Varint(b, 2, m.Timestamp)
	b = appendString(b, 3, m.Params)
	return b
}

func unmarshalClientFlowItem(m *ClientFlowItem, data []byte) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("userpb.ClientFlowItem: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("userpb.ClientFlowItem: bad event_name")
			}
			data = data[n:]
			m.EventName = string(raw)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("userpb.ClientFlowItem: bad timestamp")
			}
			data = data[n:]
			m.Timestamp = int64(v)
		case num == 3 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("userpb.ClientFlowItem: bad params")
			}
			data = data[n:]
			m.Params = string(raw)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("userpb.ClientFlowItem: %w", err)
			}
		}
	}
	return nil
}

func (m *BatchClientReportFlowRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	for _, item := range m.Items {
		b = appendMessage(b, 1, marshalClientFlowItem(item))
	}
	return b
}

func (m *BatchClientReportFlowRequest) Unmarshal(data []byte) error {
	*m = BatchClientReportFlowRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("userpb.BatchClientReportFlowRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("userpb.BatchClientReportFlowRequest: bad item")
			}
			data = data[n:]
			item := &ClientFlowItem{}
			if err := unmarshalClientFlowItem(item, raw); err != nil {
				return err
			}
			m.Items = append(m.Items, item)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("userpb.BatchClientReportFlowRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *BatchClientReportFlowReply) Marshal() []byte { return []byte{} }

func (m *BatchClientReportFlowReply) Unmarshal(data []byte) error {
	*m = BatchClientReportFlowReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("userpb.BatchClientReportFlowReply: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("userpb.BatchClientReportFlowReply: %w", err)
		}
	}
	return nil
}

func (m *SetDisplayInfoRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendString(b, 1, m.Name)
	b = appendString(b, 2, m.Signature)
	b = appendInt32(b, 3, m.Gender)
	b = appendString(b, 4, m.AvatarURL)
	return b
}

func (m *SetDisplayInfoRequest) Unmarshal(data []byte) error {
	*m = SetDisplayInfoRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("userpb.SetDisplayInfoRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("userpb.SetDisplayInfoRequest: bad name")
			}
			data = data[n:]
			m.Name = string(raw)
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("userpb.SetDisplayInfoRequest: bad signature")
			}
			data = data[n:]
			m.Signature = string(raw)
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("userpb.SetDisplayInfoRequest: bad gender")
			}
			data = data[n:]
			m.Gender = int32(v)
		case num == 4 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("userpb.SetDisplayInfoRequest: bad avatar_url")
			}
			data = data[n:]
			m.AvatarURL = string(raw)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("userpb.SetDisplayInfoRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *SetDisplayInfoReply) Unmarshal(data []byte) error {
	*m = SetDisplayInfoReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("userpb.SetDisplayInfoReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("userpb.SetDisplayInfoReply: bad basic")
			}
			data = data[n:]
			basic := &BasicInfo{}
			if err := unmarshalBasicInfo(basic, raw); err != nil {
				return err
			}
			m.Basic = basic
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("userpb.SetDisplayInfoReply: %w", err)
			}
		}
	}
	return nil
}

func (m *SetQQFriendRecommendAuthorizedRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	return appendInt64Varint(nil, 1, m.Authorized)
}

func (m *SetQQFriendRecommendAuthorizedRequest) Unmarshal(data []byte) error {
	*m = SetQQFriendRecommendAuthorizedRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("userpb.SetQQFriendRecommendAuthorizedRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("userpb.SetQQFriendRecommendAuthorizedRequest: bad authorized")
			}
			data = data[n:]
			m.Authorized = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("userpb.SetQQFriendRecommendAuthorizedRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *SetQQFriendRecommendAuthorizedReply) Unmarshal(data []byte) error {
	*m = SetQQFriendRecommendAuthorizedReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("userpb.SetQQFriendRecommendAuthorizedReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("userpb.SetQQFriendRecommendAuthorizedReply: bad authorized")
			}
			data = data[n:]
			m.Authorized = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("userpb.SetQQFriendRecommendAuthorizedReply: %w", err)
			}
		}
	}
	return nil
}

func (m *GetUserSettingsRequest) Marshal() []byte { return []byte{} }

func (m *GetUserSettingsRequest) Unmarshal(data []byte) error {
	*m = GetUserSettingsRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("userpb.GetUserSettingsRequest: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("userpb.GetUserSettingsRequest: %w", err)
		}
	}
	return nil
}

func marshalUserSettings(m *UserSettings) []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.QQFriendRecommendAuthorized)
	b = appendBool(b, 2, m.DisableNudge)
	return b
}

func unmarshalUserSettings(m *UserSettings, data []byte) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("userpb.UserSettings: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("userpb.UserSettings: bad qq_friend_recommend_authorized")
			}
			data = data[n:]
			m.QQFriendRecommendAuthorized = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("userpb.UserSettings: bad disable_nudge")
			}
			data = data[n:]
			m.DisableNudge = protowire.DecodeBool(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("userpb.UserSettings: %w", err)
			}
		}
	}
	return nil
}

func (m *GetUserSettingsReply) Unmarshal(data []byte) error {
	*m = GetUserSettingsReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("userpb.GetUserSettingsReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("userpb.GetUserSettingsReply: bad settings")
			}
			data = data[n:]
			settings := &UserSettings{}
			if err := unmarshalUserSettings(settings, raw); err != nil {
				return err
			}
			m.Settings = settings
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("userpb.GetUserSettingsReply: %w", err)
			}
		}
	}
	return nil
}

func (m *BatchGetBasicInfoRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	return appendPackedInt64s(nil, 1, m.GIDs)
}

func (m *BatchGetBasicInfoRequest) Unmarshal(data []byte) error {
	*m = BatchGetBasicInfoRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("userpb.BatchGetBasicInfoRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1:
			var err error
			data, err = consumeRepeatedInt64(num, typ, data, &m.GIDs)
			if err != nil {
				return fmt.Errorf("userpb.BatchGetBasicInfoRequest: %w", err)
			}
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("userpb.BatchGetBasicInfoRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *BatchGetBasicInfoReply) Unmarshal(data []byte) error {
	*m = BatchGetBasicInfoReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("userpb.BatchGetBasicInfoReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("userpb.BatchGetBasicInfoReply: bad user")
			}
			data = data[n:]
			basic := &BasicInfo{}
			if err := unmarshalBasicInfo(basic, raw); err != nil {
				return err
			}
			m.Users = append(m.Users, basic)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("userpb.BatchGetBasicInfoReply: %w", err)
			}
		}
	}
	return nil
}
