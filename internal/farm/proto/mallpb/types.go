// Package mallpb provides hand-written mall service protobuf types (protoc unavailable).
// Source: internal/farm/proto/mallpb.proto
package mallpb

import "github.com/MQEnergy/go-skeleton/internal/farm/proto/corepb"

// GetMallListBySlotTypeRequest fetches mall goods for a slot type.
type GetMallListBySlotTypeRequest struct {
	SlotType int32
}

// GetMallListBySlotTypeResponse lists serialized mall goods.
type GetMallListBySlotTypeResponse struct {
	GoodsList [][]byte
	Timestamp int64
}

// MallGoods is one mall product (decoded from goods_list bytes).
type MallGoods struct {
	GoodsID   int32
	Name      string
	Type      int32
	ItemIDs   []byte
	Price     []byte
	IsFree    bool
	Limit     []byte
	IsLimited bool
	Discount  string
}

// PurchaseRequest buys mall goods.
type PurchaseRequest struct {
	GoodsID int32
	Count   int32
}

// PurchaseResponse is the purchase result.
type PurchaseResponse struct {
	GoodsID    int32
	Count      int32
	RewardInfo []byte
	Result     []byte
}

// GetMonthCardInfosRequest fetches month card info (stub).
type GetMonthCardInfosRequest struct{}

// MonthCardInfo is one month card entry (stub).
type MonthCardInfo struct {
	GoodsID  int32
	Reward   *corepb.Item
	CanClaim bool
}

// GetMonthCardInfosReply lists month cards (stub).
type GetMonthCardInfosReply struct {
	Infos []MonthCardInfo
}

// ClaimMonthCardRewardRequest claims month card reward (stub).
type ClaimMonthCardRewardRequest struct {
	GoodsID int32
}

// ClaimMonthCardRewardReply is the claim result (stub).
type ClaimMonthCardRewardReply struct {
	Items []*corepb.Item
}

// ProductsHasChangedNotify is a server push when mall products change.
type ProductsHasChangedNotify struct {
	SlotType int32
}

// NeedNotify is a server need notification.
type NeedNotify struct {
	NeedType int32
}
