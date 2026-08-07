package game

import (
	"context"

	"github.com/MQEnergy/go-skeleton/internal/farm/proto/paypb"
)

func (a *API) sendPay(ctx context.Context, method string, body []byte) ([]byte, error) {
	if err := a.requireSender(); err != nil {
		return nil, err
	}
	raw, _, err := a.Sender.Send(ctx, payService, method, nonNilBody(body))
	return raw, err
}

// GetRechargeInfo fetches recharge catalog info for a platform/version.
func (a *API) GetRechargeInfo(ctx context.Context, platform, version int64) (*paypb.GetRechargeInfoReply, error) {
	req := &paypb.GetRechargeInfoRequest{Platform: platform, Version: version}
	raw, err := a.sendPay(ctx, "GetRechargeInfo", req.Marshal())
	if err != nil {
		return nil, err
	}
	reply := &paypb.GetRechargeInfoReply{}
	if err := reply.Unmarshal(raw); err != nil {
		return nil, err
	}
	return reply, nil
}
