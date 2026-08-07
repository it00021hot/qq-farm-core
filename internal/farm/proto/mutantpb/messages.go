package mutantpb

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

func (m *ReadMutantBookRequest) Marshal() []byte { return []byte{} }

func (m *ReadMutantBookRequest) Unmarshal(data []byte) error {
	*m = ReadMutantBookRequest{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("mutantpb.ReadMutantBookRequest: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("mutantpb.ReadMutantBookRequest: %w", err)
		}
	}
	return nil
}

func (m *ReadMutantBookReply) Unmarshal(data []byte) error {
	*m = ReadMutantBookReply{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("mutantpb.ReadMutantBookReply: bad tag")
		}
		data = data[n:]
		var err error
		data, err = skipField(num, typ, data)
		if err != nil {
			return fmt.Errorf("mutantpb.ReadMutantBookReply: %w", err)
		}
	}
	return nil
}
