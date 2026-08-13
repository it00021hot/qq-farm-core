package logic

import (
	"fmt"
	"sort"

	"github.com/it00021hot/qq-farm-core/internal/farm/proto/corepb"
)

// UIBagItem is one merged bag row for the personal panel.
type UIBagItem struct {
	ID              int64  `json:"id"`
	Count           int64  `json:"count"`
	Name            string `json:"name"`
	Image           string `json:"image,omitempty"`
	Category        string `json:"category,omitempty"`
	ItemType        int64  `json:"itemType,omitempty"`
	Sellable        bool   `json:"sellable,omitempty"`
	SellStatus      string `json:"sellStatus,omitempty"`
	SellCondition   string `json:"sellCondition,omitempty"`
	PriceID         int64  `json:"priceId,omitempty"`
	Price           int64  `json:"price,omitempty"`
	PriceUnit       string `json:"priceUnit,omitempty"`
	Level           int64  `json:"level,omitempty"`
	InteractionType string `json:"interactionType,omitempty"`
	HoursText       string `json:"hoursText,omitempty"`
}

// BagUIResponse is returned by GET /farm/bag.
type BagUIResponse struct {
	TotalKinds    int              `json:"totalKinds"`
	Items         []UIBagItem      `json:"items"`
	OriginalItems []map[string]any `json:"originalItems"`
}

func priceUnit(priceID int64) string {
	switch priceID {
	case 1005:
		return "金豆豆"
	case 1002:
		return "点券"
	default:
		return "金"
	}
}

func bagItemName(id int64, info *ItemInfo) (name, category string) {
	if id == 1 || id == 1001 {
		return "金币", "gold"
	}
	if id == 1101 {
		return "经验", "exp"
	}
	if plant := GetPlantByFruitID(id); plant != nil {
		n := GetPlantName(plant.ID)
		if n == "" {
			n = "未知"
		}
		return n + "果实", "fruit"
	}
	if plant := GetPlantBySeedID(id); plant != nil {
		n := plant.Name
		if n == "" {
			n = "未知"
		}
		return n + "种子", "seed"
	}
	if info != nil && info.Name != "" {
		return info.Name, "item"
	}
	return fmt.Sprintf("物品%d", id), "item"
}

// FormatBagResponse merges raw bag items into UI rows.
func FormatBagResponse(rawItems []corepb.Item) BagUIResponse {
	original := make([]map[string]any, 0, len(rawItems))
	for _, it := range rawItems {
		if it.Id <= 0 || it.Count <= 0 {
			continue
		}
		original = append(original, map[string]any{
			"id": it.Id, "count": it.Count, "uid": it.Uid,
		})
	}

	merged := map[int64]*UIBagItem{}
	for _, it := range rawItems {
		if it.Id <= 0 || it.Count <= 0 {
			continue
		}
		info := GetItemByID(it.Id)
		name, category := bagItemName(it.Id, info)
		sellInfo := GetEffectiveSellInfo(info)
		var priceID, price int64
		if len(sellInfo.Sells) > 0 {
			priceID = sellInfo.Sells[0].CurrencyID
			price = sellInfo.Sells[0].Price
		}
		var itemType, level int64
		interactionType := ""
		if info != nil {
			itemType = info.Type
			if info.Level != nil {
				level = *info.Level
			}
			interactionType = info.InteractionType
		}
		row, ok := merged[it.Id]
		if !ok {
			row = &UIBagItem{
				ID: it.Id, Name: name, Image: SeedImagePath(it.Id), Category: category,
				ItemType: itemType, Sellable: sellInfo.Sellable,
				SellStatus: string(sellInfo.Status), SellCondition: sellInfo.Condition,
				PriceID: priceID, Price: price, PriceUnit: priceUnit(priceID),
				Level: level, InteractionType: interactionType,
			}
			merged[it.Id] = row
		}
		row.Count += it.Count
	}

	items := make([]UIBagItem, 0, len(merged))
	for _, row := range merged {
		if row.InteractionType == "fertilizerbucket" && row.Count > 0 {
			hours := float64(row.Count) / 3600
			hours = float64(int(hours*10)) / 10
			row.HoursText = fmt.Sprintf("%.1f小时", hours)
		}
		items = append(items, *row)
	}

	typePriority := map[int64]int{17: 0, 5: 1, 6: 2}
	sort.Slice(items, func(i, j int) bool {
		ti, tj := items[i].ItemType, items[j].ItemType
		pi, oki := typePriority[ti]
		pj, okj := typePriority[tj]
		if !oki {
			if ti > 0 {
				pi = 1000 + int(ti)
			} else {
				pi = int(^uint(0) >> 1)
			}
		}
		if !okj {
			if tj > 0 {
				pj = 1000 + int(tj)
			} else {
				pj = int(^uint(0) >> 1)
			}
		}
		if pi != pj {
			return pi < pj
		}
		if items[i].Count != items[j].Count {
			return items[i].Count > items[j].Count
		}
		return items[i].ID < items[j].ID
	})

	return BagUIResponse{
		TotalKinds: len(items), Items: items, OriginalItems: original,
	}
}
