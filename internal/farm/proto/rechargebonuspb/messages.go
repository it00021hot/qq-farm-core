package rechargebonuspb

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

func (m *GetConfigRequest) Marshal() []byte { return []byte{} }

func (m *GetConfigRequest) Unmarshal(data []byte) error {
	*m = GetConfigRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("rechargebonuspb.GetConfigRequest: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("rechargebonuspb.GetConfigRequest: %w", err)
		}
	}
	return nil
}

func (m *GetConfigReply) Unmarshal(data []byte) error {
	*m = GetConfigReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("rechargebonuspb.GetConfigReply: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("rechargebonuspb.GetConfigReply: %w", err)
		}
	}
	return nil
}
