package logic

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// PlantItem is a Plant.json row (fields used by logic).
type PlantItem struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	SeedID        *int64 `json:"seed_id"`
	Size          *int64 `json:"size"`
	Seasons       int64  `json:"seasons"`
	Exp           int64  `json:"exp"`
	GrowPhases    string `json:"grow_phases"`
	LandLevelNeed int64  `json:"land_level_need"`
	Fruit         struct {
		ID    int64 `json:"id"`
		Count int64 `json:"count"`
	} `json:"fruit"`
}

// LandConfigItem is a Land.json row.
type LandConfigItem struct {
	ID    int64 `json:"id"`
	GridX int   `json:"grid_x"`
	GridY int   `json:"grid_y"`
}

// ItemInfo is an ItemInfo.json row.
type ItemInfo struct {
	ID              int64   `json:"id"`
	Type            int64   `json:"type"`
	Name            string  `json:"name"`
	InteractionType string  `json:"interaction_type"`
	Sells           *string `json:"sells"`
	SellCond        *string `json:"sell_cond"`
	CondSells       *string `json:"cond_sells"`
	Level           *int64  `json:"level"`
	TargetID        int64   `json:"target_id"`
	AssetName       string  `json:"asset_name"`
	IconRes         string  `json:"icon_res"`
	MaxCount        int64   `json:"max_count"`
	MaxOwn          int64   `json:"max_own"`
	CanUse          int64   `json:"can_use"`
	Desc            string  `json:"desc"`
	EffectDesc      string  `json:"effectDesc"`
	TraitID         int64   `json:"trait_id"`
	Layer           int64   `json:"layer"`
	Rarity          int64   `json:"rarity"`
	RarityColor     string  `json:"rarity_color"`
	Jumps           string  `json:"jumps"`
}

// SeedInfo is the catalog shape returned for seeds (aligned with qq-farm-bot).
type SeedInfo struct {
	SeedID        int64  `json:"seedId"`
	Name          string `json:"name"`
	RequiredLevel int64  `json:"requiredLevel"`
	Price         int64  `json:"price"`
	PriceID       int64  `json:"priceId"`
	Image         string `json:"image"`
	Seasons       int64  `json:"seasons"`
	Exp           int64  `json:"exp"`
	GrowPhases    string `json:"growPhases"`
	GrowTime      int64  `json:"growTime"`
	Size          int64  `json:"size"`
	HarvestCount  int64  `json:"harvestCount"`
}

// SellEntry is one currency:price pair from sells / cond_sells.
type SellEntry struct {
	CurrencyID int64 `json:"currencyId"`
	Price      int64 `json:"price"`
}

// GameConfig holds loaded Plant/Land/ItemInfo lookups.
type GameConfig struct {
	mu               sync.RWMutex
	dir              string
	plantByID        map[int64]*PlantItem
	plantBySeed      map[int64]*PlantItem
	plantByFruit     map[int64]*PlantItem
	itemByID         map[int64]*ItemInfo
	seedItemByID     map[int64]*ItemInfo
	landByID         map[int64]*LandConfigItem
	landByCoordinate map[string]*LandConfigItem
	levelExpTable    map[int64]int64 // RoleLevel.json: level → cumulative exp at level start
}

// GlobalGameConfig is the process-wide config (loaded via LoadGameConfig).
var GlobalGameConfig = &GameConfig{
	plantByID:        make(map[int64]*PlantItem),
	plantBySeed:      make(map[int64]*PlantItem),
	plantByFruit:     make(map[int64]*PlantItem),
	itemByID:         make(map[int64]*ItemInfo),
	seedItemByID:     make(map[int64]*ItemInfo),
	landByID:         make(map[int64]*LandConfigItem),
	landByCoordinate: make(map[string]*LandConfigItem),
}

// LoadGameConfig loads Plant.json, Land.json and ItemInfo.json from dir.
func LoadGameConfig(dir string) error {
	return GlobalGameConfig.Load(dir)
}

// Dir returns the config directory last loaded.
func (g *GameConfig) Dir() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.dir
}

// Load populates g from JSON files under dir.
func (g *GameConfig) Load(dir string) error {
	plantPath := filepath.Join(dir, "Plant.json")
	landPath := filepath.Join(dir, "Land.json")
	itemPath := filepath.Join(dir, "ItemInfo.json")

	plantRaw, err := os.ReadFile(plantPath)
	if err != nil {
		return fmt.Errorf("load Plant.json: %w", err)
	}
	var plants []PlantItem
	if err := json.Unmarshal(plantRaw, &plants); err != nil {
		return fmt.Errorf("parse Plant.json: %w", err)
	}

	landRaw, err := os.ReadFile(landPath)
	if err != nil {
		return fmt.Errorf("load Land.json: %w", err)
	}
	var lands []LandConfigItem
	if err := json.Unmarshal(landRaw, &lands); err != nil {
		return fmt.Errorf("parse Land.json: %w", err)
	}

	itemRaw, err := os.ReadFile(itemPath)
	if err != nil {
		return fmt.Errorf("load ItemInfo.json: %w", err)
	}
	items, err := parseItemInfoList(itemRaw)
	if err != nil {
		return fmt.Errorf("parse ItemInfo.json: %w", err)
	}

	plantByID := make(map[int64]*PlantItem, len(plants))
	plantBySeed := make(map[int64]*PlantItem)
	plantByFruit := make(map[int64]*PlantItem)
	for i := range plants {
		p := &plants[i]
		plantByID[p.ID] = p
		if p.SeedID != nil && *p.SeedID > 0 {
			plantBySeed[*p.SeedID] = p
		}
		if p.Fruit.ID > 0 {
			plantByFruit[p.Fruit.ID] = p
		}
	}
	landByID := make(map[int64]*LandConfigItem, len(lands))
	landByCoord := make(map[string]*LandConfigItem)
	for i := range lands {
		l := &lands[i]
		if l.ID <= 0 {
			continue
		}
		landByID[l.ID] = l
		landByCoord[landCoordKey(l.GridX, l.GridY)] = l
	}
	itemByID := make(map[int64]*ItemInfo, len(items))
	seedItemByID := make(map[int64]*ItemInfo)
	for i := range items {
		it := &items[i]
		if it.ID <= 0 {
			continue
		}
		itemByID[it.ID] = it
		if it.Type == 5 {
			seedItemByID[it.ID] = it
		}
	}

	levelExpTable := loadRoleLevelTable(dir)

	g.mu.Lock()
	g.dir = dir
	g.plantByID = plantByID
	g.plantBySeed = plantBySeed
	g.plantByFruit = plantByFruit
	g.itemByID = itemByID
	g.seedItemByID = seedItemByID
	g.landByID = landByID
	g.landByCoordinate = landByCoord
	g.levelExpTable = levelExpTable
	g.mu.Unlock()
	return nil
}

type roleLevelRow struct {
	Level int64 `json:"level"`
	Exp   int64 `json:"exp"`
}

func loadRoleLevelTable(dir string) map[int64]int64 {
	raw, err := os.ReadFile(filepath.Join(dir, "RoleLevel.json"))
	if err != nil {
		return map[int64]int64{}
	}
	var rows []roleLevelRow
	if err := json.Unmarshal(raw, &rows); err != nil {
		return map[int64]int64{}
	}
	out := make(map[int64]int64, len(rows))
	for _, row := range rows {
		if row.Level > 0 {
			out[row.Level] = row.Exp
		}
	}
	return out
}

// LevelExpProgress is the current-level exp bar (bot getLevelExpProgress).
type LevelExpProgress struct {
	Current int64 `json:"current"`
	Needed  int64 `json:"needed"`
	Level   int64 `json:"level"`
}

// GetLevelExpProgress returns exp within the current level and exp needed for next.
func (g *GameConfig) GetLevelExpProgress(level, totalExp int64) LevelExpProgress {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if level <= 0 || len(g.levelExpTable) == 0 {
		return LevelExpProgress{Level: level}
	}
	start := g.levelExpTable[level]
	next, ok := g.levelExpTable[level+1]
	if !ok {
		next = start + 100000
	}
	needed := next - start
	if needed < 0 {
		needed = 0
	}
	current := totalExp - start
	if current < 0 {
		current = 0
	}
	if needed > 0 && current > needed {
		current = needed
	}
	return LevelExpProgress{Current: current, Needed: needed, Level: level}
}

// GetLevelExpProgress uses GlobalGameConfig.
func GetLevelExpProgress(level, totalExp int64) LevelExpProgress {
	return GlobalGameConfig.GetLevelExpProgress(level, totalExp)
}

// parseItemInfoList tolerates null string/number fields and the "trait_id " typo.
func parseItemInfoList(raw []byte) ([]ItemInfo, error) {
	var rows []map[string]any
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	out := make([]ItemInfo, 0, len(rows))
	for _, row := range rows {
		if v, ok := row["trait_id "]; ok {
			if _, has := row["trait_id"]; !has {
				row["trait_id"] = v
			}
			delete(row, "trait_id ")
		}
		out = append(out, ItemInfo{
			ID:              toInt64(row["id"]),
			Type:            toInt64(row["type"]),
			Name:            toString(row["name"]),
			InteractionType: toString(row["interaction_type"]),
			Sells:           toStringPtr(row["sells"]),
			SellCond:        toStringPtr(row["sell_cond"]),
			CondSells:       toStringPtr(row["cond_sells"]),
			Level:           toInt64Ptr(row["level"]),
			TargetID:        toInt64(row["target_id"]),
			AssetName:       toString(row["asset_name"]),
			IconRes:         toString(row["icon_res"]),
			MaxCount:        toInt64(row["max_count"]),
			MaxOwn:          toInt64(row["max_own"]),
			CanUse:          toInt64(row["can_use"]),
			Desc:            toString(row["desc"]),
			EffectDesc:      toString(row["effectDesc"]),
			TraitID:         toInt64(row["trait_id"]),
			Layer:           toInt64(row["layer"]),
			Rarity:          toInt64(row["rarity"]),
			RarityColor:     toString(row["rarity_color"]),
			Jumps:           toString(row["jumps"]),
		})
	}
	return out, nil
}

func toString(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case json.Number:
		return t.String()
	default:
		return fmt.Sprint(t)
	}
}

func toStringPtr(v any) *string {
	if v == nil {
		return nil
	}
	s := toString(v)
	return &s
}

func toInt64(v any) int64 {
	if v == nil {
		return 0
	}
	switch t := v.(type) {
	case float64:
		return int64(t)
	case json.Number:
		n, _ := t.Int64()
		return n
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		return n
	case int64:
		return t
	case int:
		return int64(t)
	default:
		return 0
	}
}

func toInt64Ptr(v any) *int64 {
	if v == nil {
		return nil
	}
	n := toInt64(v)
	return &n
}

func landCoordKey(x, y int) string { return strconv.Itoa(x) + "," + strconv.Itoa(y) }

// GetPlantByID looks up a plant.
func (g *GameConfig) GetPlantByID(id int64) *PlantItem {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.plantByID[id]
}

// GetPlantBySeedID looks up plant by seed id.
func (g *GameConfig) GetPlantBySeedID(seedID int64) *PlantItem {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.plantBySeed[seedID]
}

// GetPlantByFruitID looks up plant by fruit id.
func (g *GameConfig) GetPlantByFruitID(fruitID int64) *PlantItem {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.plantByFruit[fruitID]
}

// GetPlantName returns display name.
func (g *GameConfig) GetPlantName(plantID int64) string {
	p := g.GetPlantByID(plantID)
	if p == nil {
		return fmt.Sprintf("植物%d", plantID)
	}
	return p.Name
}

// GetPlantNameBySeedID returns display name for a seed.
func (g *GameConfig) GetPlantNameBySeedID(seedID int64) string {
	p := g.GetPlantBySeedID(seedID)
	if p == nil {
		return fmt.Sprintf("种子%d", seedID)
	}
	return p.Name
}

// GetPlantExp returns exp for plant id.
func (g *GameConfig) GetPlantExp(plantID int64) int64 {
	p := g.GetPlantByID(plantID)
	if p == nil {
		return 0
	}
	return p.Exp
}

var growPhaseRe = regexp.MustCompile(`:(\d+)`)

// GetPlantGrowTime sums grow_phases seconds.
func (g *GameConfig) GetPlantGrowTime(plantID int64) int64 {
	p := g.GetPlantByID(plantID)
	if p == nil || p.GrowPhases == "" {
		return 0
	}
	var total int64
	for _, part := range strings.Split(p.GrowPhases, ";") {
		if part == "" {
			continue
		}
		m := growPhaseRe.FindStringSubmatch(part)
		if len(m) == 2 {
			n, _ := strconv.ParseInt(m[1], 10, 64)
			total += n
		}
	}
	return total
}

// GetLandConfigByID returns land grid config.
func (g *GameConfig) GetLandConfigByID(id int64) *LandConfigItem {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.landByID[id]
}

// GetLandConfigByCoordinate returns land at grid position.
func (g *GameConfig) GetLandConfigByCoordinate(gridX, gridY int) *LandConfigItem {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.landByCoordinate[landCoordKey(gridX, gridY)]
}

// GetItemByID returns an item info row.
func (g *GameConfig) GetItemByID(id int64) *ItemInfo {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.itemByID[id]
}

// ActivitySellCond captures a sell_cond that gates selling on an activity.
type ActivitySellCond struct {
	ActivityID string // referenced activity id, e.g. "2026080102"
}

// ParseActivitySellCond parses "活动结束后:<activityId>" style sell_cond.
// Returns nil when the condition is not activity-restricted.
func ParseActivitySellCond(sellCond *string) *ActivitySellCond {
	if sellCond == nil {
		return nil
	}
	raw := strings.TrimSpace(*sellCond)
	const prefix = "活动结束后:"
	if !strings.HasPrefix(raw, prefix) {
		return nil
	}
	id := strings.TrimSpace(strings.TrimPrefix(raw, prefix))
	if id == "" {
		return nil
	}
	return &ActivitySellCond{ActivityID: id}
}

// IsActivityRestrictedForSale reports whether itemID must NOT be sold at now
// (Unix seconds) because its sell_cond references an activity that is still
// running. Unknown activities are treated as active (conservative skip).
//
// Recurring activities like 青梅 get a fresh id every run while the item
// config may keep a stale id in sell_cond. When the referenced activity has
// ended but another instance of the same activity type is ongoing, the item is
// still treated as restricted.
func IsActivityRestrictedForSale(itemID int64, now int64) bool {
	item := GetItemByID(itemID)
	if item == nil {
		return false
	}
	cond := ParseActivitySellCond(item.SellCond)
	if cond == nil {
		return false
	}
	if ActivityActive(cond.ActivityID, now) {
		return true
	}
	ref, ok := activityByID(cond.ActivityID)
	if !ok || ref.Type <= 0 {
		return false
	}
	for _, act := range ActivityRegistrySnapshot() {
		if act.Type != ref.Type {
			continue
		}
		if act.ActivityID == cond.ActivityID {
			continue
		}
		if ActivityActive(act.ActivityID, now) {
			return true
		}
	}
	return false
}

// ParseSells parses "currencyId:price;..." strings.
func ParseSells(sells string) []SellEntry {
	sells = strings.TrimSpace(sells)
	if sells == "" {
		return nil
	}
	parts := strings.Split(sells, ";")
	out := make([]SellEntry, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, ":", 2)
		if len(kv) != 2 {
			continue
		}
		cid, _ := strconv.ParseInt(strings.TrimSpace(kv[0]), 10, 64)
		price, _ := strconv.ParseInt(strings.TrimSpace(kv[1]), 10, 64)
		out = append(out, SellEntry{CurrencyID: cid, Price: price})
	}
	return out
}

func sellsOrCond(item *ItemInfo) []SellEntry {
	if item == nil {
		return nil
	}
	if item.Sells != nil {
		if list := ParseSells(*item.Sells); len(list) > 0 {
			return list
		}
	}
	if item.CondSells != nil {
		return ParseSells(*item.CondSells)
	}
	return nil
}

// EffectiveSellStatus classifies how (and whether) an item can be sold,
// mirroring the reference bot's getEffectiveSellInfo.
type EffectiveSellStatus string

const (
	SellStatusAvailable   EffectiveSellStatus = "available"
	SellStatusConditional EffectiveSellStatus = "conditional"
	SellStatusUnavailable EffectiveSellStatus = "unavailable"
)

// EffectiveSellInfo is the resolved sell state of an item.
type EffectiveSellInfo struct {
	Sellable  bool                `json:"sellable"`
	Status    EffectiveSellStatus `json:"sellStatus"`
	Condition string              `json:"sellCondition,omitempty"`
	Sells     []SellEntry         `json:"sells"`
}

// GetEffectiveSellInfo resolves whether item can be sold right now.
//
// An item is directly sellable only when its plain sells list carries a real
// price. Conditional sells (cond_sells) alone do NOT make an item sellable —
// those prices only apply once the referenced activity/condition is met, so
// they must not be shown or used for direct selling. This mirrors the bot's
// getEffectiveSellInfo fix for fruits that appeared sellable with a cond_sells
// fallback price.
func GetEffectiveSellInfo(item *ItemInfo) EffectiveSellInfo {
	if item == nil {
		return EffectiveSellInfo{Status: SellStatusUnavailable}
	}
	var normal []SellEntry
	if item.Sells != nil {
		for _, sell := range ParseSells(*item.Sells) {
			if sell.CurrencyID > 0 && sell.Price > 0 {
				normal = append(normal, sell)
			}
		}
	}
	condition := ""
	if item.SellCond != nil {
		condition = strings.TrimSpace(*item.SellCond)
	}
	var conditional []SellEntry
	if item.CondSells != nil {
		for _, sell := range ParseSells(*item.CondSells) {
			if sell.CurrencyID > 0 && sell.Price > 0 {
				conditional = append(conditional, sell)
			}
		}
	}
	if len(normal) > 0 {
		return EffectiveSellInfo{Sellable: true, Status: SellStatusAvailable, Condition: condition, Sells: normal}
	}
	if condition != "" && len(conditional) > 0 {
		return EffectiveSellInfo{Status: SellStatusConditional, Condition: condition}
	}
	return EffectiveSellInfo{Status: SellStatusUnavailable, Condition: condition}
}

// GetSeedPrice returns the primary sell price for a seed.
func (g *GameConfig) GetSeedPrice(seedID int64) int64 {
	g.mu.RLock()
	item := g.seedItemByID[seedID]
	g.mu.RUnlock()
	list := sellsOrCond(item)
	if len(list) == 0 {
		return 0
	}
	return list[0].Price
}

// GetFruitPrice returns the primary sell price for a fruit item.
func (g *GameConfig) GetFruitPrice(fruitID int64) int64 {
	if fruitID <= 0 {
		return 0
	}
	g.mu.RLock()
	item := g.itemByID[fruitID]
	g.mu.RUnlock()
	list := sellsOrCond(item)
	if len(list) == 0 {
		return 0
	}
	return list[0].Price
}

// SeedImagePath returns the public static URL for a seed/item icon.
func SeedImagePath(id int64) string {
	if id <= 0 {
		return ""
	}
	return fmt.Sprintf("/game-config/seed_images_named/%d.png", id)
}

// GetAllSeeds returns catalog seed rows.
func (g *GameConfig) GetAllSeeds() []SeedInfo {
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := make([]SeedInfo, 0, len(g.plantBySeed))
	for seedID, p := range g.plantBySeed {
		level := p.LandLevelNeed
		var priceID, price int64
		if si := g.seedItemByID[seedID]; si != nil {
			if si.Level != nil {
				level = *si.Level
			}
			list := sellsOrCond(si)
			if len(list) > 0 {
				priceID = list[0].CurrencyID
				price = list[0].Price
			}
		}
		var size int64
		if p.Size != nil {
			size = *p.Size
		}
		out = append(out, SeedInfo{
			SeedID:        seedID,
			Name:          p.Name,
			RequiredLevel: level,
			Price:         price,
			PriceID:       priceID,
			Image:         SeedImagePath(seedID),
			Seasons:       p.Seasons,
			Exp:           p.Exp,
			GrowPhases:    p.GrowPhases,
			GrowTime:      growTimeFromPhases(p.GrowPhases),
			Size:          size,
			HarvestCount:  p.Fruit.Count,
		})
	}
	return out
}

func growTimeFromPhases(phases string) int64 {
	if phases == "" {
		return 0
	}
	var total int64
	for _, part := range strings.Split(phases, ";") {
		if part == "" {
			continue
		}
		m := growPhaseRe.FindStringSubmatch(part)
		if len(m) == 2 {
			n, _ := strconv.ParseInt(m[1], 10, 64)
			total += n
		}
	}
	return total
}

// GetAllFruits returns type=6 items with plant linkage fields.
func (g *GameConfig) GetAllFruits() []map[string]any {
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := make([]map[string]any, 0)
	for _, item := range g.itemByID {
		if item.Type != 6 {
			continue
		}
		list := sellsOrCond(item)
		var price, priceID int64
		if len(list) > 0 {
			price = list[0].Price
			priceID = list[0].CurrencyID
		}
		var plantID, seedID any
		var plantName any
		if plant := g.plantByFruit[item.ID]; plant != nil {
			plantID = plant.ID
			plantName = plant.Name
			if plant.SeedID != nil {
				seedID = *plant.SeedID
			}
		}
		var level int64
		if item.Level != nil {
			level = *item.Level
		}
		maxCount := item.MaxCount
		if maxCount == 0 {
			maxCount = 9999
		}
		maxOwn := item.MaxOwn
		if maxOwn == 0 {
			maxOwn = 9999
		}
		out = append(out, map[string]any{
			"id":         item.ID,
			"name":       item.Name,
			"type":       item.Type,
			"price":      price,
			"priceId":    priceID,
			"sellCond":   item.SellCond,
			"condSells":  item.CondSells,
			"level":      level,
			"assetName":  item.AssetName,
			"desc":       item.Desc,
			"effectDesc": item.EffectDesc,
			"rarity":     item.Rarity,
			"maxCount":   maxCount,
			"maxOwn":     maxOwn,
			"plantId":    plantID,
			"seedId":     seedID,
			"plantName":  plantName,
			"image":      SeedImagePath(item.ID),
		})
	}
	return out
}

// GetAllItems returns non-seed/non-fruit items; optional type filter.
func (g *GameConfig) GetAllItems(typeFilter int64) []map[string]any {
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := make([]map[string]any, 0)
	for _, item := range g.itemByID {
		if typeFilter > 0 {
			if item.Type != typeFilter {
				continue
			}
		} else if item.Type == 5 || item.Type == 6 {
			continue
		}
		list := sellsOrCond(item)
		var price, priceID int64
		if len(list) > 0 {
			price = list[0].Price
			priceID = list[0].CurrencyID
		}
		var level int64
		if item.Level != nil {
			level = *item.Level
		}
		maxCount := item.MaxCount
		if maxCount == 0 {
			maxCount = 9999
		}
		maxOwn := item.MaxOwn
		if maxOwn == 0 {
			maxOwn = 9999
		}
		out = append(out, map[string]any{
			"id":              item.ID,
			"type":            item.Type,
			"name":            item.Name,
			"interactionType": item.InteractionType,
			"priceId":         priceID,
			"price":           price,
			"sellCond":        item.SellCond,
			"condSells":       item.CondSells,
			"level":           level,
			"assetName":       item.AssetName,
			"iconRes":         item.IconRes,
			"maxCount":        maxCount,
			"maxOwn":          maxOwn,
			"canUse":          item.CanUse,
			"desc":            item.Desc,
			"effectDesc":      item.EffectDesc,
			"traitId":         item.TraitID,
			"layer":           item.Layer,
			"rarity":          item.Rarity,
			"rarityColor":     item.RarityColor,
			"jumps":           item.Jumps,
			"image":           SeedImagePath(item.ID),
		})
	}
	return out
}

// GetAllPlants returns plant catalog rows for UI.
func (g *GameConfig) GetAllPlants() []map[string]any {
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := make([]map[string]any, 0, len(g.plantByID))
	for _, p := range g.plantByID {
		var seedID, fruitID any
		var price int64
		var image string
		level := p.LandLevelNeed
		if p.SeedID != nil && *p.SeedID > 0 {
			seedID = *p.SeedID
			image = SeedImagePath(*p.SeedID)
			if si := g.seedItemByID[*p.SeedID]; si != nil {
				if si.Level != nil {
					level = *si.Level
				}
				list := sellsOrCond(si)
				if len(list) > 0 {
					price = list[0].Price
				}
			}
		}
		if p.Fruit.ID > 0 {
			fruitID = p.Fruit.ID
		}
		out = append(out, map[string]any{
			"plantId":       p.ID,
			"name":          p.Name,
			"seedId":        seedID,
			"fruitId":       fruitID,
			"fruitCount":    p.Fruit.Count,
			"landLevelNeed": level,
			"seasons":       p.Seasons,
			"growPhases":    p.GrowPhases,
			"exp":           p.Exp,
			"price":         price,
			"image":         image,
		})
	}
	return out
}

// ItemTypes returns static item type options for UI selects.
func ItemTypes() []map[string]any {
	return []map[string]any{
		{"value": 1, "label": "特殊道具"},
		{"value": 2, "label": "货币"},
		{"value": 3, "label": "经验"},
		{"value": 4, "label": "农场工具"},
		{"value": 7, "label": "化肥"},
		{"value": 8, "label": "宠物"},
		{"value": 9, "label": "宠物食品"},
		{"value": 10, "label": "头像框"},
		{"value": 11, "label": "礼品盒"},
		{"value": 12, "label": "收藏点"},
		{"value": 13, "label": "活跃点"},
		{"value": 14, "label": "解锁卡"},
		{"value": 15, "label": "高级货币"},
		{"value": 16, "label": "自选礼包"},
		{"value": 17, "label": "变异果实"},
		{"value": 18, "label": "皮肤/装饰"},
		{"value": 23, "label": "虫虫道具"},
	}
}

// Package-level helpers use GlobalGameConfig.

func GetPlantName(plantID int64) string        { return GlobalGameConfig.GetPlantName(plantID) }
func GetPlantExp(plantID int64) int64          { return GlobalGameConfig.GetPlantExp(plantID) }
func GetPlantBySeedID(seedID int64) *PlantItem { return GlobalGameConfig.GetPlantBySeedID(seedID) }
func GetPlantNameBySeedID(seedID int64) string {
	return GlobalGameConfig.GetPlantNameBySeedID(seedID)
}
func GetPlantGrowTime(plantID int64) int64 { return GlobalGameConfig.GetPlantGrowTime(plantID) }
func GetLandConfigByID(id int64) *LandConfigItem {
	return GlobalGameConfig.GetLandConfigByID(id)
}
func GetLandConfigByCoordinate(x, y int) *LandConfigItem {
	return GlobalGameConfig.GetLandConfigByCoordinate(x, y)
}
func GetItemByID(id int64) *ItemInfo { return GlobalGameConfig.GetItemByID(id) }
func GetAllSeeds() []SeedInfo        { return GlobalGameConfig.GetAllSeeds() }
func GetAllFruits() []map[string]any { return GlobalGameConfig.GetAllFruits() }
func GetAllItems(typeFilter int64) []map[string]any {
	return GlobalGameConfig.GetAllItems(typeFilter)
}
func GetAllPlants() []map[string]any { return GlobalGameConfig.GetAllPlants() }
func GetPlantByFruitID(fruitID int64) *PlantItem {
	return GlobalGameConfig.GetPlantByFruitID(fruitID)
}

// FormatGrowTime formats seconds like the Node helper.
func FormatGrowTime(seconds int64) string {
	if seconds < 60 {
		return fmt.Sprintf("%d秒", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%d分钟", seconds/60)
	}
	hours := seconds / 3600
	mins := (seconds % 3600) / 60
	if mins > 0 {
		return fmt.Sprintf("%d小时%d分", hours, mins)
	}
	return fmt.Sprintf("%d小时", hours)
}
