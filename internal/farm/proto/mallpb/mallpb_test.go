package mallpb_test

import (
	"bytes"
	"testing"

	"github.com/MQEnergy/go-skeleton/internal/farm/proto/mallpb"
	"google.golang.org/protobuf/encoding/protowire"
)

func TestPurchaseRequestRoundTrip(t *testing.T) {
	orig := &mallpb.PurchaseRequest{
		GoodsID: 1002,
		Count:   10,
	}
	raw := orig.Marshal()
	var back mallpb.PurchaseRequest
	if err := back.Unmarshal(raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	raw2 := back.Marshal()
	if !bytes.Equal(raw, raw2) {
		t.Fatalf("round-trip bytes differ:\n  orig=% x\n  back=% x", raw, raw2)
	}
	if back.GoodsID != 1002 || back.Count != 10 {
		t.Fatalf("decoded=%+v", back)
	}
}

func TestParseMallPrice(t *testing.T) {
	price := protowire.AppendTag(nil, 2, protowire.VarintType)
	price = protowire.AppendVarint(price, 50)
	if got := mallpb.ParseMallPrice(price); got != 50 {
		t.Fatalf("ParseMallPrice=%d want 50", got)
	}
}

func TestMallGoodsRoundTrip(t *testing.T) {
	price := protowire.AppendTag(nil, 2, protowire.VarintType)
	price = protowire.AppendVarint(price, 99)
	orig := &mallpb.MallGoods{
		GoodsID: 1003,
		Name:    "无机化肥",
		Type:    1,
		Price:   price,
		IsFree:  false,
	}
	raw := orig.Marshal()
	var back mallpb.MallGoods
	if err := back.Unmarshal(raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.GoodsID != 1003 || back.Name != "无机化肥" {
		t.Fatalf("decoded=%+v", back)
	}
}
