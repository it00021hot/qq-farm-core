package shoppb_test

import (
	"bytes"
	"testing"

	"github.com/MQEnergy/go-skeleton/internal/farm/proto/shoppb"
)

func TestBuyGoodsRequestRoundTrip(t *testing.T) {
	orig := &shoppb.BuyGoodsRequest{
		GoodsID: 501,
		Num:     3,
		Price:   120,
	}
	raw := orig.Marshal()
	var back shoppb.BuyGoodsRequest
	if err := back.Unmarshal(raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	raw2 := back.Marshal()
	if !bytes.Equal(raw, raw2) {
		t.Fatalf("round-trip bytes differ:\n  orig=% x\n  back=% x", raw, raw2)
	}
	if back.GoodsID != 501 || back.Num != 3 || back.Price != 120 {
		t.Fatalf("decoded=%+v", back)
	}
}
