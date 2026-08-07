package game

import (
	"context"

	"github.com/MQEnergy/go-skeleton/internal/farm/proto/randomdroppb"
)

func (a *API) sendRandomDrop(ctx context.Context, method string, body []byte) ([]byte, error) {
	if err := a.requireSender(); err != nil {
		return nil, err
	}
	raw, _, err := a.Sender.Send(ctx, randomDropService, method, nonNilBody(body))
	return raw, err
}

// GetRandomDropActivityInfo fetches random-drop activity info.
func (a *API) GetRandomDropActivityInfo(ctx context.Context) (*randomdroppb.GetActivityInfoReply, error) {
	raw, err := a.sendRandomDrop(ctx, "GetActivityInfo", marshalMessage(&randomdroppb.GetActivityInfoRequest{}))
	if err != nil {
		return nil, err
	}
	reply := &randomdroppb.GetActivityInfoReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}
