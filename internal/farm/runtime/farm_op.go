package runtime

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/farm/game"
	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/corepb"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/itempb"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/plantpb"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/shoppb"
	"github.com/it00021hot/qq-farm-core/internal/farm/stats"
)

// FarmOperationOption customizes one farm operation.
type FarmOperationOption func(*farmOperationOptions)

// FarmLogEntry 对齐 rust infra/panel_log：一条面板日志（tag/event 为 rust
// PanelEvent 体系的中文 tag + snake_case key，前端 log-events.ts 按此渲染）。
type FarmLogEntry struct {
	Tag    string // 农场/收获/种植/施肥/仓库/解锁/升级/好友/系统/错误
	Event  string // farm_cycle / harvest_crop / plant_seed / fertilize / ...
	Msg    string
	Module string // farm / warehouse / friend / task / system（空默认 farm）
	IsWarn bool
}

type farmOperationOptions struct {
	accountID   uint64
	playerLevel int64
	gold        int64
	limitsSink  func([]*plantpb.OperationLimit)
	logSink     func(FarmLogEntry)
}

// emit 向面板发一条日志（无 sink 或消息为空时静默）。
func (o *farmOperationOptions) emit(entry FarmLogEntry) {
	if o == nil || o.logSink == nil || entry.Msg == "" {
		return
	}
	if entry.Module == "" {
		entry.Module = "farm"
	}
	o.logSink(entry)
}

// WithLogSink 注入面板日志出口（Session 侧桥接到 hub 广播 + 环形缓冲）。
func WithLogSink(sink func(FarmLogEntry)) FarmOperationOption {
	return func(o *farmOperationOptions) { o.logSink = sink }
}

// WithStatsAccount enables daily cn_farm_stats increments for this operation.
func WithStatsAccount(accountID uint64) FarmOperationOption {
	return func(o *farmOperationOptions) {
		o.accountID = accountID
	}
}

// WithPlayerState supplies the login state used to filter and afford seed-shop purchases.
func WithPlayerState(level, gold int64) FarmOperationOption {
	return func(o *farmOperationOptions) {
		o.playerLevel = level
		o.gold = gold
	}
}

// WithOperationLimitsSink feeds AllLands (and other) OperationLimits into friend help/steal state,
// matching bot setOperationLimitsCallback from farm AllLands.
func WithOperationLimitsSink(sink func([]*plantpb.OperationLimit)) FarmOperationOption {
	return func(o *farmOperationOptions) {
		o.limitsSink = sink
	}
}

func feedOperationLimits(opts farmOperationOptions, limits []*plantpb.OperationLimit) {
	if opts.limitsSink == nil || len(limits) == 0 {
		return
	}
	opts.limitsSink(limits)
}

// RunFarmOperation performs a manual or automated own-farm operation.
// Supported operations are all, cycle, harvest, clear, plant, upgrade, unlock,
// water, weed, bug, insecticide, fertilize, and remove.
func RunFarmOperation(ctx context.Context, api *game.API, cfg logic.AccountConfig, op string, opts ...FarmOperationOption) (hadWork bool, actions []string, lands []logic.LandInfo, err error) {
	options := farmOperationOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	if api == nil {
		return false, nil, nil, errors.New("farm API is unavailable")
	}
	switch op {
	case "all", "cycle", "harvest", "clear", "plant", "upgrade", "unlock",
		"water", "weed", "bug", "insecticide", "fertilize", "remove":
	default:
		return false, nil, nil, fmt.Errorf("unsupported farm operation %q", op)
	}
	if op == "cycle" {
		// rust operate: "cycle" is an alias of the full "all" round.
		op = "all"
	}

	lands, allLandsReply, err := api.AllLands(ctx)
	if err != nil {
		return false, nil, nil, fmt.Errorf("load lands: %w", err)
	}
	if allLandsReply != nil {
		feedOperationLimits(options, allLandsReply.OperationLimits)
	}
	if len(lands) == 0 {
		return false, nil, lands, nil
	}
	analysis := logic.AnalyzeLands(lands)
	var opErrs []error
	record := func(label string, count int, call func() error) {
		if count == 0 {
			return
		}
		if callErr := call(); callErr != nil {
			opErrs = append(opErrs, fmt.Errorf("%s: %w", label, callErr))
			return
		}
		actions = append(actions, fmt.Sprintf("%s%d", label, count))
		recordOpCount(options, label, count)
	}

	// ----- 单步操作（rust scheduler op_* 语义：单次 AllLands 分析后只做该步） -----
	switch op {
	case "water":
		// op_water：dry_num>0 的地块全部浇水。
		record("浇水", len(analysis.NeedWater), func() error {
			return api.WaterLand(ctx, analysis.NeedWater)
		})
		return len(actions) > 0, actions, lands, errors.Join(opErrs...)
	case "weed", "bug", "insecticide":
		// op_weed / op_insecticide：长草或生虫的地块一键务农（一次清两种）。
		ids := make([]int64, 0, len(analysis.NeedWeed)+len(analysis.NeedBug))
		for i := range lands {
			plant := lands[i].Plant
			if plant == nil {
				continue
			}
			if len(plant.WeedOwners) > 0 || len(plant.InsectOwners) > 0 {
				ids = append(ids, lands[i].ID)
			}
		}
		ids = uniqueLandIDs(ids)
		label := "除草"
		if op != "weed" {
			label = "除虫"
		}
		record(label, len(ids), func() error {
			return api.Farming(ctx, ids)
		})
		return len(actions) > 0, actions, lands, errors.Join(opErrs...)
	case "fertilize":
		// op_fertilize：对当前所有种植地块按化肥配置跑一轮施肥，返回 normal/organic 计数。
		var planted []int64
		for i := range lands {
			if lands[i].Plant != nil {
				planted = append(planted, lands[i].ID)
			}
		}
		if len(planted) > 0 {
			normal, organic := runFertilizerByConfig(ctx, api, cfg, planted, false, false, &actions, &opErrs, options)
			actions = append(actions, fmt.Sprintf("施肥(普通肥%d/有机肥%d)", normal, organic))
		}
		return len(actions) > 0, actions, lands, errors.Join(opErrs...)
	case "remove":
		// op_remove：铲除所有枯死地块。
		record("铲除", len(analysis.Dead), func() error {
			return api.RemovePlant(ctx, analysis.Dead)
		})
		return len(actions) > 0, actions, lands, errors.Join(opErrs...)
	case "unlock":
		// op_unlock：解锁第一块可解锁土地（rust 返回该 landId，这里记入 actions）。
		if len(analysis.Unlockable) > 0 {
			id := analysis.Unlockable[0]
			if callErr := api.UnlockLand(ctx, id, false); callErr != nil {
				opErrs = append(opErrs, fmt.Errorf("解锁土地%d: %w", id, callErr))
			} else {
				actions = append(actions, fmt.Sprintf("解锁%d", id))
			}
		}
		return len(actions) > 0, actions, lands, errors.Join(opErrs...)
	}

	if op == "all" || op == "clear" {
		// 互动道具地块（黄金虫/足球/乌云）并入一键务农目标（rust scheduler 同链）。
		farmingIDs := uniqueLandIDs(analysis.NeedWeed, analysis.NeedBug, analysis.NeedWater, analysis.NeedInteraction)
		if !(op == "all" && cfg.Automation.SkipOwnWeedBug) {
			weedN, bugN, waterN := len(analysis.NeedWeed), len(analysis.NeedBug), len(analysis.NeedWater)
			itemN := len(analysis.NeedInteraction)
			if total := len(farmingIDs); total > 0 {
				if callErr := api.Farming(ctx, farmingIDs); callErr != nil {
					opErrs = append(opErrs, fmt.Errorf("务农: %w", callErr))
				} else {
					// 对齐 rust 一键务农明细：草N/虫N/水N/道具N。
					parts := make([]string, 0, 4)
					if weedN > 0 {
						parts = append(parts, fmt.Sprintf("草%d", weedN))
					}
					if bugN > 0 {
						parts = append(parts, fmt.Sprintf("虫%d", bugN))
					}
					if waterN > 0 {
						parts = append(parts, fmt.Sprintf("水%d", waterN))
					}
					if itemN > 0 {
						parts = append(parts, fmt.Sprintf("道具%d", itemN))
					}
					label := fmt.Sprintf("务农%d", len(farmingIDs))
					if len(parts) > 0 {
						label += fmt.Sprintf("(%s)", strings.Join(parts, "/"))
					}
					actions = append(actions, label)
					recordOpCount(options, "务农", len(farmingIDs))
				}
			}
		}
	}

	harvested := []int64(nil)
	var harvestReplyLands []logic.LandInfo
	if op == "all" || op == "harvest" {
		record("收获", len(analysis.Harvestable), func() error {
			replyLands, callErr := api.Harvest(ctx, analysis.Harvestable)
			if callErr != nil {
				return callErr
			}
			harvested = append(harvested, analysis.Harvestable...)
			harvestReplyLands = replyLands
			return nil
		})
		// Prefer bot-style harvest log with crop names when available.
		if len(analysis.Harvestable) > 0 && len(actions) > 0 {
			last := actions[len(actions)-1]
			if strings.HasPrefix(last, "收获") {
				if named := formatHarvestAction(analysis.HarvestableInfo); named != "" {
					actions[len(actions)-1] = named
				}
			}
		}
		// 对齐 rust scheduler：收获成功独立面板条目（收获完成 N 块土地）。
		if len(harvested) > 0 {
			options.emit(FarmLogEntry{Tag: "收获", Event: "harvest_crop",
				Msg: fmt.Sprintf("收获完成 %d 块土地", len(harvested))})
		}
	}
	if len(harvested) > 0 && cfg.Automation.Sell {
		sold, gold, names, sellErr := sellAllFruitsDetailed(ctx, api)
		if sellErr != nil {
			opErrs = append(opErrs, fmt.Errorf("出售果实: %w", sellErr))
			options.emit(FarmLogEntry{Tag: "仓库", Event: "sell_done", Module: "warehouse",
				Msg: "出售失败: " + sellErr.Error(), IsWarn: true})
		} else if sold > 0 {
			if gold > 0 {
				actions = append(actions, fmt.Sprintf("出售%d(+%d金)", sold, gold))
				recordOpCount(options, "sell", 1)
				if options.accountID > 0 {
					stats.RecordExpGold(options.accountID, 0, 0, gold)
				}
				// 对齐 rust warehouse：出售成功独立条目（module=warehouse）。
				options.emit(FarmLogEntry{Tag: "仓库", Event: "sell_success", Module: "warehouse",
					Msg: fmt.Sprintf("出售 %s，获得 %d 金币", strings.Join(names, ", "), gold)})
			} else {
				actions = append(actions, fmt.Sprintf("出售%d", sold))
				recordOpCount(options, "sell", 1)
				options.emit(FarmLogEntry{Tag: "仓库", Event: "sell_done", Module: "warehouse",
					Msg: fmt.Sprintf("出售 %s", strings.Join(names, ", "))})
			}
		}
	}

	var postHarvestGrowing []int64
	if op == "all" || op == "plant" {
		removable := append([]int64(nil), analysis.Dead...)
		if op == "all" && len(harvested) > 0 {
			if callErr := waitFarmDelay(ctx, time.Second); callErr != nil {
				opErrs = append(opErrs, callErr)
			} else {
				var refreshed []logic.LandInfo
				var refreshReply *plantpb.AllLandsReply
				refreshed, refreshReply, refreshErr := api.AllLands(ctx)
				if refreshErr != nil {
					opErrs = append(opErrs, fmt.Errorf("refresh harvested lands: %w", refreshErr))
					// Classify from harvest reply only — never shovel all harvested on refresh fail.
					resolved := logic.ResolveRemovableHarvestedLandsPure(harvested, harvestReplyLands, nil)
					removable = uniqueLandIDs(removable, resolved.Removable)
					postHarvestGrowing = resolved.Growing
				} else {
					if refreshReply != nil {
						feedOperationLimits(options, refreshReply.OperationLimits)
					}
					lands = refreshed
					resolved := logic.ResolveRemovableHarvestedLandsPure(harvested, harvestReplyLands, refreshed)
					removable = uniqueLandIDs(removable, resolved.Removable)
					postHarvestGrowing = resolved.Growing
				}
			}
		}
		removed := len(removable) == 0
		record("铲除", len(removable), func() error {
			if callErr := api.RemovePlant(ctx, removable); callErr != nil {
				return callErr
			}
			removed = true
			return nil
		})

		// Bot autoPlantEmptyLands: after shovel, re-fetch AllLands and plant analyze.empty only.
		// On refresh fail keep original empties and do NOT plant former dead/removable IDs.
		available := uniqueLandIDs(analysis.Empty)
		if removed && len(removable) > 0 {
			refreshed, reply, refreshErr := api.AllLands(ctx)
			if refreshErr != nil {
				opErrs = append(opErrs, fmt.Errorf("铲除后确认土地: %w", refreshErr))
			} else {
				if reply != nil {
					feedOperationLimits(options, reply.OperationLimits)
				}
				lands = refreshed
				available = uniqueLandIDs(logic.AnalyzeLands(refreshed).Empty)
			}
		}
		if len(available) > 0 {
			plantedIDs, plantNotes, plantErrs := plantAvailableLands(ctx, api, cfg, available, options.playerLevel, options.gold)
			opErrs = append(opErrs, plantErrs...)
			actions = append(actions, plantNotes...)
			// 多格预留独立条目（rust plant_from_bag_seeds：预留 N 块空地等凑布局）。
			for _, note := range plantNotes {
				if strings.HasPrefix(note, "预留") {
					options.emit(FarmLogEntry{Tag: "种植", Event: "plant_seed",
						Msg: note + " 块空地等待凑齐布局"})
				}
			}
			if len(plantedIDs) > 0 {
				actions = append(actions, fmt.Sprintf("种植%d", len(plantedIDs)))
				recordOpCount(options, "plant", len(plantedIDs))
				// 对齐 rust scheduler：种植成功独立面板条目。
				options.emit(FarmLogEntry{Tag: "种植", Event: "plant_seed",
					Msg: fmt.Sprintf("种植完成 %d 块土地", len(plantedIDs))})
				// Both 改由巡田末尾统一跑全场，种完后不再立刻补肥，避免同一轮打两遍。
				if cfg.Automation.Fertilizer != logic.FertilizerBoth {
					runFertilizerByConfig(ctx, api, cfg, plantedIDs, false, false, &actions, &opErrs, options)
				}
			}
		}
	}

	if op == "all" && cfg.Automation.FertilizerMultiSeason && len(postHarvestGrowing) > 0 &&
		cfg.Automation.Fertilizer != logic.FertilizerBoth {
		normal, organic := runFertilizerByConfig(ctx, api, cfg, postHarvestGrowing, false, true, &actions, &opErrs, options)
		// 对齐 rust scheduler：多季补肥完成 普通 a / 有机 b。
		if normal+organic > 0 {
			options.emit(FarmLogEntry{Tag: "施肥", Event: "fertilize",
				Msg: fmt.Sprintf("多季补肥完成 普通%d / 有机%d", normal, organic)})
		}
	}

	shouldUpgrade := op == "upgrade" || (op == "all" && cfg.Automation.LandUpgrade)
	if shouldUpgrade {
		unlocked := 0
		for _, id := range analysis.Unlockable {
			if callErr := api.UnlockLand(ctx, id, false); callErr != nil {
				opErrs = append(opErrs, fmt.Errorf("解锁土地%d: %w", id, callErr))
				continue
			}
			unlocked++
			// 对齐 rust scheduler：土地#N 解锁成功。
			options.emit(FarmLogEntry{Tag: "解锁", Event: "unlock_land",
				Msg: fmt.Sprintf("土地#%d 解锁成功", id)})
		}
		if unlocked > 0 {
			actions = append(actions, fmt.Sprintf("解锁%d", unlocked))
		}
		upgraded := 0
		for _, id := range analysis.Upgradable {
			if callErr := api.UpgradeLand(ctx, id); callErr != nil {
				opErrs = append(opErrs, fmt.Errorf("升级土地%d: %w", id, callErr))
				continue
			}
			upgraded++
			options.emit(FarmLogEntry{Tag: "升级", Event: "upgrade_land",
				Msg: fmt.Sprintf("土地#%d 升级成功", id)})
		}
		if upgraded > 0 {
			actions = append(actions, fmt.Sprintf("升级%d", upgraded))
			recordOpCount(options, "upgrade", upgraded)
		}
	}

	// End fertilizer pass: smart skips normal (organic on fast-mature lands only),
	// Both runs the full-field pass here (normal + organic ripening, bot scheduler).
	if op == "all" && (cfg.Automation.Fertilizer == logic.FertilizerSmart || cfg.Automation.Fertilizer == logic.FertilizerBoth) {
		normal, organic := runFertilizerByConfig(ctx, api, cfg, nil, cfg.Automation.Fertilizer == logic.FertilizerSmart, false, &actions, &opErrs, options)
		// 对齐 rust scheduler：巡田施肥完成 普通 a / 有机 b。
		if normal+organic > 0 {
			options.emit(FarmLogEntry{Tag: "施肥", Event: "fertilize",
				Msg: fmt.Sprintf("巡田施肥完成 普通%d / 有机%d", normal, organic)})
		}
	}

	// 对齐 rust scheduler run_one_cycle：巡田汇总只在有动作时记录，
	// 消息为状态摘要 [收:N 农:N 水:N 枯:N 空:N 解:N 升:N 长:N] → 动作列表。
	if (op == "all" || op == "cycle") && len(actions) > 0 {
		options.emit(FarmLogEntry{Tag: "农场", Event: "farm_cycle",
			Msg: farmCycleSummary(analysis, actions)})
	}

	return len(actions) > 0, actions, lands, errors.Join(opErrs...)
}

// plantAvailableLands plants empty lands and returns confirmed planted master land IDs.
// bagSeedLandTypes 非空时（rust plant_from_bag_seeds）：先拉一次最新地块建
// landId→类型映射，受限种子先种且只落在命中类型的空地；解析失败 logWarn
// 按不限制处理，不能导致整轮种不下去。多格预留开关打开时同样复用这次全量
// 地块拉取取全部已解锁土地（rust all_unlocked_land_ids，开关默认关闭零额外请求）。
// 第二个返回值是附加动作说明（如多格预留），由调用方并入本轮 actions。
func plantAvailableLands(ctx context.Context, api *game.API, cfg logic.AccountConfig, available []int64, playerLevel, gold int64) ([]int64, []string, []error) {
	remaining := uniqueLandIDs(available)
	var (
		plantedIDs []int64
		notes      []string
		errs       []error
	)

	fallbackAllowed := true
	var bagSeeds []logic.BagSeed
	useBag := cfg.PlantingStrategy == logic.StrategyBagPriority
	// 背包种子土地类型限制上下文：限制表为空且未开多格预留时零额外请求；
	// 否则才拉最新土地（rust resolve_land_type_map_for_bag_seeds + 预留全量土地）。
	landTypeAvailable := false
	landTypeByID := map[int64]string{}
	var allUnlockedLandIDs []int64
	if useBag {
		needAllLands := len(cfg.BagSeedLandTypes) > 0 || cfg.BagSeedMultiLandReservationEnabled
		if needAllLands {
			latest, _, latestErr := api.AllLands(ctx)
			if latestErr != nil {
				// rust：解析失败仅 warn，按不限制处理；本轮不做多格预留。
				slog.Warn("解析土地类型失败，本轮背包种子按不限制处理", "err", latestErr)
			} else {
				landTypeByID = landTypes(latest)
				landTypeAvailable = len(landTypeByID) > 0
				if cfg.BagSeedMultiLandReservationEnabled {
					for _, l := range latest {
						if l.Unlocked && l.ID > 0 {
							allUnlockedLandIDs = append(allUnlockedLandIDs, l.ID)
						}
					}
				}
			}
		}
		var bagErr error
		bagSeeds, bagErr = api.BagSeeds(ctx)
		if bagErr != nil {
			errs = append(errs, fmt.Errorf("读取背包种子: %w", bagErr))
			return plantedIDs, notes, errs
		}
	}
	if useBag {
		levelLocked := map[int64]bool{}
		for _, seed := range bagSeeds {
			if playerLevel > 0 && seed.RequiredLevel > playerLevel {
				levelLocked[seed.SeedID] = true
			}
		}
		// 有土地限制的种子排在最前先种，否则不限种子会把受限种子唯一兼容的地块占光。
		orderedSeeds := logic.SortSeedsRestrictedFirst(
			logic.SortBagSeedsForPlanting(bagSeeds, cfg.BagSeedPriority), cfg.BagSeedLandTypes)
		// 多格预留每轮最多一次（bot futureLayoutReserved）。
		reservedAnchor := int64(0)
		for _, seed := range orderedSeeds {
			if !fallbackAllowed || len(remaining) == 0 {
				break
			}
			// 受限种子只在命中类型的空地上装箱；未命中的空地留给后续种子。
			allowedTarget := remaining
			if landTypeAvailable {
				if seedTypes, restricted := logic.ResolveSeedLandTypes(cfg.BagSeedLandTypes, seed.SeedID); restricted {
					allowedTarget = logic.FilterLandIDsByTypes(remaining, landTypeByID, seedTypes)
					if len(allowedTarget) == 0 {
						slog.Info("背包种子无匹配类型空地，已跳过",
							"seedId", seed.SeedID, "name", seed.Name)
						continue
					}
				}
			}
			_, layouts := logic.PlanBagPlantingLayouts(allowedTarget, seed.PlantSize, seed.Count)
			// 多格预留（bot 96fdb39 + ee4de82）：为排在前面的多格种子保住
			// 尚未凑齐布局的空地，只预留当前已空出的部分；每轮最多一次，
			// 且只对显式优先级种子生效。
			if len(layouts) == 0 && cfg.BagSeedMultiLandReservationEnabled && reservedAnchor == 0 &&
				seed.PlantSize > 1 && slices.Contains(cfg.BagSeedPriority, seed.SeedID) {
				allEligible := allUnlockedLandIDs
				if landTypeAvailable {
					if seedTypes, restricted := logic.ResolveSeedLandTypes(cfg.BagSeedLandTypes, seed.SeedID); restricted {
						allEligible = logic.FilterLandIDsByTypes(allUnlockedLandIDs, landTypeByID, seedTypes)
					}
				}
				if reservation, ok := logic.SelectFutureLayoutReservation(allowedTarget, allEligible, seed.PlantSize); ok {
					remaining = removeLandIDs(remaining, reservation.ReservedLandIDs)
					reservedAnchor = reservation.Layout.AnchorLandID
					note := fmt.Sprintf("预留%d", len(reservation.ReservedLandIDs))
					notes = append(notes, note)
					slog.Info("背包种子预留未来布局",
						"seedId", seed.SeedID, "name", seed.Name,
						"anchorLandId", reservation.Layout.AnchorLandID,
						"layoutLandIds", reservation.Layout.LandIDs,
						"reservedLandIds", reservation.ReservedLandIDs)
				}
			}
			stopSeed := false
			for _, layout := range layouts {
				if !fallbackAllowed || len(remaining) == 0 {
					break
				}
				masterID, occupied, uncertain, plantErr := plantOneLayout(ctx, api, seed.SeedID, layout)
				if plantErr != nil {
					errs = append(errs, fmt.Errorf("种植背包种子%d: %w", seed.SeedID, plantErr))
					remaining = removeLandIDs(remaining, layout.LandIDs)
					if levelLocked[seed.SeedID] {
						// Level-locked failure must not block shop fallback (bot planting.ts).
						stopSeed = true
						break
					}
					fallbackAllowed = false
					stopSeed = true
					break
				}
				if uncertain {
					errs = append(errs, fmt.Errorf("种植背包种子%d: footprint uncertain on land %d", seed.SeedID, layout.AnchorLandID))
					remaining = removeLandIDs(remaining, layout.LandIDs)
					if levelLocked[seed.SeedID] {
						stopSeed = true
						break
					}
					// Uncertain bag plant: skip shop fallback to avoid mis-buying (bot).
					fallbackAllowed = false
					stopSeed = true
					break
				}
				plantedIDs = append(plantedIDs, masterID)
				remaining = removeLandIDs(remaining, occupied)
				remaining = removeLandIDs(remaining, layout.LandIDs)
			}
			if stopSeed && !fallbackAllowed {
				break
			}
		}
	}
	if !fallbackAllowed || len(remaining) == 0 {
		return uniqueLandIDs(plantedIDs), notes, errs
	}

	strategy := cfg.PlantingStrategy
	if strategy == logic.StrategyBagPriority {
		strategy = cfg.BagSeedFallbackStrategy
		if strategy == "" {
			strategy = logic.StrategyLevel
		}
	}
	shop, err := api.ShopInfo(ctx, 2)
	if err != nil {
		return uniqueLandIDs(plantedIDs), notes, append(errs, fmt.Errorf("读取种子商店: %w", err))
	}
	candidates := shopSeedCandidates(shop, playerLevel)
	candidates = logic.SortSeedCandidatesByStrategy(candidates, strategy, cfg.PreferredSeedID, playerLevel, rankingsForStrategy(strategy))
	for _, candidate := range candidates {
		if len(remaining) == 0 {
			break
		}
		layouts, units, needCount, _ := logic.ComputeShopPurchaseLayouts(candidate, remaining, gold)
		if units <= 0 || len(layouts) == 0 {
			continue
		}
		reply, err := api.BuyGoods(ctx, candidate.GoodsID, units, candidate.Price)
		if err != nil {
			errs = append(errs, fmt.Errorf("购买种子%d: %w", candidate.SeedID, err))
			break // purchase outcome is unknown; do not risk duplicate buys
		}
		seedID := candidate.SeedID
		if len(reply.GetItems) > 0 && reply.GetItems[0] != nil && reply.GetItems[0].Id > 0 {
			seedID = reply.GetItems[0].Id
			if reply.GetItems[0].Count > 0 && reply.GetItems[0].Count < needCount {
				needCount = reply.GetItems[0].Count
				layouts = layouts[:needCount]
			}
		}
		gold -= candidate.Price * units
		stop := false
		for _, layout := range layouts {
			masterID, occupied, uncertain, plantErr := plantOneLayout(ctx, api, seedID, layout)
			if plantErr != nil {
				errs = append(errs, fmt.Errorf("种植商店种子%d: %w", seedID, plantErr))
				stop = true
				break
			}
			if uncertain {
				errs = append(errs, fmt.Errorf("种植商店种子%d: footprint uncertain on land %d", seedID, layout.AnchorLandID))
				remaining = removeLandIDs(remaining, layout.LandIDs)
				stop = true
				break
			}
			plantedIDs = append(plantedIDs, masterID)
			remaining = removeLandIDs(remaining, occupied)
			remaining = removeLandIDs(remaining, layout.LandIDs)
		}
		if stop {
			break
		}
	}
	return uniqueLandIDs(plantedIDs), notes, errs
}

// plantOneLayout plants one layout, confirms footprint, and returns master + occupied IDs.
// uncertain=true means planting should stop; reserved layout IDs should be removed from remaining.
func plantOneLayout(ctx context.Context, api *game.API, seedID int64, layout logic.PlantingLayout) (masterID int64, occupied []int64, uncertain bool, err error) {
	replyLands, err := api.Plant(ctx, seedID, layout.LandIDs)
	if err != nil {
		return 0, nil, true, err
	}
	masterID, occupied = logic.ResolveOccupiedLandIDs(layout.AnchorLandID, replyLands)
	if masterID == 0 {
		masterID = layout.AnchorLandID
	}
	confirmed := logic.ConfirmsPlantedFootprint(layout.LandIDs, masterID, occupied, replyLands)
	if !confirmed {
		latest, _, refreshErr := api.AllLands(ctx)
		if refreshErr != nil {
			return masterID, occupied, true, nil
		}
		masterID, occupied = logic.ResolveOccupiedLandIDs(layout.AnchorLandID, latest)
		if masterID == 0 {
			masterID = layout.AnchorLandID
		}
		confirmed = logic.ConfirmsPlantedFootprint(layout.LandIDs, masterID, occupied, latest)
	}
	if !confirmed {
		return masterID, occupied, true, nil
	}
	return masterID, occupied, false, nil
}

// runFertilizerByConfig mirrors planting.fertilize_by_config_ex:
// normal/both/smart get normal fert; organic targets depend on mode —
// Both ripens all immature lands (until-mature loop), Organic loops over
// lands that can still take organic fertilizer (restricted to the planted
// multi-season lands), Smart targets soon-to-mature lands.
// Lands are re-fetched here; on fetch failure the whole round is skipped
// (fail-closed, stricter than bot). Returns applied normal/organic counts
// (rust FertilizeResult{normal, organic}).
func runFertilizerByConfig(ctx context.Context, api *game.API, cfg logic.AccountConfig, planted []int64, skipNormal bool, multiSeason bool, actions *[]string, errs *[]error, opts farmOperationOptions) (normal int, organic int) {
	mode := cfg.Automation.Fertilizer
	if mode == logic.FertilizerNone {
		return 0, 0
	}
	selectedTypes := logic.NormalizeFertilizerLandTypes(cfg.Automation.FertilizerLandTypes)
	if len(selectedTypes) == 0 {
		return 0, 0
	}
	plantedIDs := uniqueLandIDs(planted)
	if len(plantedIDs) == 0 && mode == logic.FertilizerNormal {
		return 0, 0
	}
	reason := "常规施肥"
	if multiSeason {
		reason = "多季补肥"
	}

	latestLands, reply, refreshErr := api.AllLands(ctx)
	if refreshErr != nil {
		*errs = append(*errs, fmt.Errorf("%s：获取土地信息失败，已跳过本轮施肥: %w", reason, refreshErr))
		return 0, 0
	}
	if len(latestLands) == 0 {
		*errs = append(*errs, fmt.Errorf("%s：获取土地信息失败，已跳过本轮施肥", reason))
		return 0, 0
	}
	if reply != nil {
		feedOperationLimits(opts, reply.OperationLimits)
	}
	types := landTypes(latestLands)

	// Both 常规化肥按「本季还能施普通肥的全部地块」取目标，其余模式只施本次地块。
	normalSource := plantedIDs
	if mode == logic.FertilizerBoth {
		normalSource = logic.GetNormalFertilizerTargetsFromLands(latestLands)
	}
	normalTargets := normalSource
	if len(types) > 0 {
		normalTargets = logic.FilterLandIDsByTypes(normalSource, types, selectedTypes)
	}

	total := 0
	if !skipNormal && (mode == logic.FertilizerNormal || mode == logic.FertilizerBoth || mode == logic.FertilizerSmart) {
		appliedNormal := fertilizeNormalStep(ctx, api, normalTargets, mode == logic.FertilizerBoth, reason, actions, errs)
		normal = appliedNormal
		total += appliedNormal
	}

	if mode == logic.FertilizerOrganic || mode == logic.FertilizerBoth || mode == logic.FertilizerSmart {
		var organicTargets []int64
		switch mode {
		case logic.FertilizerBoth:
			// Both：全场未成熟地催熟。
			organicTargets = logic.GetImmatureCropTargetsFromLands(latestLands)
			organicTargets = logic.FilterLandIDsByTypes(organicTargets, types, selectedTypes)
		case logic.FertilizerOrganic:
			organicTargets = logic.GetOrganicFertilizerTargetsFromLands(latestLands)
			// bot：多季补肥时有机肥目标限定到本次多季地块。
			if multiSeason && len(plantedIDs) > 0 {
				organicTargets = intersectLandIDs(organicTargets, plantedIDs)
			}
			organicTargets = logic.FilterLandIDsByTypes(organicTargets, types, selectedTypes)
		case logic.FertilizerSmart:
			smartSecs := int64(cfg.Automation.FertilizerSmartSeconds)
			if smartSecs <= 0 {
				smartSecs = 300
			}
			organicTargets = logic.GetFastMatureLands(latestLands, smartSecs)
		}
		if len(organicTargets) > 0 {
			var (
				count        int
				remainingSec int64
				hasRemaining bool
			)
			if mode == logic.FertilizerBoth {
				count, remainingSec, hasRemaining, _ = api.FertilizeOrganicUntilMature(ctx, organicTargets)
			} else {
				count, remainingSec, hasRemaining, _ = api.FertilizeOrganicLoop(ctx, organicTargets)
			}
			if count > 0 {
				*actions = append(*actions, fmt.Sprintf("有机肥%d%s", count, remainingHoursLabel(remainingSec, hasRemaining)))
			}
			if limit := game.OrganicOperationLimit(len(organicTargets)); count >= limit {
				*actions = append(*actions, fmt.Sprintf("有机肥循环达到单次上限%d，已停止继续请求", limit))
			}
			organic = count
			total += count
		}
	}
	if total > 0 && opts.accountID > 0 {
		stats.RecordOp(opts.accountID, 0, "fertilize", total)
	}
	return normal, organic
}

// fertilizeNormalStep applies normal fertilizer land by land. continueOnError
// is the Both mode behavior: one failing land must not block the rest.
func fertilizeNormalStep(ctx context.Context, api *game.API, normalTargets []int64, continueOnError bool, reason string, actions *[]string, errs *[]error) int {
	if len(normalTargets) == 0 {
		return 0
	}
	normal := 0
	var (
		remainingSec int64
		hasRemaining bool
	)
	for i, landID := range normalTargets {
		res, err := api.Fertilize(ctx, []int64{landID}, game.NormalFertilizerID)
		if err != nil {
			if continueOnError {
				continue
			}
			break
		}
		if res.HasRemaining {
			remainingSec = res.RemainingSecs
			hasRemaining = true
		}
		normal++
		if i+1 < len(normalTargets) {
			if delayErr := waitFarmDelay(ctx, 50*time.Millisecond); delayErr != nil {
				break
			}
		}
	}
	if normal > 0 {
		*actions = append(*actions, fmt.Sprintf("普通肥%d/%d%s", normal, len(normalTargets), remainingHoursLabel(remainingSec, hasRemaining)))
	}
	return normal
}

// remainingHoursLabel mirrors the Rust "，剩 X.Xh" suffix (container seconds → hours).
func remainingHoursLabel(remainingSec int64, hasRemaining bool) string {
	if !hasRemaining || remainingSec <= 0 {
		return ""
	}
	return fmt.Sprintf("(剩%.1fh)", float64(remainingSec)/3600)
}

// intersectLandIDs keeps ids present in both slices (positive, deduped).
func intersectLandIDs(ids, keep []int64) []int64 {
	set := make(map[int64]struct{}, len(keep))
	for _, id := range keep {
		if id > 0 {
			set[id] = struct{}{}
		}
	}
	seen := make(map[int64]struct{}, len(ids))
	var out []int64
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := set[id]; !ok {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func shopSeedCandidates(reply *shoppb.ShopInfoReply, playerLevel int64) []logic.SeedCandidate {
	if reply == nil {
		return nil
	}
	out := make([]logic.SeedCandidate, 0, len(reply.GoodsList))
	for _, goods := range reply.GoodsList {
		if goods == nil || !goods.Unlocked || goods.Id <= 0 || goods.ItemId <= 0 || goods.Price <= 0 {
			continue
		}
		requiredLevel := int64(0)
		for _, condition := range goods.Conds {
			if condition.Type == 1 && condition.Param > requiredLevel {
				requiredLevel = condition.Param
			}
		}
		if playerLevel > 0 && requiredLevel > playerLevel {
			continue
		}
		if goods.LimitCount > 0 && goods.BoughtNum >= goods.LimitCount {
			continue
		}
		maxPurchase := float64(0)
		if goods.LimitCount > 0 {
			maxPurchase = float64(goods.LimitCount - goods.BoughtNum)
		} else {
			maxPurchase = math.Inf(1)
		}
		out = append(out, logic.SeedCandidate{
			GoodsID: goods.Id, SeedID: goods.ItemId, Price: goods.Price,
			RequiredLevel: requiredLevel, UnitItemCount: max(goods.ItemCount, 1), MaxPurchaseCount: maxPurchase,
		})
	}
	return out
}

func rankingsForStrategy(strategy string) []logic.RankingRow {
	sortBy := map[string]string{
		logic.StrategyMaxExp: "exp", logic.StrategyMaxFertExp: "fert",
		logic.StrategyMaxProfit: "profit", logic.StrategyMaxFertProfit: "fert_profit",
	}[strategy]
	if sortBy == "" {
		return nil
	}
	rankings := logic.GetPlantRankings(sortBy)
	out := make([]logic.RankingRow, 0, len(rankings))
	for _, row := range rankings {
		level := float64(0)
		if row.Level != nil {
			level = float64(*row.Level)
		}
		out = append(out, logic.RankingRow{SeedID: row.SeedID, Level: level})
	}
	return out
}

func sellAllFruits(ctx context.Context, api *game.API) (int, error) {
	sold, _, _, err := sellAllFruitsDetailed(ctx, api)
	return sold, err
}

// sellAllFruitsDetailed sells bag fruits and returns kind count, gold earned, and fruit names.
func sellAllFruitsDetailed(ctx context.Context, api *game.API) (soldKinds int, gold int64, names []string, err error) {
	bag, err := api.Bag(ctx)
	if err != nil {
		return 0, 0, nil, err
	}
	items := game.GetBagItems(bag)
	fruits := make([]corepb.Item, 0)
	nameSet := make(map[string]struct{})
	now := time.Now().Unix()
	for _, item := range items {
		if item.Count <= 0 || logic.GetPlantByFruitID(item.Id) == nil {
			continue
		}
		// bot sellAllFruits：跳过锁定物品，按 sell_cond + 过期时间评估可售性。
		if item.Locked {
			continue
		}
		info := logic.GetItemByID(item.Id)
		sellInfo := logic.GetEffectiveSellInfoAt(info, logic.DefaultSellConditionContext(now), logic.ToTimeSec(item.ExpireTime))
		if !sellInfo.Sellable {
			continue
		}
		fruits = append(fruits, item)
		name := ""
		if info := logic.GetItemByID(item.Id); info != nil {
			name = strings.TrimSpace(info.Name)
		}
		if name == "" {
			if plant := logic.GetPlantByFruitID(item.Id); plant != nil {
				name = plant.Name
			}
		}
		if name == "" {
			name = fmt.Sprintf("果实%d", item.Id)
		}
		label := fmt.Sprintf("%sx%d", name, item.Count)
		if _, ok := nameSet[label]; !ok {
			nameSet[label] = struct{}{}
			names = append(names, label)
		}
	}
	for start := 0; start < len(fruits); start += 15 {
		end := start + 15
		if end > len(fruits) {
			end = len(fruits)
		}
		reply, sellErr := api.Sell(ctx, fruits[start:end])
		if sellErr != nil {
			return soldKinds, gold, names, sellErr
		}
		soldKinds += end - start
		gold += goldFromSellReply(reply)
	}
	return soldKinds, gold, names, nil
}

func goldFromSellReply(reply *itempb.SellReply) int64 {
	if reply == nil {
		return 0
	}
	var total int64
	for _, it := range reply.GetItems {
		if it == nil || it.Count <= 0 {
			continue
		}
		// Gold currency items commonly use id 1 / 1001 (same as ItemNotify).
		if it.Id == 1 || it.Id == 1001 {
			total += it.Count
		}
	}
	return total
}

func removeLandIDs(landIDs, removed []int64) []int64 {
	remove := make(map[int64]struct{}, len(removed))
	for _, id := range removed {
		remove[id] = struct{}{}
	}
	out := make([]int64, 0, len(landIDs))
	for _, id := range landIDs {
		if _, ok := remove[id]; !ok {
			out = append(out, id)
		}
	}
	return out
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func recordOpCount(opts farmOperationOptions, label string, count int) {
	if opts.accountID == 0 || count <= 0 {
		return
	}
	op := statsOpFromLabel(label)
	if op == "" {
		return
	}
	stats.RecordOp(opts.accountID, 0, op, count)
}

func statsOpFromLabel(label string) string {
	switch label {
	case "收获", "harvest":
		return "harvest"
	case "务农", "farming":
		return "farming"
	case "种植", "plant":
		return "plant"
	case "普通肥", "有机肥", "fertilize":
		return "fertilize"
	case "出售", "sell":
		return "sell"
	case "升级", "upgrade":
		return "upgrade"
	default:
		return ""
	}
}

// formatHarvestAction mirrors bot harvest log: 收获N(作物A/作物B).
func formatHarvestAction(infos []logic.HarvestableInfo) string {
	if len(infos) == 0 {
		return ""
	}
	seen := make(map[string]struct{}, len(infos))
	names := make([]string, 0, len(infos))
	for _, info := range infos {
		name := strings.TrimSpace(info.Name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	if len(names) == 0 {
		return fmt.Sprintf("收获%d", len(infos))
	}
	return fmt.Sprintf("收获%d(%s)", len(infos), strings.Join(names, "/"))
}

func landTypes(lands []logic.LandInfo) map[int64]string {
	out := make(map[int64]string, len(lands))
	for _, land := range lands {
		out[land.ID] = logic.GetLandTypeByLevel(land.Level)
	}
	return out
}

func uniqueLandIDs(groups ...[]int64) []int64 {
	seen := make(map[int64]struct{})
	var out []int64
	for _, group := range groups {
		for _, id := range group {
			if id <= 0 {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out
}

// farmCycleSummary 对齐 rust scheduler farm_cycle_status_parts + farming_action：
// 巡田汇总消息 `[收:N 农:N 水:N 枯:N 空:N 解:N 升:N 长:N] → 动作/动作`，
// 零值部分省略（rust 同款）。
func farmCycleSummary(analysis logic.LandAnalysis, actions []string) string {
	parts := make([]string, 0, 8)
	add := func(label string, n int) {
		if n > 0 {
			parts = append(parts, fmt.Sprintf("%s:%d", label, n))
		}
	}
	add("收", len(analysis.Harvestable))
	add("农", len(analysis.NeedWeed)+len(analysis.NeedBug)+len(analysis.NeedInteraction))
	add("水", len(analysis.NeedWater))
	add("枯", len(analysis.Dead))
	add("空", len(analysis.Empty))
	add("解", len(analysis.Unlockable))
	add("升", len(analysis.Upgradable))
	add("长", len(analysis.Growing))
	summary := ""
	if len(parts) > 0 {
		summary = "[" + strings.Join(parts, " ") + "] → "
	}
	return summary + strings.Join(actions, "/")
}

func waitFarmDelay(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
