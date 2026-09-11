package activitycenter

// 图鉴 V2 快照（对齐 bot services/illustrated.ts）：
// 普通(1)/超变(2)两本，按官方客户端顺序串行请求，单飞合并并发读取。

import (
	"context"
	"sync"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/farm/game"
	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/illustratedpb"
)

// illustratedRewardDTO mirrors bot rewardDto.
func illustratedRewardDTO(reward *illustratedpb.IllustratedReward) map[string]any {
	if reward == nil || reward.ItemId <= 0 {
		return nil
	}
	name := ""
	if info := logic.GetItemByID(reward.ItemId); info != nil {
		name = info.Name
	}
	if name == "" {
		name = "TODO"
	}
	return map[string]any{
		"itemId": reward.ItemId,
		"count":  reward.Count,
		"name":   name,
		"image":  logic.SeedImagePath(reward.ItemId),
	}
}

// illustratedMutantGroup mirrors bot getMutantGroup.
func illustratedMutantGroup(seedID int64) string {
	info := logic.GlobalGameConfig.GetIllustratedConfig(seedID)
	if info == nil {
		return "gold"
	}
	switch info.Type {
	case "装扮果实":
		return "decoration"
	case "活动果实":
		return "activity"
	default:
		return "gold"
	}
}

// illustratedItemDTO mirrors bot itemDto.
func illustratedItemDTO(input *illustratedpb.IllustratedItem) map[string]any {
	seedID := input.GetSeedId()
	name := ""
	if info := logic.GetItemByID(seedID); info != nil {
		name = info.Name
	}
	if name == "" {
		if plant := logic.GetPlantBySeedID(seedID); plant != nil {
			name = plant.Name
		}
	}
	attributes := []map[string]any{}
	for _, attr := range input.GetAttributes() {
		if attr == nil {
			continue
		}
		if attr.Type == 0 && attr.Value == 0 {
			continue
		}
		attributes = append(attributes, map[string]any{
			"type": attr.Type, "param": attr.Param, "value": attr.Value,
		})
	}
	sortKey := int64(0)
	if cfg := logic.GlobalGameConfig.GetIllustratedConfig(seedID); cfg != nil {
		sortKey = cfg.Sort
	}
	return map[string]any{
		"seedId":        seedID,
		"name":          name,
		"image":         logic.SeedImagePath(seedID),
		"rewardCategory": input.GetRewardCategory(),
		"group":         illustratedMutantGroup(seedID),
		"sort":          sortKey,
		"cropCategory":  input.GetCropCategory(),
		"unlocked":      input.GetUnlocked(),
		"progress":      input.GetProgress(),
		"isNew":         input.GetIsNew(),
		"reward":        illustratedRewardDTO(input.GetReward()),
		"attributes":    attributes,
	}
}

// illustratedBuffDTO mirrors bot illustratedBuffDto.
func illustratedBuffDTO(entry logic.BuffConfigItem) map[string]any {
	valueType := "quantity"
	if entry.AttrValue > 10 {
		valueType = "probability"
	}
	return map[string]any{
		"id": entry.ID, "level": entry.SourceParam, "name": entry.AttrID,
		"value": entry.AttrValue, "valueType": valueType,
	}
}

// normalizeIllustratedBook mirrors bot normalizeBook.
func normalizeIllustratedBook(bookType int32, listReply *illustratedpb.GetIllustratedListV2Reply, levelReply *illustratedpb.GetIllustratedLevelListV2Reply) map[string]any {
	levels := []map[string]any{}
	for _, entry := range levelReply.GetLevels() {
		if entry == nil {
			continue
		}
		rewards := []map[string]any{}
		for _, reward := range entry.GetRewards() {
			if dto := illustratedRewardDTO(reward); dto != nil {
				rewards = append(rewards, dto)
			}
		}
		levels = append(levels, map[string]any{
			"level": entry.GetLevel(), "progress": entry.GetProgress(),
			"claimed": entry.GetClaimed(), "rewards": rewards,
		})
	}
	currentLevel := listReply.GetLevel()
	if levelReply.GetLevel() > currentLevel {
		currentLevel = levelReply.GetLevel()
	}
	currentProgress := listReply.GetProgress()
	if levelReply.GetProgress() > currentProgress {
		currentProgress = levelReply.GetProgress()
	}
	nextProgress := listReply.GetNextLevelProgress()
	if nextProgress == 0 {
		for _, entry := range levels {
			if lvl, _ := entry["level"].(int32); lvl > listReply.GetLevel() {
				if p, _ := entry["progress"].(int64); p > nextProgress {
					nextProgress = p
				}
			}
		}
	}
	attributeBonuses := []map[string]any{}
	for _, bonus := range listReply.GetAttributeBonuses() {
		if dto := illustratedRewardDTO(bonus); dto != nil {
			attributeBonuses = append(attributeBonuses, dto)
		}
	}
	items := make([]map[string]any, 0, len(listReply.GetItems()))
	for _, item := range listReply.GetItems() {
		if item != nil {
			items = append(items, illustratedItemDTO(item))
		}
	}
	// 按 sort 升序（稳定）。
	for i := 1; i < len(items); i++ {
		for j := i; j > 0; j-- {
			a, _ := items[j-1]["sort"].(int64)
			b, _ := items[j]["sort"].(int64)
			if a > b {
				items[j-1], items[j] = items[j], items[j-1]
			} else {
				break
			}
		}
	}
	var buffs, currentBuffs []map[string]any
	if bookType == 2 {
		for _, entry := range logic.GlobalGameConfig.IllustratedBuffs() {
			buffs = append(buffs, illustratedBuffDTO(entry))
		}
		for _, entry := range logic.GlobalGameConfig.IllustratedBuffsByLevel(int64(currentLevel)) {
			currentBuffs = append(currentBuffs, illustratedBuffDTO(entry))
		}
	}
	if buffs == nil {
		buffs = []map[string]any{}
	}
	if currentBuffs == nil {
		currentBuffs = []map[string]any{}
	}
	return map[string]any{
		"type":              bookType,
		"level":             currentLevel,
		"progress":          currentProgress,
		"nextLevelProgress": nextProgress,
		"currentBonus":      illustratedRewardDTO(listReply.GetCurrentBonus()),
		"attributeBonuses":  attributeBonuses,
		"buffs":             buffs,
		"currentBuffs":      currentBuffs,
		"items":             items,
		"levels":            levels,
	}
}

// single-flight snapshot (bot pendingSnapshot).
var illustratedMu sync.Mutex
var illustratedInFlight chan illustratedSnapResult

type illustratedSnapResult struct {
	value map[string]any
	err   error
}

// BuildIllustratedSnapshot builds the crop + mutant books.
func BuildIllustratedSnapshot(ctx context.Context, api *game.API) (map[string]any, error) {
	illustratedMu.Lock()
	if illustratedInFlight != nil {
		ch := illustratedInFlight
		illustratedMu.Unlock()
		res := <-ch
		return res.value, res.err
	}
	ch := make(chan illustratedSnapResult, 1)
	illustratedInFlight = ch
	illustratedMu.Unlock()

	value, err := buildIllustratedOnce(ctx, api)
	illustratedMu.Lock()
	illustratedInFlight = nil
	illustratedMu.Unlock()
	ch <- illustratedSnapResult{value: value, err: err}
	return value, err
}

func buildIllustratedOnce(ctx context.Context, api *game.API) (map[string]any, error) {
	// 按官方客户端顺序串行请求：normal 优先级并发槽有限，一次性并发 4 个请求
	// 会把自己的请求挤进队列并触发网关压力告警。
	cropList, err := api.GetIllustratedListV2(ctx, 1)
	if err != nil {
		return nil, err
	}
	cropLevels, err := api.GetIllustratedLevelListV2(ctx, 1)
	if err != nil {
		return nil, err
	}
	mutantList, err := api.GetIllustratedListV2(ctx, 2)
	if err != nil {
		return nil, err
	}
	mutantLevels, err := api.GetIllustratedLevelListV2(ctx, 2)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"crop":     normalizeIllustratedBook(1, cropList, cropLevels),
		"mutant":   normalizeIllustratedBook(2, mutantList, mutantLevels),
		"updatedAt": time.Now().UnixMilli(),
	}, nil
}
