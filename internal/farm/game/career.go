package game

import (
	"context"

	"github.com/MQEnergy/go-skeleton/internal/farm/proto/careerpb"
)

func (a *API) sendCareer(ctx context.Context, method string, body []byte) ([]byte, error) {
	if err := a.requireSender(); err != nil {
		return nil, err
	}
	raw, _, err := a.Sender.Send(ctx, careerService, method, nonNilBody(body))
	return raw, err
}

// CareerInfoGet fetches career/lifetime info.
func (a *API) CareerInfoGet(ctx context.Context) (*careerpb.CareerInfoGetReply, error) {
	raw, err := a.sendCareer(ctx, "CareerInfoGet", marshalMessage(&careerpb.CareerInfoGetRequest{}))
	if err != nil {
		return nil, err
	}
	reply := &careerpb.CareerInfoGetReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}
