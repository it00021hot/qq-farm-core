package game

import (
	"context"

	"github.com/MQEnergy/go-skeleton/internal/farm/proto/mutantpb"
)

func (a *API) sendMutant(ctx context.Context, method string, body []byte) ([]byte, error) {
	if err := a.requireSender(); err != nil {
		return nil, err
	}
	raw, _, err := a.Sender.Send(ctx, mutantService, method, nonNilBody(body))
	return raw, err
}

// ReadMutantBook reads the mutant book (empty request/reply).
func (a *API) ReadMutantBook(ctx context.Context) (*mutantpb.ReadMutantBookReply, error) {
	raw, err := a.sendMutant(ctx, "ReadMutantBook", (&mutantpb.ReadMutantBookRequest{}).Marshal())
	if err != nil {
		return nil, err
	}
	reply := &mutantpb.ReadMutantBookReply{}
	if err := reply.Unmarshal(raw); err != nil {
		return nil, err
	}
	return reply, nil
}
