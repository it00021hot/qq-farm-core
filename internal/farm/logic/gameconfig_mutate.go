package logic

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SeedWriteReq fields for add/modify seed.
type SeedWriteReq struct {
	SeedID        int64  `json:"seedId"`
	Name          string `json:"name"`
	GrowPhases    string `json:"growPhases"`
	LandLevelNeed int64  `json:"landLevelNeed"`
	Seasons       int64  `json:"seasons"`
	FruitCount    int64  `json:"fruitCount"`
	Price         int64  `json:"price"`
	PriceID       int64  `json:"priceId"`
	Exp           int64  `json:"exp"`
	Size          int64  `json:"size"`
}

// FruitWriteReq fields for add/modify fruit.
type FruitWriteReq struct {
	ID         int64  `json:"id"`
	PlantID    int64  `json:"plantId"`
	Name       string `json:"name"`
	Price      int64  `json:"price"`
	PriceID    int64  `json:"priceId"`
	Desc       string `json:"desc"`
	EffectDesc string `json:"effectDesc"`
	Rarity     int64  `json:"rarity"`
	MaxCount   int64  `json:"maxCount"`
	Level      int64  `json:"level"`
	FruitCount int64  `json:"fruitCount"`
	AssetName  string `json:"assetName"`
}

// ItemWriteReq fields for add/modify generic item.
type ItemWriteReq struct {
	ID              int64  `json:"id"`
	Type            int64  `json:"type"`
	Name            string `json:"name"`
	Price           int64  `json:"price"`
	PriceID         int64  `json:"priceId"`
	InteractionType string `json:"interactionType"`
	CanUse          int64  `json:"canUse"`
	Desc            string `json:"desc"`
	EffectDesc      string `json:"effectDesc"`
	Rarity          int64  `json:"rarity"`
	MaxCount        int64  `json:"maxCount"`
	Level           int64  `json:"level"`
	AssetName       string `json:"assetName"`
}

func readJSONArray(path string) ([]map[string]any, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rows []map[string]any
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

func writeJSONArray(path string, rows []map[string]any) error {
	raw, err := json.MarshalIndent(rows, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

func formatSells(priceID, price int64) any {
	if price <= 0 {
		return nil
	}
	if priceID <= 0 {
		priceID = 1001
	}
	return fmt.Sprintf("%d:%d", priceID, price)
}

// AddSeed creates plant + seed + fruit entries (aligned with qq-farm-bot).
func (g *GameConfig) AddSeed(req SeedWriteReq) (map[string]any, error) {
	if req.SeedID <= 0 {
		return nil, fmt.Errorf("种子ID必须为正整数")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("作物名称不能为空")
	}
	growPhases := strings.TrimSpace(req.GrowPhases)
	if growPhases == "" {
		return nil, fmt.Errorf("生长阶段不能为空")
	}
	if req.LandLevelNeed <= 0 {
		return nil, fmt.Errorf("等级要求必须为正整数")
	}
	if req.Seasons == 0 {
		req.Seasons = 1
	}
	if req.Seasons != 1 && req.Seasons != 2 {
		return nil, fmt.Errorf("季节数必须为1或2")
	}
	if req.FruitCount <= 0 {
		return nil, fmt.Errorf("收获数量必须为正整数")
	}
	if req.Price < 0 {
		return nil, fmt.Errorf("种子价格不能为负数")
	}
	if req.PriceID <= 0 {
		req.PriceID = 1001
	}
	if g.GetPlantBySeedID(req.SeedID) != nil {
		return nil, fmt.Errorf("种子ID %d 已存在", req.SeedID)
	}

	g.mu.Lock()
	defer g.mu.Unlock()
	dir := g.dir
	if dir == "" {
		return nil, fmt.Errorf("游戏配置未加载")
	}

	plantPath := filepath.Join(dir, "Plant.json")
	itemPath := filepath.Join(dir, "ItemInfo.json")
	plants, err := readJSONArray(plantPath)
	if err != nil {
		return nil, err
	}
	items, err := readJSONArray(itemPath)
	if err != nil {
		return nil, err
	}

	plantID := 1000000 + req.SeedID
	fruitID := 20000 + req.SeedID
	assetName := fmt.Sprintf("Crop_%d", req.SeedID)

	plantEntry := map[string]any{
		"id":                       plantID,
		"name":                     name,
		"mutant_effect_plant":      nil,
		"special_fruit":            nil,
		"fruit":                    map[string]any{"id": fruitID, "count": req.FruitCount},
		"seed_id":                  req.SeedID,
		"land_level_need":          req.LandLevelNeed,
		"seasons":                  req.Seasons,
		"grow_phases":              growPhases,
		"exp":                      req.Exp,
		"size":                     nil,
		"offsetPosition":           map[string]any{"x": 0, "y": 0},
		"mutantEffectScale":        map[string]any{"x": 1, "y": 1},
		"harvestOffsetPosition":    map[string]any{"x": -35, "y": 40},
		"harvestRandom":            nil,
		"harvestAllSpineRes":       nil,
		"harvestAllOffsetPosition": nil,
		"harvestAniName":           "anim_harvest_putong",
		"all_state_spine":          nil,
		"mature_effect":            "effect/prefab/effect_plant_maturation",
		"mature_effect_offset":     map[string]any{"x": 0, "y": 0},
		"rare_plant_light_pos":     nil,
		"exp_root":                 0,
		"exp_alter":                0,
		"fruit_root":               0,
		"fruit_alter":              0,
	}
	if req.Size > 0 {
		plantEntry["size"] = req.Size
	}

	seedItem := map[string]any{
		"id":               req.SeedID,
		"type":             5,
		"name":             name + "种子",
		"interaction_type": "plant",
		"sells":            formatSells(req.PriceID, req.Price),
		"sell_cond":        nil,
		"cond_sells":       nil,
		"level":            req.LandLevelNeed,
		"target_id":        0,
		"asset_name":       assetName,
		"icon_res":         "",
		"max_count":        9999,
		"max_own":          9999,
		"duration":         nil,
		"can_use":          0,
		"desc":             fmt.Sprintf("种植后，可以收获一定数量的%s。", name),
		"effectDesc":       name,
		"trait_id ":        0,
		"layer":            13,
		"rarity":           1,
		"rarity_color":     "D2C5AC",
		"jumps":            "",
		"ware_scale":       nil,
	}
	fruitPrice := int64(0)
	if req.Price > 0 {
		fruitPrice = req.Price / 4
		if fruitPrice == 0 {
			fruitPrice = 1
		}
	}
	fruitItem := map[string]any{
		"id":               fruitID,
		"type":             6,
		"name":             name,
		"interaction_type": "",
		"sells":            formatSells(req.PriceID, fruitPrice),
		"sell_cond":        nil,
		"cond_sells":       nil,
		"level":            req.LandLevelNeed,
		"target_id":        0,
		"asset_name":       assetName,
		"icon_res":         "",
		"max_count":        999,
		"max_own":          999,
		"duration":         nil,
		"can_use":          0,
		"desc":             fmt.Sprintf("%s的果实，可以出售换取金币。", name),
		"effectDesc":       name,
		"trait_id ":        0,
		"layer":            0,
		"rarity":           1,
		"rarity_color":     "D2C5AC",
		"jumps":            "",
		"ware_scale":       nil,
	}

	plants = append(plants, plantEntry)
	items = append(items, seedItem, fruitItem)
	if err := writeJSONArray(plantPath, plants); err != nil {
		return nil, err
	}
	if err := writeJSONArray(itemPath, items); err != nil {
		return nil, err
	}
	if err := g.reloadLocked(dir); err != nil {
		return nil, err
	}
	return map[string]any{
		"plantId": plantID,
		"seedId":  req.SeedID,
		"fruitId": fruitID,
		"name":    name,
	}, nil
}

// ModifySeed updates plant/seed/fruit for an existing seed.
func (g *GameConfig) ModifySeed(req SeedWriteReq) (map[string]any, error) {
	if req.SeedID <= 0 {
		return nil, fmt.Errorf("无效的种子ID")
	}
	plant := g.GetPlantBySeedID(req.SeedID)
	if plant == nil {
		return nil, fmt.Errorf("种子ID %d 不存在", req.SeedID)
	}

	g.mu.Lock()
	defer g.mu.Unlock()
	dir := g.dir
	plantPath := filepath.Join(dir, "Plant.json")
	itemPath := filepath.Join(dir, "ItemInfo.json")
	plants, err := readJSONArray(plantPath)
	if err != nil {
		return nil, err
	}
	items, err := readJSONArray(itemPath)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(req.Name)
	for _, p := range plants {
		if toInt64(p["id"]) != plant.ID {
			continue
		}
		if name != "" {
			p["name"] = name
		}
		if gp := strings.TrimSpace(req.GrowPhases); gp != "" {
			p["grow_phases"] = gp
		}
		if req.LandLevelNeed > 0 {
			p["land_level_need"] = req.LandLevelNeed
		}
		if req.Seasons == 1 || req.Seasons == 2 {
			p["seasons"] = req.Seasons
		}
		if req.Exp > 0 || req.Name != "" {
			p["exp"] = req.Exp
		}
		if req.Size > 0 {
			p["size"] = req.Size
		}
		if req.FruitCount > 0 {
			fruit, _ := p["fruit"].(map[string]any)
			if fruit == nil {
				fruit = map[string]any{}
			}
			fruit["count"] = req.FruitCount
			p["fruit"] = fruit
		}
		break
	}

	fruitID := plant.Fruit.ID
	for _, item := range items {
		id := toInt64(item["id"])
		typ := toInt64(item["type"])
		if id == req.SeedID && typ == 5 {
			if name != "" {
				item["name"] = name + "种子"
				item["effectDesc"] = name
				item["desc"] = fmt.Sprintf("种植后，可以收获一定数量的%s。", name)
			}
			if req.Price >= 0 && (req.Price > 0 || req.PriceID > 0 || req.Name != "") {
				priceID := req.PriceID
				if priceID <= 0 {
					priceID = 1001
				}
				item["sells"] = formatSells(priceID, req.Price)
			}
			if req.LandLevelNeed > 0 {
				item["level"] = req.LandLevelNeed
			}
		}
		if fruitID > 0 && id == fruitID && typ == 6 && name != "" {
			item["name"] = name
			item["effectDesc"] = name
			item["desc"] = fmt.Sprintf("%s的果实，可以出售换取金币。", name)
		}
	}

	if err := writeJSONArray(plantPath, plants); err != nil {
		return nil, err
	}
	if err := writeJSONArray(itemPath, items); err != nil {
		return nil, err
	}
	if err := g.reloadLocked(dir); err != nil {
		return nil, err
	}
	if name == "" {
		name = plant.Name
	}
	return map[string]any{"seedId": req.SeedID, "name": name}, nil
}

// DeleteSeed removes plant + seed + fruit for a seed id.
func (g *GameConfig) DeleteSeed(seedID int64) (map[string]any, error) {
	if seedID <= 0 {
		return nil, fmt.Errorf("无效的种子ID")
	}
	plant := g.GetPlantBySeedID(seedID)
	if plant == nil {
		return nil, fmt.Errorf("种子ID %d 不存在", seedID)
	}
	plantID := plant.ID
	fruitID := plant.Fruit.ID
	name := plant.Name

	g.mu.Lock()
	defer g.mu.Unlock()
	dir := g.dir
	plantPath := filepath.Join(dir, "Plant.json")
	itemPath := filepath.Join(dir, "ItemInfo.json")
	plants, err := readJSONArray(plantPath)
	if err != nil {
		return nil, err
	}
	items, err := readJSONArray(itemPath)
	if err != nil {
		return nil, err
	}

	newPlants := make([]map[string]any, 0, len(plants))
	for _, p := range plants {
		if toInt64(p["id"]) == plantID {
			continue
		}
		newPlants = append(newPlants, p)
	}
	newItems := make([]map[string]any, 0, len(items))
	for _, item := range items {
		id := toInt64(item["id"])
		typ := toInt64(item["type"])
		if id == seedID && typ == 5 {
			continue
		}
		if fruitID > 0 && id == fruitID && typ == 6 {
			continue
		}
		newItems = append(newItems, item)
	}
	if err := writeJSONArray(plantPath, newPlants); err != nil {
		return nil, err
	}
	if err := writeJSONArray(itemPath, newItems); err != nil {
		return nil, err
	}
	if err := g.reloadLocked(dir); err != nil {
		return nil, err
	}
	return map[string]any{"seedId": seedID, "plantId": plantID, "fruitId": fruitID, "name": name}, nil
}

// AddFruit adds a fruit item linked to a plant.
func (g *GameConfig) AddFruit(req FruitWriteReq) (map[string]any, error) {
	if req.PlantID <= 0 {
		return nil, fmt.Errorf("请选择关联的植物")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("果实名称不能为空")
	}
	plant := g.GetPlantByID(req.PlantID)
	if plant == nil {
		return nil, fmt.Errorf("植物ID %d 不存在", req.PlantID)
	}
	if req.PriceID <= 0 {
		req.PriceID = 1001
	}
	if req.MaxCount <= 0 {
		req.MaxCount = 9999
	}

	g.mu.Lock()
	defer g.mu.Unlock()
	dir := g.dir
	itemPath := filepath.Join(dir, "ItemInfo.json")
	plantPath := filepath.Join(dir, "Plant.json")
	items, err := readJSONArray(itemPath)
	if err != nil {
		return nil, err
	}
	existingFruitID := plant.Fruit.ID
	if existingFruitID > 0 {
		for _, item := range items {
			if toInt64(item["id"]) == existingFruitID && toInt64(item["type"]) == 6 {
				return nil, fmt.Errorf("植物「%s」已有果实（ID: %d）", plant.Name, existingFruitID)
			}
		}
	}
	fruitID := existingFruitID
	if fruitID <= 0 {
		base := plant.ID
		if plant.SeedID != nil {
			base = *plant.SeedID
		}
		fruitID = 40000 + base
	}
	assetName := strings.TrimSpace(req.AssetName)
	if assetName == "" {
		sid := plant.ID
		if plant.SeedID != nil {
			sid = *plant.SeedID
		}
		assetName = fmt.Sprintf("Crop_%d", sid)
	}
	level := req.Level
	if level <= 0 {
		level = plant.LandLevelNeed
	}
	fruitEntry := map[string]any{
		"id":               fruitID,
		"type":             6,
		"name":             name,
		"interaction_type": "",
		"sells":            formatSells(req.PriceID, req.Price),
		"sell_cond":        nil,
		"cond_sells":       nil,
		"level":            level,
		"target_id":        0,
		"asset_name":       assetName,
		"icon_res":         "",
		"max_count":        req.MaxCount,
		"max_own":          req.MaxCount,
		"duration":         nil,
		"can_use":          0,
		"desc":             req.Desc,
		"effectDesc":       req.EffectDesc,
		"trait_id ":        0,
		"layer":            0,
		"rarity":           req.Rarity,
		"rarity_color":     "",
		"jumps":            "",
		"ware_scale":       nil,
	}
	if fruitEntry["desc"] == "" {
		fruitEntry["desc"] = fmt.Sprintf("%s的果实，可以出售换取金币。", name)
	}
	if fruitEntry["effectDesc"] == "" {
		fruitEntry["effectDesc"] = name
	}
	if req.Rarity > 0 {
		fruitEntry["rarity_color"] = "D2C5AC"
	}
	items = append(items, fruitEntry)
	plants, err := readJSONArray(plantPath)
	if err != nil {
		return nil, err
	}
	for _, p := range plants {
		if toInt64(p["id"]) != req.PlantID {
			continue
		}
		count := req.FruitCount
		if count <= 0 {
			if fruit, ok := p["fruit"].(map[string]any); ok {
				count = toInt64(fruit["count"])
			}
			if count <= 0 {
				count = 200
			}
		}
		p["fruit"] = map[string]any{"id": fruitID, "count": count}
		break
	}
	if err := writeJSONArray(itemPath, items); err != nil {
		return nil, err
	}
	if err := writeJSONArray(plantPath, plants); err != nil {
		return nil, err
	}
	if err := g.reloadLocked(dir); err != nil {
		return nil, err
	}
	return map[string]any{"fruitId": fruitID, "plantId": req.PlantID, "name": name, "assetName": assetName}, nil
}

// ModifyFruit updates a fruit item.
func (g *GameConfig) ModifyFruit(req FruitWriteReq) (map[string]any, error) {
	if req.ID <= 0 {
		return nil, fmt.Errorf("无效的果实ID")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	dir := g.dir
	itemPath := filepath.Join(dir, "ItemInfo.json")
	items, err := readJSONArray(itemPath)
	if err != nil {
		return nil, err
	}
	found := false
	var name string
	for _, item := range items {
		if toInt64(item["id"]) != req.ID || toInt64(item["type"]) != 6 {
			continue
		}
		found = true
		if n := strings.TrimSpace(req.Name); n != "" {
			item["name"] = n
			item["effectDesc"] = n
			item["desc"] = fmt.Sprintf("%s的果实，可以出售换取金币。", n)
			name = n
		} else {
			name = toString(item["name"])
		}
		if req.Desc != "" {
			item["desc"] = req.Desc
		}
		if req.Price >= 0 && (req.Price > 0 || req.PriceID > 0) {
			priceID := req.PriceID
			if priceID <= 0 {
				priceID = 1001
			}
			item["sells"] = formatSells(priceID, req.Price)
		}
		if req.Rarity > 0 || req.Name != "" {
			item["rarity"] = req.Rarity
		}
		if req.Level > 0 {
			item["level"] = req.Level
		}
		break
	}
	if !found {
		return nil, fmt.Errorf("果实ID %d 不存在", req.ID)
	}
	if err := writeJSONArray(itemPath, items); err != nil {
		return nil, err
	}
	if err := g.reloadLocked(dir); err != nil {
		return nil, err
	}
	return map[string]any{"fruitId": req.ID, "name": name}, nil
}

// DeleteFruit removes a fruit item and clears plant.fruit.
func (g *GameConfig) DeleteFruit(fruitID int64) (map[string]any, error) {
	if fruitID <= 0 {
		return nil, fmt.Errorf("无效的果实ID")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	dir := g.dir
	itemPath := filepath.Join(dir, "ItemInfo.json")
	plantPath := filepath.Join(dir, "Plant.json")
	items, err := readJSONArray(itemPath)
	if err != nil {
		return nil, err
	}
	var name string
	newItems := make([]map[string]any, 0, len(items))
	found := false
	for _, item := range items {
		if toInt64(item["id"]) == fruitID && toInt64(item["type"]) == 6 {
			found = true
			name = toString(item["name"])
			continue
		}
		newItems = append(newItems, item)
	}
	if !found {
		return nil, fmt.Errorf("果实ID %d 不存在", fruitID)
	}
	plants, err := readJSONArray(plantPath)
	if err != nil {
		return nil, err
	}
	for _, p := range plants {
		fruit, _ := p["fruit"].(map[string]any)
		if fruit != nil && toInt64(fruit["id"]) == fruitID {
			p["fruit"] = map[string]any{"id": 0, "count": 0}
		}
	}
	if err := writeJSONArray(itemPath, newItems); err != nil {
		return nil, err
	}
	if err := writeJSONArray(plantPath, plants); err != nil {
		return nil, err
	}
	if err := g.reloadLocked(dir); err != nil {
		return nil, err
	}
	return map[string]any{"fruitId": fruitID, "name": name}, nil
}

// AddItem adds a non-seed/non-fruit item.
func (g *GameConfig) AddItem(req ItemWriteReq) (map[string]any, error) {
	if req.ID <= 0 {
		return nil, fmt.Errorf("物品ID必须为正整数")
	}
	if req.Type <= 0 {
		return nil, fmt.Errorf("请选择物品类型")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("物品名称不能为空")
	}
	if req.Type == 5 || req.Type == 6 {
		return nil, fmt.Errorf("种子(type=5)和果实(type=6)请使用对应的录入功能")
	}
	if g.GetItemByID(req.ID) != nil {
		return nil, fmt.Errorf("物品ID %d 已存在", req.ID)
	}
	if req.PriceID <= 0 {
		req.PriceID = 1001
	}
	if req.MaxCount <= 0 {
		req.MaxCount = 9999
	}
	effectDesc := strings.TrimSpace(req.EffectDesc)
	if effectDesc == "" {
		effectDesc = name
	}

	g.mu.Lock()
	defer g.mu.Unlock()
	dir := g.dir
	itemPath := filepath.Join(dir, "ItemInfo.json")
	items, err := readJSONArray(itemPath)
	if err != nil {
		return nil, err
	}
	entry := map[string]any{
		"id":               req.ID,
		"type":             req.Type,
		"name":             name,
		"interaction_type": req.InteractionType,
		"sells":            formatSells(req.PriceID, req.Price),
		"sell_cond":        nil,
		"cond_sells":       nil,
		"level":            req.Level,
		"target_id":        0,
		"asset_name":       req.AssetName,
		"icon_res":         "",
		"max_count":        req.MaxCount,
		"max_own":          req.MaxCount,
		"duration":         nil,
		"can_use":          req.CanUse,
		"desc":             req.Desc,
		"effectDesc":       effectDesc,
		"trait_id ":        0,
		"layer":            0,
		"rarity":           req.Rarity,
		"rarity_color":     "",
		"jumps":            "",
		"ware_scale":       nil,
	}
	if req.Rarity > 0 {
		entry["rarity_color"] = "D2C5AC"
	}
	items = append(items, entry)
	if err := writeJSONArray(itemPath, items); err != nil {
		return nil, err
	}
	if err := g.reloadLocked(dir); err != nil {
		return nil, err
	}
	return map[string]any{"itemId": req.ID, "type": req.Type, "name": name}, nil
}

// ModifyItem updates a non-seed/non-fruit item.
func (g *GameConfig) ModifyItem(req ItemWriteReq) (map[string]any, error) {
	if req.ID <= 0 {
		return nil, fmt.Errorf("无效的物品ID")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	dir := g.dir
	itemPath := filepath.Join(dir, "ItemInfo.json")
	items, err := readJSONArray(itemPath)
	if err != nil {
		return nil, err
	}
	found := false
	var name string
	for _, item := range items {
		if toInt64(item["id"]) != req.ID {
			continue
		}
		typ := toInt64(item["type"])
		if typ == 5 || typ == 6 {
			return nil, fmt.Errorf("种子和果实请使用对应的修改功能")
		}
		found = true
		if n := strings.TrimSpace(req.Name); n != "" {
			item["name"] = n
			name = n
		} else {
			name = toString(item["name"])
		}
		if req.Price >= 0 && (req.Price > 0 || req.PriceID > 0) {
			priceID := req.PriceID
			if priceID <= 0 {
				priceID = 1001
			}
			item["sells"] = formatSells(priceID, req.Price)
		}
		if req.InteractionType != "" || req.Name != "" {
			item["interaction_type"] = req.InteractionType
		}
		item["can_use"] = req.CanUse
		if req.Desc != "" {
			item["desc"] = req.Desc
		}
		if req.EffectDesc != "" {
			item["effectDesc"] = req.EffectDesc
		}
		item["rarity"] = req.Rarity
		if req.MaxCount > 0 {
			item["max_count"] = req.MaxCount
			item["max_own"] = req.MaxCount
		}
		if req.Level > 0 {
			item["level"] = req.Level
		}
		break
	}
	if !found {
		return nil, fmt.Errorf("物品ID %d 不存在", req.ID)
	}
	if err := writeJSONArray(itemPath, items); err != nil {
		return nil, err
	}
	if err := g.reloadLocked(dir); err != nil {
		return nil, err
	}
	return map[string]any{"itemId": req.ID, "name": name}, nil
}

// DeleteItem removes a non-seed/non-fruit item.
func (g *GameConfig) DeleteItem(itemID int64) (map[string]any, error) {
	if itemID <= 0 {
		return nil, fmt.Errorf("无效的物品ID")
	}
	existing := g.GetItemByID(itemID)
	if existing == nil {
		return nil, fmt.Errorf("物品ID %d 不存在", itemID)
	}
	if existing.Type == 5 || existing.Type == 6 {
		return nil, fmt.Errorf("种子和果实请使用对应的删除功能")
	}
	name := existing.Name

	g.mu.Lock()
	defer g.mu.Unlock()
	dir := g.dir
	itemPath := filepath.Join(dir, "ItemInfo.json")
	items, err := readJSONArray(itemPath)
	if err != nil {
		return nil, err
	}
	newItems := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if toInt64(item["id"]) == itemID {
			continue
		}
		newItems = append(newItems, item)
	}
	if err := writeJSONArray(itemPath, newItems); err != nil {
		return nil, err
	}
	if err := g.reloadLocked(dir); err != nil {
		return nil, err
	}
	return map[string]any{"itemId": itemID, "name": name}, nil
}

// reloadLocked reloads maps; caller must hold g.mu write lock.
func (g *GameConfig) reloadLocked(dir string) error {
	// Temporarily unlock is unsafe; reimplement inline load without lock.
	plantPath := filepath.Join(dir, "Plant.json")
	landPath := filepath.Join(dir, "Land.json")
	itemPath := filepath.Join(dir, "ItemInfo.json")

	plantRaw, err := os.ReadFile(plantPath)
	if err != nil {
		return err
	}
	var plants []PlantItem
	if err := json.Unmarshal(plantRaw, &plants); err != nil {
		return err
	}
	landRaw, err := os.ReadFile(landPath)
	if err != nil {
		return err
	}
	var lands []LandConfigItem
	if err := json.Unmarshal(landRaw, &lands); err != nil {
		return err
	}
	itemRaw, err := os.ReadFile(itemPath)
	if err != nil {
		return err
	}
	items, err := parseItemInfoList(itemRaw)
	if err != nil {
		return err
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
	g.dir = dir
	g.plantByID = plantByID
	g.plantBySeed = plantBySeed
	g.plantByFruit = plantByFruit
	g.itemByID = itemByID
	g.seedItemByID = seedItemByID
	g.landByID = landByID
	g.landByCoordinate = landByCoord
	return nil
}
