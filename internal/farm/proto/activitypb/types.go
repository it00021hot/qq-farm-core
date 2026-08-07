// Package activitypb provides hand-written activity service protobuf types.
package activitypb

const (
	OperateExchangeShop       int64 = 1
	OperateQueryShop          int64 = 7
	OperateLightConstellation int64 = 21
)

type ActivityItem struct {
	ItemID int64
	Count  int64
}

type StarSandGoods struct {
	GoodsID      int64
	Item         *ActivityItem
	Cost         *ActivityItem
	Status       int64
	Owned        bool
	SortOrder    int64
	Name         []byte
	ResourceJSON []byte
	Field10      int64
	Field11      int64
	Category     []byte
}

type StarSandGoodsList struct {
	Goods []StarSandGoods
}

type ConstellationNode struct {
	NodeID  int64
	Field2  bool
	Field3  bool
	Field4  bool
	Rewards []ActivityItem
}

type ConstellationData struct {
	Field1 int64
	Field2 int64
	Field3 int64
	Nodes  []ConstellationNode
}

type ActivityContent struct {
	ActivityID int64
	Type       int64
	Name       []byte
	BeginTime  int64
	EndTime    int64
}

type ActivityData struct {
	Activity      *ActivityContent
	Catalog       *StarSandGoodsList
	Constellation *ConstellationData
}

type QueryActivityRequest struct {
	ActivityID  int64
	OperateType int64
}

type ExchangeShopOperateParams struct {
	GoodsID int64
	Count   int64
}

type ExchangeShopRequest struct {
	ActivityID          int64
	OperateType         int64
	ExchangeShopOperate *ExchangeShopOperateParams
}

type OperateConstellationRequest struct {
	ActivityID  int64
	OperateType int64
}

type ActivityOperateReply struct {
	ActivityID  int64
	OperateType int64
	Data        *ActivityData
}

// ActiviesChangeNotify is a server push when activities change (proto typo preserved).
type ActiviesChangeNotify struct {
	Activities []ActivityContent
}
