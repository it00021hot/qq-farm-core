package dogpb

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

func (m *DogInfo) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.ID)
	b = appendString(b, 2, m.Name)
	b = appendInt64Varint(b, 3, m.Price)
	b = appendInt64Varint(b, 4, m.Status)
	b = appendInt64Varint(b, 5, m.Level)
	return b
}

func (m *DogInfo) Unmarshal(data []byte) error {
	*m = DogInfo{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("dogpb.DogInfo: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("dogpb.DogInfo: bad id")
			}
			data = data[n:]
			m.ID = int64(v)
		case num == 2 && typ == protowire.BytesType:
			v, n := protowire.ConsumeString(data)
			if n < 0 {
				return fmt.Errorf("dogpb.DogInfo: bad name")
			}
			data = data[n:]
			m.Name = v
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("dogpb.DogInfo: bad price")
			}
			data = data[n:]
			m.Price = int64(v)
		case num == 4 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("dogpb.DogInfo: bad status")
			}
			data = data[n:]
			m.Status = int64(v)
		case num == 5 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("dogpb.DogInfo: bad level")
			}
			data = data[n:]
			m.Level = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("dogpb.DogInfo: %w", err)
			}
		}
	}
	return nil
}

func (m *DogItem) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt64Varint(b, 1, m.ID)
	b = appendInt64Varint(b, 2, m.Duration)
	b = appendInt64Varint(b, 3, m.Status)
	return b
}

func (m *DogItem) Unmarshal(data []byte) error {
	*m = DogItem{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("dogpb.DogItem: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("dogpb.DogItem: bad id")
			}
			data = data[n:]
			m.ID = int64(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("dogpb.DogItem: bad duration")
			}
			data = data[n:]
			m.Duration = int64(v)
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("dogpb.DogItem: bad status")
			}
			data = data[n:]
			m.Status = int64(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("dogpb.DogItem: %w", err)
			}
		}
	}
	return nil
}

func (m *GetDogInfoRequest) Marshal() []byte { return []byte{} }

func (m *GetDogInfoRequest) Unmarshal(data []byte) error {
	*m = GetDogInfoRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("dogpb.GetDogInfoRequest: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("dogpb.GetDogInfoRequest: %w", err)
		}
	}
	return nil
}

func (m *GetDogInfoReply) Unmarshal(data []byte) error {
	*m = GetDogInfoReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("dogpb.GetDogInfoReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("dogpb.GetDogInfoReply: bad dog")
			}
			data = data[n:]
			dog := &DogInfo{}
			if err := dog.Unmarshal(raw); err != nil {
				return err
			}
			m.Dogs = append(m.Dogs, dog)
		case num == 4 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("dogpb.GetDogInfoReply: bad protect_duration")
			}
			data = data[n:]
			m.ProtectDuration = int64(v)
		case num == 5 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("dogpb.GetDogInfoReply: bad item")
			}
			data = data[n:]
			item := &DogItem{}
			if err := item.Unmarshal(raw); err != nil {
				return err
			}
			m.Items = append(m.Items, item)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("dogpb.GetDogInfoReply: %w", err)
			}
		}
	}
	return nil
}
