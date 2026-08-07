package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/MQEnergy/go-skeleton/internal/app/model"
	"github.com/MQEnergy/go-skeleton/internal/farm/game"
	"github.com/MQEnergy/go-skeleton/internal/farm/hub"
	"github.com/MQEnergy/go-skeleton/internal/farm/logic"
	"github.com/MQEnergy/go-skeleton/internal/farm/proto/corepb"
	"github.com/MQEnergy/go-skeleton/internal/farm/proto/friendpb"
	"github.com/MQEnergy/go-skeleton/internal/farm/proto/plantpb"
	"github.com/MQEnergy/go-skeleton/internal/farm/stats"
	"github.com/MQEnergy/go-skeleton/internal/vars"
	"gorm.io/gorm/clause"
)

const (
	friendOpPutWeed   int64 = 10003
	friendOpPutBug    int64 = 10004
	friendOpWeed      int64 = 10005
	friendOpBug       int64 = 10006
	friendOpWater     int64 = 10007
	friendOpSteal     int64 = 10008
	enterReasonFriend int32 = 2
)

// SyncFriendsToDB persists the game friend list used by the friend HTTP views.
// myGID, when > 0, is excluded and any stale self row is deleted (game APIs may return self).
func SyncFriendsToDB(accountID uint64, myGID int64, friends []friendpb.GameFriend) {
	if accountID == 0 {
		return
	}
	db := vars.DB
	if db == nil {
		return
	}
	if myGID > 0 {
		if err := db.Where("account_id = ? AND gid = ?", accountID, myGID).Delete(&model.FarmFriendGid{}).Error; err != nil {
			slog.Warn("farm friend self gid purge failed", "account", accountID, "gid", myGID, "err", err)
		}
	}
	if len(friends) == 0 {
		return
	}
	now := uint(time.Now().Unix())
	for _, friend := range friends {
		if friend.Gid <= 0 || (myGID > 0 && friend.Gid == myGID) {
			continue
		}
		nickname := friend.Remark
		if nickname == "" {
			nickname = friend.Name
		}
		row := model.FarmFriendGid{
			AccountID: accountID, Gid: friend.Gid,
			Nickname: nickname, SyncedAt: now, CreatedAt: now, UpdatedAt: now,
		}
		if err := db.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "account_id"}, {Name: "gid"}},
			DoUpdates: clause.Assignments(map[string]any{
				"nickname": nickname, "synced_at": now, "updated_at": now,
			}),
		}).Create(&row).Error; err != nil {
			slog.Warn("farm friend gid sync failed", "account", accountID, "gid", friend.Gid, "err", err)
		}
	}
}

// AcceptPendingFriends fetches pending applications and accepts them (bot auto-accept mirror).
func AcceptPendingFriends(ctx context.Context, api *game.API) (accepted int, err error) {
	if api == nil {
		return 0, fmt.Errorf("farm API is unavailable")
	}
	reply, err := api.GetApplications(ctx)
	if err != nil {
		return 0, err
	}
	gids := make([]int64, 0, len(reply.Applications))
	for _, app := range reply.Applications {
		if app != nil && app.Gid > 0 {
			gids = append(gids, app.Gid)
		}
	}
	if len(gids) == 0 {
		return 0, nil
	}
	acceptedReply, err := api.AcceptFriends(ctx, gids)
	if err != nil {
		return 0, err
	}
	if acceptedReply != nil {
		return len(acceptedReply.Friends), nil
	}
	return len(gids), nil
}

// RunStealTick visits bubble friends first, then probes ceil(n/4) unvisited zero-bubble friends.
func RunStealTick(ctx context.Context, s *Session, visited map[int64]struct{}) (actions int, err error) {
	if s == nil {
		return 0, fmt.Errorf("farm session is unavailable")
	}
	api := s.GameAPI()
	cfg := s.Config()
	myGID := s.GID()
	accountID := parseAccountID(s.id)
	if api == nil {
		return 0, fmt.Errorf("farm API is unavailable")
	}
	friends, err := loadFriends(ctx, s, api, cfg)
	if err != nil {
		return 0, err
	}
	SyncFriendsToDB(accountID, myGID, friends)
	blacklist := makeIDSet(cfg.FriendBlacklist)
	if visited == nil {
		visited = make(map[int64]struct{})
	}
	targets := buildStealPatrolTargets(friends, myGID, blacklist, visited)
	helpState := s.ensureHelpState()
	for _, gid := range targets {
		if shouldAbortFriendPatrol(ctx, s) {
			break
		}
		if !helpState.canOperate(friendOpSteal) {
			break
		}
		outcome, visitErr := stealFriend(ctx, s, api, cfg, myGID, gid)
		markPatrolVisited(visited, gid)
		if visitErr != nil {
			if handleFriendEnterError(s, gid, visitErr) {
				continue
			}
			if isTransientNetworkError(visitErr) {
				abortFriendPatrol(s, accountID, "steal", visitErr)
				break
			}
			writeInteractLog(accountID, 0, gid, "steal", "error", map[string]any{"error": friendlyNetworkError(visitErr)})
			continue
		}
		if outcome.Count > 0 {
			actions += outcome.Count
			writeInteractLog(accountID, 0, gid, "steal", "ok", map[string]any{
				"count":   outcome.Count,
				"plants":  outcome.Plants,
				"summary": outcome.Summary,
				"score":   outcome.Score,
				"value":   outcome.Value,
			})
			if outcome.Score > 0 {
				writeInteractLog(accountID, 0, gid, "steal_score", "ok", map[string]any{
					"count":   int(outcome.Score),
					"summary": fmt.Sprintf("获得积分x%d", outcome.Score),
				})
			}
		}
	}
	if actions > 0 {
		stats.RecordOp(accountID, 0, "steal", actions)
		if cfg.Automation.Sell {
			sold, gold, names, sellErr := sellAllFruitsDetailed(ctx, api)
			if sellErr != nil {
				slog.Warn("steal sell failed", "account", accountID, "err", sellErr)
			} else if sold > 0 {
				stats.RecordOp(accountID, 0, "sell", 1)
				if gold > 0 {
					stats.RecordExpGold(accountID, 0, 0, gold)
				}
				logSellFruits(s, accountID, names, gold, sold)
			}
		}
	}
	return actions, nil
}

// RunHelpTick visits help-bubble friends first, then probes ceil(n/4) unvisited zero-bubble friends.
func RunHelpTick(ctx context.Context, s *Session, visited map[int64]struct{}) (actions int, err error) {
	if s == nil {
		return 0, fmt.Errorf("farm session is unavailable")
	}
	api := s.GameAPI()
	cfg := s.Config()
	myGID := s.GID()
	accountID := parseAccountID(s.id)
	if api == nil {
		return 0, fmt.Errorf("farm API is unavailable")
	}
	helpState := s.ensureHelpState()
	if !cfg.Automation.FriendHelpExpLimit {
		helpState.setCanGetHelpExp(true)
	} else if !helpState.getCanGetHelpExp() {
		return 0, nil
	}
	friends, err := loadFriends(ctx, s, api, cfg)
	if err != nil {
		return 0, err
	}
	SyncFriendsToDB(accountID, myGID, friends)
	blacklist := makeIDSet(cfg.FriendBlacklist)
	if visited == nil {
		visited = make(map[int64]struct{})
	}
	targets := buildHelpPatrolTargets(friends, myGID, blacklist, visited)
	for _, gid := range targets {
		if shouldAbortFriendPatrol(ctx, s) {
			break
		}
		if cfg.Automation.FriendHelpExpLimit && !helpState.getCanGetHelpExp() {
			break
		}
		outcome, limitReached, visitErr := helpFriend(ctx, s, api, cfg, gid)
		markPatrolVisited(visited, gid)
		if visitErr != nil {
			if handleFriendEnterError(s, gid, visitErr) {
				continue
			}
			if isTransientNetworkError(visitErr) {
				abortFriendPatrol(s, accountID, "help", visitErr)
				break
			}
			writeInteractLog(accountID, 0, gid, "help", "error", map[string]any{"error": friendlyNetworkError(visitErr)})
		} else if outcome.Count > 0 {
			actions += outcome.Count
			writeInteractLog(accountID, 0, gid, "help", "ok", map[string]any{
				"count":   outcome.Count,
				"summary": outcome.Summary,
				"weed":    outcome.Weed,
				"bug":     outcome.Bug,
				"water":   outcome.Water,
			})
		}
		if cfg.Automation.FriendHelpExpLimit && (limitReached || !helpState.getCanGetHelpExp()) {
			break
		}
	}
	if actions > 0 {
		stats.RecordOp(accountID, 0, "help", actions)
	}
	return actions, nil
}

// RunBadOnce visits up to 20 zero-bubble friends (highest level first) and puts weeds/bugs.
func RunBadOnce(ctx context.Context, s *Session) (actions int, err error) {
	if s == nil {
		return 0, fmt.Errorf("farm session is unavailable")
	}
	api := s.GameAPI()
	cfg := s.Config()
	myGID := s.GID()
	accountID := parseAccountID(s.id)
	if api == nil {
		return 0, fmt.Errorf("farm API is unavailable")
	}
	friends, err := loadFriends(ctx, s, api, cfg)
	if err != nil {
		return 0, err
	}
	SyncFriendsToDB(accountID, myGID, friends)
	blacklist := makeIDSet(cfg.FriendBlacklist)
	targets := buildBadFriendTargets(friends, myGID, blacklist, badFriendTopN)
	helpState := s.ensureHelpState()
	for _, gid := range targets {
		if shouldAbortFriendPatrol(ctx, s) {
			break
		}
		if helpState.isBadOperationLimitReached() {
			break
		}
		if !helpState.canOperate(friendOpPutBug) && !helpState.canOperate(friendOpPutWeed) {
			break
		}
		outcome, visitErr := badFriend(ctx, s, api, myGID, gid)
		if visitErr != nil {
			if handleFriendEnterError(s, gid, visitErr) {
				continue
			}
			if isTransientNetworkError(visitErr) {
				abortFriendPatrol(s, accountID, "bad", visitErr)
				break
			}
			writeInteractLog(accountID, 0, gid, "bad", "error", map[string]any{"error": friendlyNetworkError(visitErr)})
			continue
		}
		if outcome.Count > 0 {
			actions += outcome.Count
			writeInteractLog(accountID, 0, gid, "bad", "ok", map[string]any{
				"count":   outcome.Count,
				"summary": outcome.Summary,
				"putBug":  outcome.PutBug,
				"putWeed": outcome.PutWeed,
			})
		}
	}
	return actions, nil
}

const qqFriendListBatchSize = 35

func loadFriends(ctx context.Context, s *Session, api *game.API, cfg logic.AccountConfig) ([]friendpb.GameFriend, error) {
	platform := ""
	myGID := int64(0)
	if s != nil {
		s.mu.Lock()
		platform = s.cfg.Platform
		myGID = s.gid
		s.mu.Unlock()
	}
	if platform == "" {
		platform = "qq"
	}

	var friends []friendpb.GameFriend
	if platform != "qq" {
		reply, getErr := api.GetAllFriends(ctx)
		if getErr == nil {
			friends = dedupeFriendsByGID(reply.GameFriends)
		} else if len(cfg.KnownFriendGids) == 0 {
			return nil, fmt.Errorf("get all friends: %w", getErr)
		} else {
			fallback, fallbackErr := api.GetGameFriends(ctx, excludeGID(cfg.KnownFriendGids, myGID))
			if fallbackErr != nil {
				return nil, fmt.Errorf("get all friends: %w; get known friends: %v", getErr, fallbackErr)
			}
			friends = dedupeFriendsByGID(fallback.GameFriends)
		}
		return excludeSelfFriends(friends, myGID), nil
	}

	// QQ path (gid-manager.ts): InteractRecords → merge visitor GIDs → GetGameFriends;
	// fallback SyncAll then GetAll.
	known := mergeVisitorGIDsFromInteract(ctx, s, api, cfg, myGID)
	if friends = fetchQQFriendsByKnownGIDs(ctx, api, known); len(friends) > 0 {
		mergeKnownFriendGIDs(s, friendGIDs(friends))
		return excludeSelfFriends(friends, myGID), nil
	}

	legacy, legacyErr := fetchQQFriendsLegacy(ctx, api)
	if legacyErr != nil {
		if len(known) == 0 {
			return nil, fmt.Errorf("qq friend list failed (maintain knownFriendGids): %w", legacyErr)
		}
		return nil, legacyErr
	}
	if len(legacy) > 0 {
		mergeKnownFriendGIDs(s, friendGIDs(legacy))
	}
	return excludeSelfFriends(legacy, myGID), nil
}

func mergeVisitorGIDsFromInteract(ctx context.Context, s *Session, api *game.API, cfg logic.AccountConfig, myGID int64) []int64 {
	known := excludeGID(normalizeFriendGIDs(cfg.KnownFriendGids), myGID)
	blacklist := makeIDSet(cfg.FriendBlacklist)
	reply, err := api.InteractRecords(ctx)
	if err != nil {
		slog.Debug("interact records for qq friends failed", "err", err)
		return filterBlacklistedGIDs(known, blacklist)
	}
	visitorGIDs := make([]int64, 0, len(reply.Records))
	for _, rec := range reply.Records {
		if rec == nil || rec.VisitorGid <= 0 || (myGID > 0 && rec.VisitorGid == myGID) {
			continue
		}
		if _, blocked := blacklist[rec.VisitorGid]; blocked {
			continue
		}
		visitorGIDs = append(visitorGIDs, rec.VisitorGid)
	}
	merged := normalizeFriendGIDs(append(append([]int64(nil), known...), visitorGIDs...))
	if added := len(merged) - len(known); added > 0 {
		mergeKnownFriendGIDs(s, merged)
		slog.Info("merged visitor gids into known friends", "account", sessionID(s), "added", added, "total", len(merged))
	}
	return filterBlacklistedGIDs(merged, blacklist)
}

func fetchQQFriendsByKnownGIDs(ctx context.Context, api *game.API, known []int64) []friendpb.GameFriend {
	if len(known) == 0 {
		return nil
	}
	all := make([]*friendpb.GameFriend, 0, len(known))
	for i := 0; i < len(known); i += qqFriendListBatchSize {
		end := i + qqFriendListBatchSize
		if end > len(known) {
			end = len(known)
		}
		batch := known[i:end]
		reply, err := api.GetGameFriends(ctx, batch)
		if err != nil {
			slog.Debug("GetGameFriends batch failed", "from", i+1, "to", end, "err", err)
		} else if reply != nil {
			all = append(all, reply.GameFriends...)
		}
		if end < len(known) {
			select {
			case <-ctx.Done():
				return dedupeFriendsByGID(all)
			case <-time.After(500 * time.Millisecond):
			}
		}
	}
	return dedupeFriendsByGID(all)
}

func fetchQQFriendsLegacy(ctx context.Context, api *game.API) ([]friendpb.GameFriend, error) {
	var errs []string
	if syncReply, err := api.SyncAll(ctx, nil); err == nil {
		return dedupeFriendsByGID(syncReply.GameFriends), nil
	} else {
		errs = append(errs, fmt.Sprintf("SyncAll: %v", err))
	}
	if allReply, err := api.GetAllFriends(ctx); err == nil {
		return dedupeFriendsByGID(allReply.GameFriends), nil
	} else {
		errs = append(errs, fmt.Sprintf("GetAll: %v", err))
	}
	return nil, fmt.Errorf("%s", strings.Join(errs, " | "))
}

func mergeKnownFriendGIDs(s *Session, gids []int64) {
	if s == nil || len(gids) == 0 {
		return
	}
	normalized := normalizeFriendGIDs(gids)
	s.mu.Lock()
	myGID := s.gid
	cfg := s.cfg.AccountConfig
	merged := excludeGID(normalizeFriendGIDs(append(append([]int64(nil), cfg.KnownFriendGids...), normalized...)), myGID)
	same := len(merged) == len(cfg.KnownFriendGids)
	if same {
		for i := range merged {
			if merged[i] != cfg.KnownFriendGids[i] {
				same = false
				break
			}
		}
	}
	if same {
		s.mu.Unlock()
		return
	}
	cfg.KnownFriendGids = merged
	s.cfg.AccountConfig = cfg
	s.mu.Unlock()
	persistKnownFriendGIDs(parseAccountID(s.id), cfg)
}

func normalizeFriendGIDs(values []int64) []int64 {
	out := make([]int64, 0, len(values))
	seen := make(map[int64]struct{}, len(values))
	for _, gid := range values {
		if gid <= 0 {
			continue
		}
		if _, ok := seen[gid]; ok {
			continue
		}
		seen[gid] = struct{}{}
		out = append(out, gid)
	}
	return out
}

func excludeGID(gids []int64, skip int64) []int64 {
	if skip <= 0 || len(gids) == 0 {
		return gids
	}
	out := make([]int64, 0, len(gids))
	for _, gid := range gids {
		if gid == skip {
			continue
		}
		out = append(out, gid)
	}
	return out
}

func excludeSelfFriends(friends []friendpb.GameFriend, myGID int64) []friendpb.GameFriend {
	if myGID <= 0 || len(friends) == 0 {
		return friends
	}
	out := make([]friendpb.GameFriend, 0, len(friends))
	for _, f := range friends {
		if f.Gid == myGID {
			continue
		}
		out = append(out, f)
	}
	return out
}

func filterBlacklistedGIDs(gids []int64, blacklist map[int64]struct{}) []int64 {
	if len(blacklist) == 0 {
		return gids
	}
	out := make([]int64, 0, len(gids))
	for _, gid := range gids {
		if _, blocked := blacklist[gid]; blocked {
			continue
		}
		out = append(out, gid)
	}
	return out
}

func friendGIDs(friends []friendpb.GameFriend) []int64 {
	out := make([]int64, 0, len(friends))
	for _, f := range friends {
		if f.Gid > 0 {
			out = append(out, f.Gid)
		}
	}
	return out
}

func dedupeFriendsByGID(friends []*friendpb.GameFriend) []friendpb.GameFriend {
	out := make([]friendpb.GameFriend, 0, len(friends))
	seen := make(map[int64]struct{}, len(friends))
	for _, f := range friends {
		if f == nil || f.Gid <= 0 {
			continue
		}
		if _, ok := seen[f.Gid]; ok {
			continue
		}
		seen[f.Gid] = struct{}{}
		out = append(out, *f)
	}
	return out
}

func sessionID(s *Session) string {
	if s == nil {
		return ""
	}
	return s.id
}

// friendVisitOutcome carries bot-style log details for one friend visit.
type friendVisitOutcome struct {
	Count   int
	Plants  []string
	Summary string
	Score   int64
	Value   int64
	Weed    int
	Bug     int
	Water   int
	PutBug  int
	PutWeed int
}

func stealFriend(ctx context.Context, s *Session, api *game.API, cfg logic.AccountConfig, myGID, gid int64) (friendVisitOutcome, error) {
	var out friendVisitOutcome
	lands, err := api.VisitEnter(ctx, gid, enterReasonFriend)
	if err != nil {
		if handleFriendEnterError(s, gid, err) {
			return out, nil
		}
		return out, err
	}
	defer func() {
		if leaveErr := api.VisitLeave(ctx, gid); err == nil && leaveErr != nil {
			err = leaveErr
		}
	}()
	blacklist := makeIDSet(cfg.PlantBlacklist)
	activityOnly := cfg.Automation.FriendStealActivityOnly
	landsMap := logic.BuildLandMap(lands)
	targets := make([]int64, 0, len(lands))
	plantByLand := make(map[int64]string, len(lands))
	for i := range lands {
		land := &lands[i]
		if logic.IsOccupiedSlaveLand(land, landsMap) {
			continue
		}
		if land.Plant == nil || !land.Plant.Stealable || !isMature(*land) {
			continue
		}
		if isPlantBlacklistedBySeed(blacklist, land.Plant.ID) {
			continue
		}
		if activityOnly && !logic.IsActivityPlant(land.Plant) {
			continue
		}
		targets = append(targets, land.ID)
		displayID, name := logic.ResolvePlantDisplayName(land.Plant)
		if name == "" || name == "未知" {
			name = strings.TrimSpace(land.Plant.Name)
		}
		if name != "" {
			plantByLand[land.ID] = name
		}
		if displayID > 0 && displayID != land.Plant.ID {
			slog.Info("steal mutant plant resolved",
				"account", sessionID(s),
				"friend_gid", gid,
				"land", land.ID,
				"plant_id", land.Plant.ID,
				"fruit_id", land.Plant.FruitID,
				"mutant_ids", land.Plant.MutantConfigIDs,
				"display_id", displayID,
				"display_name", name,
			)
		}
	}
	if len(targets) == 0 {
		return out, nil
	}
	// Bot checkCanOperate: on error fail-open (canOperate=true).
	canOperate, canSteal, checkErr := api.CheckCanOperate(ctx, gid, friendOpSteal)
	if checkErr != nil {
		canOperate, canSteal = true, 0
	}
	if !canOperate {
		return out, nil
	}
	if canSteal > 0 && int64(len(targets)) > canSteal {
		targets = targets[:canSteal]
	}
	stolenIDs, items, harvestErr := friendHarvestWithFallback(ctx, s, api, gid, targets)
	if harvestErr != nil && len(stolenIDs) == 0 {
		return out, harvestErr
	}
	out.Count = len(stolenIDs)
	if out.Count > 0 {
		out.Plants = mergeUniqueNames(
			uniquePlantNames(plantByLand, stolenIDs),
			plantNamesFromHarvestItems(items),
		)
		out.Score, out.Value = summarizeHarvestRewards(items)
		out.Summary = formatStealSummary(out.Count, out.Plants, out.Score, out.Value)
	}
	return out, nil
}

func mergeUniqueNames(parts ...[]string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, list := range parts {
		for _, name := range list {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			// Prefer mutant label over base when both appear (黄金·艾草 vs 艾草).
			if base, ok := baseNameIfMutant(name); ok {
				if _, hasBase := seen[base]; hasBase {
					// replace base with mutant in out
					for i, existing := range out {
						if existing == base {
							out[i] = name
							delete(seen, base)
							seen[name] = struct{}{}
							break
						}
					}
					continue
				}
			}
			seen[name] = struct{}{}
			out = append(out, name)
		}
	}
	return out
}

func baseNameIfMutant(name string) (string, bool) {
	n := strings.TrimSpace(name)
	for _, prefix := range []string{"黄金·", "黄金"} {
		if strings.HasPrefix(n, prefix) && len(n) > len(prefix) {
			return n[len(prefix):], true
		}
	}
	return "", false
}

func plantNamesFromHarvestItems(items []*corepb.Item) []string {
	out := make([]string, 0)
	seen := map[string]struct{}{}
	for _, it := range items {
		if it == nil || it.Id <= 0 || it.Count <= 0 {
			continue
		}
		if isActivityScoreItemID(it.Id) {
			continue
		}
		name := ""
		if p := logic.GetPlantByFruitID(it.Id); p != nil {
			name = strings.TrimSpace(p.Name)
		}
		if name == "" {
			if info := logic.GetItemByID(it.Id); info != nil {
				name = strings.TrimSpace(info.Name)
			}
		}
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}

func uniquePlantNames(plantByLand map[int64]string, landIDs []int64) []string {
	seen := make(map[string]struct{}, len(landIDs))
	out := make([]string, 0, len(landIDs))
	for _, id := range landIDs {
		name := plantByLand[id]
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}

func formatStealSummary(count int, plants []string, score, value int64) string {
	if count <= 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "偷%d", count)
	if len(plants) > 0 {
		fmt.Fprintf(&b, "(%s)", strings.Join(plants, "/"))
	}
	if score > 0 {
		fmt.Fprintf(&b, "+积分x%d", score)
	}
	if value > 0 {
		fmt.Fprintf(&b, "，价值%d金", value)
	}
	return b.String()
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type harvestRewardAccum struct {
	stolenIDs []int64
	items     []*corepb.Item
}

// friendHarvestWithFallback mirrors bot stealLandsWithRewardLog: batch then per-land retry; 1001040 = unstealable.
func friendHarvestWithFallback(ctx context.Context, s *Session, api *game.API, gid int64, targets []int64) (stolenIDs []int64, items []*corepb.Item, err error) {
	if len(targets) == 0 {
		return nil, nil, nil
	}
	acc := &harvestRewardAccum{}
	reply, batchErr := api.FriendHarvest(ctx, gid, targets)
	if batchErr == nil {
		if s != nil && reply != nil {
			s.ensureHelpState().updateLimits(reply.OperationLimits)
		}
		applyHarvestReply(acc, targets, reply)
		return acc.stolenIDs, acc.items, nil
	}
	if isTransientNetworkError(batchErr) {
		return nil, nil, batchErr
	}
	for _, landID := range targets {
		if shouldAbortFriendPatrol(ctx, s) {
			break
		}
		one, oneErr := api.FriendHarvest(ctx, gid, []int64{landID})
		if oneErr != nil {
			if isUnstealableError(oneErr) {
				continue
			}
			if isTransientNetworkError(oneErr) {
				if len(acc.stolenIDs) == 0 {
					return nil, nil, oneErr
				}
				break
			}
			continue
		}
		if s != nil && one != nil {
			s.ensureHelpState().updateLimits(one.OperationLimits)
		}
		applyHarvestReply(acc, []int64{landID}, one)
		_ = waitFarmDelay(ctx, 100*time.Millisecond)
	}
	return acc.stolenIDs, acc.items, nil
}

func applyHarvestReply(acc *harvestRewardAccum, requested []int64, reply *plantpb.HarvestReply) {
	if acc == nil {
		return
	}
	succeeded := requested
	if reply != nil && len(reply.Land) > 0 {
		reqSet := make(map[int64]struct{}, len(requested))
		for _, id := range requested {
			reqSet[id] = struct{}{}
		}
		succeeded = succeeded[:0]
		for _, land := range reply.Land {
			if land == nil || land.Id <= 0 {
				continue
			}
			if _, ok := reqSet[land.Id]; ok {
				succeeded = append(succeeded, land.Id)
			}
		}
		if len(succeeded) == 0 {
			succeeded = requested
		}
	}
	seen := make(map[int64]struct{}, len(acc.stolenIDs)+len(succeeded))
	for _, id := range acc.stolenIDs {
		seen[id] = struct{}{}
	}
	for _, id := range succeeded {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		acc.stolenIDs = append(acc.stolenIDs, id)
	}
	if reply != nil {
		for _, it := range reply.Items {
			if it == nil || it.Id <= 0 || it.Count <= 0 {
				continue
			}
			cp := *it
			acc.items = append(acc.items, &cp)
		}
	}
}

func isUnstealableError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "code=1001040") || strings.Contains(msg, "1001040")
}

func isActivityScoreItemID(itemID int64) bool {
	if itemID == 1019 || itemID == 1022 {
		return true
	}
	info := logic.GetItemByID(itemID)
	if info == nil {
		return false
	}
	return strings.Contains(info.Name, "积分")
}

// summarizeHarvestRewards returns activity score + estimated fruit sell value from HarvestReply.items.
func summarizeHarvestRewards(items []*corepb.Item) (score, fruitValue int64) {
	for _, it := range items {
		if it == nil || it.Count <= 0 {
			continue
		}
		if isActivityScoreItemID(it.Id) {
			score += it.Count
			continue
		}
		if logic.GetPlantByFruitID(it.Id) == nil {
			continue
		}
		price := logic.GlobalGameConfig.GetFruitPrice(it.Id)
		if price > 0 {
			fruitValue += price * it.Count
		}
	}
	return score, fruitValue
}

func logSellFruits(s *Session, accountID uint64, names []string, gold int64, soldKinds int) {
	if accountID == 0 {
		return
	}
	namePart := strings.Join(names, ", ")
	if namePart == "" {
		namePart = fmt.Sprintf("%d种果实", soldKinds)
	}
	msg := fmt.Sprintf("出售 %s", namePart)
	if gold > 0 {
		msg = fmt.Sprintf("%s，获得 %d 金币", msg, gold)
	}
	payload := map[string]any{
		"tag":     "仓库",
		"event":   "出售果实",
		"message": msg,
		"isWarn":  false,
		"actions": []string{msg},
		"gold":    gold,
		"count":   soldKinds,
	}
	if s != nil && s.hub != nil {
		s.hub.PublishJSON("farm_operation", accountID, payload)
		return
	}
	hub.Default.PublishJSON("farm_operation", accountID, payload)
}

func helpFriend(ctx context.Context, s *Session, api *game.API, cfg logic.AccountConfig, gid int64) (friendVisitOutcome, bool, error) {
	var out friendVisitOutcome
	helpState := s.ensureHelpState()
	stopWhenExpLimit := cfg.Automation.FriendHelpExpLimit
	if !stopWhenExpLimit {
		helpState.setCanGetHelpExp(true)
	} else if !helpState.getCanGetHelpExp() {
		return out, true, nil
	}

	lands, err := api.VisitEnter(ctx, gid, enterReasonFriend)
	if err != nil {
		if handleFriendEnterError(s, gid, err) {
			return out, false, nil
		}
		return out, false, err
	}
	defer func() {
		if leaveErr := api.VisitLeave(ctx, gid); err == nil && leaveErr != nil {
			err = leaveErr
		}
	}()
	analysisWater, analysisWeed, analysisBug := logic.AnalyzeFriendHelpLands(lands)
	out.Weed = len(analysisWeed)
	out.Bug = len(analysisBug)
	out.Water = len(analysisWater)
	allHelp := uniqueLandIDs(analysisWeed, analysisBug, analysisWater)
	if len(allHelp) == 0 {
		return out, false, nil
	}

	allExpIDs := []int64{friendOpWeed, friendOpBug, friendOpWater}
	allowByExp := !stopWhenExpLimit || (helpState.canGetExpByCandidates(allExpIDs) && helpState.getCanGetHelpExp())
	if !allowByExp {
		return out, true, nil
	}

	beforeExp := s.playerExpSnapshot()
	reply, farmErr := api.FriendFarming(ctx, gid, allHelp)
	if farmErr != nil {
		if isFarmingNoopError(farmErr) {
			return out, false, nil
		}
		return out, false, farmErr
	}
	count := len(allHelp)
	if reply != nil {
		helpState.updateLimits(reply.OperationLimits)
		if landIDs := farmingResultLandIDs(reply.Results); len(landIDs) > 0 {
			count = len(landIDs)
		} else if len(reply.Results) > 0 {
			count = len(reply.Results)
		}
		if stopWhenExpLimit && count > 0 {
			_ = waitFarmDelay(ctx, 200*time.Millisecond)
			afterExp := s.playerExpSnapshot()
			if afterExp <= beforeExp {
				helpState.autoDisableHelpByExpLimit()
				slog.Info("friend help exp limit reached", "account", s.id, "friend_gid", gid)
				out.Count = count
				out.Summary = formatHelpSummary(count, out.Weed, out.Bug, out.Water)
				return out, true, nil
			}
		}
	}
	out.Count = count
	if count > 0 {
		out.Summary = formatHelpSummary(count, out.Weed, out.Bug, out.Water)
	}
	if stopWhenExpLimit && !helpState.getCanGetHelpExp() {
		return out, true, nil
	}
	return out, false, nil
}

func formatHelpSummary(landCount, weed, bug, water int) string {
	if landCount <= 0 {
		return ""
	}
	parts := make([]string, 0, 3)
	if weed > 0 {
		parts = append(parts, fmt.Sprintf("草%d", weed))
	}
	if bug > 0 {
		parts = append(parts, fmt.Sprintf("虫%d", bug))
	}
	if water > 0 {
		parts = append(parts, fmt.Sprintf("水%d", water))
	}
	ops := weed + bug + water
	if ops <= 0 {
		ops = landCount
	}
	if len(parts) == 0 {
		return fmt.Sprintf("一键务农%d块", landCount)
	}
	return fmt.Sprintf("一键务农%d块/%d项(%s)", landCount, ops, strings.Join(parts, "/"))
}

func farmingResultLandIDs(results []*plantpb.FarmingResult) []int64 {
	seen := make(map[int64]struct{}, len(results))
	out := make([]int64, 0, len(results))
	for _, r := range results {
		if r == nil || r.LandId <= 0 {
			continue
		}
		if _, ok := seen[r.LandId]; ok {
			continue
		}
		seen[r.LandId] = struct{}{}
		out = append(out, r.LandId)
	}
	return out
}

func isFarmingNoopError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "code=1001057") || strings.Contains(msg, "1001057")
}

func isEnterFarmBannedError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "1002003")
}

// isTransientNetworkError mirrors bot visit-strategy isTransientNetworkError, plus Go protocol errors.
func isTransientNetworkError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	msg := err.Error()
	if msg == "" {
		return false
	}
	keywords := []string{
		"protocol: client closed",
		"protocol: connection closed",
		"protocol: not connected",
		"protocol: read:",
		"protocol: write:",
		"连接未打开",
		"请求超时",
		"请求已中断",
		"连接关闭",
		"连接已在加密途中关闭",
		"worker exited",
		"context canceled",
		"context deadline exceeded",
	}
	for _, kw := range keywords {
		if strings.Contains(msg, kw) {
			return true
		}
	}
	return false
}

func friendlyNetworkError(err error) string {
	if err == nil {
		return ""
	}
	if isTransientNetworkError(err) {
		return "连接关闭"
	}
	return err.Error()
}

func parseRPCErrorCode(err error) int {
	if err == nil {
		return 0
	}
	msg := err.Error()
	idx := strings.Index(strings.ToLower(msg), "code=")
	if idx < 0 {
		return 0
	}
	n := 0
	for _, c := range msg[idx+5:] {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func isInvalidFriendAccessError(err error) bool {
	if err == nil || isEnterFarmBannedError(err) || isTransientNetworkError(err) {
		return false
	}
	msg := strings.ToLower(err.Error())
	if msg == "" {
		return false
	}
	hasInvalid := false
	for _, kw := range []string{"无效", "不存在", "删除", "关系", "not found", "invalid", "not friend", "friend"} {
		if strings.Contains(msg, strings.ToLower(kw)) {
			hasInvalid = true
			break
		}
	}
	return hasInvalid && parseRPCErrorCode(err) > 0
}

func (s *Session) isQuiescing() bool {
	if s == nil {
		return false
	}
	st := s.Status()
	return st == StatusStopping || st == StatusStopped || st == StatusError
}

func shouldAbortFriendPatrol(ctx context.Context, s *Session) bool {
	if ctx != nil && ctx.Err() != nil {
		return true
	}
	return s != nil && s.isQuiescing()
}

// abortFriendPatrol stops the batch after a transport/session exit error.
// While stopping/kicked, stay silent (bot quiesceBot); otherwise log once.
func abortFriendPatrol(s *Session, accountID uint64, kind string, err error) {
	if s != nil && s.isQuiescing() {
		slog.Info("friend patrol aborted (session exiting)", "account", accountID, "kind", kind, "err", friendlyNetworkError(err))
		return
	}
	label := "偷菜"
	switch kind {
	case "help":
		label = "帮助"
	case "bad":
		label = "捣乱"
	}
	msg := fmt.Sprintf("%s巡查中断: %s", label, friendlyNetworkError(err))
	slog.Warn("friend patrol aborted", "account", accountID, "kind", kind, "err", err)
	if s == nil || s.hub == nil || accountID == 0 {
		return
	}
	s.hub.PublishJSON("friend_interact", accountID, map[string]any{
		"action":  kind,
		"result":  "error",
		"event":   "好友巡查中断",
		"message": msg,
		"isWarn":  true,
		"tag":     "好友",
	})
}

// handleFriendEnterError mirrors bot: 1002003 → blacklist; invalid friend → drop known gid.
func handleFriendEnterError(s *Session, gid int64, err error) bool {
	if s == nil || err == nil || gid <= 0 {
		return false
	}
	if isEnterFarmBannedError(err) {
		if addFriendToBlacklist(s, gid, err.Error()) {
			slog.Warn("friend enter banned, auto blacklisted", "account", s.id, "friend_gid", gid, "err", err)
			writeInteractLog(parseAccountID(s.id), 0, gid, "steal", "error", map[string]any{
				"error": "好友已自动加入黑名单",
			})
		}
		return true
	}
	if isInvalidFriendAccessError(err) {
		removeKnownFriendGID(s, gid)
		slog.Info("invalid friend gid removed", "account", s.id, "friend_gid", gid, "err", err)
		return true
	}
	return false
}

func removeKnownFriendGID(s *Session, gid int64) {
	if s == nil || gid <= 0 {
		return
	}
	s.mu.Lock()
	cfg := s.cfg.AccountConfig
	next := make([]int64, 0, len(cfg.KnownFriendGids))
	changed := false
	for _, id := range cfg.KnownFriendGids {
		if id == gid {
			changed = true
			continue
		}
		next = append(next, id)
	}
	if changed {
		cfg.KnownFriendGids = next
		s.cfg.AccountConfig = cfg
	}
	s.mu.Unlock()
	if !changed {
		return
	}
	persistKnownFriendGIDs(parseAccountID(s.id), cfg)
}

func persistKnownFriendGIDs(accountID uint64, cfg logic.AccountConfig) {
	if accountID == 0 {
		return
	}
	db := vars.DB
	if db == nil {
		return
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return
	}
	now := uint(time.Now().Unix())
	_ = db.Model(&model.FarmAccountConfig{}).
		Where("account_id = ?", accountID).
		Updates(map[string]any{
			"config_json": string(raw),
			"updated_at":  now,
		}).Error
}

func addFriendToBlacklist(s *Session, gid int64, reason string) bool {
	if s == nil || gid <= 0 {
		return false
	}
	s.mu.Lock()
	cfg := s.cfg.AccountConfig
	for _, existing := range cfg.FriendBlacklist {
		if existing == gid {
			s.mu.Unlock()
			return false
		}
	}
	cfg.FriendBlacklist = append(append([]int64(nil), cfg.FriendBlacklist...), gid)
	s.cfg.AccountConfig = cfg
	s.mu.Unlock()
	persistFriendBlacklist(parseAccountID(s.id), cfg)
	_ = reason
	return true
}

func persistFriendBlacklist(accountID uint64, cfg logic.AccountConfig) {
	if accountID == 0 {
		return
	}
	db := vars.DB
	if db == nil {
		return
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return
	}
	now := uint(time.Now().Unix())
	_ = db.Model(&model.FarmAccountConfig{}).
		Where("account_id = ?", accountID).
		Updates(map[string]any{
			"config_json": string(raw),
			"updated_at":  now,
		}).Error
}

func manualHelpFriend(ctx context.Context, s *Session, api *game.API, gid int64, op string) (friendVisitOutcome, bool, error) {
	var out friendVisitOutcome
	lands, err := api.VisitEnter(ctx, gid, enterReasonFriend)
	if err != nil {
		if handleFriendEnterError(s, gid, err) {
			return out, false, nil
		}
		return out, false, err
	}
	defer func() {
		if leaveErr := api.VisitLeave(ctx, gid); err == nil && leaveErr != nil {
			err = leaveErr
		}
	}()
	needWater, needWeed, needBug := logic.AnalyzeFriendHelpLands(lands)
	out.Weed = len(needWeed)
	out.Bug = len(needBug)
	out.Water = len(needWater)
	switch op {
	case "help":
		allHelp := uniqueLandIDs(needWeed, needBug, needWater)
		if len(allHelp) == 0 {
			return out, false, nil
		}
		reply, farmErr := api.FriendFarming(ctx, gid, allHelp)
		if farmErr != nil {
			if isFarmingNoopError(farmErr) {
				return out, false, nil
			}
			return out, false, farmErr
		}
		if s != nil && reply != nil {
			s.ensureHelpState().updateLimits(reply.OperationLimits)
		}
		count := len(allHelp)
		if reply != nil {
			if landIDs := farmingResultLandIDs(reply.Results); len(landIDs) > 0 {
				count = len(landIDs)
			}
		}
		out.Count = count
		out.Summary = formatHelpSummary(count, out.Weed, out.Bug, out.Water)
		return out, false, nil
	case "water":
		if len(needWater) == 0 {
			return out, false, nil
		}
		reply, waterErr := api.FriendWater(ctx, gid, needWater)
		if waterErr != nil {
			return out, false, waterErr
		}
		if s != nil && reply != nil {
			s.ensureHelpState().updateLimits(reply.OperationLimits)
		}
		out.Count = len(needWater)
		out.Summary = fmt.Sprintf("浇水%d", out.Count)
		return out, false, nil
	case "weed":
		if len(needWeed) == 0 {
			return out, false, nil
		}
		reply, farmErr := api.FriendFarming(ctx, gid, needWeed)
		if farmErr != nil {
			if isFarmingNoopError(farmErr) {
				return out, false, nil
			}
			return out, false, farmErr
		}
		if s != nil && reply != nil {
			s.ensureHelpState().updateLimits(reply.OperationLimits)
		}
		out.Count = len(needWeed)
		out.Summary = fmt.Sprintf("除草%d", out.Count)
		return out, false, nil
	case "bug":
		if len(needBug) == 0 {
			return out, false, nil
		}
		reply, farmErr := api.FriendFarming(ctx, gid, needBug)
		if farmErr != nil {
			if isFarmingNoopError(farmErr) {
				return out, false, nil
			}
			return out, false, farmErr
		}
		if s != nil && reply != nil {
			s.ensureHelpState().updateLimits(reply.OperationLimits)
		}
		out.Count = len(needBug)
		out.Summary = fmt.Sprintf("除虫%d", out.Count)
		return out, false, nil
	default:
		return out, false, fmt.Errorf("unsupported help operation %q", op)
	}
}

func badFriend(ctx context.Context, s *Session, api *game.API, myGID, gid int64) (friendVisitOutcome, error) {
	var out friendVisitOutcome
	helpState := s.ensureHelpState()
	if helpState.isBadOperationLimitReached() {
		return out, nil
	}
	lands, err := api.VisitEnter(ctx, gid, enterReasonFriend)
	if err != nil {
		if handleFriendEnterError(s, gid, err) {
			return out, nil
		}
		return out, err
	}
	defer func() {
		if leaveErr := api.VisitLeave(ctx, gid); err == nil && leaveErr != nil {
			err = leaveErr
		}
	}()
	weedTargets, bugTargets := collectBadLandTargets(lands, myGID)

	// Bot: slice by remaining times; insect failure must not block weeds; 1001046 marks day limit.
	if len(bugTargets) > 0 && helpState.canOperate(friendOpPutBug) {
		remaining := helpState.getRemainingTimes(friendOpPutBug)
		if remaining > 0 {
			if remaining < len(bugTargets) {
				bugTargets = bugTargets[:remaining]
			}
			canOperate, _, checkErr := api.CheckCanOperate(ctx, gid, friendOpPutBug)
			if checkErr != nil || canOperate {
				reply, putErr := api.PutInsects(ctx, gid, bugTargets)
				if putErr != nil {
					if isBadOpLimitError(putErr) {
						helpState.markBadOperationLimitReached()
					} else {
						slog.Debug("put insects failed", "account", sessionID(s), "friend_gid", gid, "err", putErr)
					}
				} else {
					if reply != nil {
						helpState.updateLimits(reply.OperationLimits)
					}
					out.PutBug = len(bugTargets)
					out.Count += out.PutBug
				}
			}
		}
	}
	if helpState.isBadOperationLimitReached() {
		if out.Count > 0 {
			out.Summary = formatBadSummary(out.PutBug, out.PutWeed)
		}
		return out, nil
	}
	if len(weedTargets) > 0 && helpState.canOperate(friendOpPutWeed) {
		remaining := helpState.getRemainingTimes(friendOpPutWeed)
		if remaining > 0 {
			if remaining < len(weedTargets) {
				weedTargets = weedTargets[:remaining]
			}
			canOperate, _, checkErr := api.CheckCanOperate(ctx, gid, friendOpPutWeed)
			if checkErr != nil || canOperate {
				reply, putErr := api.PutWeeds(ctx, gid, weedTargets)
				if putErr != nil {
					if isBadOpLimitError(putErr) {
						helpState.markBadOperationLimitReached()
					} else {
						slog.Debug("put weeds failed", "account", sessionID(s), "friend_gid", gid, "err", putErr)
					}
				} else {
					if reply != nil {
						helpState.updateLimits(reply.OperationLimits)
					}
					out.PutWeed = len(weedTargets)
					out.Count += out.PutWeed
				}
			}
		}
	}
	if out.Count > 0 {
		out.Summary = formatBadSummary(out.PutBug, out.PutWeed)
	}
	return out, nil
}

func isBadOpLimitError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "code=1001046") || strings.Contains(msg, "1001046")
}

func formatBadSummary(putBug, putWeed int) string {
	parts := make([]string, 0, 2)
	if putBug > 0 {
		parts = append(parts, fmt.Sprintf("放虫%d", putBug))
	}
	if putWeed > 0 {
		parts = append(parts, fmt.Sprintf("放草%d", putWeed))
	}
	return strings.Join(parts, "/")
}

// collectBadLandTargets mirrors bot canPutWeed/canPutBug: owners < 2 and not already put by me.
func collectBadLandTargets(lands []logic.LandInfo, myGID int64) (weedTargets, bugTargets []int64) {
	landsMap := logic.BuildLandMap(lands)
	for i := range lands {
		land := &lands[i]
		if logic.IsOccupiedSlaveLand(land, landsMap) {
			continue
		}
		if land.Plant == nil || !land.Unlocked || !isGrowing(land) {
			continue
		}
		weedOwners := land.Plant.WeedOwners
		insectOwners := land.Plant.InsectOwners
		if len(weedOwners) < 2 && !containsID(weedOwners, myGID) {
			weedTargets = append(weedTargets, land.ID)
		}
		if len(insectOwners) < 2 && !containsID(insectOwners, myGID) {
			bugTargets = append(bugTargets, land.ID)
		}
	}
	return weedTargets, bugTargets
}

// isPlantBlacklistedBySeed compares plant.ID → SeedID against the seed-id blacklist.
func isPlantBlacklistedBySeed(blacklist map[int64]struct{}, plantID int64) bool {
	if plantID <= 0 || len(blacklist) == 0 {
		return false
	}
	plant := logic.GlobalGameConfig.GetPlantByID(plantID)
	if plant == nil || plant.SeedID == nil || *plant.SeedID <= 0 {
		return false
	}
	return hasID(blacklist, *plant.SeedID)
}

func isMature(land logic.LandInfo) bool {
	return land.Plant != nil && logic.GetCurrentPhase(land.Plant.Phases) != nil &&
		logic.GetCurrentPhase(land.Plant.Phases).Phase == logic.PhaseMature
}

func isGrowing(land *logic.LandInfo) bool {
	if land == nil || land.Plant == nil {
		return false
	}
	phase := logic.GetCurrentPhase(land.Plant.Phases)
	return phase != nil && phase.Phase != logic.PhaseMature && phase.Phase != logic.PhaseDead
}

func makeIDSet(ids []int64) map[int64]struct{} {
	out := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		out[id] = struct{}{}
	}
	return out
}

func hasID(ids map[int64]struct{}, id int64) bool {
	_, ok := ids[id]
	return ok
}

func containsID(ids []int64, id int64) bool {
	for _, candidate := range ids {
		if candidate == id {
			return true
		}
	}
	return false
}

func writeInteractLog(accountID uint64, _ uint64, targetGID int64, action, result string, detail any) {
	if accountID == 0 {
		return
	}
	db := vars.DB
	if db == nil {
		return
	}
	friendName := lookupFriendNickname(accountID, targetGID)
	payload, err := json.Marshal(detail)
	if err != nil {
		payload = []byte(`{}`)
	}
	row := model.FarmInteractLog{
		AccountID: accountID, TargetGid: targetGID,
		Action: action, Result: result, DetailJSON: string(payload), CreatedAt: uint(time.Now().Unix()),
	}
	if err := db.Create(&row).Error; err != nil {
		slog.Warn("farm interaction log failed", "account", accountID, "target_gid", targetGID, "action", action, "err", err)
	}

	detailMap, _ := detail.(map[string]any)
	if detailMap == nil {
		detailMap = map[string]any{}
	}
	count, _ := detailMap["count"].(int)
	if count == 0 {
		if v, ok := detailMap["count"].(float64); ok {
			count = int(v)
		}
	}
	errText, _ := detailMap["error"].(string)
	message := formatFriendInteractMessage(friendName, targetGID, action, result, count, errText, detailMap)
	event := friendActionEvent(action)
	hub.Default.PublishJSON("friend_interact", accountID, map[string]any{
		"targetGid":  targetGID,
		"friendName": friendName,
		"action":     action,
		"result":     result,
		"event":      event,
		"message":    message,
		"isWarn":     result == "error" || errText != "",
		"detail":     detail,
		"tag":        "好友",
	})
}

func lookupFriendNickname(accountID uint64, gid int64) string {
	if accountID == 0 || gid <= 0 {
		return ""
	}
	db := vars.DB
	if db == nil {
		return ""
	}
	var row model.FarmFriendGid
	if err := db.Where("account_id = ? AND gid = ?", accountID, gid).First(&row).Error; err != nil {
		return ""
	}
	return strings.TrimSpace(row.Nickname)
}

func friendActionEvent(action string) string {
	switch action {
	case "steal":
		return "偷好友菜"
	case "steal_score":
		return "偷取积分"
	case "help", "water", "weed", "bug":
		return "帮助好友"
	case "bad":
		return "放虫放草"
	default:
		return "照顾好友"
	}
}

func formatFriendInteractMessage(friendName string, gid int64, action, result string, count int, errText string, detail map[string]any) string {
	name := friendName
	if name == "" {
		name = fmt.Sprintf("GID:%d", gid)
	}
	if result == "error" || errText != "" {
		msg := errText
		if msg == "" {
			msg = "失败"
		}
		return fmt.Sprintf("%s: %s", name, msg)
	}
	if action == "steal_score" {
		if summary := asDetailString(detail, "summary"); summary != "" {
			return summary
		}
		if count > 0 {
			return fmt.Sprintf("获得积分x%d", count)
		}
	}
	if summary := asDetailString(detail, "summary"); summary != "" {
		return fmt.Sprintf("%s: %s", name, summary)
	}
	label := action
	switch action {
	case "steal":
		label = "偷"
	case "help":
		label = "一键务农"
	case "water":
		label = "浇水"
	case "weed":
		label = "除草"
	case "bug":
		label = "除虫"
	case "bad":
		label = "捣乱"
	}
	plants := asDetailStringSlice(detail, "plants")
	if action == "steal" && count > 0 {
		score := int64(0)
		value := int64(0)
		if detail != nil {
			switch v := detail["score"].(type) {
			case int64:
				score = v
			case int:
				score = int64(v)
			case float64:
				score = int64(v)
			}
			switch v := detail["value"].(type) {
			case int64:
				value = v
			case int:
				value = int64(v)
			case float64:
				value = int64(v)
			}
		}
		return fmt.Sprintf("%s: %s", name, formatStealSummary(count, plants, score, value))
	}
	if action == "steal_score" && count > 0 {
		return fmt.Sprintf("获得积分x%d", count)
	}
	if count > 0 {
		return fmt.Sprintf("%s: %s%d", name, label, count)
	}
	return fmt.Sprintf("%s: %s", name, label)
}

func asDetailString(detail map[string]any, key string) string {
	if detail == nil {
		return ""
	}
	v, ok := detail[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

func asDetailStringSlice(detail map[string]any, key string) []string {
	if detail == nil {
		return nil
	}
	v, ok := detail[key]
	if !ok || v == nil {
		return nil
	}
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, x := range t {
			s := strings.TrimSpace(fmt.Sprint(x))
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}
