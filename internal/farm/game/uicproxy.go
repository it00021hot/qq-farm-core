package game

import (
	"context"

	"github.com/MQEnergy/go-skeleton/internal/farm/proto/uicproxypb"
)

func (a *API) sendUicProxy(ctx context.Context, method string, body []byte) ([]byte, error) {
	if err := a.requireSender(); err != nil {
		return nil, err
	}
	raw, _, err := a.Sender.Send(ctx, uicProxyService, method, nonNilBody(body))
	return raw, err
}

// BatchModerateText batch-moderates text items via UIC proxy.
func (a *API) BatchModerateText(ctx context.Context, items []*uicproxypb.TextItem) (*uicproxypb.BatchModerateTextReply, error) {
	req := &uicproxypb.BatchModerateTextRequest{Items: items}
	raw, err := a.sendUicProxy(ctx, "BatchModerateText", req.Marshal())
	if err != nil {
		return nil, err
	}
	reply := &uicproxypb.BatchModerateTextReply{}
	if err := reply.Unmarshal(raw); err != nil {
		return nil, err
	}
	return reply, nil
}
