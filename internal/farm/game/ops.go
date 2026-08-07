package game

import (
	"context"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/plantpb"
)

func (a *API) send(ctx context.Context, method string, body []byte) error {
	if err := a.requireSender(); err != nil {
		return err
	}
	_, _, err := a.Sender.Send(ctx, plantService, method, nonNilBody(body))
	return err
}

func (a *API) sendPlant(ctx context.Context, method string, body []byte) ([]byte, error) {
	if err := a.requireSender(); err != nil {
		return nil, err
	}
	raw, _, err := a.Sender.Send(ctx, plantService, method, nonNilBody(body))
	return raw, err
}

// Harvest harvests the given land IDs and returns decoded reply lands.
func (a *API) Harvest(ctx context.Context, landIDs []int64) ([]logic.LandInfo, error) {
	req := &plantpb.HarvestRequest{
		LandIds: landIDs,
		HostGid: a.GID,
		IsAll:   true,
	}
	raw, err := a.sendPlant(ctx, "Harvest", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &plantpb.HarvestReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return logic.LandsFromPlantPB(reply.Land), nil
}

// Farming weeds and removes bugs on the given lands.
func (a *API) Farming(ctx context.Context, landIDs []int64) error {
	req := &plantpb.FarmingRequest{LandIds: landIDs, HostGid: a.GID}
	return a.send(ctx, "Farming", marshalMessage(req))
}

// WaterLand waters the given lands.
func (a *API) WaterLand(ctx context.Context, landIDs []int64) error {
	req := &plantpb.WaterLandRequest{LandIds: landIDs, HostGid: a.GID}
	return a.send(ctx, "WaterLand", marshalMessage(req))
}

// Fertilize applies fertilizer one land at a time with 50ms spacing.
func (a *API) Fertilize(ctx context.Context, landIDs []int64, fertilizerID int64) (successCount int, err error) {
	if err := a.requireSender(); err != nil {
		return 0, err
	}
	for i, landID := range landIDs {
		req := &plantpb.FertilizeRequest{
			LandIds:      []int64{landID},
			FertilizerId: fertilizerID,
		}
		if sendErr := a.send(ctx, "Fertilize", marshalMessage(req)); sendErr != nil {
			return successCount, sendErr
		}
		successCount++
		if i+1 < len(landIDs) {
			select {
			case <-ctx.Done():
				return successCount, ctx.Err()
			case <-time.After(50 * time.Millisecond):
			}
		}
	}
	return successCount, nil
}

// FertilizeOrganicLoop round-robins organic fertilizer (ID 1012) across landIDs
// until a Fertilize call fails. Delay between rounds is ~1.2s (bot: 1–1.5s).
func (a *API) FertilizeOrganicLoop(ctx context.Context, landIDs []int64) (int, error) {
	ids := make([]int64, 0, len(landIDs))
	for _, id := range landIDs {
		if id > 0 {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return 0, nil
	}
	if err := a.requireSender(); err != nil {
		return 0, err
	}

	successCount := 0
	idx := 0
	for {
		req := &plantpb.FertilizeRequest{
			LandIds:      []int64{ids[idx]},
			FertilizerId: OrganicFertilizerID,
		}
		if sendErr := a.send(ctx, "Fertilize", marshalMessage(req)); sendErr != nil {
			return successCount, nil
		}
		successCount++
		idx = (idx + 1) % len(ids)
		select {
		case <-ctx.Done():
			return successCount, ctx.Err()
		case <-time.After(1200 * time.Millisecond):
		}
	}
}

// Plant sows seedID on the given lands and returns decoded reply lands.
func (a *API) Plant(ctx context.Context, seedID int64, landIDs []int64) ([]logic.LandInfo, error) {
	req := &plantpb.PlantRequest{
		Items: []*plantpb.PlantItem{{SeedId: seedID, LandIds: landIDs}},
	}
	raw, err := a.sendPlant(ctx, "Plant", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &plantpb.PlantReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return logic.LandsFromPlantPB(reply.Land), nil
}

// RemovePlant removes plants from the given lands.
func (a *API) RemovePlant(ctx context.Context, landIDs []int64) error {
	req := &plantpb.RemovePlantRequest{LandIds: landIDs}
	return a.send(ctx, "RemovePlant", marshalMessage(req))
}

// UnlockLand unlocks a land plot.
func (a *API) UnlockLand(ctx context.Context, landID int64, doShared bool) error {
	req := &plantpb.UnlockLandRequest{LandId: landID, DoShared: doShared}
	return a.send(ctx, "UnlockLand", marshalMessage(req))
}

// UpgradeLand upgrades a land plot.
func (a *API) UpgradeLand(ctx context.Context, landID int64) error {
	req := &plantpb.UpgradeLandRequest{LandId: landID}
	return a.send(ctx, "UpgradeLand", marshalMessage(req))
}

// PutInsects puts insects on the given lands (own or friend farm via HostGID).
func (a *API) PutInsects(ctx context.Context, hostGID int64, landIDs []int64) (*plantpb.PutInsectsReply, error) {
	req := &plantpb.PutInsectsRequest{HostGid: hostGID, LandIds: landIDs}
	raw, err := a.sendPlant(ctx, "PutInsects", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &plantpb.PutInsectsReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// PutWeeds puts weeds on the given lands (own or friend farm via HostGID).
func (a *API) PutWeeds(ctx context.Context, hostGID int64, landIDs []int64) (*plantpb.PutWeedsReply, error) {
	req := &plantpb.PutWeedsRequest{HostGid: hostGID, LandIds: landIDs}
	raw, err := a.sendPlant(ctx, "PutWeeds", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &plantpb.PutWeedsReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// CheckCanOperate checks whether an operation is allowed on a farm.
func (a *API) CheckCanOperate(ctx context.Context, hostGID, operationID int64) (bool, int64, error) {
	req := &plantpb.CheckCanOperateRequest{HostGid: hostGID, OperationId: operationID}
	raw, err := a.sendPlant(ctx, "CheckCanOperate", marshalMessage(req))
	if err != nil {
		return false, 0, err
	}
	reply := &plantpb.CheckCanOperateReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return false, 0, err
	}
	return reply.CanOperate, reply.CanStealNum, nil
}

// PutSocialItem places a social item (e.g. friendship fruit) on a friend's land.
func (a *API) PutSocialItem(ctx context.Context, hostGID, landID, itemID int64) (*plantpb.PutSocialItemReply, error) {
	req := &plantpb.PutSocialItemRequest{HostGid: hostGID, LandId: landID, ItemId: itemID}
	raw, err := a.sendPlant(ctx, "PutSocialItem", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &plantpb.PutSocialItemReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}
