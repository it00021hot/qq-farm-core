package runtime

import (
	"context"
	"testing"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/farm/game"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/corepb"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/plantpb"
)

func TestDogSkillGiftItemIDMatchesRustConstant(t *testing.T) {
	if game.DogSkillGiftItemID != 101351 {
		t.Fatalf("DogSkillGiftItemID = %d, want 101351", game.DogSkillGiftItemID)
	}
}

// TestFarmingSkillGiftCount 对齐 rust DogSkillGiftService::farming_skill_gift_count：
// 只统计 helpFarming 回包 results[].reward.id==101351 的数量。
func TestFarmingSkillGiftCount(t *testing.T) {
	mk := func(id, count int64) *plantpb.FarmingResult {
		return &plantpb.FarmingResult{Reward: &corepb.Item{Id: id, Count: count}}
	}
	results := []*plantpb.FarmingResult{
		mk(game.DogSkillGiftItemID, 1),
		mk(1001, 5),
		mk(game.DogSkillGiftItemID, 2),
		{},
		nil,
	}
	if got := farmingSkillGiftCount(results); got != 3 {
		t.Fatalf("farmingSkillGiftCount = %d, want 3", got)
	}
	if got := farmingSkillGiftCount(nil); got != 0 {
		t.Fatalf("farmingSkillGiftCount(nil) = %d, want 0", got)
	}
	// 数量为 0 的礼包奖励不计入。
	if got := farmingSkillGiftCount([]*plantpb.FarmingResult{
		mk(game.DogSkillGiftItemID, 0),
	}); got != 0 {
		t.Fatalf("zero-count reward must be ignored, got %d", got)
	}
}

// TestMaybeClaimDogSkillGiftsFromFarming 校验帮忙务农回包掉落礼包时触发领取、
// 无掉落时不触发；与 PendingGiftCountNotify 路径共用 dogGiftClaiming 单飞锁。
func TestMaybeClaimDogSkillGiftsFromFarming(t *testing.T) {
	sender := &farmOpSender{}
	api := &game.API{Sender: sender, GID: 1}
	s := &Session{id: "77"}
	reply := &plantpb.FarmingReply{Results: []*plantpb.FarmingResult{
		{LandId: 1, Reward: &corepb.Item{Id: game.DogSkillGiftItemID, Count: 2}},
	}}
	maybeClaimDogSkillGiftsFromFarming(s, context.Background(), api, reply)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && !sender.called("ClaimSkillGifts") {
		time.Sleep(10 * time.Millisecond)
	}
	if !sender.called("ClaimSkillGifts") {
		t.Fatal("expected ClaimSkillGifts after gift drop in help farming reply")
	}

	// 无礼包掉落 → 不触发。
	sender2 := &farmOpSender{}
	api2 := &game.API{Sender: sender2, GID: 1}
	plain := &plantpb.FarmingReply{Results: []*plantpb.FarmingResult{
		{LandId: 1, Reward: &corepb.Item{Id: 1001, Count: 5}},
	}}
	maybeClaimDogSkillGiftsFromFarming(s, context.Background(), api2, plain)
	time.Sleep(50 * time.Millisecond)
	if sender2.called("ClaimSkillGifts") {
		t.Fatal("must not claim without gift drop")
	}

	// 单飞：领取进行中 → 本次跳过（不重复领取）。
	sender3 := &farmOpSender{}
	api3 := &game.API{Sender: sender3, GID: 1}
	s.dogGiftClaiming.Store(true)
	maybeClaimDogSkillGiftsFromFarming(s, context.Background(), api3, reply)
	time.Sleep(50 * time.Millisecond)
	if sender3.called("ClaimSkillGifts") {
		t.Fatal("must not double-claim while single-flight is held")
	}
}
