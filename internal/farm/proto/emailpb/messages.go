package emailpb

import (
	"fmt"

	"github.com/MQEnergy/go-skeleton/internal/farm/proto/corepb"
	"google.golang.org/protobuf/encoding/protowire"
)

func marshalEmailItem(m *EmailItem) []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendString(b, 1, m.ID)
	b = appendInt32(b, 2, m.MailType)
	b = appendString(b, 3, m.Title)
	b = appendBool(b, 4, m.Claimed)
	b = appendBool(b, 5, m.HasReward)
	b = appendString(b, 7, m.Subtitle)
	return b
}

func unmarshalEmailItem(m *EmailItem, data []byte) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("emailpb.EmailItem: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			v, n := protowire.ConsumeString(data)
			if n < 0 {
				return fmt.Errorf("emailpb.EmailItem: bad id")
			}
			data = data[n:]
			m.ID = v
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("emailpb.EmailItem: bad mail_type")
			}
			data = data[n:]
			m.MailType = int32(v)
		case num == 3 && typ == protowire.BytesType:
			v, n := protowire.ConsumeString(data)
			if n < 0 {
				return fmt.Errorf("emailpb.EmailItem: bad title")
			}
			data = data[n:]
			m.Title = v
		case num == 4 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("emailpb.EmailItem: bad claimed")
			}
			data = data[n:]
			m.Claimed = protowire.DecodeBool(v)
		case num == 5 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("emailpb.EmailItem: bad has_reward")
			}
			data = data[n:]
			m.HasReward = protowire.DecodeBool(v)
		case num == 7 && typ == protowire.BytesType:
			v, n := protowire.ConsumeString(data)
			if n < 0 {
				return fmt.Errorf("emailpb.EmailItem: bad subtitle")
			}
			data = data[n:]
			m.Subtitle = v
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("emailpb.EmailItem: %w", err)
			}
		}
	}
	return nil
}

func (m *GetEmailListRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	return appendInt32(nil, 1, m.BoxType)
}

func (m *GetEmailListRequest) Unmarshal(data []byte) error {
	*m = GetEmailListRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("emailpb.GetEmailListRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("emailpb.GetEmailListRequest: bad box_type")
			}
			data = data[n:]
			m.BoxType = int32(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("emailpb.GetEmailListRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *GetEmailListReply) Unmarshal(data []byte) error {
	*m = GetEmailListReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("emailpb.GetEmailListReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("emailpb.GetEmailListReply: bad email")
			}
			data = data[n:]
			item := EmailItem{}
			if err := unmarshalEmailItem(&item, raw); err != nil {
				return err
			}
			m.Emails = append(m.Emails, item)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("emailpb.GetEmailListReply: %w", err)
			}
		}
	}
	return nil
}

func (m *ReadEmailRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt32(b, 1, m.BoxType)
	b = appendString(b, 2, m.EmailID)
	return b
}

func (m *ReadEmailRequest) Unmarshal(data []byte) error {
	*m = ReadEmailRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("emailpb.ReadEmailRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("emailpb.ReadEmailRequest: bad box_type")
			}
			data = data[n:]
			m.BoxType = int32(v)
		case num == 2 && typ == protowire.BytesType:
			v, n := protowire.ConsumeString(data)
			if n < 0 {
				return fmt.Errorf("emailpb.ReadEmailRequest: bad email_id")
			}
			data = data[n:]
			m.EmailID = v
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("emailpb.ReadEmailRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *ReadEmailReply) Marshal() []byte { return []byte{} }

func (m *ReadEmailReply) Unmarshal(data []byte) error {
	*m = ReadEmailReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("emailpb.ReadEmailReply: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("emailpb.ReadEmailReply: %w", err)
		}
	}
	return nil
}

func (m *ClaimEmailRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt32(b, 1, m.BoxType)
	b = appendString(b, 2, m.EmailID)
	return b
}

func (m *ClaimEmailRequest) Unmarshal(data []byte) error {
	*m = ClaimEmailRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("emailpb.ClaimEmailRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("emailpb.ClaimEmailRequest: bad box_type")
			}
			data = data[n:]
			m.BoxType = int32(v)
		case num == 2 && typ == protowire.BytesType:
			v, n := protowire.ConsumeString(data)
			if n < 0 {
				return fmt.Errorf("emailpb.ClaimEmailRequest: bad email_id")
			}
			data = data[n:]
			m.EmailID = v
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("emailpb.ClaimEmailRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *ClaimEmailReply) Unmarshal(data []byte) error {
	*m = ClaimEmailReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("emailpb.ClaimEmailReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("emailpb.ClaimEmailReply: bad item")
			}
			data = data[n:]
			item := &corepb.Item{}
			if err := item.Unmarshal(raw); err != nil {
				return err
			}
			m.Items = append(m.Items, item)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("emailpb.ClaimEmailReply: %w", err)
			}
		}
	}
	return nil
}

func (m *BatchClaimEmailRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt32(b, 1, m.BoxType)
	b = appendString(b, 2, m.EmailID)
	return b
}

func (m *BatchClaimEmailReply) Unmarshal(data []byte) error {
	*m = BatchClaimEmailReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("emailpb.BatchClaimEmailReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("emailpb.BatchClaimEmailReply: bad item")
			}
			data = data[n:]
			item := &corepb.Item{}
			if err := item.Unmarshal(raw); err != nil {
				return err
			}
			m.Items = append(m.Items, item)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("emailpb.BatchClaimEmailReply: %w", err)
			}
		}
	}
	return nil
}

func (m *BatchDeleteEmailRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt32(b, 1, m.BoxType)
	for _, id := range m.EmailIDs {
		b = appendString(b, 2, id)
	}
	return b
}

func (m *BatchDeleteEmailRequest) Unmarshal(data []byte) error {
	*m = BatchDeleteEmailRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("emailpb.BatchDeleteEmailRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("emailpb.BatchDeleteEmailRequest: bad box_type")
			}
			data = data[n:]
			m.BoxType = int32(v)
		case num == 2 && typ == protowire.BytesType:
			v, n := protowire.ConsumeString(data)
			if n < 0 {
				return fmt.Errorf("emailpb.BatchDeleteEmailRequest: bad email_id")
			}
			data = data[n:]
			m.EmailIDs = append(m.EmailIDs, v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("emailpb.BatchDeleteEmailRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *BatchDeleteEmailReply) Unmarshal(data []byte) error {
	*m = BatchDeleteEmailReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("emailpb.BatchDeleteEmailReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("emailpb.BatchDeleteEmailReply: bad success")
			}
			data = data[n:]
			m.Success = protowire.DecodeBool(v)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("emailpb.BatchDeleteEmailReply: %w", err)
			}
		}
	}
	return nil
}
