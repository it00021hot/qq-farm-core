// Package userpb provides hand-written user service protobuf types (protoc unavailable).
// Source: internal/farm/proto/userpb.proto
package userpb

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

// DeviceInfo is gamepb.userpb.DeviceInfo (subset used at login).
type DeviceInfo struct {
	ClientVersion string
	SysSoftware   string
	SysHardware   string
	TelecomOper   string
	Network       string
	ScreenWidth   int64
	ScreenHeight  int64
	Density       float32
	CPU           string
	Memory        int64
	GLRender      string
	GLVersion     string
	DeviceID      string
	AndroidOAID   string
	IOSCAID       string
}

// LoginRequest is gamepb.userpb.LoginRequest (fields used by the bot).
type LoginRequest struct {
	SharerID     int64
	SharerOpenID string
	DeviceInfo   *DeviceInfo
	ShareCfgID   int64
	SceneID      string
	ReportData   *ReportData
}

// ReportData is gamepb.userpb.ReportData.
type ReportData struct {
	Callback        string
	CDExtendInfo    string
	ClickID         string
	ClueToken       string
	MinigameChannel string
	MinigamePlatID  int32
	ReqID           string
	TrackID         string
}

// BasicInfo is gamepb.userpb.BasicInfo.
// Has* flags track wire presence so partial BasicNotify (e.g. level-only) does not wipe gold/exp to 0.
type BasicInfo struct {
	GID              int64
	Name             string
	Level            int64
	Exp              int64
	Gold             int64
	OpenID           string
	AvatarURL        string
	Remark           string
	Signature        string
	Gender           int32
	AuthorizedStatus int32
	DisableNudge     bool
	HasLevel         bool
	HasExp           bool
	HasGold          bool
}

// QQGroupInfo is gamepb.userpb.QQGroupInfo.
type QQGroupInfo struct {
	QQGroupID   string
	QQGroupName string
}

// VersionInfo is gamepb.userpb.VersionInfo.
type VersionInfo struct {
	Status           int32
	VersionRecommend string
	VersionForce     string
	ResVersion       string
}

// LoginReply is gamepb.userpb.LoginReply.
type LoginReply struct {
	Basic                       *BasicInfo
	TimeNowMillis               int64
	IsFirstLogin                bool
	QQGroupInfos                []*QQGroupInfo
	VersionInfo                 *VersionInfo
	QQFriendRecommendAuthorized int64
}

// HeartbeatReply is gamepb.userpb.HeartbeatReply.
type HeartbeatReply struct {
	ServerTime  int64
	VersionInfo *VersionInfo
}

// ReportArkClickRequest is gamepb.userpb.ReportArkClickRequest.
type ReportArkClickRequest struct {
	SharerID     int64
	SharerOpenID string
	ShareCfgID   int64
	SceneID      string
}

// ReportArkClickReply is gamepb.userpb.ReportArkClickReply.
type ReportArkClickReply struct{}

// ClientFlowItem is gamepb.userpb.ClientFlowItem.
type ClientFlowItem struct {
	EventName string
	Timestamp int64
	Params    string
}

// BatchClientReportFlowRequest is gamepb.userpb.BatchClientReportFlowRequest.
type BatchClientReportFlowRequest struct {
	Items []*ClientFlowItem
}

// BatchClientReportFlowReply is gamepb.userpb.BatchClientReportFlowReply.
type BatchClientReportFlowReply struct{}

// SetDisplayInfoRequest is gamepb.userpb.SetDisplayInfoRequest.
type SetDisplayInfoRequest struct {
	Name      string
	Signature string
	Gender    int32
	AvatarURL string
}

// SetDisplayInfoReply is gamepb.userpb.SetDisplayInfoReply.
type SetDisplayInfoReply struct {
	Basic *BasicInfo
}

// SetQQFriendRecommendAuthorizedRequest is gamepb.userpb.SetQQFriendRecommendAuthorizedRequest.
type SetQQFriendRecommendAuthorizedRequest struct {
	Authorized int64
}

// SetQQFriendRecommendAuthorizedReply is gamepb.userpb.SetQQFriendRecommendAuthorizedReply.
type SetQQFriendRecommendAuthorizedReply struct {
	Authorized int64
}

// GetUserSettingsRequest is gamepb.userpb.GetUserSettingsRequest.
type GetUserSettingsRequest struct{}

// UserSettings is gamepb.userpb.UserSettings.
type UserSettings struct {
	QQFriendRecommendAuthorized int64
	DisableNudge                bool
}

// GetUserSettingsReply is gamepb.userpb.GetUserSettingsReply.
type GetUserSettingsReply struct {
	Settings *UserSettings
}

// BatchGetBasicInfoRequest is gamepb.userpb.BatchGetBasicInfoRequest.
type BatchGetBasicInfoRequest struct {
	GIDs []int64
}

// BatchGetBasicInfoReply is gamepb.userpb.BatchGetBasicInfoReply.
type BatchGetBasicInfoReply struct {
	Users []*BasicInfo
}

// BasicNotify is gamepb.userpb.BasicNotify (server push).
type BasicNotify struct {
	Basic *BasicInfo
}

// Marshal encodes LoginRequest.
func (m *LoginRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	if m.SharerID != 0 {
		b = protowire.AppendTag(b, 3, protowire.VarintType)
		b = protowire.AppendVarint(b, uint64(m.SharerID))
	}
	if m.SharerOpenID != "" {
		b = protowire.AppendTag(b, 4, protowire.BytesType)
		b = protowire.AppendString(b, m.SharerOpenID)
	}
	if m.DeviceInfo != nil {
		raw := marshalDeviceInfo(m.DeviceInfo)
		b = protowire.AppendTag(b, 5, protowire.BytesType)
		b = protowire.AppendBytes(b, raw)
	}
	if m.ShareCfgID != 0 {
		b = protowire.AppendTag(b, 6, protowire.VarintType)
		b = protowire.AppendVarint(b, uint64(m.ShareCfgID))
	}
	if m.SceneID != "" {
		b = protowire.AppendTag(b, 7, protowire.BytesType)
		b = protowire.AppendString(b, m.SceneID)
	}
	if m.ReportData != nil {
		raw := marshalReportData(m.ReportData)
		b = protowire.AppendTag(b, 8, protowire.BytesType)
		b = protowire.AppendBytes(b, raw)
	}
	return b
}

func marshalReportData(d *ReportData) []byte {
	var b []byte
	appendStr := func(field protowire.Number, s string) {
		if s == "" {
			return
		}
		b = protowire.AppendTag(b, field, protowire.BytesType)
		b = protowire.AppendString(b, s)
	}
	appendStr(1, d.Callback)
	appendStr(2, d.CDExtendInfo)
	appendStr(3, d.ClickID)
	appendStr(4, d.ClueToken)
	appendStr(5, d.MinigameChannel)
	if d.MinigamePlatID != 0 {
		b = protowire.AppendTag(b, 6, protowire.VarintType)
		b = protowire.AppendVarint(b, uint64(d.MinigamePlatID))
	}
	appendStr(7, d.ReqID)
	appendStr(8, d.TrackID)
	return b
}

// Unmarshal parses LoginReply (enough fields for session bootstrap).
func (m *LoginReply) Unmarshal(data []byte) error {
	*m = LoginReply{}
	if len(data) == 0 {
		return fmt.Errorf("userpb.LoginReply: empty body")
	}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("userpb.LoginReply: bad tag (len=%d head=%x)", len(data), data[:min(8, len(data))])
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("userpb.LoginReply: bad basic")
			}
			data = data[n:]
			basic := &BasicInfo{}
			if err := unmarshalBasicInfo(basic, raw); err != nil {
				return err
			}
			m.Basic = basic
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("userpb.LoginReply: bad time_now_millis")
			}
			data = data[n:]
			m.TimeNowMillis = int64(v)
		case num == 4 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("userpb.LoginReply: bad is_first_login")
			}
			data = data[n:]
			m.IsFirstLogin = protowire.DecodeBool(v)
		case num == 6 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("userpb.LoginReply: bad qq_group_info")
			}
			data = data[n:]
			info := &QQGroupInfo{}
			if err := unmarshalQQGroupInfo(info, raw); err != nil {
				return err
			}
			m.QQGroupInfos = append(m.QQGroupInfos, info)
		case num == 9 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("userpb.LoginReply: bad version_info")
			}
			data = data[n:]
			vi := &VersionInfo{}
			if err := unmarshalVersionInfo(vi, raw); err != nil {
				return err
			}
			m.VersionInfo = vi
		case num == 11 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("userpb.LoginReply: bad qq_friend_recommend_authorized")
			}
			data = data[n:]
			m.QQFriendRecommendAuthorized = int64(v)
		default:
			n := protowire.ConsumeFieldValue(num, typ, data)
			if n < 0 {
				return fmt.Errorf("userpb.LoginReply: bad field %d typ=%d", num, typ)
			}
			data = data[n:]
		}
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func marshalDeviceInfo(d *DeviceInfo) []byte {
	var b []byte
	appendStr := func(field numberField, s string) {
		if s == "" {
			return
		}
		b = protowire.AppendTag(b, protowire.Number(field), protowire.BytesType)
		b = protowire.AppendString(b, s)
	}
	appendStr(1, d.ClientVersion)
	appendStr(2, d.SysSoftware)
	appendStr(3, d.SysHardware)
	appendStr(4, d.TelecomOper)
	appendStr(5, d.Network)
	// Bot LoginRequest always includes screen_width (value 0).
	if d.ClientVersion != "" || d.ScreenWidth != 0 {
		b = protowire.AppendTag(b, 6, protowire.VarintType)
		b = protowire.AppendVarint(b, uint64(d.ScreenWidth))
	}
	if d.ScreenHeight != 0 {
		b = protowire.AppendTag(b, 7, protowire.VarintType)
		b = protowire.AppendVarint(b, uint64(d.ScreenHeight))
	}
	appendStr(9, d.CPU)
	if d.Memory != 0 {
		b = protowire.AppendTag(b, 10, protowire.VarintType)
		b = protowire.AppendVarint(b, uint64(d.Memory))
	}
	appendStr(11, d.GLRender)
	appendStr(12, d.GLVersion)
	appendStr(13, d.DeviceID)
	appendStr(14, d.AndroidOAID)
	appendStr(15, d.IOSCAID)
	return b
}

// MarshalHeartbeatRequest encodes gamepb.userpb.HeartbeatRequest.
func MarshalHeartbeatRequest(gid int64, clientVersion string) []byte {
	var b []byte
	if gid != 0 {
		b = protowire.AppendTag(b, 1, protowire.VarintType)
		b = protowire.AppendVarint(b, uint64(gid))
	}
	if clientVersion != "" {
		b = protowire.AppendTag(b, 2, protowire.BytesType)
		b = protowire.AppendString(b, clientVersion)
	}
	return b
}

type numberField = int

// Unmarshal decodes wire bytes into BasicInfo.
func (m *BasicInfo) Unmarshal(data []byte) error {
	*m = BasicInfo{}
	return unmarshalBasicInfo(m, data)
}

func unmarshalBasicInfo(m *BasicInfo, data []byte) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("userpb.BasicInfo: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("userpb.BasicInfo: bad gid")
			}
			data = data[n:]
			m.GID = int64(v)
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("userpb.BasicInfo: bad name")
			}
			data = data[n:]
			m.Name = string(raw)
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("userpb.BasicInfo: bad level")
			}
			data = data[n:]
			m.Level = int64(v)
			m.HasLevel = true
		case num == 4 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("userpb.BasicInfo: bad exp")
			}
			data = data[n:]
			m.Exp = int64(v)
			m.HasExp = true
		case num == 5 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("userpb.BasicInfo: bad gold")
			}
			data = data[n:]
			m.Gold = int64(v)
			m.HasGold = true
		case num == 6 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("userpb.BasicInfo: bad open_id")
			}
			data = data[n:]
			m.OpenID = string(raw)
		case num == 7 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("userpb.BasicInfo: bad avatar_url")
			}
			data = data[n:]
			m.AvatarURL = string(raw)
		case num == 8 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("userpb.BasicInfo: bad remark")
			}
			data = data[n:]
			m.Remark = string(raw)
		case num == 9 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("userpb.BasicInfo: bad signature")
			}
			data = data[n:]
			m.Signature = string(raw)
		case num == 10 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("userpb.BasicInfo: bad gender")
			}
			data = data[n:]
			m.Gender = int32(v)
		case num == 13 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("userpb.BasicInfo: bad authorized_status")
			}
			data = data[n:]
			m.AuthorizedStatus = int32(v)
		case num == 14 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("userpb.BasicInfo: bad disable_nudge")
			}
			data = data[n:]
			m.DisableNudge = protowire.DecodeBool(v)
		default:
			n := protowire.ConsumeFieldValue(num, typ, data)
			if n < 0 {
				return fmt.Errorf("userpb.BasicInfo: bad field %d", num)
			}
			data = data[n:]
		}
	}
	return nil
}

func (m *BasicNotify) Unmarshal(data []byte) error {
	*m = BasicNotify{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("userpb.BasicNotify: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("userpb.BasicNotify: bad basic")
			}
			data = data[n:]
			basic := &BasicInfo{}
			if err := unmarshalBasicInfo(basic, raw); err != nil {
				return err
			}
			m.Basic = basic
		default:
			n := protowire.ConsumeFieldValue(num, typ, data)
			if n < 0 {
				return fmt.Errorf("userpb.BasicNotify: bad field %d", num)
			}
			data = data[n:]
		}
	}
	return nil
}
