package game

import (
	"context"

	"github.com/MQEnergy/go-skeleton/internal/farm/proto/rechargebonuspb"
)

func (a *API) sendRechargeBonus(ctx context.Context, method string, body []byte) ([]byte, error) {
	if err := a.requireSender(); err != nil {
		return nil, err
	}
	raw, _, err := a.Sender.Send(ctx, rechargeBonusService, method, nonNilBody(body))
	return raw, err
}

// GetConfig fetches recharge-bonus configuration.
func (a *API) GetConfig(ctx context.Context) (*rechargebonuspb.GetConfigReply, error) {
	raw, err := a.sendRechargeBonus(ctx, "GetConfig", marshalMessage(&rechargebonuspb.GetConfigRequest{}))
	if err != nil {
		return nil, err
	}
	reply := &rechargebonuspb.GetConfigReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}
