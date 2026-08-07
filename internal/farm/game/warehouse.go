package game

import (
	"context"

	"github.com/MQEnergy/go-skeleton/internal/farm/logic"
	"github.com/MQEnergy/go-skeleton/internal/farm/proto/corepb"
	"github.com/MQEnergy/go-skeleton/internal/farm/proto/itempb"
)

func (a *API) sendItem(ctx context.Context, method string, body []byte) ([]byte, error) {
	if err := a.requireSender(); err != nil {
		return nil, err
	}
	raw, _, err := a.Sender.Send(ctx, itemService, method, nonNilBody(body))
	return raw, err
}

// Bag fetches the player bag.
func (a *API) Bag(ctx context.Context) (*itempb.BagReply, error) {
	raw, err := a.sendItem(ctx, "Bag", marshalMessage(&itempb.BagRequest{}))
	if err != nil {
		return nil, err
	}
	reply := &itempb.BagReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// Sell sells items from the bag.
func (a *API) Sell(ctx context.Context, items []corepb.Item) (*itempb.SellReply, error) {
	raw, err := a.sendItem(ctx, "Sell", marshalMessage(&itempb.SellRequest{Items: itemPointers(items)}))
	if err != nil {
		return nil, err
	}
	reply := &itempb.SellReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// Use uses a single item.
func (a *API) Use(ctx context.Context, itemID, count int64, landIDs []int64) (*itempb.UseReply, error) {
	req := &itempb.UseRequest{ItemId: itemID, Count: count, LandIds: landIDs}
	raw, err := a.sendItem(ctx, "Use", marshalMessage(req))
	if err != nil {
		return nil, err
	}
	reply := &itempb.UseReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// BatchUse uses multiple items at once.
func (a *API) BatchUse(ctx context.Context, items []corepb.Item) (*itempb.BatchUseReply, error) {
	raw, err := a.sendItem(ctx, "BatchUse", marshalMessage(&itempb.BatchUseRequest{Items: itemPointers(items)}))
	if err != nil {
		return nil, err
	}
	reply := &itempb.BatchUseReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// GetBagItems extracts flat item list from a BagReply.
func GetBagItems(reply *itempb.BagReply) []corepb.Item {
	if reply == nil || reply.ItemBag == nil {
		return nil
	}
	out := make([]corepb.Item, 0, len(reply.ItemBag.Items))
	for _, item := range reply.ItemBag.Items {
		if item == nil || item.Id <= 0 || item.Count <= 0 {
			continue
		}
		out = append(out, *item)
	}
	return out
}

// ItemsFromPointers converts decoded item pointers to values.
func ItemsFromPointers(items []*corepb.Item) []corepb.Item {
	out := make([]corepb.Item, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, *item)
	}
	return out
}

func itemPointers(items []corepb.Item) []*corepb.Item {
	out := make([]*corepb.Item, len(items))
	for i := range items {
		item := items[i]
		out[i] = &item
	}
	return out
}

// ExtractBagSeeds builds seed entries from bag items using game config.
func ExtractBagSeeds(items []corepb.Item) []logic.BagSeed {
	merged := make(map[int64]*logic.BagSeed)
	for _, item := range items {
		seedID := item.Id
		count := item.Count
		if seedID <= 0 || count <= 0 {
			continue
		}
		plant := logic.GetPlantBySeedID(seedID)
		if plant == nil {
			continue
		}
		cur, ok := merged[seedID]
		if !ok {
			requiredLevel := plant.LandLevelNeed
			if info := logic.GetItemByID(seedID); info != nil && info.Level != nil {
				requiredLevel = *info.Level
			}
			plantSize := int64(1)
			if plant.Size != nil && *plant.Size > 0 {
				plantSize = *plant.Size
			}
			name := plant.Name
			if name == "" {
				name = logic.GetPlantNameBySeedID(seedID)
			}
			cur = &logic.BagSeed{
				SeedID:        seedID,
				Name:          name,
				Count:         0,
				RequiredLevel: requiredLevel,
				PlantSize:     plantSize,
			}
			merged[seedID] = cur
		}
		cur.Count += count
	}
	out := make([]logic.BagSeed, 0, len(merged))
	for _, seed := range merged {
		out = append(out, *seed)
	}
	return out
}

// BagSeeds fetches the bag and returns seed entries.
func (a *API) BagSeeds(ctx context.Context) ([]logic.BagSeed, error) {
	reply, err := a.Bag(ctx)
	if err != nil {
		return nil, err
	}
	return ExtractBagSeeds(GetBagItems(reply)), nil
}
