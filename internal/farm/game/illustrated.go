package game

import (
	"context"

	"github.com/it00021hot/qq-farm-core/internal/farm/proto/illustratedpb"
)

// GetIllustratedListV2 fetches the illustrated book entries for a type
// (bot getIllustratedList; refresh stays false like the game client).
func (a *API) GetIllustratedListV2(ctx context.Context, bookType int32) (*illustratedpb.GetIllustratedListV2Reply, error) {
	req := &illustratedpb.GetIllustratedListV2Request{Refresh: false, Type: bookType}
	raw, err := a.sendIllustrated(ctx, "GetIllustratedListV2", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &illustratedpb.GetIllustratedListV2Reply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// GetIllustratedLevelListV2 fetches the level reward ladder for a type.
func (a *API) GetIllustratedLevelListV2(ctx context.Context, bookType int32) (*illustratedpb.GetIllustratedLevelListV2Reply, error) {
	req := &illustratedpb.GetIllustratedLevelListV2Request{Type: bookType}
	raw, err := a.sendIllustrated(ctx, "GetIllustratedLevelListV2", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &illustratedpb.GetIllustratedLevelListV2Reply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}
