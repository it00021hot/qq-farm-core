package game

import (
	"context"

	"github.com/MQEnergy/go-skeleton/internal/farm/proto/marqueepb"
)

func (a *API) sendMarquee(ctx context.Context, method string, body []byte) ([]byte, error) {
	if err := a.requireSender(); err != nil {
		return nil, err
	}
	raw, _, err := a.Sender.Send(ctx, marqueeService, method, nonNilBody(body))
	return raw, err
}

// GetMarquee fetches marquee / scrolling messages.
func (a *API) GetMarquee(ctx context.Context) (*marqueepb.GetMarqueeReply, error) {
	raw, err := a.sendMarquee(ctx, "GetMarquee", (&marqueepb.GetMarqueeRequest{}).Marshal())
	if err != nil {
		return nil, err
	}
	reply := &marqueepb.GetMarqueeReply{}
	if err := reply.Unmarshal(raw); err != nil {
		return nil, err
	}
	return reply, nil
}
