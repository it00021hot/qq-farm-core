package game

import (
	"context"

	"github.com/it00021hot/qq-farm-core/internal/farm/proto/mutantpb"
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
	raw, err := a.sendMutant(ctx, "ReadMutantBook", marshalMessage(&mutantpb.ReadMutantBookRequest{}))
	if err != nil {
		return nil, err
	}
	reply := &mutantpb.ReadMutantBookReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}
