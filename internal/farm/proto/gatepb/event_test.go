package gatepb

import (
	"testing"

	"google.golang.org/protobuf/encoding/protowire"
)

func TestKickoutNotifyUnmarshal(t *testing.T) {
	var b []byte
	b = protowire.AppendTag(b, 1, protowire.VarintType)
	b = protowire.AppendVarint(b, 7)
	b = protowire.AppendTag(b, 2, protowire.BytesType)
	b = protowire.AppendString(b, "duplicate login")

	var n KickoutNotify
	if err := n.Unmarshal(b); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if n.Reason != 7 || n.ReasonMessage != "duplicate login" {
		t.Fatalf("got %+v", n)
	}
}
