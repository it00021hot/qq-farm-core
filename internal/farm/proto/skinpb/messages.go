package skinpb

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

func (m *SkinItem) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.SkinID)
	b = appendInt64Varint(b, 2, m.SlotType)
	return b
}

func (m *SkinItem) Unmarshal(data []byte) error {
	*m = SkinItem{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("skinpb.SkinItem: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("skinpb.SkinItem: bad skin_id")
			}
			data = data[n:]
			m.SkinID = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("skinpb.SkinItem: bad slot_type")
			}
			data = data[n:]
			m.SlotType = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("skinpb.SkinItem: %w", err)
			}
		}
	}
	return nil
}

func unmarshalSkinsReply(data []byte, dst *[]*SkinItem, pkgMsg string) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("%s: bad tag", pkgMsg)
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("%s: bad skin", pkgMsg)
			}
			data = data[n:]
			item := &SkinItem{}
			if err := item.Unmarshal(raw); err != nil {
				return err
			}
			*dst = append(*dst, item)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("%s: %w", pkgMsg, err)
			}
		}
	}
	return nil
}

func (m *SkinsOwnedRequest) Marshal() []byte { return []byte{} }

func (m *SkinsOwnedRequest) Unmarshal(data []byte) error {
	*m = SkinsOwnedRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("skinpb.SkinsOwnedRequest: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("skinpb.SkinsOwnedRequest: %w", err)
		}
	}
	return nil
}

func (m *SkinsOwnedReply) Unmarshal(data []byte) error {
	*m = SkinsOwnedReply{}
	return unmarshalSkinsReply(data, &m.Skins, "skinpb.SkinsOwnedReply")
}

func (m *SkinsEquippedRequest) Marshal() []byte { return []byte{} }

func (m *SkinsEquippedRequest) Unmarshal(data []byte) error {
	*m = SkinsEquippedRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("skinpb.SkinsEquippedRequest: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("skinpb.SkinsEquippedRequest: %w", err)
		}
	}
	return nil
}

func (m *SkinsEquippedReply) Unmarshal(data []byte) error {
	*m = SkinsEquippedReply{}
	return unmarshalSkinsReply(data, &m.Skins, "skinpb.SkinsEquippedReply")
}

func (m *EquipRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.SkinID)
	b = appendInt64Varint(b, 2, m.SlotType)
	return b
}

func (m *EquipRequest) Unmarshal(data []byte) error {
	*m = EquipRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("skinpb.EquipRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("skinpb.EquipRequest: bad skin_id")
			}
			data = data[n:]
			m.SkinID = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("skinpb.EquipRequest: bad slot_type")
			}
			data = data[n:]
			m.SlotType = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("skinpb.EquipRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *EquipReply) Unmarshal(data []byte) error {
	*m = EquipReply{}
	return unmarshalSkinsReply(data, &m.Skins, "skinpb.EquipReply")
}

func (m *MarkAsViewedRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	return appendPackedInt64s(nil, 1, m.SkinIDs)
}

func (m *MarkAsViewedRequest) Unmarshal(data []byte) error {
	*m = MarkAsViewedRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("skinpb.MarkAsViewedRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1:
			var err error
			data, err = consumeRepeatedInt64(num, typ, data, &m.SkinIDs)
			if err != nil {
				return fmt.Errorf("skinpb.MarkAsViewedRequest: %w", err)
			}
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("skinpb.MarkAsViewedRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *MarkAsViewedReply) Unmarshal(data []byte) error {
	*m = MarkAsViewedReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("skinpb.MarkAsViewedReply: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("skinpb.MarkAsViewedReply: %w", err)
		}
	}
	return nil
}

func (m *SkinChangeNotify) Unmarshal(data []byte) error {
	*m = SkinChangeNotify{}
	return unmarshalSkinsReply(data, &m.Skins, "skinpb.SkinChangeNotify")
}
