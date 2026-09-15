package logic

import (
	"fmt"
	"sort"
	"strings"

	"github.com/it00021hot/qq-farm-core/internal/farm/proto/corepb"
)

// 背包出售/使用结果的汇总文案（rust item_capture gain_display_name /
// format_gains 的 Go 侧移植，供 app 层 bag sell/use 的 summary 字段使用）。

// GainEntry is one aggregated id→count delta row (rust item_capture GainEntry).
type GainEntry struct {
	ID    int64 `json:"id"`
	Count int64 `json:"count"`
}

// GainDisplayName mirrors rust gain_display_name: 常用货币固定名，其次按
// 果实 ID 反查作物名，再按物品目录名，兜底 物品#ID。
func GainDisplayName(id int64) string {
	switch id {
	case 1, 1001:
		return "金币"
	case 2, 1101:
		return "经验"
	case 1002:
		return "点券"
	case 1005:
		return "金豆"
	}
	if plant := GetPlantByFruitID(id); plant != nil {
		if name := GetPlantName(plant.ID); name != "" {
			return name
		}
	}
	if item := GetItemByID(id); item != nil && item.Name != "" {
		return item.Name
	}
	return fmt.Sprintf("物品#%d", id)
}

// ItemDisplayName is the bot error-message name: `info?.name || 物品{id}`.
func ItemDisplayName(id int64) string {
	if item := GetItemByID(id); item != nil && item.Name != "" {
		return item.Name
	}
	return fmt.Sprintf("物品#%d", id)
}

// FormatGains mirrors rust format_gains: `白萝卜×12、金币×100`。
func FormatGains(entries []GainEntry) string {
	parts := make([]string, 0, len(entries))
	for _, e := range entries {
		parts = append(parts, fmt.Sprintf("%s×%d", GainDisplayName(e.ID), e.Count))
	}
	return strings.Join(parts, "、")
}

// AggregateGains sums items by id (ascending id order, zero counts dropped) —
// mirrors rust bag_sell/bag_use 的 BTreeMap 聚合。
func AggregateGains(items []*corepb.Item) []GainEntry {
	acc := make(map[int64]int64, len(items))
	for _, it := range items {
		if it == nil || it.Count <= 0 {
			continue
		}
		acc[it.Id] += it.Count
	}
	ids := make([]int64, 0, len(acc))
	for id := range acc {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	out := make([]GainEntry, 0, len(ids))
	for _, id := range ids {
		out = append(out, GainEntry{ID: id, Count: acc[id]})
	}
	return out
}
