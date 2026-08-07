package itempb_test

import (
	"bytes"
	"testing"

	"github.com/MQEnergy/go-skeleton/internal/farm/proto/corepb"
	"github.com/MQEnergy/go-skeleton/internal/farm/proto/itempb"
)

func TestBagReplyRoundTrip(t *testing.T) {
	orig := &itempb.BagReply{
		ItemBag: &corepb.ItemBag{
			Items: []*corepb.Item{
				{ID: 2001, Count: 5, UID: 99, IsNew: true},
				{ID: 1001, Count: 12345},
			},
		},
	}
	raw := orig.Marshal()
	var back itempb.BagReply
	if err := back.Unmarshal(raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	raw2 := back.Marshal()
	if !bytes.Equal(raw, raw2) {
		t.Fatalf("round-trip bytes differ:\n  orig=% x\n  back=% x", raw, raw2)
	}
	if len(back.ItemBag.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(back.ItemBag.Items))
	}
	if back.ItemBag.Items[0].ID != 2001 || back.ItemBag.Items[0].Count != 5 || !back.ItemBag.Items[0].IsNew {
		t.Fatalf("first item mismatch: %+v", back.ItemBag.Items[0])
	}
}

func TestSellRequestRoundTrip(t *testing.T) {
	orig := &itempb.SellRequest{
		Items: []corepb.Item{
			{ID: 3001, Count: 2},
			{ID: 3002, Count: 1, UID: 7},
		},
	}
	raw := orig.Marshal()
	var back itempb.SellRequest
	if err := back.Unmarshal(raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(back.Items) != 2 || back.Items[1].UID != 7 {
		t.Fatalf("decoded=%+v", back.Items)
	}
}
