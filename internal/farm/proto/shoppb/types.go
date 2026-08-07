// Package shoppb provides hand-written shop service protobuf types (protoc unavailable).
// Source: internal/farm/proto/shoppb.proto
package shoppb

import "github.com/MQEnergy/go-skeleton/internal/farm/proto/corepb"

// Cond is a goods unlock condition.
type Cond struct {
	Type  int32
	Param int64
}

// ShopProfile is one shop overview entry.
type ShopProfile struct {
	ShopID   int64
	ShopName string
	ShopType int32
}

// GoodsInfo is one shop goods entry.
type GoodsInfo struct {
	ID         int64
	BoughtNum  int64
	Price      int64
	LimitCount int64
	Unlocked   bool
	ItemID     int64
	ItemCount  int64
	Conds      []Cond
}

// ShopProfilesRequest lists shop profiles.
type ShopProfilesRequest struct{}

// ShopProfilesReply lists shop profiles.
type ShopProfilesReply struct {
	ShopProfiles []ShopProfile
}

// ShopInfoRequest fetches goods for a shop.
type ShopInfoRequest struct {
	ShopID int64
}

// ShopInfoReply lists goods in a shop.
type ShopInfoReply struct {
	GoodsList []GoodsInfo
}

// BuyGoodsRequest purchases shop goods.
type BuyGoodsRequest struct {
	GoodsID int64
	Num     int64
	Price   int64
}

// BuyGoodsReply is the purchase result.
type BuyGoodsReply struct {
	Goods     *GoodsInfo
	GetItems  []*corepb.Item
	CostItems []*corepb.Item
}

// GoodsUnlockNotify is a server push when goods unlock.
type GoodsUnlockNotify struct {
	GoodsList []GoodsInfo
}
