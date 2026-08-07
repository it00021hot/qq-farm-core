// Package stats persists per-day farm operation counters.
package stats

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/MQEnergy/go-skeleton/internal/app/model"
	"github.com/MQEnergy/go-skeleton/internal/vars"
	"github.com/MQEnergy/go-skeleton/pkg/tenant"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// RecordExpGold upserts today's cn_farm_stats row and increments exp/gold columns.
func RecordExpGold(accountID, tenantID uint64, expDelta, goldDelta int64) {
	if accountID == 0 {
		return
	}
	if expDelta <= 0 && goldDelta <= 0 {
		return
	}
	today := time.Now().Format("2006-01-02")
	now := uint(time.Now().Unix())

	row := model.FarmStats{
		TenantID:  tenantID,
		AccountID: accountID,
		StatDate:  today,
		ExtraJSON: "{}",
		CreatedAt: now,
		UpdatedAt: now,
	}
	assigns := map[string]any{"updated_at": now}
	if expDelta > 0 {
		row.Exp = expDelta
		assigns["exp"] = gorm.Expr("exp + ?", expDelta)
	}
	if goldDelta > 0 {
		row.Gold = goldDelta
		assigns["gold"] = gorm.Expr("gold + ?", goldDelta)
	}

	db := tenant.Global(vars.DB, context.Background())
	if db == nil {
		return
	}
	if err := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "tenant_id"},
			{Name: "account_id"},
			{Name: "stat_date"},
		},
		DoUpdates: clause.Assignments(assigns),
	}).Create(&row).Error; err != nil {
		slog.Warn("farm stats RecordExpGold failed", "account", accountID, "exp", expDelta, "gold", goldDelta, "err", err)
	}
}

// RecordOp upserts today's cn_farm_stats row and increments the op counter.
// Known ops: harvest, plant, farming, steal, help, sell, upgrade, fertilize, taskClaim, levelUp, ...
// Unknown ops are stored in extra_json.operations.
func RecordOp(accountID, tenantID uint64, op string, count int) {
	if accountID == 0 || count <= 0 || op == "" {
		return
	}
	today := time.Now().Format("2006-01-02")
	now := uint(time.Now().Unix())

	row := model.FarmStats{
		TenantID:  tenantID,
		AccountID: accountID,
		StatDate:  today,
		ExtraJSON: "{}",
		CreatedAt: now,
		UpdatedAt: now,
	}

	db := tenant.Global(vars.DB, context.Background())
	if db == nil {
		return
	}

	column, extraKey := opColumn(op)
	if column != "" {
		switch column {
		case "harvest_count":
			row.HarvestCount = int64(count)
		case "plant_count":
			row.PlantCount = int64(count)
		case "steal_count":
			row.StealCount = int64(count)
		case "help_count":
			row.HelpCount = int64(count)
		}
		if err := db.Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "tenant_id"},
				{Name: "account_id"},
				{Name: "stat_date"},
			},
			DoUpdates: clause.Assignments(map[string]any{
				column:       gorm.Expr(column+" + ?", count),
				"updated_at": now,
			}),
		}).Create(&row).Error; err != nil {
			slog.Warn("farm stats RecordOp failed", "account", accountID, "op", op, "err", err)
		}
		return
	}

	if extraKey == "" {
		extraKey = op
	}
	var existing model.FarmStats
	err := db.Where("tenant_id = ? AND account_id = ? AND stat_date = ?", tenantID, accountID, today).
		First(&existing).Error
	if err != nil {
		if err := db.Create(&row).Error; err != nil {
			slog.Warn("farm stats create failed", "account", accountID, "op", op, "err", err)
			return
		}
		existing = row
	}
	extra := map[string]int{}
	if existing.ExtraJSON != "" && existing.ExtraJSON != "{}" {
		var parsed struct {
			Operations map[string]int `json:"operations"`
		}
		if json.Unmarshal([]byte(existing.ExtraJSON), &parsed) == nil && parsed.Operations != nil {
			extra = parsed.Operations
		}
	}
	extra[extraKey] += count
	payload, _ := json.Marshal(map[string]any{"operations": extra})
	if err := db.Model(&model.FarmStats{}).
		Where("id = ?", existing.ID).
		Updates(map[string]any{"extra_json": string(payload), "updated_at": now}).Error; err != nil {
		slog.Warn("farm stats extra_json update failed", "account", accountID, "op", op, "err", err)
	}
}

func opColumn(op string) (column, extraKey string) {
	switch op {
	case "harvest":
		return "harvest_count", ""
	case "plant":
		return "plant_count", ""
	case "steal":
		return "steal_count", ""
	case "help", "helpFarming":
		return "help_count", ""
	case "farming", "sell", "upgrade", "fertilize", "taskClaim", "levelUp":
		return "", op
	default:
		return "", op
	}
}

// TodayOperations returns today's op counters for the dashboard (bot status.operations).
func TodayOperations(accountID, tenantID uint64) map[string]int {
	out := map[string]int{
		"harvest": 0, "farming": 0, "fertilize": 0, "plant": 0,
		"steal": 0, "helpFarming": 0, "taskClaim": 0, "sell": 0,
		"upgrade": 0, "levelUp": 0,
	}
	if accountID == 0 {
		return out
	}
	today := time.Now().Format("2006-01-02")
	db := tenant.Global(vars.DB, context.Background())
	if db == nil {
		return out
	}
	var row model.FarmStats
	q := db.Where("account_id = ? AND stat_date = ?", accountID, today)
	if tenantID > 0 {
		q = q.Where("tenant_id = ?", tenantID)
	}
	if err := q.First(&row).Error; err != nil {
		return out
	}
	out["harvest"] = int(row.HarvestCount)
	out["plant"] = int(row.PlantCount)
	out["steal"] = int(row.StealCount)
	out["helpFarming"] = int(row.HelpCount)
	if row.ExtraJSON != "" && row.ExtraJSON != "{}" {
		var parsed struct {
			Operations map[string]int `json:"operations"`
		}
		if json.Unmarshal([]byte(row.ExtraJSON), &parsed) == nil && parsed.Operations != nil {
			for k, v := range parsed.Operations {
				if k == "help" {
					out["helpFarming"] += v
					continue
				}
				out[k] += v
			}
		}
	}
	return out
}

// TodayExpGold returns today's recorded exp/gold deltas from cn_farm_stats.
func TodayExpGold(accountID, tenantID uint64) (exp, gold int64) {
	if accountID == 0 {
		return 0, 0
	}
	today := time.Now().Format("2006-01-02")
	db := tenant.Global(vars.DB, context.Background())
	if db == nil {
		return 0, 0
	}
	var row model.FarmStats
	q := db.Where("account_id = ? AND stat_date = ?", accountID, today)
	if tenantID > 0 {
		q = q.Where("tenant_id = ?", tenantID)
	}
	if err := q.First(&row).Error; err != nil {
		return 0, 0
	}
	return row.Exp, row.Gold
}
