// Package activitycenter builds activity center snapshots from live game RPC data.
package activitycenter

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	"github.com/MQEnergy/go-skeleton/internal/farm/game"
	"github.com/MQEnergy/go-skeleton/internal/farm/proto/activitypb"
	"github.com/MQEnergy/go-skeleton/internal/farm/proto/seasonpb"
	"github.com/MQEnergy/go-skeleton/internal/farm/proto/solartermspb"
)

const (
	beijingOffsetSec = 8 * 3600
	secondsPerDay    = 86400
)

type constellationConfirmed struct {
	Opened []string
	Lit    []string
}

var (
	lastConstellationNodes     sync.Map // activityID string -> *activitypb.ConstellationData
	lastConstellationConfirmed sync.Map // activityID string -> *constellationConfirmed

	liveTravelPassMu sync.Mutex
	liveTravelPass   map[string]any
)

// Snapshot is the aggregated activity center view.
type Snapshot struct {
	Season        map[string]any            `json:"season"`
	Constellation map[string]any            `json:"constellation"`
	Shop          map[string]any            `json:"shop"`
	SolarTerms    map[string]any            `json:"solarTerms"`
	Capabilities  map[string]bool           `json:"capabilities"`
	Actions       map[string]map[string]any `json:"actions"`
	Errors        map[string]string         `json:"errors,omitempty"`
}

// BuildSnapshot loads live activity data from the game API.
func BuildSnapshot(ctx context.Context, api *game.API) Snapshot {
	out := Snapshot{
		Season:        map[string]any{},
		Constellation: map[string]any{},
		Shop:          map[string]any{},
		SolarTerms:    map[string]any{},
		Capabilities: map[string]bool{
			"claimPass":          false,
			"lightConstellation": false,
			"claimSolar":         false,
			"exchange":           false,
		},
		Actions: map[string]map[string]any{},
		Errors:  map[string]string{},
	}

	seasonReply, seasonErr := api.GetSeasonInfo(ctx)
	if seasonErr != nil {
		out.Errors["season"] = seasonErr.Error()
	} else if seasonReply != nil && seasonReply.SeasonInfo != nil {
		out.Season = normalizeSeason(seasonReply.SeasonInfo)
		constellationAct := game.FindSeasonActivity(seasonReply.SeasonInfo, game.ConstellationActivityType())
		if constellationAct != nil {
			var nodes *activitypb.ConstellationData
			if v, ok := lastConstellationNodes.Load(strconv.FormatInt(constellationAct.ActivityID, 10)); ok {
				nodes, _ = v.(*activitypb.ConstellationData)
			}
			var confirmed *constellationConfirmed
			if v, ok := lastConstellationConfirmed.Load(strconv.FormatInt(constellationAct.ActivityID, 10)); ok {
				confirmed, _ = v.(*constellationConfirmed)
			}
			out.Constellation = buildConstellation(constellationAct, seasonReply.SeasonInfo.ServerTime, nodes, confirmed)
		}
		shopAct := game.FindSeasonActivity(seasonReply.SeasonInfo, game.ShopActivityType())
		if shopAct != nil {
			shopReply, shopErr := api.QueryActivityShop(ctx, shopAct.ActivityID)
			if shopErr != nil {
				out.Errors["shop"] = shopErr.Error()
			} else {
				out.Shop = normalizeShop(seasonReply.SeasonInfo, shopAct, shopReply, apiBagBalances(ctx, api, shopReply))
			}
		}
	}

	solarReply, solarErr := api.GetSolarTerms(ctx)
	if solarErr != nil {
		out.Errors["solarTerms"] = solarErr.Error()
	} else if solarReply != nil {
		out.SolarTerms = normalizeSolarTerms(solarReply)
	}

	out.Actions = buildActions(out.Season, out.SolarTerms, out.Constellation, out.Shop)

	// capabilities remain actionable bools for Vue clients that have not migrated to actions yet
	if pass, ok := out.Season["pass"].(map[string]any); ok {
		if claimable, _ := pass["claimableCount"].(int); claimable > 0 {
			out.Capabilities["claimPass"] = true
		}
	}
	if enabled, _ := out.Actions["lightConstellation"]["enabled"].(bool); enabled {
		out.Capabilities["lightConstellation"] = true
	}
	if enabled, _ := out.Actions["claimSolar"]["enabled"].(bool); enabled {
		out.Capabilities["claimSolar"] = true
	}
	if enabled, _ := out.Actions["exchange"]["enabled"].(bool); enabled {
		out.Capabilities["exchange"] = true
	}

	return out
}

// ApplySeasonPassNotify merges a pushed/fetched SeasonPass into the live pass cache.
func ApplySeasonPassNotify(pass *seasonpb.SeasonPass) map[string]any {
	next := passDTO(pass)
	if next == nil {
		return nil
	}
	liveTravelPassMu.Lock()
	defer liveTravelPassMu.Unlock()
	if liveTravelPass != nil {
		if title, _ := liveTravelPass["title"].(string); title != "" {
			if cur, _ := next["title"].(string); cur == "" {
				next["title"] = title
			}
		}
		if activityID, _ := liveTravelPass["activityId"].(string); activityID != "" && activityID != "0" {
			if cur, _ := next["activityId"].(string); cur == "" || cur == "0" {
				next["activityId"] = activityID
			}
		}
		// Push payloads often omit nodes; keep prior nodes and recompute claim flags.
		nextNodes, _ := next["nodes"].([]map[string]any)
		prevNodes, _ := liveTravelPass["nodes"].([]map[string]any)
		if len(nextNodes) == 0 && len(prevNodes) > 0 {
			next["nodes"] = clonePassNodes(prevNodes)
			recomputePassNodeFlags(next)
		}
	}
	liveTravelPass = next
	return clonePassMap(liveTravelPass)
}

// GetLiveTravelPass returns a copy of the cached live season pass, if any.
func GetLiveTravelPass() map[string]any {
	liveTravelPassMu.Lock()
	defer liveTravelPassMu.Unlock()
	return clonePassMap(liveTravelPass)
}

// RememberConstellationNodes caches constellation operate reply nodes for snapshot merge.
func RememberConstellationNodes(activityID int64, data *activitypb.ConstellationData) {
	if data == nil {
		return
	}
	key := strconv.FormatInt(activityID, 10)
	lastConstellationNodes.Store(key, data)
	opened, lit := ConfirmedFromDynamicNodes(data.Nodes)
	RememberConstellationConfirmed(activityID, opened, lit)
}

// RememberConstellationConfirmed caches confirmed opened/lit node IDs (monotonic merge).
func RememberConstellationConfirmed(activityID int64, opened, lit []string) {
	key := strconv.FormatInt(activityID, 10)
	mergedOpened := map[string]struct{}{}
	mergedLit := map[string]struct{}{}
	if v, ok := lastConstellationConfirmed.Load(key); ok {
		if prev, _ := v.(*constellationConfirmed); prev != nil {
			for _, id := range prev.Opened {
				mergedOpened[id] = struct{}{}
			}
			for _, id := range prev.Lit {
				mergedLit[id] = struct{}{}
				mergedOpened[id] = struct{}{}
			}
		}
	}
	for _, id := range opened {
		if id != "" {
			mergedOpened[id] = struct{}{}
		}
	}
	for _, id := range lit {
		if id != "" {
			mergedLit[id] = struct{}{}
			mergedOpened[id] = struct{}{}
		}
	}
	next := &constellationConfirmed{
		Opened: sortedKeys(mergedOpened),
		Lit:    sortedKeys(mergedLit),
	}
	lastConstellationConfirmed.Store(key, next)
}

// HydrateConstellationConfirmedFromStateJSON loads persisted opened/lit nodes into memory.
func HydrateConstellationConfirmedFromStateJSON(activityID string, stateJSON string) {
	if activityID == "" || stateJSON == "" {
		return
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(stateJSON), &payload); err != nil {
		return
	}
	opened := stringSlice(payload["confirmedOpenedNodeIds"])
	lit := stringSlice(payload["confirmedLitNodeIds"])
	if len(opened) == 0 && len(lit) == 0 {
		return
	}
	id, err := strconv.ParseInt(activityID, 10, 64)
	if err != nil {
		// activity_id column may be the logical key "constellation"; try nested activityId
		if nested, ok := payload["activityId"].(string); ok && nested != "" {
			id, err = strconv.ParseInt(nested, 10, 64)
		}
		if err != nil {
			return
		}
	}
	RememberConstellationConfirmed(id, opened, lit)
}

// ConfirmedFromDynamicNodes extracts opened/lit node IDs from operate reply nodes.
func ConfirmedFromDynamicNodes(nodes []activitypb.ConstellationNode) (opened, lit []string) {
	openedSet := map[string]struct{}{}
	litSet := map[string]struct{}{}
	for _, node := range nodes {
		id := strconv.FormatInt(node.NodeID, 10)
		if id == "0" {
			continue
		}
		if node.Field2 {
			openedSet[id] = struct{}{}
		}
		if node.Field3 {
			openedSet[id] = struct{}{}
			litSet[id] = struct{}{}
		}
	}
	return sortedKeys(openedSet), sortedKeys(litSet)
}

func buildActions(season, solarTerms, constellation, shop map[string]any) map[string]map[string]any {
	pass, _ := season["pass"].(map[string]any)
	hasPass := pass != nil
	claimablePassCount := 0
	if hasPass {
		if n, ok := pass["claimableCount"].(int); ok {
			claimablePassCount = n
		} else if nodes, ok := pass["nodes"].([]map[string]any); ok {
			for _, node := range nodes {
				if node["claimable"] == true {
					claimablePassCount++
				}
			}
		}
	}

	constellationAct, _ := season["constellationActivity"].(map[string]any)
	hasConstellation := constellationAct != nil
	serverTime := parseInt64Any(season["serverTime"])
	constellationStart := parseInt64Any(constellationAct["startTime"])
	constellationEnd := parseInt64Any(constellationAct["endTime"])
	constellationActive := hasConstellation &&
		(serverTime <= 0 || constellationStart <= 0 || serverTime >= constellationStart) &&
		(serverTime <= 0 || constellationEnd <= 0 || serverTime <= constellationEnd)

	groups, _ := constellation["groups"].([]map[string]any)
	lightableCount := 0
	attemptableCount := 0
	currentDay, _ := constellation["currentDay"].(int)
	currentKnown := true
	currentSeen := false
	for _, g := range groups {
		vs, _ := g["visualState"].(string)
		if vs == "lightable" {
			lightableCount++
			attemptableCount++
		} else if vs == "claimableUnknown" {
			attemptableCount++
		}
		order, _ := g["order"].(int)
		if currentDay > 0 && order == currentDay {
			currentSeen = true
			if known, ok := g["stateKnown"].(bool); ok && !known {
				currentKnown = false
			}
		}
	}
	availabilityKnown := lightableCount > 0 || (currentSeen && currentKnown)
	catalogStatus, _ := constellation["catalogStatus"].(string)

	hasClaimableSolar := false
	if terms, ok := solarTerms["terms"].([]map[string]any); ok {
		for _, t := range terms {
			if t["canClaim"] == true {
				hasClaimableSolar = true
				break
			}
		}
	}

	shopAction, _ := shop["action"].(map[string]any)
	exchangeEnabled := shopAction != nil && shopAction["enabled"] == true
	exchangeAvailable := shopAction != nil && shopAction["available"] == true
	exchangeCount := 0
	if shopAction != nil {
		if n, ok := shopAction["count"].(int); ok {
			exchangeCount = n
		}
	}
	exchange := map[string]any{
		"supported":         true,
		"enabled":           exchangeEnabled,
		"available":         exchangeAvailable,
		"availabilityKnown": shop != nil && len(shop) > 0,
		"count":             exchangeCount,
	}
	if shop == nil || len(shop) == 0 {
		exchange["reason"] = "活动商店目录当前不可用"
	} else if reason, ok := shopAction["reason"].(string); ok && reason != "" {
		exchange["reason"] = reason
	}

	return map[string]map[string]any{
		"claimPass": {
			"supported": true,
			"enabled":   hasPass,
			"available": claimablePassCount > 0,
			"count":     claimablePassCount,
		},
		"lightConstellation": {
			"supported":         true,
			"enabled":           constellationActive && attemptableCount > 0,
			"available":         lightableCount > 0,
			"attemptable":       attemptableCount > 0,
			"availabilityKnown": constellation != nil && catalogStatus == "supported" && availabilityKnown,
			"count":             lightableCount,
			"attemptableCount":  attemptableCount,
		},
		"claimSolar": {
			"supported": true,
			"enabled":   hasClaimableSolar,
		},
		"exchange": exchange,
	}
}

func normalizeSeason(season *seasonpb.SeasonInfo) map[string]any {
	constellationAct := game.FindSeasonActivity(season, game.ConstellationActivityType())
	shopAct := game.FindSeasonActivity(season, game.ShopActivityType())
	pass := ApplySeasonPassNotify(season.Pass)
	if pass == nil {
		pass = GetLiveTravelPass()
	}
	activities := make([]map[string]any, 0, len(season.Activities))
	for i := range season.Activities {
		activities = append(activities, activityDTO(&season.Activities[i]))
	}
	result := map[string]any{
		"id":         strconv.FormatInt(season.SeasonID, 10),
		"title":      bytesText(season.Name),
		"statusCode": strconv.FormatInt(season.Status, 10),
		"startTime":  strconv.FormatInt(season.BeginTime, 10),
		"endTime":    strconv.FormatInt(season.EndTime, 10),
		"serverTime": strconv.FormatInt(season.ServerTime, 10),
		"activities": activities,
		"pass":       pass,
	}
	if constellationAct != nil {
		result["constellationActivity"] = activityDTO(constellationAct)
	} else {
		result["constellationActivity"] = nil
	}
	if shopAct != nil {
		result["shopActivity"] = activityDTO(shopAct)
	} else {
		result["shopActivity"] = nil
	}
	return result
}

func passDTO(pass *seasonpb.SeasonPass) map[string]any {
	if pass == nil {
		return nil
	}
	currentLevel := pass.CurrentLevel
	claimedThrough := pass.ClaimedThroughLevel
	nodes := make([]map[string]any, 0, len(pass.Nodes))
	claimableCount := 0
	for _, node := range pass.Nodes {
		claimed := node.NodeID <= claimedThrough
		locked := node.NodeID > currentLevel
		claimable := !locked && !claimed
		if claimable {
			claimableCount++
		}
		rewards := make([]map[string]any, 0, len(node.Rewards))
		for _, r := range node.Rewards {
			rewards = append(rewards, map[string]any{
				"id":    strconv.FormatInt(r.ItemID, 10),
				"count": strconv.FormatInt(r.Count, 10),
			})
		}
		nodes = append(nodes, map[string]any{
			"id":        strconv.FormatInt(node.NodeID, 10),
			"level":     strconv.FormatInt(node.NodeID, 10),
			"keyLevel":  node.IsKeyLevel,
			"locked":    locked,
			"claimed":   claimed,
			"claimable": claimable,
			"current":   node.NodeID != 0 && node.NodeID == currentLevel,
			"rewards":   rewards,
		})
	}
	return map[string]any{
		"activityId":          strconv.FormatInt(pass.ActivityID, 10),
		"title":               bytesText(pass.Title),
		"level":               strconv.FormatInt(currentLevel, 10),
		"progress":            strconv.FormatInt(pass.CurrentProgress, 10),
		"progressMax":         strconv.FormatInt(pass.ProgressTarget, 10),
		"claimedThroughLevel": strconv.FormatInt(claimedThrough, 10),
		"nodeCount":           strconv.FormatInt(pass.NodeCount, 10),
		"rules":               bytesText(pass.RulesJSON),
		"nodes":               nodes,
		"claimableCount":      claimableCount,
	}
}

func recomputePassNodeFlags(pass map[string]any) {
	level := parseInt64Any(pass["level"])
	claimedThrough := parseInt64Any(pass["claimedThroughLevel"])
	nodes, _ := pass["nodes"].([]map[string]any)
	claimableCount := 0
	for _, node := range nodes {
		nodeID := parseInt64Any(node["id"])
		claimed := nodeID <= claimedThrough
		locked := nodeID > level
		claimable := !locked && !claimed
		node["claimed"] = claimed
		node["locked"] = locked
		node["claimable"] = claimable
		node["current"] = nodeID != 0 && nodeID == level
		if claimable {
			claimableCount++
		}
	}
	pass["claimableCount"] = claimableCount
}

func activityDTO(act *seasonpb.SeasonActivity) map[string]any {
	return map[string]any{
		"id":        strconv.FormatInt(act.ActivityID, 10),
		"typeCode":  strconv.FormatInt(act.Type, 10),
		"name":      bytesText(act.Name),
		"startTime": strconv.FormatInt(act.BeginTime, 10),
		"endTime":   strconv.FormatInt(act.EndTime, 10),
	}
}

func normalizeSolarTerms(reply *solartermspb.GetSolarTermsReply) map[string]any {
	serverTime := reply.ServerTime
	terms := make([]map[string]any, 0, len(reply.Terms))
	var currentTermID string
	for _, term := range reply.Terms {
		dto := solarTermDTO(&term)
		terms = append(terms, dto)
		start, _ := strconv.ParseInt(dto["startTime"].(string), 10, 64)
		end, _ := strconv.ParseInt(dto["endTime"].(string), 10, 64)
		if serverTime > 0 && start <= serverTime && serverTime <= end {
			currentTermID = dto["id"].(string)
		}
	}
	result := map[string]any{
		"serverTime":    strconv.FormatInt(serverTime, 10),
		"currentTermId": nil,
		"terms":         terms,
	}
	if currentTermID != "" {
		result["currentTermId"] = currentTermID
	}
	return result
}

func solarTermDTO(term *solartermspb.SolarTermInfo) map[string]any {
	rewards := make([]map[string]any, 0, len(term.Rewards))
	for _, r := range term.Rewards {
		rewards = append(rewards, map[string]any{
			"id":    strconv.FormatInt(r.ItemID, 10),
			"count": strconv.FormatInt(r.Count, 10),
		})
	}
	status := strconv.FormatInt(term.Status, 10)
	return map[string]any{
		"id":         strconv.FormatInt(term.TermID, 10),
		"name":       bytesText(term.Name),
		"statusCode": status,
		"canClaim":   term.Status == 2,
		"startTime":  strconv.FormatInt(term.BeginTime, 10),
		"endTime":    strconv.FormatInt(term.EndTime, 10),
		"rewards":    rewards,
	}
}

func normalizeShop(season *seasonpb.SeasonInfo, shopAct *seasonpb.SeasonActivity, reply *activitypb.ActivityOperateReply, balances map[int64]int64) map[string]any {
	goods := reply.Data.Catalog.Goods
	goodsDTOs := make([]map[string]any, 0, len(goods))
	affordable := 0
	exchangeable := 0
	categories := map[string]struct{}{}
	for _, g := range goods {
		costID := int64(0)
		costCount := int64(0)
		if g.Cost != nil {
			costID = g.Cost.ItemID
			costCount = g.Cost.Count
		}
		canExchange := costID != 0 && costCount > 0
		if canExchange {
			exchangeable++
		}
		maxCount := "0"
		balanceKnown := balances != nil
		if canExchange && balanceKnown {
			bal := balances[costID]
			if bal >= costCount {
				maxCount = strconv.FormatInt(bal/costCount, 10)
				if bal/costCount > 0 {
					affordable++
				}
			}
		} else if canExchange {
			affordable++
		}
		itemDTO := map[string]any{"id": "0", "count": "0"}
		if g.Item != nil {
			itemDTO = map[string]any{
				"id":    strconv.FormatInt(g.Item.ItemID, 10),
				"count": strconv.FormatInt(g.Item.Count, 10),
			}
		}
		costDTO := map[string]any{"id": "0", "count": "0"}
		if g.Cost != nil {
			costDTO = map[string]any{
				"id":    strconv.FormatInt(g.Cost.ItemID, 10),
				"count": strconv.FormatInt(g.Cost.Count, 10),
			}
		}
		category := bytesText(g.Category)
		if category != "" {
			categories[category] = struct{}{}
		}
		goodsDTOs = append(goodsDTOs, map[string]any{
			"id":                    strconv.FormatInt(g.GoodsID, 10),
			"activityId":            strconv.FormatInt(reply.ActivityID, 10),
			"name":                  bytesText(g.Name),
			"category":              category,
			"item":                  itemDTO,
			"cost":                  costDTO,
			"sortOrder":             strconv.FormatInt(g.SortOrder, 10),
			"statusCode":            strconv.FormatInt(g.Status, 10),
			"owned":                 g.Owned,
			"exchangeable":          canExchange,
			"soldOut":               false,
			"balanceKnown":          balanceKnown,
			"maxExchangeCount":      maxCount,
			"maxExchangeCountKnown": balanceKnown,
		})
	}
	categoryList := make([]string, 0, len(categories))
	for c := range categories {
		categoryList = append(categoryList, c)
	}
	action := map[string]any{
		"supported":         true,
		"enabled":           affordable > 0,
		"available":         affordable > 0,
		"count":             affordable,
		"availabilityKnown": true,
	}
	if exchangeable == 0 {
		action["reason"] = "当前目录没有明确可兑换的商品"
	} else if affordable == 0 {
		action["reason"] = "当前余额不足以兑换目录商品"
	}
	return map[string]any{
		"activityId":   strconv.FormatInt(reply.ActivityID, 10),
		"name":         bytesText(shopAct.Name),
		"startTime":    strconv.FormatInt(shopAct.BeginTime, 10),
		"endTime":      strconv.FormatInt(shopAct.EndTime, 10),
		"serverTime":   strconv.FormatInt(season.ServerTime, 10),
		"balanceKnown": balances != nil,
		"categories":   categoryList,
		"goods":        goodsDTOs,
		"action":       action,
	}
}

func buildConstellation(act *seasonpb.SeasonActivity, serverTime int64, dynamic *activitypb.ConstellationData, confirmed *constellationConfirmed) map[string]any {
	catalog := LoadConstellationCatalog()
	activityID := strconv.FormatInt(act.ActivityID, 10)
	base := map[string]any{
		"activityId":  activityID,
		"typeCode":    strconv.FormatInt(act.Type, 10),
		"displayName": bytesText(act.Name),
		"serverName":  bytesText(act.Name),
		"startTime":   strconv.FormatInt(act.BeginTime, 10),
		"endTime":     strconv.FormatInt(act.EndTime, 10),
		"serverTime":  strconv.FormatInt(serverTime, 10),
	}
	if catalog == nil || catalog.ActivityID != activityID {
		base["catalogStatus"] = "unsupported"
		base["catalogVersion"] = nil
		base["currentDay"] = nil
		base["groups"] = []map[string]any{}
		return base
	}
	currentDay := constellationDay(act.BeginTime, serverTime)
	dynamicNodes := map[int64]*activitypb.ConstellationNode{}
	if dynamic != nil {
		for i := range dynamic.Nodes {
			n := &dynamic.Nodes[i]
			dynamicNodes[n.NodeID] = n
		}
	}
	openedSet := map[string]struct{}{}
	litSet := map[string]struct{}{}
	if confirmed != nil {
		for _, id := range confirmed.Opened {
			openedSet[id] = struct{}{}
		}
		for _, id := range confirmed.Lit {
			litSet[id] = struct{}{}
			openedSet[id] = struct{}{}
		}
	}
	groups := make([]map[string]any, 0, len(catalog.Groups))
	for _, g := range catalog.Groups {
		nodeID, _ := strconv.ParseInt(g.NodeID, 10, 64)
		dn := dynamicNodes[nodeID]
		confirmedOpened := false
		confirmedLit := false
		if _, ok := openedSet[g.NodeID]; ok {
			confirmedOpened = true
		}
		if _, ok := litSet[g.NodeID]; ok {
			confirmedLit = true
		}
		dynamicOpened := dn != nil && dn.Field2
		dynamicLit := dn != nil && dn.Field3
		dynamicLightable := dynamicOpened && dn != nil && !dn.Field3

		var (
			opened       any
			lit          any
			stateKnown   bool
			visual       string
			statusSource string
		)
		if confirmedLit || dynamicLit {
			opened = true
			lit = true
			stateKnown = true
			visual = "lit"
			if confirmedLit {
				statusSource = "persisted"
			} else {
				statusSource = "authoritative"
			}
		} else if dynamicLightable {
			opened = true
			lit = false
			stateKnown = true
			visual = "lightable"
			statusSource = "authoritative"
		} else if currentDay > 0 && g.Order > currentDay {
			opened = false
			lit = false
			stateKnown = false
			visual = "locked"
			statusSource = "schedule"
		} else if currentDay > 0 && g.Order == currentDay {
			if confirmedOpened || dynamicOpened {
				opened = true
			} else {
				opened = nil
			}
			lit = nil
			stateKnown = false
			visual = "claimableUnknown"
			if confirmedOpened {
				statusSource = "persisted"
			} else if dynamicOpened {
				statusSource = "authoritative"
			} else {
				statusSource = "schedule"
			}
		} else {
			if confirmedOpened || dynamicOpened {
				opened = true
			} else {
				opened = nil
			}
			lit = nil
			stateKnown = false
			visual = "unknown"
			if confirmedOpened {
				statusSource = "persisted"
			} else if dynamicOpened {
				statusSource = "authoritative"
			} else {
				statusSource = "schedule"
			}
		}
		groups = append(groups, map[string]any{
			"id":           g.ID,
			"nodeId":       g.NodeID,
			"name":         g.Name,
			"category":     g.Category,
			"order":        g.Order,
			"visualState":  visual,
			"opened":       opened,
			"lit":          lit,
			"stateKnown":   stateKnown,
			"statusSource": statusSource,
			"rewards":      g.Rewards,
		})
	}
	base["displayName"] = catalog.DisplayName
	if catalog.ServerName != "" {
		base["serverName"] = catalog.ServerName
	}
	base["catalogStatus"] = "supported"
	base["catalogVersion"] = catalog.CatalogVersion
	base["currentDay"] = currentDay
	base["groups"] = groups
	base["confirmedOpenedNodeIds"] = sortedKeys(openedSet)
	base["confirmedLitNodeIds"] = sortedKeys(litSet)
	return base
}

func constellationDay(startTime, serverTime int64) int {
	if startTime <= 0 || serverTime < startTime {
		return 0
	}
	startDate := (startTime + beijingOffsetSec) / secondsPerDay
	serverDate := (serverTime + beijingOffsetSec) / secondsPerDay
	day := int(serverDate-startDate) + 1
	if day < 1 {
		return 1
	}
	if day > 28 {
		return 28
	}
	return day
}

func apiBagBalances(ctx context.Context, api *game.API, shopReply *activitypb.ActivityOperateReply) map[int64]int64 {
	if shopReply == nil || shopReply.Data == nil || shopReply.Data.Catalog == nil {
		return nil
	}
	currencyIDs := map[int64]struct{}{}
	for _, g := range shopReply.Data.Catalog.Goods {
		if g.Cost != nil && g.Cost.ItemID > 0 {
			currencyIDs[g.Cost.ItemID] = struct{}{}
		}
	}
	if len(currencyIDs) == 0 {
		return map[int64]int64{}
	}
	bag, err := api.Bag(ctx)
	if err != nil {
		return nil
	}
	items := game.GetBagItems(bag)
	out := map[int64]int64{}
	for _, item := range items {
		if _, ok := currencyIDs[item.ID]; ok {
			out[item.ID] += item.Count
		}
	}
	return out
}

func bytesText(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return string(b)
}

func parseInt64Any(v any) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case float64:
		return int64(t)
	case string:
		n, _ := strconv.ParseInt(t, 10, 64)
		return n
	default:
		return 0
	}
}

func stringSlice(v any) []string {
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func sortedKeys(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	// small sets; insertion order is fine for persistence equality via sets on merge
	sortStrings(out)
	return out
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		j := i
		for j > 0 && values[j-1] > values[j] {
			values[j-1], values[j] = values[j], values[j-1]
			j--
		}
	}
}

func clonePassNodes(nodes []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(nodes))
	for _, node := range nodes {
		cp := make(map[string]any, len(node))
		for k, v := range node {
			cp[k] = v
		}
		out = append(out, cp)
	}
	return out
}

func clonePassMap(pass map[string]any) map[string]any {
	if pass == nil {
		return nil
	}
	out := make(map[string]any, len(pass))
	for k, v := range pass {
		if k == "nodes" {
			if nodes, ok := v.([]map[string]any); ok {
				out[k] = clonePassNodes(nodes)
				continue
			}
		}
		out[k] = v
	}
	return out
}

// StateJSONFromSnapshot serializes key flags for DB persistence.
func StateJSONFromSnapshot(kind string, data map[string]any) string {
	payload := map[string]any{"kind": kind}
	switch kind {
	case "pass":
		if pass, ok := data["pass"].(map[string]any); ok {
			payload["claimableCount"] = pass["claimableCount"]
			payload["level"] = pass["level"]
		}
	case "constellation":
		payload["currentDay"] = data["currentDay"]
		payload["activityId"] = data["activityId"]
		payload["confirmedOpenedNodeIds"] = data["confirmedOpenedNodeIds"]
		payload["confirmedLitNodeIds"] = data["confirmedLitNodeIds"]
	case "shop":
		if action, ok := data["action"].(map[string]any); ok {
			payload["available"] = action["available"]
		}
	case "solar":
		payload["canClaim"] = data["canClaim"]
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Sprintf(`{"kind":%q}`, kind)
	}
	return string(raw)
}
