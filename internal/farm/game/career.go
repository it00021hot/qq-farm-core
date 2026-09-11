package game

import (
	"context"

	"github.com/it00021hot/qq-farm-core/internal/farm/proto/careerpb"
)

func (a *API) sendCareer(ctx context.Context, method string, body []byte) ([]byte, error) {
	if err := a.requireSender(); err != nil {
		return nil, err
	}
	raw, _, err := a.Sender.Send(ctx, careerService, method, nonNilBody(body))
	return raw, err
}

// CareerInfoGet fetches the account's own career/lifetime info.
func (a *API) CareerInfoGet(ctx context.Context) (*careerpb.CareerInfoGetReply, error) {
	return a.CareerInfoGetForGID(ctx, 0)
}

// CareerInfoGetForGID fetches career/lifetime info for the given GID.
func (a *API) CareerInfoGetForGID(ctx context.Context, gid int64) (*careerpb.CareerInfoGetReply, error) {
	raw, err := a.sendCareer(ctx, "CareerInfoGet", marshalMessage(&careerpb.CareerInfoGetRequest{Gid: gid}))
	if err != nil {
		return nil, err
	}
	reply := &careerpb.CareerInfoGetReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}
