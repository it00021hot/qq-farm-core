package game

import (
	"context"

	"github.com/it00021hot/qq-farm-core/internal/farm/proto/shoppb"
)

func (a *API) sendShop(ctx context.Context, method string, body []byte) ([]byte, error) {
	if err := a.requireSender(); err != nil {
		return nil, err
	}
	raw, _, err := a.Sender.Send(ctx, shopService, method, nonNilBody(body))
	return raw, err
}

// ShopInfo fetches goods for a shop.
func (a *API) ShopInfo(ctx context.Context, shopID int64) (*shoppb.ShopInfoReply, error) {
	req := &shoppb.ShopInfoRequest{ShopId: shopID}
	raw, err := a.sendShop(ctx, "ShopInfo", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &shoppb.ShopInfoReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// BuyGoods purchases shop goods.
func (a *API) BuyGoods(ctx context.Context, goodsID, num, price int64) (*shoppb.BuyGoodsReply, error) {
	req := &shoppb.BuyGoodsRequest{
		GoodsId: goodsID,
		Num:     num,
		Price:   price,
	}
	raw, err := a.sendShop(ctx, "BuyGoods", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &shoppb.BuyGoodsReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}
