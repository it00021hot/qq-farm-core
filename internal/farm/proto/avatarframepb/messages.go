package avatarframepb

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

func (m *AvatarFrame) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.FrameID)
	b = appendInt64Varint(b, 2, m.Status)
	b = appendInt64Varint(b, 3, m.Equipped)
	b = appendInt64Varint(b, 4, m.ExpireTime)
	return b
}

func (m *AvatarFrame) Unmarshal(data []byte) error {
	*m = AvatarFrame{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("avatarframepb.AvatarFrame: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("avatarframepb.AvatarFrame: bad frame_id")
			}
			data = data[n:]
			m.FrameID = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("avatarframepb.AvatarFrame: bad status")
			}
			data = data[n:]
			m.Status = int64(v)
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("avatarframepb.AvatarFrame: bad equipped")
			}
			data = data[n:]
			m.Equipped = int64(v)
		case num == 4 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("avatarframepb.AvatarFrame: bad expire_time")
			}
			data = data[n:]
			m.ExpireTime = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("avatarframepb.AvatarFrame: %w", err)
			}
		}
	}
	return nil
}

func (m *AvatarFramesOwnedRequest) Marshal() []byte { return []byte{} }

func (m *AvatarFramesOwnedRequest) Unmarshal(data []byte) error {
	*m = AvatarFramesOwnedRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("avatarframepb.AvatarFramesOwnedRequest: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("avatarframepb.AvatarFramesOwnedRequest: %w", err)
		}
	}
	return nil
}

func (m *AvatarFramesOwnedReply) Unmarshal(data []byte) error {
	*m = AvatarFramesOwnedReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("avatarframepb.AvatarFramesOwnedReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("avatarframepb.AvatarFramesOwnedReply: bad frame")
			}
			data = data[n:]
			frame := &AvatarFrame{}
			if err := frame.Unmarshal(raw); err != nil {
				return err
			}
			m.Frames = append(m.Frames, frame)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("avatarframepb.AvatarFramesOwnedReply: %w", err)
			}
		}
	}
	return nil
}

func (m *AvatarFrameRedDotNotify) Unmarshal(data []byte) error {
	*m = AvatarFrameRedDotNotify{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("avatarframepb.AvatarFrameRedDotNotify: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("avatarframepb.AvatarFrameRedDotNotify: %w", err)
		}
	}
	return nil
}
