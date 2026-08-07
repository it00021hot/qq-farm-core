package bulletinboardpb

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

func (m *BulletinItem) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.ID)
	b = appendString(b, 2, m.Title)
	b = appendInt64Varint(b, 3, m.Status)
	b = appendInt64Varint(b, 5, m.Type)
	return b
}

func (m *BulletinItem) Unmarshal(data []byte) error {
	*m = BulletinItem{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("bulletinboardpb.BulletinItem: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("bulletinboardpb.BulletinItem: bad id")
			}
			data = data[n:]
			m.ID = int64(v)
		case num == 2 && typ == protowire.BytesType:
			v, n := protowire.ConsumeString(data)
			if n < 0 {
				return fmt.Errorf("bulletinboardpb.BulletinItem: bad title")
			}
			data = data[n:]
			m.Title = v
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("bulletinboardpb.BulletinItem: bad status")
			}
			data = data[n:]
			m.Status = int64(v)
		case num == 5 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("bulletinboardpb.BulletinItem: bad type")
			}
			data = data[n:]
			m.Type = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("bulletinboardpb.BulletinItem: %w", err)
			}
		}
	}
	return nil
}

func (m *GetBulletinListRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	return appendInt64Varint(nil, 1, m.Count)
}

func (m *GetBulletinListRequest) Unmarshal(data []byte) error {
	*m = GetBulletinListRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("bulletinboardpb.GetBulletinListRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("bulletinboardpb.GetBulletinListRequest: bad count")
			}
			data = data[n:]
			m.Count = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("bulletinboardpb.GetBulletinListRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *GetBulletinListReply) Unmarshal(data []byte) error {
	*m = GetBulletinListReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("bulletinboardpb.GetBulletinListReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("bulletinboardpb.GetBulletinListReply: bad bulletin")
			}
			data = data[n:]
			item := &BulletinItem{}
			if err := item.Unmarshal(raw); err != nil {
				return err
			}
			m.Bulletins = append(m.Bulletins, item)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("bulletinboardpb.GetBulletinListReply: %w", err)
			}
		}
	}
	return nil
}

func (m *GetBulletinDetailRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	return appendInt64Varint(nil, 1, m.ID)
}

func (m *GetBulletinDetailRequest) Unmarshal(data []byte) error {
	*m = GetBulletinDetailRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("bulletinboardpb.GetBulletinDetailRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("bulletinboardpb.GetBulletinDetailRequest: bad id")
			}
			data = data[n:]
			m.ID = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("bulletinboardpb.GetBulletinDetailRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *BulletinDetail) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.ID)
	b = appendString(b, 2, m.Title)
	b = appendString(b, 3, m.Content)
	b = appendInt64Varint(b, 4, m.Status)
	b = appendInt64Varint(b, 5, m.Type)
	return b
}

func (m *BulletinDetail) Unmarshal(data []byte) error {
	*m = BulletinDetail{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("bulletinboardpb.BulletinDetail: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("bulletinboardpb.BulletinDetail: bad id")
			}
			data = data[n:]
			m.ID = int64(v)
		case num == 2 && typ == protowire.BytesType:
			v, n := protowire.ConsumeString(data)
			if n < 0 {
				return fmt.Errorf("bulletinboardpb.BulletinDetail: bad title")
			}
			data = data[n:]
			m.Title = v
		case num == 3 && typ == protowire.BytesType:
			v, n := protowire.ConsumeString(data)
			if n < 0 {
				return fmt.Errorf("bulletinboardpb.BulletinDetail: bad content")
			}
			data = data[n:]
			m.Content = v
		case num == 4 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("bulletinboardpb.BulletinDetail: bad status")
			}
			data = data[n:]
			m.Status = int64(v)
		case num == 5 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("bulletinboardpb.BulletinDetail: bad type")
			}
			data = data[n:]
			m.Type = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("bulletinboardpb.BulletinDetail: %w", err)
			}
		}
	}
	return nil
}

func (m *GetBulletinDetailReply) Unmarshal(data []byte) error {
	*m = GetBulletinDetailReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("bulletinboardpb.GetBulletinDetailReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("bulletinboardpb.GetBulletinDetailReply: bad detail")
			}
			data = data[n:]
			d := &BulletinDetail{}
			if err := d.Unmarshal(raw); err != nil {
				return err
			}
			m.Detail = d
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("bulletinboardpb.GetBulletinDetailReply: %w", err)
			}
		}
	}
	return nil
}

func (m *BulletinListChangedNTF) Unmarshal(data []byte) error {
	*m = BulletinListChangedNTF{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("bulletinboardpb.BulletinListChangedNTF: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("bulletinboardpb.BulletinListChangedNTF: bad bulletin")
			}
			data = data[n:]
			item := &BulletinItem{}
			if err := item.Unmarshal(raw); err != nil {
				return err
			}
			m.Bulletins = append(m.Bulletins, item)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("bulletinboardpb.BulletinListChangedNTF: %w", err)
			}
		}
	}
	return nil
}
