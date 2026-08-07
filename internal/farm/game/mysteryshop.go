package game

import (
	"context"

	"github.com/it00021hot/qq-farm-core/internal/farm/proto/mysteryshoppb"
)

func (a *API) sendMysteryShop(ctx context.Context, method string, body []byte) ([]byte, error) {
	if err := a.requireSender(); err != nil {
		return nil, err
	}
	raw, _, err := a.Sender.Send(ctx, mysteryShopService, method, nonNilBody(body))
	return raw, err
}

// GetActiveNPC fetches the current mystery shop merchant offer.
func (a *API) GetActiveNPC(ctx context.Context) (*mysteryshoppb.GetActiveNPCReply, error) {
	req := &mysteryshoppb.GetActiveNPCRequest{}
	raw, err := a.sendMysteryShop(ctx, "GetActiveNPC", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &mysteryshoppb.GetActiveNPCReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}
