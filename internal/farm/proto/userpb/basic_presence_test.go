package userpb

import (
	"testing"

	"google.golang.org/protobuf/encoding/protowire"
)

func TestBasicInfoPresenceFlags(t *testing.T) {
	var b []byte
	b = protowire.AppendTag(b, 3, protowire.VarintType)
	b = protowire.AppendVarint(b, 42) // level only

	var info BasicInfo
	if err := info.Unmarshal(b); err != nil {
		t.Fatal(err)
	}
	if !info.HasLevel || info.Level != 42 {
		t.Fatalf("level presence: has=%v level=%d", info.HasLevel, info.Level)
	}
	if info.HasExp || info.HasGold {
		t.Fatalf("exp/gold should be absent, hasExp=%v hasGold=%v", info.HasExp, info.HasGold)
	}
}

func TestBasicInfoPresenceWithGold(t *testing.T) {
	var b []byte
	b = protowire.AppendTag(b, 5, protowire.VarintType)
	b = protowire.AppendVarint(b, 12345)

	var info BasicInfo
	if err := info.Unmarshal(b); err != nil {
		t.Fatal(err)
	}
	if !info.HasGold || info.Gold != 12345 {
		t.Fatalf("gold presence: has=%v gold=%d", info.HasGold, info.Gold)
	}
	if info.HasExp || info.HasLevel {
		t.Fatalf("unexpected presence flags")
	}
}
