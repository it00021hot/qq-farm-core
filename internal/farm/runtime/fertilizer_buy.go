package runtime

import (
	"context"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/farm/game"
	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/corepb"
)

// 事件驱动化肥补充（对齐 bot 2026-09-11：购买检测不再跑周期定时器，
// 施肥轮结束/登录后/施肥模式变更即检测，账号级节流 60 秒）。
const eventFertilizerBuyMinInterval = 60 * time.Second

// getContainerHoursFromBagItems mirrors warehouse.get_container_hours_from_bag_items:
// container item counts are remaining seconds; hours = seconds / 3600.
func getContainerHoursFromBagItems(items []corepb.Item) (normalHours, organicHours int64) {
	var normalSecs, organicSecs int64
	for _, it := range items {
		switch it.Id {
		case logic.NormalContainerID:
			normalSecs = it.Count
		case logic.OrganicContainerID:
			organicSecs = it.Count
		}
	}
	return normalSecs / 3600, organicSecs / 3600
}

// maybeEventFertilizerBuy runs the threshold check at most once per minute per
// account (bot maybe_event_fertilizer_buy throttle).
func (s *Session) maybeEventFertilizerBuy(ctx context.Context) {
	s.mu.Lock()
	last := s.lastFertBuyCheckAt
	s.mu.Unlock()
	now := time.Now()
	if now.Sub(last) < eventFertilizerBuyMinInterval {
		return
	}
	s.mu.Lock()
	s.lastFertBuyCheckAt = now
	s.mu.Unlock()
	s.checkFertilizerBuyOnce(ctx)
}

// checkFertilizerBuyOnce compares container hours against the configured
// thresholds and buys through the mall when below (bot check_fertilizer_buy_once
// + CommerceService.check_and_buy_fertilizer_both).
func (s *Session) checkFertilizerBuyOnce(ctx context.Context) {
	cfg := s.Config()
	buyOrganic, buyNormal := cfg.Automation.FertilizerBuyOrganic, cfg.Automation.FertilizerBuyNormal
	if !buyOrganic && !buyNormal {
		return
	}
	api := s.GameAPI()
	if api == nil {
		return
	}
	buyCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	items, err := s.GetBagItems(buyCtx)
	if err != nil {
		s.publishFertilizerBuyLog("检测化肥容器失败: "+err.Error(), 0, 0, true)
		return
	}
	normalHours, organicHours := getContainerHoursFromBagItems(items)

	var organicBought, normalBought int32
	if buyOrganic && cfg.FertilizerBuyOrganicCount > 0 && cfg.FertilizerBuyOrganicThresholdHours > 0 &&
		organicHours < int64(cfg.FertilizerBuyOrganicThresholdHours) {
		organicBought = s.autoBuyFertilizer(buyCtx, api, game.FertilizerKindOrganic, int32(cfg.FertilizerBuyOrganicCount))
	}

	// 1000-2000ms 随机延迟，避免双类型购买被风控关联（bot）。
	if organicBought > 0 && buyNormal && cfg.FertilizerBuyNormalCount > 0 && cfg.FertilizerBuyNormalThresholdHours > 0 {
		delay := time.Duration(1000+rand.IntN(1000)) * time.Millisecond
		timer := time.NewTimer(delay)
		select {
		case <-buyCtx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}

	if buyNormal && cfg.FertilizerBuyNormalCount > 0 && cfg.FertilizerBuyNormalThresholdHours > 0 &&
		normalHours < int64(cfg.FertilizerBuyNormalThresholdHours) {
		normalBought = s.autoBuyFertilizer(buyCtx, api, game.FertilizerKindNormal, int32(cfg.FertilizerBuyNormalCount))
	}

	if organicBought > 0 || normalBought > 0 {
		// 购买结果进面板日志：前端据此即时刷新化肥桶。
		s.publishFertilizerBuyLog(formatFertilizerBought(organicBought, normalBought, organicHours, normalHours), organicBought, normalBought, false)
	}
}

// autoBuyFertilizer wraps the mall purchase with the 10-minute cooldown slot
// (bot MallService.acquire_buy_slot(force=true) + daily paused-no-gold latch).
func (s *Session) autoBuyFertilizer(ctx context.Context, api *game.API, kind game.MallFertilizerKind, targetCount int32) int32 {
	now := time.Now()
	dateKey := now.Format("2006-01-02")
	s.mu.Lock()
	if date, ok := s.mallBuyPausedNoGoldDate[kind.MallGoodsID()]; ok && date == dateKey {
		s.mu.Unlock()
		return 0
	}
	s.mu.Unlock()

	bought, err := api.AutoBuyFertilizerViaMall(ctx, kind, targetCount)
	if game.IsInsufficientBalance(err) {
		s.mu.Lock()
		if s.mallBuyPausedNoGoldDate == nil {
			s.mallBuyPausedNoGoldDate = map[int32]string{}
		}
		s.mallBuyPausedNoGoldDate[kind.MallGoodsID()] = dateKey
		s.mu.Unlock()
		s.publishFertilizerBuyLog("余额不足，今日自动购买化肥已暂停", 0, 0, true)
		return bought
	}
	if bought > 0 {
		s.publishFertilizerBuyLog("自动购买"+kind.TypeName()+" x"+strconv.FormatInt(int64(bought), 10), 0, 0, false)
	}
	return bought
}

func formatFertilizerBought(organicBought, normalBought int32, organicHours, normalHours int64) string {
	var b strings.Builder
	b.WriteString("已自动补充化肥：有机 x")
	b.WriteString(strconv.FormatInt(int64(organicBought), 10))
	b.WriteString("（剩")
	b.WriteString(strconv.FormatInt(organicHours, 10))
	b.WriteString("h）/ 普通 x")
	b.WriteString(strconv.FormatInt(int64(normalBought), 10))
	b.WriteString("（剩")
	b.WriteString(strconv.FormatInt(normalHours, 10))
	b.WriteString("h）")
	return b.String()
}

// publishFertilizerBuyLog pushes the purchase result to the dashboard log.
func (s *Session) publishFertilizerBuyLog(message string, organicBought, normalBought int32, isWarn bool) {
	if s.hub == nil {
		return
	}
	s.hub.PublishJSON("runtime_log", parseAccountID(s.id), map[string]any{
		"tag":       "商城",
		"event":     "化肥补充",
		"message":   message,
		"isWarn":    isWarn,
		"accountId": parseAccountID(s.id),
	})
}
