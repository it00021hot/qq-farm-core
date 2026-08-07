package runtime

import (
	"context"
	"fmt"
	"time"

	"github.com/MQEnergy/go-skeleton/internal/farm/game"
	"github.com/MQEnergy/go-skeleton/internal/farm/proto/taskpb"
)

// DailyGiftCard is one card in the personal「每日礼包 & 任务」grid (bot getDailyGiftOverview gifts[]).
type DailyGiftCard struct {
	Key            string `json:"key"`
	Label          string `json:"label"`
	Enabled        bool   `json:"enabled"`
	DoneToday      bool   `json:"doneToday"`
	LastAt         int64  `json:"lastAt,omitempty"`
	CompletedCount int64  `json:"completedCount,omitempty"`
	TotalCount     int64  `json:"totalCount,omitempty"`
	Mode           string `json:"mode,omitempty"`
	CheckStatus    string `json:"checkStatus,omitempty"`
	CanShare       *bool  `json:"canShare,omitempty"`
	HasGift        *bool  `json:"hasGift,omitempty"`
	CanClaim       *bool  `json:"canClaim,omitempty"`
	HasCard        *bool  `json:"hasCard,omitempty"`
	HasClaimable   *bool  `json:"hasClaimable,omitempty"`
	Result         string `json:"result,omitempty"`
}

// GrowthTaskRow is one growth-task list row for the personal task panel.
type GrowthTaskRow struct {
	ID            int64  `json:"id"`
	Desc          string `json:"desc"`
	Progress      int64  `json:"progress"`
	TotalProgress int64  `json:"totalProgress"`
	IsClaimed     bool   `json:"isClaimed"`
	IsUnlocked    bool   `json:"isUnlocked"`
	IsCompleted   bool   `json:"isCompleted"`
}

// GrowthTaskOverview mirrors bot growth block.
type GrowthTaskOverview struct {
	Key            string          `json:"key"`
	Label          string          `json:"label"`
	DoneToday      bool            `json:"doneToday"`
	CompletedCount int             `json:"completedCount"`
	TotalCount     int             `json:"totalCount"`
	Tasks          []GrowthTaskRow `json:"tasks"`
}

// DailyGiftOverview mirrors bot GET /api/daily-gifts payload.
type DailyGiftOverview struct {
	Date   string              `json:"date"`
	Growth GrowthTaskOverview  `json:"growth"`
	Gifts  []DailyGiftCard     `json:"gifts"`
}

// DailyGiftOverview builds the personal task-panel payload (bot getDailyGiftOverview).
func (s *Session) DailyGiftOverview(ctx context.Context) (DailyGiftOverview, error) {
	s.farmOpMu.Lock()
	defer s.farmOpMu.Unlock()

	s.mu.Lock()
	api := s.gameAPI
	cfg := s.cfg.AccountConfig
	state := &s.dailyState
	s.mu.Unlock()
	if api == nil {
		return DailyGiftOverview{}, fmt.Errorf("farm session is not connected")
	}

	now := time.Now()
	today := localDateKey(now)
	out := DailyGiftOverview{
		Date:  today,
		Gifts: make([]DailyGiftCard, 0, 6),
	}

	taskCard, growth := buildTaskGiftCards(ctx, api)
	out.Growth = growth
	taskCard.Enabled = cfg.Automation.Task
	out.Gifts = append(out.Gifts, taskCard)

	state.mu.Lock()
	emailDone := state.emailDoneDateKey == today
	emailAt := unixMs(state.emailLastCheck)
	shareDone := state.shareDoneDateKey == today
	shareAt := unixMs(state.shareLastCheck)
	freeDone := state.freeGiftDoneDateKey == today
	freeAt := unixMs(state.freeGiftLastCheck)
	vipDone := state.vipDoneDateKey == today
	vipAt := unixMs(state.vipLastCheck)
	monthDone := state.monthCardDoneDateKey == today
	monthAt := unixMs(state.monthCardLastCheck)
	state.mu.Unlock()

	out.Gifts = append(out.Gifts,
		DailyGiftCard{Key: "email_rewards", Label: "邮箱奖励", Enabled: true, DoneToday: emailDone, LastAt: emailAt},
		DailyGiftCard{Key: "mall_free_gifts", Label: "商城免费礼包", Enabled: true, DoneToday: freeDone, LastAt: freeAt},
		DailyGiftCard{Key: "daily_share", Label: "分享礼包", Enabled: true, Mode: "auto_claim", DoneToday: shareDone, LastAt: shareAt},
	)

	vipCard := DailyGiftCard{Key: "vip_daily_gift", Label: "会员礼包", Enabled: true, DoneToday: vipDone, LastAt: vipAt}
	if status, err := api.GetDailyGiftStatus(ctx); err == nil && status != nil {
		hasGift := status.HasGift
		canClaim := status.CanClaim
		vipCard.HasGift = &hasGift
		vipCard.CanClaim = &canClaim
		if !hasGift {
			vipCard.DoneToday = false
		} else if canClaim {
			vipCard.DoneToday = false
		}
	}
	out.Gifts = append(out.Gifts, vipCard)

	monthCard := DailyGiftCard{Key: "month_card_gift", Label: "月卡礼包", Enabled: true, DoneToday: monthDone, LastAt: monthAt}
	if rep, err := api.GetMonthCardInfos(ctx); err == nil && rep != nil {
		hasCard := len(rep.Infos) > 0
		hasClaimable := false
		for _, info := range rep.Infos {
			if info != nil && info.CanClaim {
				hasClaimable = true
				break
			}
		}
		monthCard.HasCard = &hasCard
		monthCard.HasClaimable = &hasClaimable
		if !hasCard {
			monthCard.DoneToday = false
		} else if hasClaimable {
			monthCard.DoneToday = false
		}
	}
	out.Gifts = append(out.Gifts, monthCard)

	return out, nil
}

func buildTaskGiftCards(ctx context.Context, api *game.API) (DailyGiftCard, GrowthTaskOverview) {
	taskCard := DailyGiftCard{
		Key:        "task_claim",
		Label:      "每日任务",
		Enabled:    true,
		TotalCount: 3,
	}
	growth := GrowthTaskOverview{
		Key:   "growth_task",
		Label: "成长任务",
		Tasks: []GrowthTaskRow{},
	}

	reply, err := api.TaskInfo(ctx)
	if err != nil || reply == nil || reply.TaskInfo == nil {
		return taskCard, growth
	}
	normalized := normalizeTaskInfo(reply.TaskInfo)

	completedDaily := 0
	for _, t := range normalized.dailyTasks {
		if t != nil && t.TotalProgress > 0 && t.Progress >= t.TotalProgress {
			completedDaily++
		}
	}
	if completedDaily > 3 {
		completedDaily = 3
	}
	taskCard.CompletedCount = int64(completedDaily)
	taskCard.DoneToday = completedDaily >= 3

	rows := make([]GrowthTaskRow, 0, len(normalized.growthTasks))
	completedGrowth := 0
	for _, t := range normalized.growthTasks {
		if t == nil {
			continue
		}
		row := growthTaskRowFromPB(t)
		if row.IsCompleted {
			completedGrowth++
		}
		rows = append(rows, row)
	}
	growth.Tasks = rows
	growth.TotalCount = len(rows)
	growth.CompletedCount = completedGrowth
	growth.DoneToday = growth.TotalCount > 0 && growth.CompletedCount >= growth.TotalCount
	return taskCard, growth
}

func growthTaskRowFromPB(t *taskpb.Task) GrowthTaskRow {
	desc := t.Desc
	if desc == "" {
		desc = fmt.Sprintf("成长任务#%d", t.Id)
	}
	completed := t.TotalProgress > 0 && t.Progress >= t.TotalProgress
	return GrowthTaskRow{
		ID:            t.Id,
		Desc:          desc,
		Progress:      t.Progress,
		TotalProgress: t.TotalProgress,
		IsClaimed:     t.IsClaimed,
		IsUnlocked:    t.IsUnlocked,
		IsCompleted:   completed,
	}
}

func unixMs(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixMilli()
}
