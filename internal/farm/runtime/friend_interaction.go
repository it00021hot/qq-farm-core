package runtime

// 特殊互动道具：库存发现与顺序批量使用（对齐 bot friend-interaction-items.ts 核心路径），
// 以及删除好友（bot deleteFriend）。

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/app/model"
	"github.com/it00021hot/qq-farm-core/internal/farm/game"
	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/corepb"
	"github.com/it00021hot/qq-farm-core/internal/vars"
)

const (
	interactionItemType     = 23
	specialInteractionType  = "additemuseitem"
	interactionMaxBatchLand = 48
)

// selfUsableInteractionItems mirrors bot SELF_USABLE_INTERACTION_ITEM_IDS.
var selfUsableInteractionItems = map[int64]struct{}{
	5003:   {}, // 闪电变异瓶
	301103: {}, // 七夕活动土地道具
}

// friendFarmItemIDs mirrors bot FRIEND_FARM_ITEM_IDS.
var friendFarmItemIDs = map[int64]struct{}{5005: {}}

// interactionMutationMu serializes interaction batches (bot serializeMutation).
var interactionMutationMu sync.Mutex

func isLandInteractionInfo(info *logic.ItemInfo) bool {
	return info != nil &&
		info.Type == interactionItemType &&
		info.CanUse > 0 &&
		strings.EqualFold(strings.TrimSpace(info.InteractionType), specialInteractionType)
}

func isFriendLandInteractionInfo(info *logic.ItemInfo) bool {
	if !isLandInteractionInfo(info) {
		return false
	}
	switch info.ID {
	case 301101, 301102, 301103:
		return true
	}
	desc := info.Desc + " " + info.EffectDesc
	return strings.Contains(desc, "好友") || strings.Contains(desc, "他人")
}

func isFriendFarmInteractionInfo(info *logic.ItemInfo) bool {
	if info == nil {
		return false
	}
	if info.Type != interactionItemType || info.CanUse <= 0 {
		return false
	}
	_, ok := friendFarmItemIDs[info.ID]
	return ok
}

func isSelfLandInteractionInfo(info *logic.ItemInfo) bool {
	if !isLandInteractionInfo(info) {
		return false
	}
	_, ok := selfUsableInteractionItems[info.ID]
	return ok
}

// interactionStack is one usable bag stack sorted by expiry.
type interactionStack struct {
	uid       int64
	remaining int64
	expireAt  int64
}

// sellConditionSatisfied mirrors rust stack_sale_condition_satisfied:
// 道具的 sell_cond 在当前时间 + 过期时间下是否已满足。
func sellConditionSatisfied(info *logic.ItemInfo, expireAt int64) bool {
	if info == nil || info.SellCond == nil || strings.TrimSpace(*info.SellCond) == "" {
		return false
	}
	ctx := logic.DefaultSellConditionContext(logic.GetServerTimeSec())
	ctx.ExpireTime = expireAt
	return logic.IsSellConditionSatisfied(*info.SellCond, ctx)
}

// collectInteractionStacks returns usable (unlocked, positive) stacks for the item.
func collectInteractionStacks(items []corepb.Item, itemID int64) []interactionStack {
	stacks := make([]interactionStack, 0, 4)
	for _, item := range items {
		if item.Id != itemID || item.Count <= 0 || item.Locked {
			continue
		}
		stacks = append(stacks, interactionStack{uid: item.Uid, remaining: item.Count, expireAt: item.ExpireTime})
	}
	sort.SliceStable(stacks, func(i, j int) bool {
		ei, ej := stacks[i].expireAt, stacks[j].expireAt
		if ei <= 0 {
			ei = 1<<62 - 1
		}
		if ej <= 0 {
			ej = 1<<62 - 1
		}
		return ei < ej
	})
	return stacks
}

// InteractionItemDTO is the panel row for one interaction item.
type InteractionItemDTO struct {
	ItemID     int64  `json:"itemId"`
	Name       string `json:"name"`
	Image      string `json:"image"`
	Count      int64  `json:"count"`
	TargetKind string `json:"targetKind"`
	SelfUsable bool   `json:"selfUsable"`
	// 对齐 rust item_dto：道具说明与满足出售条件的库存数（前端展示用）。
	Description                 string `json:"description"`
	SaleConditionSatisfiedCount int64  `json:"saleConditionSatisfiedCount"`
}

// GetFriendInteractionItems lists usable friend-land / friend-farm interaction items.
func GetFriendInteractionItems(ctx context.Context, api *game.API) ([]InteractionItemDTO, error) {
	bag, err := api.Bag(ctx)
	if err != nil {
		return nil, err
	}
	items := game.GetBagItems(bag)
	seen := map[int64]struct{}{}
	dto := make([]InteractionItemDTO, 0, 4)
	for _, item := range items {
		if item.Count <= 0 {
			continue
		}
		if _, dup := seen[item.Id]; dup {
			continue
		}
		seen[item.Id] = struct{}{}
		info := logic.GetItemByID(item.Id)
		kind := ""
		switch {
		case isFriendLandInteractionInfo(info):
			kind = "land"
		case isFriendFarmInteractionInfo(info):
			kind = "farm"
		default:
			continue
		}
		stacks := collectInteractionStacks(items, item.Id)
		total := int64(0)
		for _, s := range stacks {
			total += s.remaining
		}
		if total <= 0 {
			continue
		}
		name := fmt.Sprintf("物品%d", item.Id)
		if info != nil && info.Name != "" {
			name = info.Name
		}
		_, selfUsable := selfUsableInteractionItems[item.Id]
		description, saleSatisfied := "", int64(0)
		if info != nil {
			description = info.Desc
			for _, stack := range collectInteractionStacks(items, item.Id) {
				if sellConditionSatisfied(info, stack.expireAt) {
					saleSatisfied += stack.remaining
				}
			}
		}
		dto = append(dto, InteractionItemDTO{
			ItemID: item.Id, Name: name, Image: logic.SeedImagePath(item.Id),
			Count: total, TargetKind: kind, SelfUsable: selfUsable,
			Description: description, SaleConditionSatisfiedCount: saleSatisfied,
		})
	}
	return dto, nil
}

// UseFriendInteractionItemBatch enters the friend farm and applies the item to
// each land in order (bot useFriendInteractionItemBatch).
// GetSelfInteractionItems lists self-usable interaction items (SELF_USABLE whitelist).
func GetSelfInteractionItems(ctx context.Context, api *game.API) ([]InteractionItemDTO, error) {
	items, err := GetFriendInteractionItems(ctx, api)
	if err != nil {
		return nil, err
	}
	out := make([]InteractionItemDTO, 0, len(items))
	for _, item := range items {
		if item.SelfUsable {
			out = append(out, item)
		}
	}
	return out, nil
}

func UseFriendInteractionItemBatch(ctx context.Context, s *Session, api *game.API, friendGID, itemID int64, landIDs []int64) (used, failed int, err error) {
	if friendGID <= 0 || itemID <= 0 || len(landIDs) == 0 {
		return 0, 0, fmt.Errorf("参数无效")
	}
	if len(landIDs) > interactionMaxBatchLand {
		return 0, 0, fmt.Errorf("单次最多选择 %d 块地", interactionMaxBatchLand)
	}
	info := logic.GetItemByID(itemID)
	if !isFriendLandInteractionInfo(info) {
		return 0, 0, fmt.Errorf("该物品不是可用于好友土地的特殊互动道具")
	}

	interactionMutationMu.Lock()
	defer interactionMutationMu.Unlock()

	bag, err := api.Bag(ctx)
	if err != nil {
		return 0, 0, err
	}
	stacks := collectInteractionStacks(game.GetBagItems(bag), itemID)
	available := int64(0)
	for _, st := range stacks {
		available += st.remaining
	}
	if available <= 0 {
		return 0, 0, fmt.Errorf("%s 当前没有可提交服务器校验的库存", info.Name)
	}
	if int64(len(landIDs)) > available {
		return 0, 0, fmt.Errorf("已选择 %d 块地，但当前只有 %d 个%s", len(landIDs), available, info.Name)
	}

	detail, err := api.VisitEnterDetailed(ctx, friendGID, enterReasonFriend)
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = api.VisitLeave(ctx, friendGID) }()
	if s != nil {
		// Enter 回包顺手记录狗信息（零额外 RPC）。
		s.recordFriendDogFromEnter(friendGID, detail)
	}

	stackIdx := 0
	for _, landID := range landIDs {
		for stackIdx < len(stacks) && stacks[stackIdx].remaining <= 0 {
			stackIdx++
		}
		if stackIdx >= len(stacks) {
			failed++
			continue
		}
		if _, useErr := api.UseTargeted(ctx, itemID, stacks[stackIdx].uid, friendGID, []int64{landID}); useErr != nil {
			failed++
			continue
		}
		stacks[stackIdx].remaining--
		used++
	}
	return used, failed, nil
}

// UseSelfInteractionItemBatch applies a self-usable item to own lands in order.
func UseSelfInteractionItemBatch(ctx context.Context, s *Session, api *game.API, itemID int64, landIDs []int64) (used, failed int, err error) {
	if api == nil || itemID <= 0 || len(landIDs) == 0 {
		return 0, 0, fmt.Errorf("参数无效")
	}
	if len(landIDs) > interactionMaxBatchLand {
		return 0, 0, fmt.Errorf("单次最多选择 %d 块地", interactionMaxBatchLand)
	}
	info := logic.GetItemByID(itemID)
	if !isSelfLandInteractionInfo(info) {
		return 0, 0, fmt.Errorf("该道具只能在好友农场使用，不能对自己的农场使用")
	}

	interactionMutationMu.Lock()
	defer interactionMutationMu.Unlock()

	bag, err := api.Bag(ctx)
	if err != nil {
		return 0, 0, err
	}
	stacks := collectInteractionStacks(game.GetBagItems(bag), itemID)
	available := int64(0)
	for _, st := range stacks {
		available += st.remaining
	}
	if int64(len(landIDs)) > available {
		return 0, 0, fmt.Errorf("已选择 %d 块地，但当前只有 %d 个%s", len(landIDs), available, info.Name)
	}

	myGID := api.GID
	stackIdx := 0
	for _, landID := range landIDs {
		for stackIdx < len(stacks) && stacks[stackIdx].remaining <= 0 {
			stackIdx++
		}
		if stackIdx >= len(stacks) {
			failed++
			continue
		}
		if _, useErr := api.UseTargeted(ctx, itemID, stacks[stackIdx].uid, myGID, []int64{landID}); useErr != nil {
			failed++
			continue
		}
		stacks[stackIdx].remaining--
		used++
	}
	return used, failed, nil
}

// DeleteFriend removes a friend and updates local state (bot deleteFriend):
// drop from list cache + known GIDs + dog cache, then add to blacklist.
func DeleteFriend(ctx context.Context, s *Session, api *game.API, friendGID int64) error {
	if api == nil || friendGID <= 0 {
		return fmt.Errorf("无效的好友 GID")
	}
	name := fmt.Sprintf("GID:%d", friendGID)
	if s != nil {
		if cache := s.friendsCache; len(cache) > 0 {
			s.friendsMu.Lock()
			for i := range s.friendsCache {
				if s.friendsCache[i].Gid == friendGID {
					if nm := strings.TrimSpace(s.friendsCache[i].Remark); nm != "" {
						name = nm
					} else if nm := strings.TrimSpace(s.friendsCache[i].Name); nm != "" {
						name = nm
					}
					s.friendsCache = append(s.friendsCache[:i], s.friendsCache[i+1:]...)
					break
				}
			}
			s.friendsMu.Unlock()
		}
	}
	if err := api.DelFriend(ctx, friendGID); err != nil {
		return err
	}
	_ = name // name retained for panel logging at the service layer
	if s != nil {
		mergeKnownFriendGIDs(s, excludeGID(s.Config().KnownFriendGids, friendGID))
		if s.petCache != nil {
			s.petCache.forget(friendGID)
		}
		// bot deleteFriend：删除后加入好友黑名单，防止对方再次申请。
		addFriendBlacklist(s, friendGID)
	}
	return nil
}

// addFriendBlacklist appends the gid to the account blacklist and persists it.
func addFriendBlacklist(s *Session, gid int64) {
	if s == nil || gid <= 0 || vars.DB == nil {
		return
	}
	cfg := s.Config()
	for _, existing := range cfg.FriendBlacklist {
		if existing == gid {
			return
		}
	}
	cfg.FriendBlacklist = append(cfg.FriendBlacklist, gid)
	accountID := parseAccountID(s.id)
	var row model.FarmAccountConfig
	if err := vars.DB.Where("account_id = ?", accountID).First(&row).Error; err != nil {
		return
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return
	}
	if err := vars.DB.Model(&row).Updates(map[string]any{
		"config_json": string(raw),
		"updated_at":  uint(time.Now().Unix()),
	}).Error; err != nil {
		slog.Warn("delete friend blacklist persist failed", "account", accountID, "err", err)
		return
	}
	s.ApplyConfig(cfg)
}
