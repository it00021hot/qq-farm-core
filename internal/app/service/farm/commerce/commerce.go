package commerce

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/app/model"
	"github.com/it00021hot/qq-farm-core/internal/app/service"
	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/corepb"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/mallpb"
	farmruntime "github.com/it00021hot/qq-farm-core/internal/farm/runtime"
	farmtypes "github.com/it00021hot/qq-farm-core/internal/types/farm"
	"github.com/it00021hot/qq-farm-core/internal/vars"
	"github.com/gofiber/fiber/v3"
)

type Service struct {
	service.Service
}

var Commerce = &Service{}

func (s *Service) Mall(ctx fiber.Ctx, req farmtypes.MallListReq) (farmtypes.MallCatalog, error) {
	session, err := s.session(ctx, req.AccountID)
	if err != nil {
		return farmtypes.MallCatalog{}, friendlyFarmErr(err)
	}
	if req.SlotType == 0 {
		req.SlotType = 1
	}
	callCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	result, err := s.mallCatalog(callCtx, session, req.SlotType, req.SubSlotType)
	return result, friendlyFarmErr(err)
}

func (s *Service) Purchase(ctx fiber.Ctx, req farmtypes.MallPurchaseReq) (farmtypes.MallPurchaseResult, error) {
	session, err := s.session(ctx, req.AccountID)
	if err != nil {
		return farmtypes.MallPurchaseResult{}, friendlyFarmErr(err)
	}
	api := session.GameAPI()
	if api == nil {
		return farmtypes.MallPurchaseResult{}, errors.New("账号未连接游戏")
	}
	callCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	reply, err := api.Purchase(callCtx, req.GoodsID, req.Count)
	if err != nil {
		return farmtypes.MallPurchaseResult{}, friendlyFarmErr(err)
	}
	catalog, err := s.mallCatalog(callCtx, session, 1, 0)
	if err != nil {
		return farmtypes.MallPurchaseResult{}, friendlyFarmErr(err)
	}
	return farmtypes.MallPurchaseResult{
		Purchase: farmtypes.MallPurchase{
			GoodsID: reply.GetGoodsId(),
			Count:   reply.GetCount(),
			Rewards: commerceItems(reply.GetRewardItems()),
			Limit:   purchaseLimit(reply.GetPurchaseLimit()),
		},
		Catalog: catalog,
	}, nil
}

func (s *Service) MysteryShop(ctx fiber.Ctx, req farmtypes.CommerceAccountReq) (farmtypes.MysteryShop, error) {
	session, err := s.session(ctx, req.AccountID)
	if err != nil {
		return farmtypes.MysteryShop{}, friendlyFarmErr(err)
	}
	callCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	result, err := s.loadMysteryShop(callCtx, session)
	return result, friendlyFarmErr(err)
}

func (s *Service) PurchaseMystery(ctx fiber.Ctx, req farmtypes.MysteryPurchaseReq) (farmtypes.MysteryPurchaseResult, error) {
	session, err := s.session(ctx, req.AccountID)
	if err != nil {
		return farmtypes.MysteryPurchaseResult{}, friendlyFarmErr(err)
	}
	api := session.GameAPI()
	if api == nil {
		return farmtypes.MysteryPurchaseResult{}, errors.New("账号未连接游戏")
	}
	callCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	before, err := s.loadMysteryShop(callCtx, session)
	if err != nil {
		return farmtypes.MysteryPurchaseResult{}, friendlyFarmErr(err)
	}
	offer := before.NPC
	if !before.Active || offer == nil || offer.ID != req.NpcID {
		return farmtypes.MysteryPurchaseResult{}, errors.New("神秘商人商品已失效")
	}
	if offer.Stock <= 0 {
		return farmtypes.MysteryPurchaseResult{}, errors.New("神秘商人商品已售罄")
	}
	if offer.Price.Balance != nil && *offer.Price.Balance < offer.Price.Count {
		return farmtypes.MysteryPurchaseResult{}, errors.New("货币余额不足")
	}

	if err := api.Buy(callCtx, req.NpcID); err != nil {
		return farmtypes.MysteryPurchaseResult{}, friendlyFarmErr(err)
	}
	shop, err := s.loadMysteryShop(callCtx, session)
	if err != nil {
		return farmtypes.MysteryPurchaseResult{}, friendlyFarmErr(err)
	}
	if shop.Active && shop.NPC != nil && shop.NPC.ID == req.NpcID && shop.NPC.Stock >= offer.Stock {
		return farmtypes.MysteryPurchaseResult{}, errors.New("神秘商人购买未确认")
	}
	return farmtypes.MysteryPurchaseResult{
		Purchase: farmtypes.MysteryPurchase{
			NpcID:          offer.ID,
			Reward:          offer.Reward,
			Price:           offer.Price,
			OriginalPrice:   offer.OriginalPrice,
			DiscountPercent: offer.DiscountPercent,
		},
		Shop: shop,
	}, nil
}

func (s *Service) loadMysteryShop(ctx context.Context, session *farmruntime.Session) (farmtypes.MysteryShop, error) {
	api := session.GameAPI()
	if api == nil {
		return farmtypes.MysteryShop{}, errors.New("账号未连接游戏")
	}
	reply, err := api.GetActiveNPC(ctx)
	if err != nil {
		return farmtypes.MysteryShop{}, err
	}
	result := farmtypes.MysteryShop{
		Active:     reply.GetIsActive() && reply.GetNpc() != nil,
		ServerTime: time.Now().UnixMilli(),
	}
	npc := reply.GetNpc()
	if !result.Active || npc == nil {
		return result, nil
	}
	balances := bagBalances(ctx, session)
	rewardCount := int64(max32(npc.GetRewardCount()))
	unitPrice := max64(npc.GetPrice())
	unitOriginal := max64(npc.GetOriginalPrice())
	result.ActiveTime = millis(reply.GetActiveTime())
	result.ExpireTime = millis(reply.GetExpireTime())
	result.NPC = &farmtypes.MysteryNPC{
		ID:                npc.GetNpcId(),
		Reward:            commerceItem(npc.GetRewardItemId(), rewardCount, "神秘商品"),
		Stock:             0,
		Price:             commercePrice(npc.GetCurrencyItemId(), unitPrice*rewardCount, balances),
		OriginalPrice:     unitOriginal * rewardCount,
		UnitPrice:         unitPrice,
		UnitOriginalPrice: unitOriginal,
		DiscountPercent:   max32(npc.GetDiscountPercent()),
	}
	return result, nil
}

func (s *Service) Diamond(ctx fiber.Ctx, req farmtypes.CommerceAccountReq) (farmtypes.DiamondBalance, error) {
	session, err := s.session(ctx, req.AccountID)
	if err != nil {
		return farmtypes.DiamondBalance{}, friendlyFarmErr(err)
	}
	api := session.GameAPI()
	if api == nil {
		return farmtypes.DiamondBalance{}, errors.New("账号未连接游戏")
	}
	callCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	balance, err := api.GetDiamondBalance(callCtx)
	if err != nil {
		return farmtypes.DiamondBalance{}, friendlyFarmErr(err)
	}
	return farmtypes.DiamondBalance{Diamond: balance}, nil
}

func (s *Service) mallCatalog(ctx context.Context, session *farmruntime.Session, slotType, subSlotType int32) (farmtypes.MallCatalog, error) {
	api := session.GameAPI()
	if api == nil {
		return farmtypes.MallCatalog{}, errors.New("账号未连接游戏")
	}
	reply, err := api.GetMallListBySlot(ctx, slotType, subSlotType)
	if err != nil {
		return farmtypes.MallCatalog{}, err
	}
	balances := bagBalances(ctx, session)
	currencyIDs := make([]int64, 0)
	seenCurrency := make(map[int64]struct{})
	goods := make([]farmtypes.MallGoods, 0, len(reply.GetGoodsList()))
	for _, raw := range reply.GetGoodsList() {
		if raw == nil {
			continue
		}
		priceID := raw.GetPrice().GetId()
		if priceID > 0 {
			if _, ok := seenCurrency[priceID]; !ok {
				seenCurrency[priceID] = struct{}{}
				currencyIDs = append(currencyIDs, priceID)
			}
		}
		limit := purchaseLimit(raw.GetPurchaseLimit())
		available := raw.GetIsAvailable()
		purchasable := available && (limit == nil || limit.Remaining == nil || *limit.Remaining > 0)
		price := commercePrice(priceID, raw.GetPrice().GetCount(), balances)
		goods = append(goods, farmtypes.MallGoods{
			ID:              raw.GetGoodsId(),
			Name:            raw.GetName(),
			Type:            raw.GetGoodsType(),
			Rewards:         commerceItems(raw.GetRewardItems()),
			Price:           price,
			IsFree:          raw.GetIsFree() || price.ID == 0 || price.Count == 0,
			Limit:           limit,
			IsLimited:       raw.GetIsLimited(),
			DiscountText:    raw.GetDiscountText(),
			IsDiscounted:    raw.GetIsDiscounted(),
			DiscountEndTime: millis(raw.GetDiscountEndTime()),
			Available:       available,
			Purchasable:     purchasable,
		})
	}
	currencies := make([]farmtypes.MallCurrency, 0, len(currencyIDs))
	for _, id := range currencyIDs {
		balance, known := balances[id]
		currencies = append(currencies, farmtypes.MallCurrency{
			CommerceItem: commerceItem(id, balance, ""),
			BalanceKnown: known,
		})
	}
	return farmtypes.MallCatalog{
		SlotType:         slotType,
		SubSlotType:      subSlotType,
		ServerTime:       time.Now().UnixMilli(),
		RefreshCountdown: max64(reply.GetRefreshCountdown()),
		Currencies:       currencies,
		Goods:            goods,
	}, nil
}

func commerceItems(items []*corepb.Item) []farmtypes.CommerceItem {
	result := make([]farmtypes.CommerceItem, 0, len(items))
	for _, item := range items {
		if item != nil {
			result = append(result, commerceItem(item.GetId(), item.GetCount(), ""))
		}
	}
	return result
}

func commerceItem(id, count int64, fallbackName string) farmtypes.CommerceItem {
	name := fallbackName
	var rarity int64
	if item := logic.GetItemByID(id); item != nil {
		if item.Name != "" {
			name = item.Name
		}
		rarity = item.Rarity
	}
	if name == "" {
		name = fmt.Sprintf("物品 #%d", id)
	}
	return farmtypes.CommerceItem{
		ID:     max64(id),
		Count:  max64(count),
		Name:   name,
		Image:  logic.SeedImagePath(id),
		Rarity: max64(rarity),
	}
}

func commercePrice(id, count int64, balances map[int64]int64) farmtypes.MallPrice {
	item := commerceItem(id, count, "")
	var balance *int64
	if value, ok := balances[id]; ok {
		value = max64(value)
		balance = &value
	}
	return farmtypes.MallPrice{CommerceItem: item, Balance: balance}
}

func purchaseLimit(limit *mallpb.PurchaseLimit) *farmtypes.PurchaseLimit {
	if limit == nil {
		return nil
	}
	bought := max32(limit.GetBoughtCount())
	max := max32(limit.GetLimitCount())
	var remaining *int32
	if max > 0 {
		value := max - bought
		if value < 0 {
			value = 0
		}
		remaining = &value
	}
	return &farmtypes.PurchaseLimit{
		Type:      max32(limit.GetLimitType()),
		Bought:    bought,
		Max:       max,
		Remaining: remaining,
	}
}

func bagBalances(ctx context.Context, session *farmruntime.Session) map[int64]int64 {
	result := make(map[int64]int64)
	items, err := session.GetBagItems(ctx)
	if err != nil {
		return result
	}
	for _, item := range items {
		if item.Id > 0 && item.Count > 0 {
			result[item.Id] += item.Count
		}
	}
	return result
}

func friendlyFarmErr(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "EOF") ||
		strings.Contains(msg, "i/o timeout") {
		return errors.New("游戏连接已断开，请重新启动账号")
	}
	return err
}

func (s *Service) session(ctx fiber.Ctx, accountID uint64) (*farmruntime.Session, error) {
	var account model.FarmAccount
	if err := vars.DB.Where("id = ?", accountID).First(&account).Error; err != nil {
		return nil, errors.New("账号不存在")
	}
	session, ok := farmruntime.Default.Session(accountID)
	if !ok || session.Status() != farmruntime.StatusRunning {
		return nil, errors.New("账号未运行")
	}
	return session, nil
}

func max64(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}

func max32(value int32) int32 {
	if value < 0 {
		return 0
	}
	return value
}

func millis(seconds int64) int64 {
	return max64(seconds) * int64(time.Second/time.Millisecond)
}
