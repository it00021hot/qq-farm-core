package game

import (
	"context"

	"github.com/MQEnergy/go-skeleton/internal/farm/proto/dogpb"
)

func (a *API) sendDog(ctx context.Context, method string, body []byte) ([]byte, error) {
	if err := a.requireSender(); err != nil {
		return nil, err
	}
	raw, _, err := a.Sender.Send(ctx, dogService, method, nonNilBody(body))
	return raw, err
}

// GetDogInfo fetches dog info and protect items.
func (a *API) GetDogInfo(ctx context.Context) (*dogpb.GetDogInfoReply, error) {
	raw, err := a.sendDog(ctx, "GetDogInfo", (&dogpb.GetDogInfoRequest{}).Marshal())
	if err != nil {
		return nil, err
	}
	reply := &dogpb.GetDogInfoReply{}
	if err := reply.Unmarshal(raw); err != nil {
		return nil, err
	}
	return reply, nil
}
