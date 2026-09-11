package game

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/farm/proto/mallpb"
)

const (
	OrganicMallGoodsID   int32 = 1002
	InorganicMallGoodsID int32 = 1003
)

// Mall fertilizer purchase caps mirroring bot mall.ts.
const (
	MallBuyMaxRounds = 100 // 单次最大轮次
	MallBuyPerRound  = 10  // 单批购买数
)

// MallFertilizerKind selects which fertilizer container to buy for.
type MallFertilizerKind int

const (
	FertilizerKindOrganic MallFertilizerKind = iota
	FertilizerKindNormal
)

// TypeName is the Chinese display name used in logs.
func (k MallFertilizerKind) TypeName() string {
	if k == FertilizerKindNormal {
		return "无机化肥"
	}
	return "有机化肥"
}

// MallGoodsID maps the kind to its mall goods id.
func (k MallFertilizerKind) MallGoodsID() int32 {
	if k == FertilizerKindNormal {
		return InorganicMallGoodsID
	}
	return OrganicMallGoodsID
}

func (a *API) sendMall(ctx context.Context, method string, body []byte) ([]byte, error) {
	if err := a.requireSender(); err != nil {
		return nil, err
	}
	raw, _, err := a.Sender.Send(ctx, mallService, method, nonNilBody(body))
	return raw, err
}

// GetMallList fetches mall goods for a slot type (sub-slot defaults to 0).
func (a *API) GetMallList(ctx context.Context, slotType int32) (*mallpb.GetMallListBySlotTypeResponse, error) {
	return a.GetMallListBySlot(ctx, slotType, 0)
}

// GetMallListBySlot fetches mall goods for a slot and sub-slot.
func (a *API) GetMallListBySlot(ctx context.Context, slotType, subSlotType int32) (*mallpb.GetMallListBySlotTypeResponse, error) {
	req := &mallpb.GetMallListBySlotTypeRequest{SlotType: slotType, SubSlotType: subSlotType}
	raw, err := a.sendMall(ctx, "GetMallListBySlotType", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &mallpb.GetMallListBySlotTypeResponse{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// GetMallGoods decodes mall goods from a slot type listing.
func (a *API) GetMallGoods(ctx context.Context, slotType int32) ([]*mallpb.MallGoods, error) {
	reply, err := a.GetMallList(ctx, slotType)
	if err != nil {
		return nil, err
	}
	return reply.GoodsList, nil
}

// Purchase buys mall goods.
func (a *API) Purchase(ctx context.Context, goodsID, count int32) (*mallpb.PurchaseResponse, error) {
	req := &mallpb.PurchaseRequest{GoodsId: goodsID, Count: count}
	raw, err := a.sendMall(ctx, "Purchase", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &mallpb.PurchaseResponse{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// IsInsufficientBalance reports whether err is the mall "not enough currency"
// business error (bot is_insufficient_balance: 余额不足/点券不足/code=1000019).
func IsInsufficientBalance(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "余额不足") ||
		strings.Contains(msg, "点券不足") ||
		strings.Contains(msg, "code=1000019")
}

// AutoBuyFertilizerViaMall buys fertilizer goods in batches (BUY_PER_ROUND=10,
// MAX_ROUNDS=100, 120ms spacing). targetCount<=0 means "until the caps or the
// balance runs out". Returns the bought count. Mirrors bot
// MallService.auto_buy_fertilizer_via_mall including the insufficient-balance
// downgrade (batch → 1 → pause for today, signalled by ErrMallBuyPausedNoGold).
func (a *API) AutoBuyFertilizerViaMall(ctx context.Context, kind MallFertilizerKind, targetCount int32) (int32, error) {
	goodsList, err := a.GetMallGoods(ctx, 1)
	if err != nil {
		return 0, err
	}
	target := kind.MallGoodsID()
	var goods *mallpb.MallGoods
	for _, g := range goodsList {
		if g != nil && g.GoodsId == target {
			goods = g
			break
		}
	}
	if goods == nil || goods.GoodsId <= 0 {
		return 0, nil
	}
	singlePrice := int64(0)
	if goods.Price != nil {
		singlePrice = goods.Price.Count
	}
	var totalBought int32
	perRound := int32(MallBuyPerRound)
	remainingToBuy := targetCount

	for round := 0; round < MallBuyMaxRounds; round++ {
		if targetCount > 0 && totalBought >= remainingToBuy {
			break
		}
		if singlePrice > 0 && perRound == 0 {
			return totalBought, ErrMallBuyPausedNoGold
		}
		buyCount := perRound
		if targetCount > 0 && buyCount > remainingToBuy-totalBought {
			buyCount = remainingToBuy - totalBought
		}
		if buyCount <= 0 {
			break
		}
		if _, buyErr := a.Purchase(ctx, goods.GoodsId, buyCount); buyErr != nil {
			if IsInsufficientBalance(buyErr) {
				if perRound > 1 {
					perRound = 1
					continue
				}
				return totalBought, ErrMallBuyPausedNoGold
			}
			return totalBought, buyErr
		}
		totalBought += buyCount
		// 模拟 sleep(120) 抗频控
		select {
		case <-ctx.Done():
			return totalBought, ctx.Err()
		case <-time.After(120 * time.Millisecond):
		}
	}
	return totalBought, nil
}

// ErrMallBuyPausedNoGold marks "balance exhausted, auto-buy paused for today".
var ErrMallBuyPausedNoGold = errors.New("mall: balance insufficient, buy paused")
