package migrate

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/MQEnergy/go-skeleton/internal/app/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Models 需要自动迁移的业务表（与历史 SQL 一致，PG 兼容类型）
func Models() []any {
	return []any{
		&sysAdmin{},
		&sysRole{},
		&sysResource{},
		&sysRoleAuth{},
		&sysCasbinRule{},
		&attachment{},
	}
}

// AutoMigrate 根据 model 自动建表/补齐字段
func AutoMigrate(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if err := db.AutoMigrate(Models()...); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	if err := ensureIndexes(db); err != nil {
		return fmt.Errorf("ensure indexes: %w", err)
	}
	slog.Info("Database auto migrate completed")
	return nil
}

func ensureIndexes(db *gorm.DB) error {
	stmts := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_casbin_rule ON cn_sys_casbin_rule (ptype, v0, v1, v2, v3, v4, v5)`,
	}
	for _, sql := range stmts {
		if err := db.Exec(sql).Error; err != nil {
			return err
		}
	}
	return nil
}

// Seed 幂等写入初始化数据（仅在对应表为空或关键记录不存在时插入）
func Seed(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if err := seedRole(db); err != nil {
		return err
	}
	if err := seedAdmin(db); err != nil {
		return err
	}
	if err := seedCasbinRules(db); err != nil {
		return err
	}
	slog.Info("Database seed completed")
	return nil
}

func seedRole(db *gorm.DB) error {
	var count int64
	if err := db.Model(&model.SysRole{}).Where("id = ?", 1).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	now := uint(time.Now().Unix())
	role := model.SysRole{
		ID:        1,
		MchID:     0,
		Name:      "超级管理员",
		Code:      "role_superadmin",
		Desc:      "超级管理员",
		IsSys:     1,
		RoleType:  1,
		Status:    1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return db.Clauses(clause.OnConflict{DoNothing: true}).Create(&role).Error
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
	// 与历史 SQL 保持一致：account=admin，可用原密码登录
	admin := model.SysAdmin{
		ID:        1,
		UUID:      "1859045070490324993",
		DeptID:    1,
		NickName:  "admin",
		RealName:  "admin",
		Desc:      "",
		Gender:    2,
		Account:   "admin",
		Password:  "acce31f66e319f31f7b3c603cb76dd3ee1abd6bde53fcecef7fc61a35186138f",
		Phone:     "13595026776",
		Email:     "",
		Avatar:    "",
		Salt:      "02e443a7-9ff6-8b81-1823-b141b318",
		RoleIds:   "1",
		Type:      1,
		IsMain:    1,
		IsAuth:    2,
		MfaSecret: "FWQQ5ESYDYXGV5HAXDFA27P3L4JU7DRQ",
		Status:    1,
		CreatedBy: "",
		CreatedAt: now,
		UpdatedBy: "",
		UpdatedAt: now,
	}
	return db.Clauses(clause.OnConflict{DoNothing: true}).Create(&admin).Error
}

func seedCasbinRules(db *gorm.DB) error {
	var count int64
	if err := db.Model(&model.SysCasbinRule{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	strPtr := func(s string) *string { return &s }
	rules := []model.SysCasbinRule{
		{Ptype: strPtr("p"), V0: strPtr("1"), V1: strPtr("/backend/auth/*"), V2: strPtr("GET,POST")},
		{Ptype: strPtr("p"), V0: strPtr("1"), V1: strPtr("/backend/attachment/*"), V2: strPtr("GET,POST")},
		{Ptype: strPtr("p"), V0: strPtr("1"), V1: strPtr("/backend/role/*"), V2: strPtr("GET,POST")},
		{Ptype: strPtr("p"), V0: strPtr("1"), V1: strPtr("/backend/menu/*"), V2: strPtr("GET,POST")},
		{Ptype: strPtr("p"), V0: strPtr("1"), V1: strPtr("/backend/permission/*"), V2: strPtr("GET,POST")},
		{Ptype: strPtr("p"), V0: strPtr("1"), V1: strPtr("/backend/admin/*"), V2: strPtr("GET,POST")},
	}
	for i := range rules {
		rules[i].V6 = ""
		rules[i].V7 = ""
	}
	return db.Create(&rules).Error
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
