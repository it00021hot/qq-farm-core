package runtime

// 神秘商人自动购买（对齐 bot mystery-shop-auto.ts 决策引擎 + 2 小时定时）。

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/farm/game"
	"github.com/it00021hot/qq-farm-core/internal/farm/protocol"
	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
	"github.com/it00021hot/qq-farm-core/internal/farm/push"
)

// currency item ids (bot GOLD_ITEM_ID 等).
const (
	msGoldID     int64 = 1001
	msCouponID   int64 = 1002
	msDiamondID  int64 = 1004
	msGoldBeanID int64 = 1005
)

// msAutoBuyInterval mirrors bot AUTO_BUY_CHECK_INTERVAL_MS.
const msAutoBuyInterval = 2 * time.Hour

// mysteryShopAutoState dedups arrival/purchase notifications per visit.
type mysteryShopAutoState struct {
	lastArrivalKey  string
	lastPurchaseKey string
}

// mysteryShopDecision mirrors bot MysteryShopDecision.
type mysteryShopDecision struct {
	skipReason    string // inactive | disabled
	visitKey      string
	notifyArrival bool
	shouldBuy     bool
	skipBuyReason string // auto_buy_off | currency_not_allowed | balance_unknown | insufficient
}

// msOffer is the resolved offer shape used by the decision engine.
type msOffer struct {
	npcID         int64
	rewardItemID  int64
	rewardCount   int64
	currencyID    int64
	totalPrice    int64
	balance       int64
	balanceKnown  bool
	activeTime    int64
	expireTime    int64
	currencyName  string
}

// msIsCurrencyAllowed maps the currency id to its allow switch.
func msIsCurrencyAllowed(currencyID int64, cfg logic.AccountConfig) bool {
	switch currencyID {
	case msGoldID:
		return cfg.Automation.MysteryShopAllowGold
	case msCouponID:
		return cfg.Automation.MysteryShopAllowCoupon
	case msDiamondID:
		return cfg.Automation.MysteryShopAllowDiamond
	case msGoldBeanID:
		return cfg.Automation.MysteryShopAllowGoldBean
	}
	return false
}

func msCurrencyName(currencyID int64) string {
	switch currencyID {
	case msGoldID:
		return "金币"
	case msCouponID:
		return "点券"
	case msDiamondID:
		return "钻石"
	case msGoldBeanID:
		return "金豆"
	}
	return "货币"
}

// decideMysteryShopTick mirrors bot decideMysteryShopTick.
func decideMysteryShopTick(offer *msOffer, cfg logic.AccountConfig, state *mysteryShopAutoState) mysteryShopDecision {
	if offer == nil || offer.npcID <= 0 || offer.rewardCount <= 0 {
		return mysteryShopDecision{skipReason: "inactive"}
	}
	visitKey := fmt.Sprintf("%d:%d", offer.npcID, offer.activeTime)
	decision := mysteryShopDecision{
		visitKey:      visitKey,
		notifyArrival: cfg.Automation.MysteryShopArrivalNotify && state.lastArrivalKey != visitKey,
	}
	if !cfg.Automation.MysteryShopAutoBuy {
		decision.skipBuyReason = "auto_buy_off"
		return decision
	}
	if !msIsCurrencyAllowed(offer.currencyID, cfg) {
		decision.skipBuyReason = "currency_not_allowed"
		return decision
	}
	if !offer.balanceKnown {
		decision.skipBuyReason = "balance_unknown"
		return decision
	}
	if offer.balance < offer.totalPrice {
		decision.skipBuyReason = "insufficient"
		return decision
	}
	decision.shouldBuy = true
	return decision
}

// resolveMysteryShopPushFlags mirrors bot resolveMysteryShopPushFlags.
func resolveMysteryShopPushFlags(decision mysteryShopDecision, bought bool, cfg logic.AccountConfig, state *mysteryShopAutoState) (arrival, purchase bool) {
	if decision.visitKey == "" {
		return false, false
	}
	purchase = bought && cfg.Automation.MysteryShopPurchaseNotify && state.lastPurchaseKey != decision.visitKey
	return decision.notifyArrival, purchase
}

// buildMysteryShopPush mirrors bot buildMysteryShopPush.
func buildMysteryShopPush(offer *msOffer, arrival, purchase bool) (title, content string, ok bool) {
	if !arrival && !purchase {
		return "", "", false
	}
	item := fmt.Sprintf("神秘商品 x%d", offer.rewardCount)
	if info := logic.GetItemByID(offer.rewardItemID); info != nil && info.Name != "" {
		item = fmt.Sprintf("%s x%d", info.Name, offer.rewardCount)
	}
	price := fmt.Sprintf("%d %s", offer.totalPrice, msCurrencyName(offer.currencyID))
	remain := ""
	if offer.expireTime > 0 {
		diff := time.Until(time.UnixMilli(offer.expireTime))
		if diff > 0 {
			remain = fmt.Sprintf("\n剩余 %d小时%d分", int(diff.Hours()), int(diff.Minutes())%60)
		}
	}
	switch {
	case arrival && purchase:
		return "神秘商人已自动购买", fmt.Sprintf("到货 %s\n花费 %s%s", item, price, remain), true
	case purchase:
		return "神秘商人已自动购买", fmt.Sprintf("购买 %s\n花费 %s", item, price), true
	default:
		return "神秘商人到货", fmt.Sprintf("%s\n价格 %s%s", item, price, remain), true
	}
}

// loadMysteryShopOffer reads the active NPC and resolves the currency balance.
func loadMysteryShopOffer(ctx context.Context, api *game.API) (*msOffer, error) {
	reply, err := api.GetActiveNPC(ctx)
	if err != nil {
		return nil, err
	}
	if !reply.GetIsActive() || reply.GetNpc() == nil {
		return nil, nil
	}
	npc := reply.GetNpc()
	offer := &msOffer{
		npcID:        npc.GetNpcId(),
		rewardItemID: npc.GetRewardItemId(),
		rewardCount:  int64(npc.GetRewardCount()),
		currencyID:   npc.GetCurrencyItemId(),
		totalPrice:   npc.GetPrice() * int64(npc.GetRewardCount()),
		activeTime:   reply.GetActiveTime(),
		expireTime:   reply.GetExpireTime(),
	}
	if bag, bagErr := api.Bag(ctx); bagErr == nil {
		for _, item := range game.GetBagItems(bag) {
			if item.Id == offer.currencyID {
				offer.balance = item.Count
				offer.balanceKnown = true
				break
			}
		}
	}
	return offer, nil
}

// runMysteryShopAutoTick runs one auto-buy decision round.
func (s *Session) runMysteryShopAutoTick(ctx context.Context) {
	cfg := s.Config()
	api := s.GameAPI()
	if api == nil {
		return
	}
	if !cfg.Automation.MysteryShopAutoBuy && !cfg.Automation.MysteryShopArrivalNotify {
		return
	}
	offer, err := loadMysteryShopOffer(ctx, api)
	if err != nil {
		slog.Debug("神秘商人检查失败", "account", s.id, "err", err)
		return
	}
	decision := decideMysteryShopTick(offer, cfg, &s.msAutoState)
	if decision.skipReason == "inactive" || decision.skipReason == "disabled" {
		return
	}

	bought := false
	if decision.shouldBuy {
		if buyErr := api.Buy(ctx, offer.npcID); buyErr != nil {
			slog.Warn("神秘商人自动购买失败", "account", s.id, "err", buyErr)
		} else {
			bought = true
			slog.Info("神秘商人自动购买成功",
				"account", s.id,
				"item", offer.rewardItemID,
				"count", offer.rewardCount,
				"price", offer.totalPrice,
				"currency", msCurrencyName(offer.currencyID))
			s.publishShopLog("神秘商人自动购买",
				fmt.Sprintf("自动购买 %s x%d，花费 %d %s",
					msItemLabel(offer.rewardItemID), offer.rewardCount, offer.totalPrice, msCurrencyName(offer.currencyID)),
				false)
		}
	} else if decision.skipBuyReason == "currency_not_allowed" || decision.skipBuyReason == "insufficient" || decision.skipBuyReason == "balance_unknown" {
		name := msCurrencyName(offer.currencyID)
		var msg string
		switch decision.skipBuyReason {
		case "currency_not_allowed":
			msg = "神秘商人自动购买已跳过：未允许使用" + name
		case "insufficient":
			msg = "神秘商人自动购买已跳过：" + name + "余额不足"
		default:
			msg = "神秘商人自动购买已跳过：未能读取" + name + "余额"
		}
		slog.Info(msg, "account", s.id)
	}

	arrival, purchase := resolveMysteryShopPushFlags(decision, bought, cfg, &s.msAutoState)
	if arrival {
		s.msAutoState.lastArrivalKey = decision.visitKey
	}
	if purchase {
		s.msAutoState.lastPurchaseKey = decision.visitKey
	}
	if arrival || purchase {
		if title, content, ok := buildMysteryShopPush(offer, arrival, purchase); ok {
			s.publishShopLog(title, content, false)
			go pushAccountNotice(s, title, content)
		}
	}
}

func msItemLabel(itemID int64) string {
	if info := logic.GetItemByID(itemID); info != nil && info.Name != "" {
		return info.Name
	}
	return fmt.Sprintf("物品#%d", itemID)
}

func (s *Session) publishShopLog(event, message string, isWarn bool) {
	if s.hub == nil {
		return
	}
	s.hub.PublishJSON("runtime_log", parseAccountID(s.id), map[string]any{
		"tag":       "商城",
		"event":     "mystery_shop_watch",
		"module":    "warehouse",
		"message":   message,
		"isWarn":    isWarn,
		"accountId": parseAccountID(s.id),
	})
}

// mysteryShopAutoLoop ticks the auto-buy decision every 2 hours.
func (s *Session) mysteryShopAutoLoop(ctx context.Context) {
	ticker := time.NewTicker(msAutoBuyInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runMysteryShopAutoTick(protocol.WithRequestClass(ctx, protocol.ClassFarm))
		}
	}
}

// pushAccountNotice sends an account-scoped notice through the configured
// push channel (webhook today; QQ bot / DingTalk / WeChat in the push module).
func pushAccountNotice(s *Session, title, content string) {
	if s == nil {
		return
	}
	push.NotifyAll(s.cfg.PushWebhook, title, content)
}
