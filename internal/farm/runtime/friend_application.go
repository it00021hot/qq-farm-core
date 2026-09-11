package runtime

// 好友申请自动接受过滤器（对齐 bot application-filter.ts + scheduler.processFriendApplications）。

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/farm/game"
	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/friendpb"
)

// applicationFilterConfig mirrors bot ApplicationFilterConfig.
type applicationFilterConfig struct {
	MinLevel             int64
	RequireOwnLevel      bool
	OwnLevel             int64
	HarvestStealEnabled  bool
	HarvestPart          int64
	StealPart            int64
}

// applicationDecision mirrors bot FilterDecision.
type applicationDecision struct {
	Accept bool
	Reason string
}

func isHarvestStealFilterEnabled(cfg applicationFilterConfig) bool {
	return cfg.HarvestStealEnabled && cfg.HarvestPart > 0
}

func effectiveMinLevel(cfg applicationFilterConfig) int64 {
	manual := cfg.MinLevel
	if manual < 0 {
		manual = 0
	}
	own := int64(0)
	if cfg.RequireOwnLevel && cfg.OwnLevel > 0 {
		own = cfg.OwnLevel
	}
	if own > manual {
		return own
	}
	return manual
}

// evaluateLevelFilter mirrors bot evaluateLevelFilter.
func evaluateLevelFilter(applicantLevel int64, cfg applicationFilterConfig) applicationDecision {
	minLevel := effectiveMinLevel(cfg)
	if minLevel <= 0 {
		return applicationDecision{Accept: true}
	}
	if applicantLevel < 0 {
		applicantLevel = 0
	}
	if applicantLevel >= minLevel {
		return applicationDecision{Accept: true}
	}
	parts := make([]string, 0, 2)
	if cfg.MinLevel > 0 {
		parts = append(parts, fmt.Sprintf("手动最低%d级", cfg.MinLevel))
	}
	if cfg.RequireOwnLevel {
		parts = append(parts, fmt.Sprintf("自己%d级", cfg.OwnLevel))
	}
	return applicationDecision{Accept: false, Reason: fmt.Sprintf("等级 %d < %d（%s）",
		applicantLevel, minLevel, strings.Join(parts, "，"))}
}

// evaluateHarvestStealFilter mirrors bot evaluateHarvestStealFilter.
func evaluateHarvestStealFilter(harvestCount, stealCount int64, cfg applicationFilterConfig) applicationDecision {
	if !isHarvestStealFilterEnabled(cfg) {
		return applicationDecision{Accept: true}
	}
	harvestPart := cfg.HarvestPart
	if harvestPart < 0 {
		harvestPart = 0
	}
	stealPart := cfg.StealPart
	if stealPart < 1 {
		stealPart = 1
	}
	if stealCount <= 0 {
		return applicationDecision{Accept: true}
	}
	if harvestCount*stealPart >= stealCount*harvestPart {
		return applicationDecision{Accept: true}
	}
	return applicationDecision{Accept: false, Reason: fmt.Sprintf("收偷比 %d:%d 低于 %d:%d",
		harvestCount, stealCount, harvestPart, stealPart)}
}

// applicationFilterConfigOf reads the account config into the filter shape.
func applicationFilterConfigOf(cfg logic.AccountConfig) applicationFilterConfig {
	return applicationFilterConfig{
		MinLevel:            int64(cfg.AutoAcceptFriendMinLevel),
		RequireOwnLevel:     cfg.AutoAcceptRequireOwnLevel,
		HarvestStealEnabled: cfg.AutoAcceptHarvestStealEnabled,
		HarvestPart:         int64(cfg.AutoAcceptHarvestStealHarvest),
		StealPart:           int64(cfg.AutoAcceptHarvestStealSteal),
	}
}

// careerHarvestSteal fetches lifetime harvest/steal counts for the ratio filter.
func careerHarvestSteal(ctx context.Context, api *game.API, gid int64) (harvest, steal int64, err error) {
	reply, err := api.CareerInfoGetForGID(ctx, gid)
	if err != nil {
		return 0, 0, err
	}
	return reply.TotalHarvestCount, reply.TotalStealCount, nil
}

// ProcessFriendApplications applies the filters and accepts/rejects in bulk
// (bot processFriendApplications). Returns accepted count.
func ProcessFriendApplications(ctx context.Context, s *Session, api *game.API, applications []*friendpb.Application) (int, error) {
	if len(applications) == 0 {
		return 0, nil
	}
	cfg := logic.AccountConfig{}
	ownLevel := int64(0)
	blacklist := map[int64]struct{}{}
	if s != nil {
		cfg = s.Config()
		ownLevel = s.Level()
		blacklist = makeIDSet(cfg.FriendBlacklist)
	}
	filter := applicationFilterConfigOf(cfg)
	filter.OwnLevel = ownLevel
	checkRatio := isHarvestStealFilterEnabled(filter)

	toAccept := make([]int64, 0, len(applications))
	type rejection struct {
		gid    int64
		name   string
		reason string
	}
	toReject := make([]rejection, 0)

	for i, app := range applications {
		if app == nil || app.Gid <= 0 {
			continue
		}
		name := strings.TrimSpace(app.Name)
		if name == "" {
			name = fmt.Sprintf("GID:%d", app.Gid)
		}
		if _, blocked := blacklist[app.Gid]; blocked {
			toReject = append(toReject, rejection{app.Gid, name, "已在本地黑名单"})
			continue
		}
		if decision := evaluateLevelFilter(app.Level, filter); !decision.Accept {
			toReject = append(toReject, rejection{app.Gid, name, decision.Reason})
			continue
		}
		if !checkRatio {
			toAccept = append(toAccept, app.Gid)
			continue
		}
		harvest, steal, err := careerHarvestSteal(ctx, api, app.Gid)
		if err != nil {
			slog.Warn("生涯查询失败，暂不处理", "account", sessionID(s), "friend", name, "err", err)
			continue
		}
		if decision := evaluateHarvestStealFilter(harvest, steal, filter); !decision.Accept {
			toReject = append(toReject, rejection{app.Gid, name, decision.Reason})
		} else {
			toAccept = append(toAccept, app.Gid)
		}
		if i < len(applications)-1 && checkRatio {
			_ = waitFarmDelay(ctx, time.Duration(150+rand.Intn(150))*time.Millisecond)
		}
	}

	for _, item := range toReject {
		slog.Info("拒绝好友申请", "account", sessionID(s), "friend", item.name, "reason", item.reason)
	}
	if len(toReject) > 0 {
		gids := make([]int64, 0, len(toReject))
		for _, item := range toReject {
			gids = append(gids, item.gid)
		}
		if _, err := api.RejectFriends(ctx, gids); err != nil {
			slog.Warn("拒绝好友申请失败", "account", sessionID(s), "err", err)
		} else {
			slog.Info("已拒绝好友申请", "account", sessionID(s), "count", len(gids))
		}
	}
	if len(toAccept) == 0 {
		return 0, nil
	}
	reply, err := api.AcceptFriends(ctx, toAccept)
	if err != nil {
		return 0, err
	}
	accepted := len(reply.GetFriends())
	if accepted > 0 {
		names := make([]string, 0, accepted)
		for _, f := range reply.GetFriends() {
			if f == nil {
				continue
			}
			name := f.Remark
			if name == "" {
				name = f.Name
			}
			if name == "" {
				name = fmt.Sprintf("GID:%d", f.Gid)
			}
			names = append(names, name)
		}
		slog.Info("已同意好友申请", "account", sessionID(s), "count", accepted, "friends", strings.Join(names, ", "))
	}
	return accepted, nil
}

// AcceptPendingFriends fetches pending applications and runs the filters
// (bot checkAndAcceptApplications). Gate: automation.friend_auto_accept.
func AcceptPendingFriends(ctx context.Context, s *Session, api *game.API) (accepted int, err error) {
	if api == nil {
		return 0, fmt.Errorf("farm API is unavailable")
	}
	if s != nil {
		cfg := s.Config()
		if !cfg.Automation.FriendAutoAccept {
			return 0, nil
		}
	}
	reply, err := api.GetApplications(ctx)
	if err != nil {
		return 0, err
	}
	if len(reply.Applications) == 0 {
		return 0, nil
	}
	return ProcessFriendApplications(ctx, s, api, reply.Applications)
}
