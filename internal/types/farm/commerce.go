package farm

// MallListReq requests one game mall tab for a running account.
type MallListReq struct {
	AccountID   uint64 `json:"accountId" query:"accountId" validate:"required"`
	SlotType    int32  `json:"slotType" query:"slotType" validate:"omitempty,min=1,max=100"`
	SubSlotType int32  `json:"subSlotType" query:"subSlotType" validate:"omitempty,min=0,max=100"`
}

// MallPurchaseReq purchases goods from the default game mall tab.
type MallPurchaseReq struct {
	AccountID uint64 `json:"accountId" validate:"required"`
	GoodsID   int32  `json:"goodsId" validate:"required,min=1"`
	Count     int32  `json:"count" validate:"required,min=1,max=9999"`
}

// CommerceAccountReq requests commerce data for a running account.
type CommerceAccountReq struct {
	AccountID uint64 `json:"accountId" query:"accountId" validate:"required"`
}

// CommerceItem is an item enriched with local game metadata.
type CommerceItem struct {
	ID     int64  `json:"id"`
	Count  int64  `json:"count"`
	Name   string `json:"name"`
	Image  string `json:"image"`
	Rarity int64  `json:"rarity"`
}

type PurchaseLimit struct {
	Type      int32  `json:"type"`
	Bought    int32  `json:"bought"`
	Max       int32  `json:"max"`
	Remaining *int32 `json:"remaining"`
}

type MallPrice struct {
	CommerceItem
	Balance *int64 `json:"balance"`
}

type MallGoods struct {
	ID              int32          `json:"id"`
	Name            string         `json:"name"`
	Type            int32          `json:"type"`
	Rewards         []CommerceItem `json:"rewards"`
	Price           MallPrice      `json:"price"`
	IsFree          bool           `json:"isFree"`
	Limit           *PurchaseLimit `json:"limit"`
	IsLimited       bool           `json:"isLimited"`
	DiscountText    string         `json:"discountText"`
	IsDiscounted    bool           `json:"isDiscounted"`
	DiscountEndTime int64          `json:"discountEndTime"`
	Available       bool           `json:"available"`
	Purchasable     bool           `json:"purchasable"`
}

type MallCurrency struct {
	CommerceItem
	BalanceKnown bool `json:"balanceKnown"`
}

type MallCatalog struct {
	SlotType         int32          `json:"slotType"`
	SubSlotType      int32          `json:"subSlotType"`
	ServerTime       int64          `json:"serverTime"`
	RefreshCountdown int64          `json:"refreshCountdown"`
	Currencies       []MallCurrency `json:"currencies"`
	Goods            []MallGoods    `json:"goods"`
}

type MallPurchase struct {
	GoodsID int32          `json:"goodsId"`
	Count   int32          `json:"count"`
	Rewards []CommerceItem `json:"rewards"`
	Limit   *PurchaseLimit `json:"limit"`
}

type MallPurchaseResult struct {
	Purchase MallPurchase `json:"purchase"`
	Catalog  MallCatalog  `json:"catalog"`
}

type MysteryNPC struct {
	ID              int64        `json:"id"`
	Reward          CommerceItem `json:"reward"`
	Stock           int32        `json:"stock"`
	Price           MallPrice    `json:"price"`
	OriginalPrice   int64        `json:"originalPrice"`
	DiscountPercent int32        `json:"discountPercent"`
}

type MysteryShop struct {
	Active     bool        `json:"active"`
	ServerTime int64       `json:"serverTime"`
	ActiveTime int64       `json:"activeTime,omitempty"`
	ExpireTime int64       `json:"expireTime,omitempty"`
	NPC        *MysteryNPC `json:"npc"`
}

type DiamondBalance struct {
	Diamond int64 `json:"diamond"`
}
