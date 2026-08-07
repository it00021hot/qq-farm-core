package marqueepb

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

func (m *GetMarqueeRequest) Marshal() []byte { return []byte{} }

func (m *GetMarqueeRequest) Unmarshal(data []byte) error {
	*m = GetMarqueeRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("marqueepb.GetMarqueeRequest: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("marqueepb.GetMarqueeRequest: %w", err)
		}
	}
	return nil
}

func (m *GetMarqueeReply) Unmarshal(data []byte) error {
	*m = GetMarqueeReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("marqueepb.GetMarqueeReply: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("marqueepb.GetMarqueeReply: %w", err)
		}
	}
	return nil
}
