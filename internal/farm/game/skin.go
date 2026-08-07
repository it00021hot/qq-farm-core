package game

import (
	"context"

	"github.com/it00021hot/qq-farm-core/internal/farm/proto/skinpb"
)

func (a *API) sendSkin(ctx context.Context, method string, body []byte) ([]byte, error) {
	if err := a.requireSender(); err != nil {
		return nil, err
	}
	raw, _, err := a.Sender.Send(ctx, skinService, method, nonNilBody(body))
	return raw, err
}

// SkinsOwned fetches owned skins.
func (a *API) SkinsOwned(ctx context.Context) (*skinpb.SkinsOwnedReply, error) {
	raw, err := a.sendSkin(ctx, "SkinsOwned", marshalMessage(&skinpb.SkinsOwnedRequest{}))
	if err != nil {
		return nil, err
	}
	reply := &skinpb.SkinsOwnedReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// SkinsEquipped fetches equipped skins.
func (a *API) SkinsEquipped(ctx context.Context) (*skinpb.SkinsEquippedReply, error) {
	raw, err := a.sendSkin(ctx, "SkinsEquipped", marshalMessage(&skinpb.SkinsEquippedRequest{}))
	if err != nil {
		return nil, err
	}
	reply := &skinpb.SkinsEquippedReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// Equip equips a skin into a slot.
func (a *API) Equip(ctx context.Context, skinID, slotType int64) (*skinpb.EquipReply, error) {
	req := &skinpb.EquipRequest{SkinId: skinID, SlotType: slotType}
	raw, err := a.sendSkin(ctx, "Equip", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &skinpb.EquipReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// MarkAsViewed marks skins as viewed.
func (a *API) MarkAsViewed(ctx context.Context, skinIDs []int64) (*skinpb.MarkAsViewedReply, error) {
	req := &skinpb.MarkAsViewedRequest{SkinIds: skinIDs}
	raw, err := a.sendSkin(ctx, "MarkAsViewed", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &skinpb.MarkAsViewedReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}
