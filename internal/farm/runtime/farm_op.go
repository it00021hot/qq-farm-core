package runtime

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/MQEnergy/go-skeleton/internal/farm/game"
	"github.com/MQEnergy/go-skeleton/internal/farm/logic"
	"github.com/MQEnergy/go-skeleton/internal/farm/proto/corepb"
	"github.com/MQEnergy/go-skeleton/internal/farm/proto/itempb"
	"github.com/MQEnergy/go-skeleton/internal/farm/proto/plantpb"
	"github.com/MQEnergy/go-skeleton/internal/farm/proto/shoppb"
	"github.com/MQEnergy/go-skeleton/internal/farm/stats"
)

// FarmOperationOption customizes one farm operation.
type FarmOperationOption func(*farmOperationOptions)

type farmOperationOptions struct {
	accountID   uint64
	playerLevel int64
	gold        int64
	limitsSink  func([]*plantpb.OperationLimit)
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
// Supported operations are all, harvest, clear, plant, and upgrade.
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
	case "all", "harvest", "clear", "plant", "upgrade":
	default:
		return false, nil, nil, fmt.Errorf("unsupported farm operation %q", op)
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

	if op == "all" || op == "clear" {
		farmingIDs := uniqueLandIDs(analysis.NeedWeed, analysis.NeedBug, analysis.NeedWater)
		if !(op == "all" && cfg.Automation.SkipOwnWeedBug) {
			weedN, bugN, waterN := len(analysis.NeedWeed), len(analysis.NeedBug), len(analysis.NeedWater)
			record("务农", len(farmingIDs), func() error { return api.Farming(ctx, farmingIDs) })
			if len(farmingIDs) > 0 && len(actions) > 0 && strings.HasPrefix(actions[len(actions)-1], "务农") {
				parts := make([]string, 0, 3)
				if weedN > 0 {
					parts = append(parts, fmt.Sprintf("草%d", weedN))
				}
				if bugN > 0 {
					parts = append(parts, fmt.Sprintf("虫%d", bugN))
				}
				if waterN > 0 {
					parts = append(parts, fmt.Sprintf("水%d", waterN))
				}
				if len(parts) > 0 {
					actions[len(actions)-1] = fmt.Sprintf("务农%d(%s)", len(farmingIDs), strings.Join(parts, "/"))
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
	}
	if len(harvested) > 0 && cfg.Automation.Sell {
		sold, gold, _, sellErr := sellAllFruitsDetailed(ctx, api)
		if sellErr != nil {
			opErrs = append(opErrs, fmt.Errorf("出售果实: %w", sellErr))
		} else if sold > 0 {
			if gold > 0 {
				actions = append(actions, fmt.Sprintf("出售%d(+%d金)", sold, gold))
				recordOpCount(options, "sell", 1)
				if options.accountID > 0 {
					stats.RecordExpGold(options.accountID, 0, 0, gold)
				}
			} else {
				actions = append(actions, fmt.Sprintf("出售%d", sold))
				recordOpCount(options, "sell", 1)
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
			plantedIDs, plantErrs := plantAvailableLands(ctx, api, cfg, available, options.playerLevel, options.gold)
			opErrs = append(opErrs, plantErrs...)
			if len(plantedIDs) > 0 {
				actions = append(actions, fmt.Sprintf("种植%d", len(plantedIDs)))
				recordOpCount(options, "plant", len(plantedIDs))
				runFertilizerByConfig(ctx, api, cfg, plantedIDs, false, &actions, &opErrs, options)
			}
		}
	}

	if op == "all" && cfg.Automation.FertilizerMultiSeason && len(postHarvestGrowing) > 0 {
		runFertilizerByConfig(ctx, api, cfg, postHarvestGrowing, false, &actions, &opErrs, options)
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
		}
		if upgraded > 0 {
			actions = append(actions, fmt.Sprintf("升级%d", upgraded))
			recordOpCount(options, "upgrade", upgraded)
		}
	}

	// End fertilizer pass: bot only runs smart organic on fast-mature lands (scheduler.ts).
	if op == "all" && cfg.Automation.Fertilizer == logic.FertilizerSmart {
		runFertilizerByConfig(ctx, api, cfg, nil, true, &actions, &opErrs, options)
	}

	return len(actions) > 0, actions, lands, errors.Join(opErrs...)
}

// plantAvailableLands plants empty lands and returns confirmed planted master land IDs.
func plantAvailableLands(ctx context.Context, api *game.API, cfg logic.AccountConfig, available []int64, playerLevel, gold int64) ([]int64, []error) {
	remaining := uniqueLandIDs(available)
	var (
		plantedIDs []int64
		errs       []error
	)

	fallbackAllowed := true
	var bagSeeds []logic.BagSeed
	useBag := cfg.PlantingStrategy == logic.StrategyBagPriority
	if useBag {
		var bagErr error
		bagSeeds, bagErr = api.BagSeeds(ctx)
		if bagErr != nil {
			errs = append(errs, fmt.Errorf("读取背包种子: %w", bagErr))
			return plantedIDs, errs
		}
	}
	if useBag {
		levelLocked := map[int64]bool{}
		for _, seed := range bagSeeds {
			if playerLevel > 0 && seed.RequiredLevel > playerLevel {
				levelLocked[seed.SeedID] = true
			}
		}
		for _, seed := range logic.SortBagSeedsForPlanting(bagSeeds, cfg.BagSeedPriority) {
			if !fallbackAllowed || len(remaining) == 0 {
				break
			}
			_, layouts := logic.PlanBagPlantingLayouts(remaining, seed.PlantSize, seed.Count)
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
		return uniqueLandIDs(plantedIDs), errs
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
		return uniqueLandIDs(plantedIDs), append(errs, fmt.Errorf("读取种子商店: %w", err))
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
	return uniqueLandIDs(plantedIDs), errs
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

// runFertilizerByConfig mirrors bot planting.runFertilizerByConfig.
// When planted is non-empty: normal/both/smart get one-pass normal fert; organic/both get organic loop.
// When skipNormal (smart end pass): only organic on fast-mature lands.
func runFertilizerByConfig(ctx context.Context, api *game.API, cfg logic.AccountConfig, planted []int64, skipNormal bool, actions *[]string, errs *[]error, opts farmOperationOptions) {
	selectedTypes := logic.NormalizeFertilizerLandTypes(cfg.Automation.FertilizerLandTypes)
	if len(selectedTypes) == 0 {
		return
	}
	mode := cfg.Automation.Fertilizer
	plantedIDs := uniqueLandIDs(planted)
	if len(plantedIDs) == 0 && mode != logic.FertilizerOrganic && mode != logic.FertilizerBoth && mode != logic.FertilizerSmart {
		return
	}

	var latestLands []logic.LandInfo
	refreshed, reply, refreshErr := api.AllLands(ctx)
	if refreshErr != nil {
		*errs = append(*errs, fmt.Errorf("施肥刷新土地: %w", refreshErr))
	} else {
		latestLands = refreshed
		if reply != nil {
			feedOperationLimits(opts, reply.OperationLimits)
		}
	}
	types := landTypes(latestLands)
	allSelected := len(selectedTypes) == len(logic.AllFertilizerLandTypes)
	if len(types) == 0 && !allSelected {
		return
	}

	normalTargets := plantedIDs
	if len(types) > 0 {
		normalTargets = logic.FilterLandIDsByTypes(plantedIDs, types, selectedTypes)
	}

	if !skipNormal && (mode == logic.FertilizerNormal || mode == logic.FertilizerBoth || mode == logic.FertilizerSmart) && len(normalTargets) > 0 {
		fertilize(ctx, api, normalTargets, game.NormalFertilizerID, "普通肥", actions, errs, opts)
	}

	switch mode {
	case logic.FertilizerOrganic, logic.FertilizerBoth:
		organicTargets := plantedIDs
		if len(latestLands) > 0 {
			organicTargets = logic.GetOrganicFertilizerTargetsFromLands(latestLands)
		}
		if len(types) > 0 {
			organicTargets = logic.FilterLandIDsByTypes(organicTargets, types, selectedTypes)
		}
		fertilizeOrganic(ctx, api, organicTargets, actions, errs, opts)
	case logic.FertilizerSmart:
		// Bot smart organic: re-fetch after normal fert so shortened mature times are visible.
		threshold := int64(cfg.Automation.FertilizerSmartSeconds)
		smartLands := latestLands
		if !skipNormal && len(normalTargets) > 0 {
			if again, againReply, againErr := api.AllLands(ctx); againErr == nil {
				smartLands = again
				if againReply != nil {
					feedOperationLimits(opts, againReply.OperationLimits)
				}
			}
		}
		organicTargets := logic.GetFastMatureLands(smartLands, threshold)
		fertilizeOrganic(ctx, api, organicTargets, actions, errs, opts)
	}
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
	for _, item := range items {
		if logic.GetPlantByFruitID(item.Id) == nil {
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

func fertilize(ctx context.Context, api *game.API, landIDs []int64, fertilizerID int64, label string, actions *[]string, errs *[]error, opts farmOperationOptions) {
	if len(landIDs) == 0 {
		return
	}
	count, err := api.Fertilize(ctx, landIDs, fertilizerID)
	if err != nil {
		*errs = append(*errs, fmt.Errorf("%s: %w", label, err))
	}
	if count > 0 {
		*actions = append(*actions, fmt.Sprintf("%s%d", label, count))
		recordOpCount(opts, "fertilize", count)
	}
}

func fertilizeOrganic(ctx context.Context, api *game.API, landIDs []int64, actions *[]string, errs *[]error, opts farmOperationOptions) {
	if len(landIDs) == 0 {
		return
	}
	count, err := api.FertilizeOrganicLoop(ctx, landIDs)
	if err != nil {
		*errs = append(*errs, fmt.Errorf("有机肥: %w", err))
	}
	if count > 0 {
		*actions = append(*actions, fmt.Sprintf("有机肥%d", count))
		recordOpCount(opts, "fertilize", count)
	}
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
