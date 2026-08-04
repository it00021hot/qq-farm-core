package migrate

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/MQEnergy/go-skeleton/internal/app/model"
	"github.com/MQEnergy/go-skeleton/internal/vars"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Models 需要自动迁移的业务表
func Models() []any {
	return []any{
		&sysTenant{},
		&sysAdminTenant{},
		&sysAdmin{},
		&sysRole{},
		&sysResource{},
		&sysRoleAuth{},
		&sysCasbinRule{},
		&attachment{},
	}
}

// AutoMigrate 根据 model 自动建表（字段注释来自 struct 的 comment: 标签）
func AutoMigrate(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if err := db.AutoMigrate(Models()...); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	slog.Info("Database auto migrate completed")
	return nil
}

// Seed 幂等写入初始化数据
func Seed(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if err := seedRoles(db); err != nil {
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

func seedRoles(db *gorm.DB) error {
	now := uint(time.Now().Unix())
	roles := []model.SysRole{
		{ID: 1, ParentID: 0, Level: 0, Name: "超级管理员", Code: "role_superadmin", Desc: "平台超级管理员", IsSys: 1, RoleType: vars.RoleTypePlatform, Status: 1, CreatedAt: now, UpdatedAt: now},
		{ID: 2, ParentID: 1, Level: 1, Name: "平台运维", Code: "role_platform_ops", Desc: "平台运维（可绑定多租户）", IsSys: 1, RoleType: vars.RoleTypePlatform, Status: 1, CreatedAt: now, UpdatedAt: now},
		{ID: 3, ParentID: 0, Level: 0, Name: "租户管理员", Code: "role_tenant_admin", Desc: "租户管理员", IsSys: 1, RoleType: vars.RoleTypeTenant, Status: 1, CreatedAt: now, UpdatedAt: now},
		{ID: 4, ParentID: 3, Level: 1, Name: "普通员工", Code: "role_tenant_staff", Desc: "租户普通员工", IsSys: 1, RoleType: vars.RoleTypeTenant, Status: 1, CreatedAt: now, UpdatedAt: now},
	}
	for _, role := range roles {
		var count int64
		if err := db.Model(&model.SysRole{}).Where("id = ?", role.ID).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&role).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedAdmin(db *gorm.DB) error {
	var count int64
	if err := db.Model(&model.SysAdmin{}).Where("account = ? AND tenant_id = ?", "admin", 0).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	now := uint(time.Now().Unix())
	admin := model.SysAdmin{
		ID:        1,
		UUID:      "1859045070490324993",
		TenantID:  0,
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

func seedCasbinRules(db *gorm.DB) error {
	strPtr := func(s string) *string { return &s }
	wanted := []model.SysCasbinRule{
		{Ptype: strPtr("p"), V0: strPtr("1"), V1: strPtr("/backend/auth/*"), V2: strPtr("GET,POST")},
		{Ptype: strPtr("p"), V0: strPtr("1"), V1: strPtr("/backend/attachment/*"), V2: strPtr("GET,POST")},
		{Ptype: strPtr("p"), V0: strPtr("1"), V1: strPtr("/backend/role/*"), V2: strPtr("GET,POST,PUT")},
		{Ptype: strPtr("p"), V0: strPtr("1"), V1: strPtr("/backend/menu/*"), V2: strPtr("GET,POST,PUT")},
		{Ptype: strPtr("p"), V0: strPtr("1"), V1: strPtr("/backend/permission/*"), V2: strPtr("GET,POST")},
		{Ptype: strPtr("p"), V0: strPtr("1"), V1: strPtr("/backend/admin/*"), V2: strPtr("GET,POST,PUT")},
		{Ptype: strPtr("p"), V0: strPtr("1"), V1: strPtr("/backend/tenant/*"), V2: strPtr("GET,POST,PUT")},
		{Ptype: strPtr("p"), V0: strPtr("1"), V1: strPtr("/backend/platform-user"), V2: strPtr("POST")},
		{Ptype: strPtr("p"), V0: strPtr("3"), V1: strPtr("/backend/auth/*"), V2: strPtr("GET,POST")},
		{Ptype: strPtr("p"), V0: strPtr("3"), V1: strPtr("/backend/admin/*"), V2: strPtr("GET,POST,PUT")},
		{Ptype: strPtr("p"), V0: strPtr("3"), V1: strPtr("/backend/role/assignable"), V2: strPtr("GET")},
		{Ptype: strPtr("p"), V0: strPtr("3"), V1: strPtr("/backend/attachment/*"), V2: strPtr("GET,POST")},
		{Ptype: strPtr("p"), V0: strPtr("4"), V1: strPtr("/backend/auth/*"), V2: strPtr("GET,POST")},
		{Ptype: strPtr("p"), V0: strPtr("4"), V1: strPtr("/backend/admin/list"), V2: strPtr("GET")},
		{Ptype: strPtr("p"), V0: strPtr("4"), V1: strPtr("/backend/role/assignable"), V2: strPtr("GET")},
	}

	for _, rule := range wanted {
		var count int64
		q := db.Model(&model.SysCasbinRule{}).Where("ptype = ? AND v0 = ? AND v1 = ?", *rule.Ptype, *rule.V0, *rule.V1)
		if err := q.Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		rule.V6 = ""
		rule.V7 = ""
		if err := db.Create(&rule).Error; err != nil {
			return err
		}
	}
	return nil
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
