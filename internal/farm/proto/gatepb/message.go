// Package gatepb provides hand-written gatepb.Message types (protoc unavailable).
// Source: qq-farm-bot/core/src/proto/game.proto
package gatepb

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

// MessageType matches gatepb.MessageType.
const (
	MessageTypeNone     int32 = 0
	MessageTypeRequest  int32 = 1
	MessageTypeResponse int32 = 2
	MessageTypeNotify   int32 = 3
)

// Meta is gatepb.Meta.
type Meta struct {
	ServiceName  string
	MethodName   string
	MessageType  int32
	ClientSeq    int64
	ServerSeq    int64
	ErrorCode    int64
	ErrorMessage string
	Metadata     map[string][]byte
}

// Message is gatepb.Message — the WSS frame envelope.
type Message struct {
	Meta  *Meta
	Body  []byte
	Token string
}

// Marshal encodes Message to protobuf wire bytes.
func (m *Message) Marshal() []byte {
	if m == nil {
		return nil
	}
	var b []byte
	if m.Meta != nil {
		metaBytes := marshalMeta(m.Meta)
		b = protowire.AppendTag(b, 1, protowire.BytesType)
		b = protowire.AppendBytes(b, metaBytes)
	}
	if len(m.Body) > 0 {
		b = protowire.AppendTag(b, 2, protowire.BytesType)
		b = protowire.AppendBytes(b, m.Body)
	}
	if m.Token != "" {
		b = protowire.AppendTag(b, 3, protowire.BytesType)
		b = protowire.AppendString(b, m.Token)
	}
	return b
}

// Unmarshal parses Message from protobuf wire bytes.
func (m *Message) Unmarshal(data []byte) error {
	*m = Message{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("gatepb.Message: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("gatepb.Message: bad meta")
			}
			data = data[n:]
			meta := &Meta{}
			if err := unmarshalMeta(meta, raw); err != nil {
				return err
			}
			m.Meta = meta
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("gatepb.Message: bad body")
			}
			data = data[n:]
			m.Body = append([]byte(nil), raw...)
		case num == 3 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("gatepb.Message: bad token")
			}
			data = data[n:]
			m.Token = string(raw)
		default:
			n := protowire.ConsumeFieldValue(num, typ, data)
			if n < 0 {
				return fmt.Errorf("gatepb.Message: bad field %d", num)
			}
			data = data[n:]
		}
	}
	return nil
}

func marshalMeta(m *Meta) []byte {
	var b []byte
	if m.ServiceName != "" {
		b = protowire.AppendTag(b, 1, protowire.BytesType)
		b = protowire.AppendString(b, m.ServiceName)
	}
	if m.MethodName != "" {
		b = protowire.AppendTag(b, 2, protowire.BytesType)
		b = protowire.AppendString(b, m.MethodName)
	}
	if m.MessageType != 0 {
		b = protowire.AppendTag(b, 3, protowire.VarintType)
		b = protowire.AppendVarint(b, uint64(m.MessageType))
	}
	if m.ClientSeq != 0 {
		b = protowire.AppendTag(b, 4, protowire.VarintType)
		b = protowire.AppendVarint(b, uint64(m.ClientSeq))
	}
	if m.ServerSeq != 0 {
		b = protowire.AppendTag(b, 5, protowire.VarintType)
		b = protowire.AppendVarint(b, uint64(m.ServerSeq))
	}
	if m.ErrorCode != 0 {
		b = protowire.AppendTag(b, 6, protowire.VarintType)
		b = protowire.AppendVarint(b, uint64(m.ErrorCode))
	}
	if m.ErrorMessage != "" {
		b = protowire.AppendTag(b, 7, protowire.BytesType)
		b = protowire.AppendString(b, m.ErrorMessage)
	}
	for k, v := range m.Metadata {
		entry := marshalMetaEntry(k, v)
		b = protowire.AppendTag(b, 8, protowire.BytesType)
		b = protowire.AppendBytes(b, entry)
	}
	return b
}

func marshalMetaEntry(key string, value []byte) []byte {
	var b []byte
	if key != "" {
		b = protowire.AppendTag(b, 1, protowire.BytesType)
		b = protowire.AppendString(b, key)
	}
	if len(value) > 0 {
		b = protowire.AppendTag(b, 2, protowire.BytesType)
		b = protowire.AppendBytes(b, value)
	}
	return b
}

func unmarshalMeta(m *Meta, data []byte) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("gatepb.Meta: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("gatepb.Meta: bad service_name")
			}
			data = data[n:]
			m.ServiceName = string(raw)
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("gatepb.Meta: bad method_name")
			}
			data = data[n:]
			m.MethodName = string(raw)
		case num == 3 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("gatepb.Meta: bad message_type")
			}
			data = data[n:]
			m.MessageType = int32(v)
		case num == 4 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("gatepb.Meta: bad client_seq")
			}
			data = data[n:]
			m.ClientSeq = int64(v)
		case num == 5 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("gatepb.Meta: bad server_seq")
			}
			data = data[n:]
			m.ServerSeq = int64(v)
		case num == 6 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("gatepb.Meta: bad error_code")
			}
			data = data[n:]
			m.ErrorCode = int64(v)
		case num == 7 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("gatepb.Meta: bad error_message")
			}
			data = data[n:]
			m.ErrorMessage = string(raw)
		case num == 8 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("gatepb.Meta: bad metadata")
			}
			data = data[n:]
			k, v, err := unmarshalMetaEntry(raw)
			if err != nil {
				return err
			}
			if m.Metadata == nil {
				m.Metadata = make(map[string][]byte)
			}
			m.Metadata[k] = v
		default:
			n := protowire.ConsumeFieldValue(num, typ, data)
			if n < 0 {
				return fmt.Errorf("gatepb.Meta: bad field %d", num)
			}
			data = data[n:]
		}
	}
	return nil
}

func unmarshalMetaEntry(data []byte) (string, []byte, error) {
	var key string
	var value []byte
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return "", nil, fmt.Errorf("gatepb.Meta.metadata: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return "", nil, fmt.Errorf("gatepb.Meta.metadata: bad key")
			}
			data = data[n:]
			key = string(raw)
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return "", nil, fmt.Errorf("gatepb.Meta.metadata: bad value")
			}
			data = data[n:]
			value = append([]byte(nil), raw...)
		default:
			n := protowire.ConsumeFieldValue(num, typ, data)
			if n < 0 {
				return "", nil, fmt.Errorf("gatepb.Meta.metadata: bad field %d", num)
			}
			data = data[n:]
		}
	}
	return key, value, nil
}
