package game

import (
	"context"

	"github.com/it00021hot/qq-farm-core/internal/farm/proto/paypb"
)

const DefaultRechargeSource = "MallUI"

func (a *API) sendPay(ctx context.Context, method string, body []byte) ([]byte, error) {
	if err := a.requireSender(); err != nil {
		return nil, err
	}
	raw, _, err := a.Sender.Send(ctx, payService, method, nonNilBody(body))
	return raw, err
}

// GetRechargeInfo fetches recharge catalog info for a source such as "MallUI".
func (a *API) GetRechargeInfo(ctx context.Context, source string) (*paypb.GetRechargeInfoReply, error) {
	if source == "" {
		source = DefaultRechargeSource
	}
	req := &paypb.GetRechargeInfoRequest{Source: source}
	raw, err := a.sendPay(ctx, "GetRechargeInfo", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &paypb.GetRechargeInfoReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// GetDiamondBalance returns the first recharge info balance (diamond).
func (a *API) GetDiamondBalance(ctx context.Context) (int64, error) {
	reply, err := a.GetRechargeInfo(ctx, DefaultRechargeSource)
	if err != nil {
		return 0, err
	}
	if len(reply.RechargeInfos) == 0 || reply.RechargeInfos[0] == nil {
		return 0, nil
	}
	bal := reply.RechargeInfos[0].Balance
	if bal < 0 {
		return 0, nil
	}
	return bal, nil
}
