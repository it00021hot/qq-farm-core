package uicproxypb

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

func (m *TextResult) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendInt32(b, 1, m.Status)
	b = appendString(b, 2, m.Text)
	b = appendString(b, 4, m.UID)
	return b
}

func (m *TextResult) Unmarshal(data []byte) error {
	*m = TextResult{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("uicproxypb.TextResult: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("uicproxypb.TextResult: bad status")
			}
			data = data[n:]
			m.Status = int32(v)
		case num == 2 && typ == protowire.BytesType:
			v, n := protowire.ConsumeString(data)
			if n < 0 {
				return fmt.Errorf("uicproxypb.TextResult: bad text")
			}
			data = data[n:]
			m.Text = v
		case num == 4 && typ == protowire.BytesType:
			v, n := protowire.ConsumeString(data)
			if n < 0 {
				return fmt.Errorf("uicproxypb.TextResult: bad uid")
			}
			data = data[n:]
			m.UID = v
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("uicproxypb.TextResult: %w", err)
			}
		}
	}
	return nil
}

func (m *TextItem) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	b = appendString(b, 1, m.UID)
	b = appendString(b, 2, m.Text)
	return b
}

func (m *TextItem) Unmarshal(data []byte) error {
	*m = TextItem{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("uicproxypb.TextItem: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			v, n := protowire.ConsumeString(data)
			if n < 0 {
				return fmt.Errorf("uicproxypb.TextItem: bad uid")
			}
			data = data[n:]
			m.UID = v
		case num == 2 && typ == protowire.BytesType:
			v, n := protowire.ConsumeString(data)
			if n < 0 {
				return fmt.Errorf("uicproxypb.TextItem: bad text")
			}
			data = data[n:]
			m.Text = v
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("uicproxypb.TextItem: %w", err)
			}
		}
	}
	return nil
}

func (m *BatchModerateTextRequest) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	for _, item := range m.Items {
		if item == nil {
			continue
		}
		b = appendMessage(b, 1, item.Marshal())
	}
	return b
}

func (m *BatchModerateTextRequest) Unmarshal(data []byte) error {
	*m = BatchModerateTextRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("uicproxypb.BatchModerateTextRequest: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("uicproxypb.BatchModerateTextRequest: bad item")
			}
			data = data[n:]
			item := &TextItem{}
			if err := item.Unmarshal(raw); err != nil {
				return err
			}
			m.Items = append(m.Items, item)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("uicproxypb.BatchModerateTextRequest: %w", err)
			}
		}
	}
	return nil
}

func (m *BatchModerateTextReply) Unmarshal(data []byte) error {
	*m = BatchModerateTextReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("uicproxypb.BatchModerateTextReply: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("uicproxypb.BatchModerateTextReply: bad result")
			}
			data = data[n:]
			r := &TextResult{}
			if err := r.Unmarshal(raw); err != nil {
				return err
			}
			m.Results = append(m.Results, r)
		default:
			var err error
			data, err = skipField(num, typ, data)
			if err != nil {
				return fmt.Errorf("uicproxypb.BatchModerateTextReply: %w", err)
			}
		}
	}
	return nil
}
