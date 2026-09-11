package game

import (
	"context"
	"math/rand/v2"
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

// FertilizeResult carries the container remaining seconds (the reply's
// fertilizer item count) and the reply lands (used by Both ripening to kick
// lands that reached mature).
type FertilizeResult struct {
	RemainingSecs int64
	HasRemaining  bool
	Lands         []logic.LandInfo
}

// Fertilize applies fertilizer one land at a time with 50ms spacing and
// returns the last decoded reply info (remaining seconds + reply lands).
func (a *API) Fertilize(ctx context.Context, landIDs []int64, fertilizerID int64) (FertilizeResult, error) {
	var out FertilizeResult
	if err := a.requireSender(); err != nil {
		return out, err
	}
	for i, landID := range landIDs {
		req := &plantpb.FertilizeRequest{
			LandIds:      []int64{landID},
			FertilizerId: fertilizerID,
		}
		raw, sendErr := a.sendPlant(ctx, "Fertilize", marshalMessage(req))
		if sendErr != nil {
			return out, sendErr
		}
		reply := &plantpb.FertilizeReply{}
		if err := unmarshalMessage(raw, reply); err == nil {
			// Presence mirrors prost Option: the message field being present on
			// the wire means "count carries the container remaining seconds"
			// (an explicit 0 is the out-of-fertilizer signal for UntilMature).
			if reply.Fertilizer != nil {
				out.RemainingSecs = reply.Fertilizer.Count
				out.HasRemaining = true
			}
			out.Lands = logic.LandsFromPlantPB(reply.Land)
		}
		if i+1 < len(landIDs) {
			select {
			case <-ctx.Done():
				return out, ctx.Err()
			case <-time.After(50 * time.Millisecond):
			}
		}
	}
	return out, nil
}

// Organic loop caps mirroring bot api.ts MAX_ORGANIC_FERTILIZE_OPERATIONS/ROUNDS:
// per-call limit = min(240, land count × 20).
const (
	MaxOrganicFertilizeOperations = 240
	MaxOrganicFertilizeRounds     = 20
)

func organicOperationLimit(landCount int) int {
	limit := MaxOrganicFertilizeRounds * landCount
	if limit > MaxOrganicFertilizeOperations {
		limit = MaxOrganicFertilizeOperations
	}
	return limit
}

// OrganicOperationLimit exposes the per-call organic cap for callers that want
// to warn when the loop stops at the cap (bot panel warning).
func OrganicOperationLimit(landCount int) int { return organicOperationLimit(landCount) }

// organicDelay mirrors bot randomDelay(1000,1500) between organic rounds.
func organicDelay(ctx context.Context) error {
	delay := time.Duration(1000+rand.IntN(500)) * time.Millisecond
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// landBecameRipe reports whether landID shows a ripe plant in the reply lands.
func landBecameRipe(lands []logic.LandInfo, landID int64) bool {
	for i := range lands {
		land := &lands[i]
		if land.ID != landID || land.Plant == nil {
			continue
		}
		current := logic.GetCurrentPhase(land.Plant.Phases)
		if current != nil && current.Phase == logic.PhaseMature {
			return true
		}
	}
	return false
}

// FertilizeOrganicLoop round-robins organic fertilizer (ID 1012) across landIDs
// until a Fertilize call fails or the per-call cap min(240, lands×20) is hit.
// Delay between rounds is 1–1.5s (bot). Returns success count and the last
// container remaining seconds from the replies.
func (a *API) FertilizeOrganicLoop(ctx context.Context, landIDs []int64) (int, int64, bool, error) {
	ids := make([]int64, 0, len(landIDs))
	for _, id := range landIDs {
		if id > 0 {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return 0, 0, false, nil
	}
	if err := a.requireSender(); err != nil {
		return 0, 0, false, err
	}

	operationLimit := organicOperationLimit(len(ids))
	successCount := 0
	var remainingSecs int64
	hasRemaining := false
	idx := 0
	for successCount < operationLimit {
		res, err := a.Fertilize(ctx, ids[idx:idx+1], OrganicFertilizerID)
		if err != nil {
			return successCount, remainingSecs, hasRemaining, nil
		}
		if res.HasRemaining {
			remainingSecs = res.RemainingSecs
			hasRemaining = true
		}
		successCount++
		idx = (idx + 1) % len(ids)
		if delayErr := organicDelay(ctx); delayErr != nil {
			return successCount, remainingSecs, hasRemaining, delayErr
		}
	}
	return successCount, remainingSecs, hasRemaining, nil
}

// FertilizeOrganicUntilMature is the desktop Both-mode ripening loop: cycle the
// given immature lands, kicking a land when it ripens or errors; stop when the
// fertilizer runs out (remaining 0s) or the shared per-call cap is reached.
func (a *API) FertilizeOrganicUntilMature(ctx context.Context, landIDs []int64) (int, int64, bool, error) {
	ids := make([]int64, 0, len(landIDs))
	for _, id := range landIDs {
		if id > 0 {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return 0, 0, false, nil
	}
	if err := a.requireSender(); err != nil {
		return 0, 0, false, err
	}

	operationLimit := organicOperationLimit(len(ids))
	successCount := 0
	var remainingSecs int64
	hasRemaining := false
	idx := 0
	for successCount < operationLimit && len(ids) > 0 {
		landID := ids[idx]
		res, err := a.Fertilize(ctx, []int64{landID}, OrganicFertilizerID)
		if err != nil {
			ids = append(ids[:idx], ids[idx+1:]...)
		} else {
			if res.HasRemaining {
				remainingSecs = res.RemainingSecs
				hasRemaining = true
			}
			successCount++
			if res.HasRemaining && res.RemainingSecs == 0 {
				break
			}
			if landBecameRipe(res.Lands, landID) {
				ids = append(ids[:idx], ids[idx+1:]...)
			} else {
				idx++
			}
		}
		if len(ids) == 0 {
			break
		}
		idx %= len(ids)
		if delayErr := organicDelay(ctx); delayErr != nil {
			return successCount, remainingSecs, hasRemaining, delayErr
		}
	}
	return successCount, remainingSecs, hasRemaining, nil
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

// PutPlantHooks lets callers sync operation limits and stop early between lands.
type PutPlantHooks struct {
	OnLimits func(limits []*plantpb.OperationLimit)
	Continue func() bool
}

// PutInsects puts insects one land at a time (matches game client / bot capture).
func (a *API) PutInsects(ctx context.Context, hostGID int64, landIDs []int64, hooks PutPlantHooks) (int, error) {
	return a.putPlantItems(ctx, landIDs, hooks, "PutInsects",
		func(landID int64) []byte {
			return marshalMessage(&plantpb.PutInsectsRequest{HostGid: hostGID, LandIds: []int64{landID}})
		},
		func(raw []byte) ([]*plantpb.LandInfo, []*plantpb.OperationLimit, error) {
			reply := &plantpb.PutInsectsReply{}
			if err := unmarshalMessage(raw, reply); err != nil {
				return nil, nil, err
			}
			return reply.Land, reply.OperationLimits, nil
		},
	)
}

// PutWeeds puts weeds one land at a time (matches game client / bot capture).
func (a *API) PutWeeds(ctx context.Context, hostGID int64, landIDs []int64, hooks PutPlantHooks) (int, error) {
	return a.putPlantItems(ctx, landIDs, hooks, "PutWeeds",
		func(landID int64) []byte {
			return marshalMessage(&plantpb.PutWeedsRequest{HostGid: hostGID, LandIds: []int64{landID}})
		},
		func(raw []byte) ([]*plantpb.LandInfo, []*plantpb.OperationLimit, error) {
			reply := &plantpb.PutWeedsReply{}
			if err := unmarshalMessage(raw, reply); err != nil {
				return nil, nil, err
			}
			return reply.Land, reply.OperationLimits, nil
		},
	)
}

func (a *API) putPlantItems(
	ctx context.Context,
	landIDs []int64,
	hooks PutPlantHooks,
	method string,
	encode func(landID int64) []byte,
	decode func(raw []byte) ([]*plantpb.LandInfo, []*plantpb.OperationLimit, error),
) (int, error) {
	ids := uniquePositiveIDs(landIDs)
	if len(ids) == 0 {
		return 0, nil
	}
	if err := a.requireSender(); err != nil {
		return 0, err
	}

	ok := 0
	for i, landID := range ids {
		if hooks.Continue != nil && !hooks.Continue() {
			break
		}
		raw, err := a.sendPlant(ctx, method, encode(landID))
		if err != nil {
			return ok, err
		}
		lands, limits, err := decode(raw)
		if err != nil {
			return ok, err
		}
		if hooks.OnLimits != nil && len(limits) > 0 {
			hooks.OnLimits(limits)
		}
		confirmed := false
		for _, land := range lands {
			if land != nil && land.Id == landID {
				confirmed = true
				break
			}
		}
		if confirmed {
			ok++
		}
		if i+1 < len(ids) {
			if hooks.Continue != nil && !hooks.Continue() {
				break
			}
			select {
			case <-ctx.Done():
				return ok, ctx.Err()
			case <-time.After(time.Duration(80+time.Now().UnixNano()%80) * time.Millisecond):
			}
		}
	}
	return ok, nil
}

func uniquePositiveIDs(ids []int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// CheckCanOperate used to hit PlantService.CheckCanOperate. The current game
// proto no longer exposes that RPC; quotas now come from OperationLimit on
// land replies. Callers treat this as fail-open.
func (a *API) CheckCanOperate(_ context.Context, _, _ int64) (bool, int64, error) {
	return true, 0, nil
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
