package migrate

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/app/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Models 需要自动迁移的业务表
func Models() []any {
	return []any{
		&sysAdmin{},
		&farmAccount{},
		&farmAccountConfig{},
		&farmFriendGid{},
		&farmStats{},
		&farmInteractLog{},
		&farmSystemConfig{},
		&farmGameConfig{},
		&farmActivityState{},
	}
}

// droppedRBACTables 已拆除的平台 RBAC 表（启动时 DROP）
var droppedRBACTables = []string{
	"cn_sys_resource",
	"cn_sys_role_auth",
	"cn_sys_casbin_rule",
	"cn_sys_role",
}

// AutoMigrate 根据 model 自动建表（字段注释来自 struct 的 comment: 标签）
func AutoMigrate(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if err := dropLegacyRBACTables(db); err != nil {
		return err
	}
	if err := db.AutoMigrate(Models()...); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	slog.Info("Database auto migrate completed")
	return nil
}

func dropLegacyRBACTables(db *gorm.DB) error {
	for _, table := range droppedRBACTables {
		if !db.Migrator().HasTable(table) {
			continue
		}
		if err := db.Migrator().DropTable(table); err != nil {
			return fmt.Errorf("drop table %s: %w", table, err)
		}
		slog.Info("Dropped legacy RBAC table", "table", table)
	}
	return nil
}

// Seed 幂等写入初始化数据（单机：仅 admin 账号；菜单由前端静态路由提供）
func Seed(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if err := seedAdmin(db); err != nil {
		return err
	}
	if err := syncPostgresSequences(db); err != nil {
		return err
	}
	slog.Info("Database seed completed")
	return nil
}

func syncPostgresSequences(db *gorm.DB) error {
	if db == nil || db.Dialector == nil {
		return nil
	}
	name := strings.ToLower(db.Dialector.Name())
	if name != "postgres" && name != "postgresql" {
		return nil
	}
	tables := []string{
		"cn_sys_admin",
		"cn_farm_account",
		"cn_farm_account_config",
		"cn_farm_friend_gid",
		"cn_farm_stats",
		"cn_farm_interact_log",
		"cn_farm_system_config",
		"cn_farm_game_config",
		"cn_farm_activity_state",
	}
	for _, table := range tables {
		sql := fmt.Sprintf(
			`SELECT setval(pg_get_serial_sequence('%s', 'id'), COALESCE((SELECT MAX(id) FROM %s), 1), true)`,
			table, table,
		)
		if err := db.Exec(sql).Error; err != nil {
			if strings.Contains(err.Error(), "pg_get_serial_sequence") ||
				strings.Contains(err.Error(), "null value") ||
				strings.Contains(strings.ToLower(err.Error()), "does not exist") {
				continue
			}
			return fmt.Errorf("sync sequence %s: %w", table, err)
		}
	}
	return nil
}

func seedAdmin(db *gorm.DB) error {
	var count int64
	if err := db.Model(&model.SysAdmin{}).Where("account = ?", "admin").Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	now := uint(time.Now().Unix())
	admin := model.SysAdmin{
		ID:        1,
		UUID:      "1859045070490324993",
		NickName:  "admin",
		RealName:  "admin",
		Account:   "admin",
		Password:  "acce31f66e319f31f7b3c603cb76dd3ee1abd6bde53fcecef7fc61a35186138f",
		Phone:     "13595026776",
		Email:     "",
		Salt:      "02e443a7-9ff6-8b81-1823-b141b318",
		RoleIds:   "1",
		Status:    1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return db.Clauses(clause.OnConflict{DoNothing: true}).Create(&admin).Error
}

// Run 执行迁移 + 初始化数据
func Run(db *gorm.DB) error {
	if err := AutoMigrate(db); err != nil {
		return err
	}
	return Seed(db)
}

// ReplaceDBName 将 postgres DSN 中的 dbname 替换为指定库名
func ReplaceDBName(dsn, dbName string) string {
	parts := strings.Fields(dsn)
	out := make([]string, 0, len(parts))
	replaced := false
	for _, p := range parts {
		if strings.HasPrefix(p, "dbname=") {
			out = append(out, "dbname="+dbName)
			replaced = true
			continue
		}
		out = append(out, p)
	}
	if !replaced {
		out = append(out, "dbname="+dbName)
	}
	return strings.Join(out, " ")
}

// ParseDBName 从 DSN 解析 dbname
func ParseDBName(dsn string) string {
	for _, p := range strings.Fields(dsn) {
		if strings.HasPrefix(p, "dbname=") {
			return strings.TrimPrefix(p, "dbname=")
		}
	}
	return ""
}
