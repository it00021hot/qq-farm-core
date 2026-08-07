// Package itempb provides hand-written item service protobuf types (protoc unavailable).
// Source: internal/farm/proto/itempb.proto
package itempb

import "github.com/MQEnergy/go-skeleton/internal/farm/proto/corepb"

// BagRequest fetches the player bag.
type BagRequest struct{}

// BagReply contains the bag contents.
type BagReply struct {
	ItemBag *corepb.ItemBag
}

// SellRequest sells items from the bag.
type SellRequest struct {
	Items []corepb.Item
}

// SellReply is the sell operation result.
type SellReply struct {
	SellItems []*corepb.Item
	GetItems  []*corepb.Item
}

// UseRequest uses a single item.
type UseRequest struct {
	ItemID  int64
	Count   int64
	LandIDs []int64
}

// UseReply is the use operation result.
type UseReply struct {
	Items []*corepb.Item
}

// BatchUseRequest uses multiple items at once.
type BatchUseRequest struct {
	Items []corepb.Item
}

// BatchUseReply is the batch use result.
type BatchUseReply struct {
	UsedItems []*corepb.Item
	Items     []*corepb.Item
}

// CannelNewRequest clears the "new" marker on items (proto typo preserved).
type CannelNewRequest struct {
	Items []corepb.Item
}

// CannelNewReply is the clear result.
type CannelNewReply struct{}

// ItemNotify is gamepb.itempb.ItemNotify (server push).
type ItemNotify struct {
	Items []*corepb.ItemChg
}
