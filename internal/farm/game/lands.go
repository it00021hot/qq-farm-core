package game

import (
	"context"

	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/plantpb"
)

func nonNilBody(b []byte) []byte {
	if b == nil {
		return []byte{}
	}
	return b
}

// AllLands fetches all lands and decodes the reply.
func (a *API) AllLands(ctx context.Context) ([]logic.LandInfo, *plantpb.AllLandsReply, error) {
	reply, _, err := a.allLands(ctx)
	if err != nil {
		return nil, nil, err
	}
	return logic.LandsFromPlantPB(reply.Lands), reply, nil
}

// AllLandsRaw returns the decoded reply and raw response body.
func (a *API) AllLandsRaw(ctx context.Context) (*plantpb.AllLandsReply, []byte, error) {
	return a.allLands(ctx)
}

func (a *API) allLands(ctx context.Context) (*plantpb.AllLandsReply, []byte, error) {
	if err := a.requireSender(); err != nil {
		return nil, nil, err
	}
	req := &plantpb.AllLandsRequest{HostGid: a.GID}
	body := nonNilBody(marshalMessage(req))
	raw, _, err := a.Sender.Send(ctx, plantService, "AllLands", body)
	if err != nil {
		return nil, nil, err
	}
	reply := &plantpb.AllLandsReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, raw, err
	}
	return reply, raw, nil
}
