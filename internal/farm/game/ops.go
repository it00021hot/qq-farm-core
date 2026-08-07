package game

import (
	"context"
	"time"

	"github.com/MQEnergy/go-skeleton/internal/farm/logic"
	"github.com/MQEnergy/go-skeleton/internal/farm/proto/plantpb"
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
		LandIDs: landIDs,
		HostGID: a.GID,
		IsAll:   true,
	}
	raw, err := a.sendPlant(ctx, "Harvest", req.Marshal())
	if err != nil {
		return nil, err
	}
	reply := &plantpb.HarvestReply{}
	if err := reply.Unmarshal(raw); err != nil {
		return nil, err
	}
	return plantpb.LandsToLogic(reply.Land), nil
}

// Farming weeds and removes bugs on the given lands.
func (a *API) Farming(ctx context.Context, landIDs []int64) error {
	req := &plantpb.FarmingRequest{LandIDs: landIDs, HostGID: a.GID}
	return a.send(ctx, "Farming", req.Marshal())
}

// WaterLand waters the given lands.
func (a *API) WaterLand(ctx context.Context, landIDs []int64) error {
	req := &plantpb.WaterLandRequest{LandIDs: landIDs, HostGID: a.GID}
	return a.send(ctx, "WaterLand", req.Marshal())
}

// Fertilize applies fertilizer one land at a time with 50ms spacing.
func (a *API) Fertilize(ctx context.Context, landIDs []int64, fertilizerID int64) (successCount int, err error) {
	if err := a.requireSender(); err != nil {
		return 0, err
	}
	for i, landID := range landIDs {
		req := &plantpb.FertilizeRequest{
			LandIDs:      []int64{landID},
			FertilizerID: fertilizerID,
		}
		if sendErr := a.send(ctx, "Fertilize", req.Marshal()); sendErr != nil {
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
			LandIDs:      []int64{ids[idx]},
			FertilizerID: OrganicFertilizerID,
		}
		if sendErr := a.send(ctx, "Fertilize", req.Marshal()); sendErr != nil {
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
		Items: []plantpb.PlantItem{{SeedID: seedID, LandIDs: landIDs}},
	}
	raw, err := a.sendPlant(ctx, "Plant", req.Marshal())
	if err != nil {
		return nil, err
	}
	reply := &plantpb.PlantReply{}
	if err := reply.Unmarshal(raw); err != nil {
		return nil, err
	}
	return plantpb.LandsToLogic(reply.Land), nil
}

// RemovePlant removes plants from the given lands.
func (a *API) RemovePlant(ctx context.Context, landIDs []int64) error {
	req := &plantpb.RemovePlantRequest{LandIDs: landIDs}
	return a.send(ctx, "RemovePlant", req.Marshal())
}

// UnlockLand unlocks a land plot.
func (a *API) UnlockLand(ctx context.Context, landID int64, doShared bool) error {
	req := &plantpb.UnlockLandRequest{LandID: landID, DoShared: doShared}
	return a.send(ctx, "UnlockLand", req.Marshal())
}

// UpgradeLand upgrades a land plot.
func (a *API) UpgradeLand(ctx context.Context, landID int64) error {
	req := &plantpb.UpgradeLandRequest{LandID: landID}
	return a.send(ctx, "UpgradeLand", req.Marshal())
}

// PutInsects puts insects on the given lands (own or friend farm via HostGID).
func (a *API) PutInsects(ctx context.Context, hostGID int64, landIDs []int64) error {
	req := &plantpb.PutInsectsRequest{HostGID: hostGID, LandIDs: landIDs}
	return a.send(ctx, "PutInsects", req.Marshal())
}

// PutWeeds puts weeds on the given lands (own or friend farm via HostGID).
func (a *API) PutWeeds(ctx context.Context, hostGID int64, landIDs []int64) error {
	req := &plantpb.PutWeedsRequest{HostGID: hostGID, LandIDs: landIDs}
	return a.send(ctx, "PutWeeds", req.Marshal())
}

// CheckCanOperate checks whether an operation is allowed on a farm.
func (a *API) CheckCanOperate(ctx context.Context, hostGID, operationID int64) (bool, int64, error) {
	req := &plantpb.CheckCanOperateRequest{HostGID: hostGID, OperationID: operationID}
	raw, err := a.sendPlant(ctx, "CheckCanOperate", req.Marshal())
	if err != nil {
		return false, 0, err
	}
	reply := &plantpb.CheckCanOperateReply{}
	if err := reply.Unmarshal(raw); err != nil {
		return false, 0, err
	}
	return reply.CanOperate, reply.CanStealNum, nil
}

// PutSocialItem places a social item (e.g. friendship fruit) on a friend's land.
func (a *API) PutSocialItem(ctx context.Context, hostGID, landID, itemID int64) (*plantpb.PutSocialItemReply, error) {
	req := &plantpb.PutSocialItemRequest{HostGID: hostGID, LandID: landID, ItemID: itemID}
	raw, err := a.sendPlant(ctx, "PutSocialItem", req.Marshal())
	if err != nil {
		return nil, err
	}
	reply := &plantpb.PutSocialItemReply{}
	if err := reply.Unmarshal(raw); err != nil {
		return nil, err
	}
	return reply, nil
}
