package status

import (
	"time"

	"github.com/it00021hot/qq-farm-core/internal/app/model"
	"github.com/it00021hot/qq-farm-core/internal/app/pkg/pagination"
	"github.com/it00021hot/qq-farm-core/internal/app/service"
	farmruntime "github.com/it00021hot/qq-farm-core/internal/farm/runtime"
	farmstats "github.com/it00021hot/qq-farm-core/internal/farm/stats"
	farmtypes "github.com/it00021hot/qq-farm-core/internal/types/farm"
	"github.com/it00021hot/qq-farm-core/internal/vars"
	"github.com/gofiber/fiber/v3"
)

type Service struct {
	service.Service
}

var Status = &Service{}

func (s *Service) Detail(ctx fiber.Ctx, req farmtypes.StatusDetailReq) (map[string]any, error) {
	db := vars.DB
	var acc model.FarmAccount
	if err := db.Where("id = ?", req.AccountID).First(&acc).Error; err != nil {
		return nil, err
	}

	run, errMsg := farmruntime.Default.GetStatus(req.AccountID)
	dayExp, dayGold := farmstats.TodayExpGold(req.AccountID, 0)
	out := map[string]any{
		"accountId":         req.AccountID,
		"runStatus":         run,
		"online":            run == farmruntime.RunRunning,
		"lastError":         errMsg,
		"nick":              acc.Name,
		"avatar":            acc.Avatar,
		"updatedAt":         time.Now().Unix(),
		"account":           acc,
		"error":             errMsg, // compat
		"operations":        farmstats.TodayOperations(req.AccountID, 0),
		"sessionExpGained":  dayExp,
		"sessionGoldGained": dayGold,
		"todayExp":          dayExp,
		"todayGold":         dayGold,
	}

	if sess, ok := farmruntime.Default.Session(req.AccountID); ok {
		snap := sess.Snapshot()
		out["runStatus"] = snap.RunStatus
		out["online"] = snap.Online
		out["level"] = snap.Level
		out["exp"] = snap.Exp
		out["gold"] = snap.Gold
		out["landCount"] = snap.LandCount
		out["friendCount"] = snap.FriendCount
		out["gid"] = snap.GID
		out["updatedAt"] = snap.UpdatedAt
		out["nextChecks"] = snap.NextChecks
		out["uptime"] = snap.Uptime
		out["sessionExpGained"] = snap.SessionExpGained
		out["sessionGoldGained"] = snap.SessionGoldGained
		out["levelProgress"] = snap.LevelProgress
		out["operations"] = snap.Operations
		if snap.Nick != "" {
			out["nick"] = snap.Nick
		}
		if snap.Avatar != "" {
			out["avatar"] = snap.Avatar
		}
		if snap.LastError != "" {
			out["lastError"] = snap.LastError
			out["error"] = snap.LastError
		}
		// Refresh land count when online and still zero (first paint after login).
		if snap.Online && snap.LandCount == 0 {
			if lands, err := sess.GetLands(ctx.Context()); err == nil {
				out["landCount"] = len(lands)
			}
		}
		// Friend count: prefer DB cache to avoid slow live friend RPC on every refresh.
		if snap.FriendCount == 0 {
			var n int64
			_ = db.Model(&model.FarmFriendGid{}).Where("account_id = ?", req.AccountID).Count(&n).Error
			if n > 0 {
				out["friendCount"] = int(n)
				sess.SetFriendCount(int(n))
			}
		}
	}
	return out, nil
}

func (s *Service) List(ctx fiber.Ctx, req farmtypes.StatusListReq) (map[string]any, error) {
	db := vars.DB.Model(&model.FarmAccount{})
	if req.Keyword != "" {
		kw := "%" + req.Keyword + "%"
		db = db.Where("name LIKE ? OR code LIKE ?", kw, kw)
	}
	var total int64
	_ = db.Count(&total).Error
	page := pagination.New().ParsePage(req.Current, req.Size)
	page.Total = total
	page.GetLastPage()
	var list []model.FarmAccount
	_ = db.Order("id desc").Offset(page.GetOffset()).Limit(page.GetLimit()).Find(&list).Error
	items := make([]map[string]any, 0, len(list))
	for _, a := range list {
		detail, err := s.Detail(ctx, farmtypes.StatusDetailReq{AccountID: a.ID})
		if err != nil {
			run, errMsg := farmruntime.Default.GetStatus(a.ID)
			items = append(items, map[string]any{
				"accountId": a.ID,
				"account":   a,
				"runStatus": run,
				"error":     errMsg,
				"lastError": errMsg,
			})
			continue
		}
		items = append(items, detail)
	}
	return map[string]any{"list": items, "total": total, "page": page}, nil
}
