package game

import (
	"context"

	"github.com/MQEnergy/go-skeleton/internal/farm/proto/mallpb"
)

const (
	OrganicMallGoodsID   int32 = 1002
	InorganicMallGoodsID int32 = 1003
)

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
