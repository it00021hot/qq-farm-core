package friendpb

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

func marshalPlant(m *Plant) []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.DryTimeSec)
	b = appendInt64Varint(b, 2, m.WeedTimeSec)
	b = appendInt64Varint(b, 3, m.InsectTimeSec)
	b = appendInt64Varint(b, 4, m.RipeTimeSec)
	b = appendInt64Varint(b, 5, m.RipeFruitID)
	b = appendInt64Varint(b, 6, m.StealPlantNum)
	b = appendInt64Varint(b, 7, m.DryNum)
	b = appendInt64Varint(b, 8, m.WeedNum)
	b = appendInt64Varint(b, 9, m.InsectNum)
	return b
}

func unmarshalPlant(m *Plant, data []byte) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("friendpb.Plant: bad tag")
		}
		data = data[n:]
		switch {
		case num >= 1 && num <= 9 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("friendpb.Plant: bad field %d", num)
			}
			data = data[n:]
			switch num {
			case 1:
				m.DryTimeSec = int64(v)
			case 2:
				m.WeedTimeSec = int64(v)
			case 3:
				m.InsectTimeSec = int64(v)
			case 4:
				m.RipeTimeSec = int64(v)
			case 5:
				m.RipeFruitID = int64(v)
			case 6:
				m.StealPlantNum = int64(v)
			case 7:
				m.DryNum = int64(v)
			case 8:
				m.WeedNum = int64(v)
			case 9:
				m.InsectNum = int64(v)
			}
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("friendpb.Plant: %w", err)
			}
		}
	}
	return nil
}

func marshalTags(m *Tags) []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendBool(b, 1, m.IsNew)
	b = appendBool(b, 2, m.IsFollow)
	return b
}

func unmarshalTags(m *Tags, data []byte) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("friendpb.Tags: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("friendpb.Tags: bad is_new")
			}
			data = data[n:]
			m.IsNew = protowire.DecodeBool(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("friendpb.Tags: bad is_follow")
			}
			data = data[n:]
			m.IsFollow = protowire.DecodeBool(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("friendpb.Tags: %w", err)
			}
		}
	}
	return nil
}

func (m *GameFriend) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.GID)
	b = appendString(b, 2, m.OpenID)
	b = appendString(b, 3, m.Name)
	b = appendString(b, 4, m.AvatarURL)
	b = appendString(b, 5, m.Remark)
	b = appendInt64Varint(b, 6, m.Level)
	b = appendInt64Varint(b, 7, m.Gold)
	if m.Tags != nil {
		b = appendMessage(b, 8, marshalTags(m.Tags))
	}
	if m.Plant != nil {
		b = appendMessage(b, 9, marshalPlant(m.Plant))
	}
	b = appendInt32(b, 10, m.AuthorizedStatus)
	return b
}

func (m *GameFriend) Unmarshal(data []byte) error {
	*m = GameFriend{}
	return unmarshalGameFriend(m, data)
}

func unmarshalGameFriend(m *GameFriend, data []byte) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("friendpb.GameFriend: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("friendpb.GameFriend: bad gid")
			}
			data = data[n:]
			m.GID = int64(v)
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("friendpb.GameFriend: bad open_id")
			}
			data = data[n:]
			m.OpenID = string(raw)
		case num == 3 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("friendpb.GameFriend: bad name")
			}
			data = data[n:]
			m.Name = string(raw)
		case num == 4 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("friendpb.GameFriend: bad avatar_url")
			}
			data = data[n:]
			m.AvatarURL = string(raw)
		case num == 5 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("friendpb.GameFriend: bad remark")
			}
			data = data[n:]
			m.Remark = string(raw)
		case num == 6 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("friendpb.GameFriend: bad level")
			}
			data = data[n:]
			m.Level = int64(v)
		case num == 7 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("friendpb.GameFriend: bad gold")
			}
			data = data[n:]
			m.Gold = int64(v)
		case num == 8 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("friendpb.GameFriend: bad tags")
			}
			data = data[n:]
			tags := &Tags{}
			if err := unmarshalTags(tags, raw); err != nil {
				return err
			}
			m.Tags = tags
		case num == 9 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("friendpb.GameFriend: bad plant")
			}
			data = data[n:]
			plant := &Plant{}
			if err := unmarshalPlant(plant, raw); err != nil {
				return err
			}
			m.Plant = plant
		case num == 10 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("friendpb.GameFriend: bad authorized_status")
			}
			data = data[n:]
			m.AuthorizedStatus = int32(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("friendpb.GameFriend: %w", err)
			}
		}
	}
	return nil
}

func (m *GetAllRequest) Marshal() []byte {
	return []byte{}
}

func (m *GetAllRequest) Unmarshal(data []byte) error {
	*m = GetAllRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("friendpb.GetAllRequest: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("friendpb.GetAllRequest: %w", err)
		}
	}
	return nil
}

func (m *GetAllReply) Unmarshal(data []byte) error {
	*m = GetAllReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("friendpb.GetAllReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("friendpb.GetAllReply: bad game_friend")
			}
			data = data[n:]
			friend := GameFriend{}
			if err := unmarshalGameFriend(&friend, raw); err != nil {
				return err
			}
			m.GameFriends = append(m.GameFriends, friend)
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("friendpb.GetAllReply: bad application_count")
			}
			data = data[n:]
			m.ApplicationCount = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("friendpb.GetAllReply: %w", err)
			}
		}
	}
	return nil
}

func (m *GetGameFriendsRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	return appendPackedInt64s(nil, 1, m.GIDs)
}

func (m *GetGameFriendsRequest) Unmarshal(data []byte) error {
	*m = GetGameFriendsRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("friendpb.GetGameFriendsRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1:
			var err error
			data, err = consumeRepeatedInt64(num, typ, data, &m.GIDs)
			if err != nil {
				return fmt.Errorf("friendpb.GetGameFriendsRequest: %w", err)
			}
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("friendpb.GetGameFriendsRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *GetGameFriendsReply) Unmarshal(data []byte) error {
	*m = GetGameFriendsReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("friendpb.GetGameFriendsReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("friendpb.GetGameFriendsReply: bad game_friend")
			}
			data = data[n:]
			friend := GameFriend{}
			if err := unmarshalGameFriend(&friend, raw); err != nil {
				return err
			}
			m.GameFriends = append(m.GameFriends, friend)
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("friendpb.GetGameFriendsReply: bad application_count")
			}
			data = data[n:]
			m.ApplicationCount = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("friendpb.GetGameFriendsReply: %w", err)
			}
		}
	}
	return nil
}

func (m *Application) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.GID)
	b = appendInt64Varint(b, 2, m.TimeAt)
	b = appendString(b, 3, m.OpenID)
	b = appendString(b, 4, m.Name)
	b = appendString(b, 5, m.AvatarURL)
	b = appendInt64Varint(b, 6, m.Level)
	return b
}

func unmarshalApplication(m *Application, data []byte) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("friendpb.Application: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("friendpb.Application: bad gid")
			}
			data = data[n:]
			m.GID = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("friendpb.Application: bad time_at")
			}
			data = data[n:]
			m.TimeAt = int64(v)
		case num == 3 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("friendpb.Application: bad open_id")
			}
			data = data[n:]
			m.OpenID = string(raw)
		case num == 4 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("friendpb.Application: bad name")
			}
			data = data[n:]
			m.Name = string(raw)
		case num == 5 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("friendpb.Application: bad avatar_url")
			}
			data = data[n:]
			m.AvatarURL = string(raw)
		case num == 6 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("friendpb.Application: bad level")
			}
			data = data[n:]
			m.Level = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("friendpb.Application: %w", err)
			}
		}
	}
	return nil
}

func (m *GetApplicationsRequest) Marshal() []byte {
	return []byte{}
}

func (m *GetApplicationsRequest) Unmarshal(data []byte) error {
	*m = GetApplicationsRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("friendpb.GetApplicationsRequest: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("friendpb.GetApplicationsRequest: %w", err)
		}
	}
	return nil
}

func (m *GetApplicationsReply) Unmarshal(data []byte) error {
	*m = GetApplicationsReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("friendpb.GetApplicationsReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("friendpb.GetApplicationsReply: bad application")
			}
			data = data[n:]
			app := Application{}
			if err := unmarshalApplication(&app, raw); err != nil {
				return err
			}
			m.Applications = append(m.Applications, app)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("friendpb.GetApplicationsReply: bad block_applications")
			}
			data = data[n:]
			m.BlockApplications = protowire.DecodeBool(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("friendpb.GetApplicationsReply: %w", err)
			}
		}
	}
	return nil
}

func (m *AcceptFriendsRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	return appendPackedInt64s(nil, 1, m.FriendGIDs)
}

func (m *AcceptFriendsRequest) Unmarshal(data []byte) error {
	*m = AcceptFriendsRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("friendpb.AcceptFriendsRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1:
			var err error
			data, err = consumeRepeatedInt64(num, typ, data, &m.FriendGIDs)
			if err != nil {
				return fmt.Errorf("friendpb.AcceptFriendsRequest: %w", err)
			}
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("friendpb.AcceptFriendsRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *AcceptFriendsReply) Unmarshal(data []byte) error {
	*m = AcceptFriendsReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("friendpb.AcceptFriendsReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("friendpb.AcceptFriendsReply: bad friend")
			}
			data = data[n:]
			friend := GameFriend{}
			if err := unmarshalGameFriend(&friend, raw); err != nil {
				return err
			}
			m.Friends = append(m.Friends, friend)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("friendpb.AcceptFriendsReply: %w", err)
			}
		}
	}
	return nil
}

func (m *SyncAllRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	for _, id := range m.OpenIDs {
		b = appendString(b, 2, id)
	}
	return b
}

func (m *SyncAllRequest) Unmarshal(data []byte) error {
	*m = SyncAllRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("friendpb.SyncAllRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("friendpb.SyncAllRequest: bad open_id")
			}
			data = data[n:]
			m.OpenIDs = append(m.OpenIDs, string(raw))
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("friendpb.SyncAllRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *SyncAllReply) Unmarshal(data []byte) error {
	*m = SyncAllReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("friendpb.SyncAllReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("friendpb.SyncAllReply: bad game_friend")
			}
			data = data[n:]
			friend := GameFriend{}
			if err := unmarshalGameFriend(&friend, raw); err != nil {
				return err
			}
			m.GameFriends = append(m.GameFriends, friend)
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("friendpb.SyncAllReply: bad application_count")
			}
			data = data[n:]
			m.ApplicationCount = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("friendpb.SyncAllReply: %w", err)
			}
		}
	}
	return nil
}

func (m *RejectFriendsRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	return appendPackedInt64s(nil, 1, m.FriendGIDs)
}

func (m *RejectFriendsRequest) Unmarshal(data []byte) error {
	*m = RejectFriendsRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("friendpb.RejectFriendsRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1:
			var err error
			data, err = consumeRepeatedInt64(num, typ, data, &m.FriendGIDs)
			if err != nil {
				return fmt.Errorf("friendpb.RejectFriendsRequest: %w", err)
			}
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("friendpb.RejectFriendsRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *RejectFriendsReply) Marshal() []byte { return []byte{} }

func (m *RejectFriendsReply) Unmarshal(data []byte) error {
	*m = RejectFriendsReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("friendpb.RejectFriendsReply: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("friendpb.RejectFriendsReply: %w", err)
		}
	}
	return nil
}

func (m *SetBlockApplicationsRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	return appendBool(nil, 1, m.Block)
}

func (m *SetBlockApplicationsRequest) Unmarshal(data []byte) error {
	*m = SetBlockApplicationsRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("friendpb.SetBlockApplicationsRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("friendpb.SetBlockApplicationsRequest: bad block")
			}
			data = data[n:]
			m.Block = protowire.DecodeBool(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("friendpb.SetBlockApplicationsRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *SetBlockApplicationsReply) Unmarshal(data []byte) error {
	*m = SetBlockApplicationsReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("friendpb.SetBlockApplicationsReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("friendpb.SetBlockApplicationsReply: bad block")
			}
			data = data[n:]
			m.Block = protowire.DecodeBool(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("friendpb.SetBlockApplicationsReply: %w", err)
			}
		}
	}
	return nil
}

func (m *GetShareKeyRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	return appendInt64Varint(nil, 1, m.ShareCfgID)
}

func (m *GetShareKeyRequest) Unmarshal(data []byte) error {
	*m = GetShareKeyRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("friendpb.GetShareKeyRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("friendpb.GetShareKeyRequest: bad share_cfg_id")
			}
			data = data[n:]
			m.ShareCfgID = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("friendpb.GetShareKeyRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *GetShareKeyReply) Unmarshal(data []byte) error {
	*m = GetShareKeyReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("friendpb.GetShareKeyReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("friendpb.GetShareKeyReply: bad share_key")
			}
			data = data[n:]
			m.ShareKey = string(raw)
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("friendpb.GetShareKeyReply: bad share_url")
			}
			data = data[n:]
			m.ShareURL = string(raw)
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("friendpb.GetShareKeyReply: bad share_cfg_id")
			}
			data = data[n:]
			m.ShareCfgID = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("friendpb.GetShareKeyReply: %w", err)
			}
		}
	}
	return nil
}

func (m *FriendApplicationReceivedNotify) Unmarshal(data []byte) error {
	*m = FriendApplicationReceivedNotify{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("friendpb.FriendApplicationReceivedNotify: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("friendpb.FriendApplicationReceivedNotify: bad application")
			}
			data = data[n:]
			app := Application{}
			if err := unmarshalApplication(&app, raw); err != nil {
				return err
			}
			m.Applications = append(m.Applications, app)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("friendpb.FriendApplicationReceivedNotify: %w", err)
			}
		}
	}
	return nil
}

func (m *FriendAddedNotify) Unmarshal(data []byte) error {
	*m = FriendAddedNotify{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("friendpb.FriendAddedNotify: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("friendpb.FriendAddedNotify: bad friend")
			}
			data = data[n:]
			friend := GameFriend{}
			if err := unmarshalGameFriend(&friend, raw); err != nil {
				return err
			}
			m.Friends = append(m.Friends, friend)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("friendpb.FriendAddedNotify: %w", err)
			}
		}
	}
	return nil
}
