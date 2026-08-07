package gatepb

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

// EventMessage is gatepb.EventMessage — server push payload inside Notify frames.
type EventMessage struct {
	MessageType string
	Body        []byte
}

// KickoutNotify is gatepb.KickoutNotify.
type KickoutNotify struct {
	Reason        int64
	ReasonMessage string
}

// Unmarshal parses KickoutNotify from protobuf wire bytes.
func (m *KickoutNotify) Unmarshal(data []byte) error {
	*m = KickoutNotify{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("gatepb.KickoutNotify: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("gatepb.KickoutNotify: bad reason")
			}
			data = data[n:]
			m.Reason = int64(v)
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("gatepb.KickoutNotify: bad reason_message")
			}
			data = data[n:]
			m.ReasonMessage = string(raw)
		default:
			n := protowire.ConsumeFieldValue(num, typ, data)
			if n < 0 {
				return fmt.Errorf("gatepb.KickoutNotify: bad field %d", num)
			}
			data = data[n:]
		}
	}
	return nil
}

// Unmarshal parses EventMessage from protobuf wire bytes.
func (m *EventMessage) Unmarshal(data []byte) error {
	*m = EventMessage{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("gatepb.EventMessage: bad tag")
		}
		data = data[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("gatepb.EventMessage: bad message_type")
			}
			data = data[n:]
			m.MessageType = string(raw)
		case num == 2 && typ == protowire.BytesType:
			raw, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("gatepb.EventMessage: bad body")
			}
			data = data[n:]
			m.Body = append([]byte(nil), raw...)
		default:
			n := protowire.ConsumeFieldValue(num, typ, data)
			if n < 0 {
				return fmt.Errorf("gatepb.EventMessage: bad field %d", num)
			}
			data = data[n:]
		}
	}
	return nil
}
