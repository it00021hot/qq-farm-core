package game

import (
	"context"

	"github.com/it00021hot/qq-farm-core/internal/farm/proto/avatarframepb"
)

func (a *API) sendAvatarFrame(ctx context.Context, method string, body []byte) ([]byte, error) {
	if err := a.requireSender(); err != nil {
		return nil, err
	}
	raw, _, err := a.Sender.Send(ctx, avatarFrameService, method, nonNilBody(body))
	return raw, err
}

// AvatarFramesOwned fetches owned avatar frames.
func (a *API) AvatarFramesOwned(ctx context.Context) (*avatarframepb.AvatarFramesOwnedReply, error) {
	raw, err := a.sendAvatarFrame(ctx, "AvatarFramesOwned", marshalMessage(&avatarframepb.AvatarFramesOwnedRequest{}))
	if err != nil {
		return nil, err
	}
	reply := &avatarframepb.AvatarFramesOwnedReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}
