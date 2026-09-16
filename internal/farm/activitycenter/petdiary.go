package activitycenter

// 萌宠成长日记（完整对齐 bot services/activity-center/pet-diary.ts）。
// 操作码/选择器来自小程序 1.14.0.1 编码器：领养27 喂食29 抽奖30 互动记录31
// 手记32 刷新锦囊41 装备锦囊42 夺宝43 被夺记录44 开宝箱45 夺宝补偿46
// 好友信息47 领狗48 标记手记49 跳过战斗50 种子21(mega_event_claim_all) 兑换1(shop_buy)。

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/it00021hot/qq-farm-core/internal/farm/game"
	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/activitypb"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/corepb"
	"github.com/it00021hot/qq-farm-core/internal/vars"
)

const (
	PetDiaryGroupID int64 = 2026090100
	PetDiaryPetID   int64 = 2026090101
	PetDiarySeedsID int64 = 2026090102
	PetDiaryShopID  int64 = 2026090103
	petDiamondID    int64 = 1004

	operatePetQueryLog     int64 = 31
	operatePetPlunderedLog int64 = 44
	operatePetFriendInfo   int64 = 47
)

// petDiaryCommands maps action → operate type.
var petDiaryCommands = map[string]int64{
	"initialize": 27, "feed": 29, "draw": 30, "story": 32,
	"refreshCharm": 41, "equipCharm": 42, "battle": 43, "openTreasure": 45,
	"compensation": 46, "claimDog": 48, "markStories": 49, "skipBattle": 50,
	"seeds": 21, "exchange": 1,
}

type petDiaryCatalog struct {
	ActivityPetTreasureHuntBase []struct {
		GrowthAdultThreshold int64  `json:"growth_adult_threshold"`
		DailyFeedLimit       int64  `json:"daily_feed_limit"`
		DailyTreasureLimit   int64  `json:"daily_treasure_limit"`
		FeedItems            string `json:"feed_items"`
	}
	ActivityPetTreasureHuntFight []struct {
		DailyBattleLimit int64 `json:"daily_battle_limit"`
	}
	ActivityPetTreasureCharmRefresh []struct {
		FreeRefreshDailyLimit   int64 `json:"free_refresh_daily_limit"`
		ManualRefreshDailyLimit int64 `json:"manual_refresh_daily_limit"`
		ManualRefreshCostID     int64 `json:"manual_refresh_cost_id"`
		ManualRefreshCostCount  int64 `json:"manual_refresh_cost_count"`
	}
	ActivityPetTreasureHuntCharm []struct {
		CharmID   int64  `json:"charm_id"`
		Name      string `json:"name"`
		Desc      string `json:"desc"`
		ShortDesc string `json:"short_desc"`
		UseLimit  int64  `json:"use_limit"`
		IconPath  string `json:"icon_path"`
	}
}

var (
	petDiaryMu       sync.RWMutex
	petDiaryData     *petDiaryCatalog
	petDiaryAssetMap map[string]string
)

type petDiaryAssetEntry struct {
	Path string `json:"path"`
	File string `json:"file"`
}

func loadPetDiaryCatalog() (*petDiaryCatalog, map[string]string, error) {
	petDiaryMu.RLock()
	if petDiaryData != nil {
		data, assets := petDiaryData, petDiaryAssetMap
		petDiaryMu.RUnlock()
		return data, assets, nil
	}
	petDiaryMu.RUnlock()

	petDiaryMu.Lock()
	defer petDiaryMu.Unlock()
	if petDiaryData != nil {
		return petDiaryData, petDiaryAssetMap, nil
	}
	dir := filepath.Join(dataRootOrDefault(os.Getenv("FARM_DATA_ROOT")), "activity-data")
	raw, err := os.ReadFile(filepath.Join(dir, "pet-diary-2026090101.json"))
	if err != nil {
		return nil, nil, fmt.Errorf("pet diary catalog: %w", err)
	}
	var catalog petDiaryCatalog
	if err := json.Unmarshal(raw, &catalog); err != nil {
		return nil, nil, fmt.Errorf("pet diary catalog parse: %w", err)
	}
	assetRaw, err := os.ReadFile(filepath.Join(dir, "pet-diary-assets.json"))
	assets := map[string]string{}
	if err == nil {
		var entries []petDiaryAssetEntry
		if json.Unmarshal(assetRaw, &entries) == nil {
			for _, entry := range entries {
				assets[strings.TrimSuffix(entry.Path, "/spriteFrame")] = "/activity-assets/pet-diary/" + entry.File
			}
		}
	}
	petDiaryData, petDiaryAssetMap = &catalog, assets
	return &catalog, assets, nil
}

// dataRootOrDefault resolves the farm resource root. The desktop shell extracts
// resources under vars.BasePath and may launch with an unrelated working
// directory, so the fallback must never stay CWD-relative.
func dataRootOrDefault(env string) string {
	if env != "" {
		return env
	}
	return filepath.Join(vars.BasePath, "resource", "farm")
}

func petDiaryImage(assets map[string]string, path string) string {
	return assets[strings.TrimSuffix(path, "/spriteFrame")]
}

func petDiaryErr(message string) error {
	return activityError{Code: "PET_DIARY_UNAVAILABLE", Message: message}
}

func petDiaryHeadActive(head *activitypb.PetDiaryActivityHead) bool {
	if head == nil || head.StartTime <= 0 {
		return false
	}
	now := logic.GetServerTimeSec()
	return now >= head.StartTime && now <= head.EndTime
}

func readPetDiaryGroup(ctx context.Context, api *game.API) (pet, seeds, shop *activitypb.PetDiaryActivityData, err error) {
	reply, err := api.GetPetDiaryGroup(ctx, PetDiaryGroupID)
	if err != nil {
		return nil, nil, nil, err
	}
	group := reply.GetGroup()
	if group == nil || group.Head == nil || group.Head.Id != PetDiaryGroupID {
		return nil, nil, nil, petDiaryErr("服务端未返回萌宠成长日记活动")
	}
	for _, child := range group.Children {
		if child == nil || child.Head == nil {
			continue
		}
		switch child.Head.Id {
		case PetDiaryPetID:
			pet = child
		case PetDiarySeedsID:
			seeds = child
		case PetDiaryShopID:
			shop = child
		}
	}
	if pet == nil || pet.PetTreasureHunt == nil {
		return nil, nil, nil, petDiaryErr("服务端未返回萌宠养成状态")
	}
	return pet, seeds, shop, nil
}

func petBalances(ctx context.Context, api *game.API) (map[int64]int64, error) {
	bag, err := api.Bag(ctx)
	if err != nil {
		return nil, err
	}
	out := map[int64]int64{}
	for _, item := range game.GetBagItems(bag) {
		out[item.Id] += item.Count
	}
	return out, nil
}

func petCostsAvailable(costs []*corepb.Item, bag map[int64]int64, count int64) bool {
	if bag == nil || len(costs) == 0 {
		return false
	}
	totals := map[int64]int64{}
	for _, cost := range costs {
		if cost == nil {
			continue
		}
		if cost.Id == petDiamondID || cost.Id == 0 || cost.Count <= 0 {
			return false
		}
		totals[cost.Id] += cost.Count * count
	}
	for id, amount := range totals {
		if bag[id] < amount {
			return false
		}
	}
	return true
}

func petShopCosts(items []*activitypb.PetDiaryShopItemInfo) []*corepb.Item {
	out := make([]*corepb.Item, 0, len(items))
	for _, item := range items {
		if item != nil {
			out = append(out, &corepb.Item{Id: item.Id, Count: item.Count})
		}
	}
	return out
}

func petFeedCosts(catalog *petDiaryCatalog) []*corepb.Item {
	if len(catalog.ActivityPetTreasureHuntBase) == 0 {
		return nil
	}
	out := []*corepb.Item{}
	for _, entry := range strings.Split(catalog.ActivityPetTreasureHuntBase[0].FeedItems, ";") {
		parts := strings.SplitN(entry, ":", 2)
		if len(parts) != 2 {
			continue
		}
		id, _ := strconv.ParseInt(parts[0], 10, 64)
		count, _ := strconv.ParseInt(parts[1], 10, 64)
		out = append(out, &corepb.Item{Id: id, Count: count})
	}
	return out
}

func petItemRow(id, count int64) map[string]any {
	name := fmt.Sprintf("物品#%d", id)
	if info := logic.GetItemByID(id); info != nil && info.Name != "" {
		name = info.Name
	}
	return map[string]any{
		"id":    strconv.FormatInt(id, 10),
		"count": strconv.FormatInt(count, 10),
		"name":  name,
		"image": logic.SeedImagePath(id),
	}
}

func petShopItemRows(items []*activitypb.PetDiaryShopItemInfo) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if item != nil {
			out = append(out, petItemRow(item.Id, item.Count))
		}
	}
	return out
}

func petItemRows(items []*corepb.Item) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if item != nil {
			out = append(out, petItemRow(item.Id, item.Count))
		}
	}
	return out
}

func petBalanceRows(bag map[int64]int64) []map[string]any {
	ids := []int64{1028, 1029, 80101, 80102, 80103, 1002}
	out := make([]map[string]any, 0, len(ids))
	known := bag != nil
	for _, id := range ids {
		count := int64(0)
		if bag != nil {
			count = bag[id]
		}
		row := petItemRow(id, count)
		row["known"] = known
		out = append(out, row)
	}
	return out
}

func petTreasureDTO(t *activitypb.PetTreasureHuntTreasure) map[string]any {
	previews := []map[string]any{}
	for _, p := range t.GetBattlePreviews() {
		if p == nil {
			continue
		}
		previews = append(previews, map[string]any{
			"challengeId":      strconv.FormatInt(p.ChallengeItemId, 10),
			"canStart":         p.CanStart,
			"maxProfit":        petItemRow(p.MaxProfit.GetId(), p.MaxProfit.GetCount()),
			"maxLoss":          petItemRow(p.MaxLoss.GetId(), p.MaxLoss.GetCount()),
			"plunderableCount": strconv.FormatInt(p.PlunderableCount, 10),
		})
	}
	sourceCharms := []int64{}
	for _, id := range t.SourceCharmIds {
		sourceCharms = append(sourceCharms, int64(id))
	}
	return map[string]any{
		"id":              t.Id,
		"status":          t.Status,
		"item":            petItemRow(t.ItemId, t.Count),
		"protectedCount":  strconv.FormatInt(t.ProtectedCount, 10),
		"originalCount":   strconv.FormatInt(t.OriginalCount, 10),
		"maxCount":        strconv.FormatInt(t.MaxCount, 10),
		"startTime":       t.StartAt * 1000,
		"endTime":         t.EndAt * 1000,
		"createdTime":     t.CreatedAt * 1000,
		"sourceCharmIds":  sourceCharms,
		"plunderCount":    t.PlunderCount,
		"maxPlunderCount": t.MaxPlunderCount,
		"previews":        previews,
	}
}

func petCharmDTO(catalog *petDiaryCatalog, assets map[string]string, battle *activitypb.PetTreasureHuntBattle, charmID int64) map[string]any {
	name := fmt.Sprintf("锦囊 %d", charmID)
	desc, short, useLimit, image := "", "", int64(0), ""
	for i := range catalog.ActivityPetTreasureHuntCharm {
		cfg := &catalog.ActivityPetTreasureHuntCharm[i]
		if cfg.CharmID != charmID {
			continue
		}
		name = cfg.Name
		desc = cfg.Desc
		short = cfg.ShortDesc
		if short == "" {
			short = desc
		}
		useLimit = cfg.UseLimit
		image = petDiaryImage(assets, cfg.IconPath)
		break
	}
	remaining := []int64{}
	for _, e := range battle.GetCharmEffectRemainingCount() {
		if e != nil && int64(e.CharmId) == charmID {
			remaining = append(remaining, e.RemainingCount)
		}
	}
	return map[string]any{
		"id":               charmID,
		"name":             name,
		"description":      desc,
		"shortDescription": short,
		"useLimit":         useLimit,
		"image":            image,
		"remaining":        remaining,
	}
}

// snapshot single-flight (bot pendingRead).
var petDiarySnapshotMu sync.Mutex
var petDiarySnapshotInFlight = map[string]chan petDiarySnapResult{}

type petDiarySnapResult struct {
	value map[string]any
	err   error
}

// BuildPetDiary builds the panel snapshot with single-flight merge.
func BuildPetDiary(ctx context.Context, api *game.API) (map[string]any, error) {
	petDiarySnapshotMu.Lock()
	key := fmt.Sprintf("%p", api)
	if ch, ok := petDiarySnapshotInFlight[key]; ok {
		petDiarySnapshotMu.Unlock()
		res := <-ch
		return res.value, res.err
	}
	ch := make(chan petDiarySnapResult, 1)
	petDiarySnapshotInFlight[key] = ch
	petDiarySnapshotMu.Unlock()

	value, err := buildPetDiaryOnce(ctx, api)
	petDiarySnapshotMu.Lock()
	delete(petDiarySnapshotInFlight, key)
	petDiarySnapshotMu.Unlock()
	ch <- petDiarySnapResult{value: value, err: err}
	return value, err
}
