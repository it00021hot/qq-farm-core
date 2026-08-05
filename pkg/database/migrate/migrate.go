package migrate

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/MQEnergy/go-skeleton/internal/app/model"
	"github.com/MQEnergy/go-skeleton/internal/vars"
	"github.com/MQEnergy/go-skeleton/pkg/rbac"
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
	if err := seedResources(db); err != nil {
		return err
	}
	if err := seedRoleAuth(db); err != nil {
		return err
	}
	if err := syncSeededRoleCasbin(db); err != nil {
		return err
	}
	// 种子数据常带显式主键，需把 SERIAL 序列对齐到 MAX(id)，否则后续 Create 会撞 cn_*_pkey
	if err := syncPostgresSequences(db); err != nil {
		return err
	}
	slog.Info("Database seed completed")
	return nil
}

func syncPostgresSequences(db *gorm.DB) error {
	tables := []string{
		"cn_sys_admin",
		"cn_sys_role",
		"cn_sys_tenant",
		"cn_sys_admin_tenant",
		"cn_sys_resource",
		"cn_sys_role_auth",
		"cn_sys_casbin_rule",
		"cn_attachment",
	}
	for _, table := range tables {
		sql := fmt.Sprintf(
			`SELECT setval(pg_get_serial_sequence('%s', 'id'), COALESCE((SELECT MAX(id) FROM %s), 1), true)`,
			table, table,
		)
		if err := db.Exec(sql).Error; err != nil {
			// 表尚未建序列时忽略（非 PG / 无 id 序列）
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
	// 清理旧 /backend 通配策略；非超管策略由 SyncRoleCasbin 按资源重建；超管走旁路
	return db.Where("ptype = ? AND v1 LIKE ?", "p", "/backend%").Delete(&model.SysCasbinRule{}).Error
}

func seedResources(db *gorm.DB) error {
	now := uint64(time.Now().Unix())
	show, hide := uint8(1), uint8(2)
	// 精确 b_url，禁止 /*；hide_in_menu：1显示 2隐藏
	resources := []model.SysResource{
		{ID: 1, Name: "平台管理", Alias: "platform", Desc: "平台配置目录", Icon: "mdi:office-building-cog", ParentID: 0, Path: "1", ResourceType: vars.ResourceTypeDir, HideInMenu: show, Status: 1, SortOrder: 10, CreatedAt: now, UpdatedAt: now},
		{ID: 20, Name: "系统管理", Alias: "system", Desc: "系统业务目录", Icon: "mdi:cog-outline", ParentID: 0, Path: "20", ResourceType: vars.ResourceTypeDir, HideInMenu: show, Status: 1, SortOrder: 20, CreatedAt: now, UpdatedAt: now},

		{ID: 2, Name: "用户管理", Alias: "system_admin", Desc: "用户管理", FURL: "/system/admin", Icon: "mdi:account", ParentID: 20, Path: "20-2", ResourceType: vars.ResourceTypeMenu, HideInMenu: show, Status: 1, SortOrder: 10, CreatedAt: now, UpdatedAt: now},
		{ID: 4, Name: "用户列表", Alias: "admin:list", Desc: "用户列表", BURL: "/system/admin/list", Methods: "GET", ParentID: 2, Path: "20-2-4", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 10, CreatedAt: now, UpdatedAt: now},
		{ID: 21, Name: "用户新增", Alias: "admin:add", Desc: "创建用户", BURL: "/system/admin/add", Methods: "POST", ParentID: 2, Path: "20-2-21", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 20, CreatedAt: now, UpdatedAt: now},
		{ID: 22, Name: "用户修改", Alias: "admin:modify", Desc: "更新用户", BURL: "/system/admin/modify", Methods: "POST", ParentID: 2, Path: "20-2-22", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 30, CreatedAt: now, UpdatedAt: now},
		{ID: 23, Name: "用户启停", Alias: "admin:status", Desc: "启停用户", BURL: "/system/admin/status", Methods: "POST", ParentID: 2, Path: "20-2-23", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 40, CreatedAt: now, UpdatedAt: now},
		{ID: 13, Name: "平台用户新增", Alias: "platform-user:add", Desc: "创建平台用户", BURL: "/system/platform-user/add", Methods: "POST", ParentID: 2, Path: "20-2-13", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 50, CreatedAt: now, UpdatedAt: now},

		{ID: 14, Name: "附件管理", Alias: "system_attachment", Desc: "附件", FURL: "/system/attachment", Icon: "mdi:paperclip", ParentID: 20, Path: "20-14", ResourceType: vars.ResourceTypeMenu, HideInMenu: show, Status: 1, SortOrder: 20, CreatedAt: now, UpdatedAt: now},
		{ID: 24, Name: "附件列表", Alias: "attachment:list", BURL: "/system/attachment/list", Methods: "GET", ParentID: 14, Path: "20-14-24", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 10, CreatedAt: now, UpdatedAt: now},
		{ID: 25, Name: "附件详情", Alias: "attachment:detail", BURL: "/system/attachment/detail", Methods: "GET", ParentID: 14, Path: "20-14-25", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 20, CreatedAt: now, UpdatedAt: now},
		{ID: 26, Name: "附件上传", Alias: "attachment:upload", BURL: "/system/attachment/upload", Methods: "POST", ParentID: 14, Path: "20-14-26", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 30, CreatedAt: now, UpdatedAt: now},
		{ID: 27, Name: "附件访问址", Alias: "attachment:access-url", BURL: "/system/attachment/access-url", Methods: "POST", ParentID: 14, Path: "20-14-27", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 40, CreatedAt: now, UpdatedAt: now},
		{ID: 28, Name: "附件启停", Alias: "attachment:status", BURL: "/system/attachment/status", Methods: "POST", ParentID: 14, Path: "20-14-28", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 50, CreatedAt: now, UpdatedAt: now},
		{ID: 29, Name: "附件删除", Alias: "attachment:delete", BURL: "/system/attachment/delete", Methods: "POST", ParentID: 14, Path: "20-14-29", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 60, CreatedAt: now, UpdatedAt: now},

		{ID: 11, Name: "租户", Alias: "system_tenant", Desc: "租户管理", FURL: "/system/tenant", Icon: "mdi:office-building", ParentID: 1, Path: "1-11", ResourceType: vars.ResourceTypeMenu, HideInMenu: show, Status: 1, SortOrder: 10, CreatedAt: now, UpdatedAt: now},
		{ID: 30, Name: "租户列表", Alias: "tenant:list", BURL: "/platform/tenant/list", Methods: "GET", ParentID: 11, Path: "1-11-30", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 10, CreatedAt: now, UpdatedAt: now},
		{ID: 31, Name: "租户详情", Alias: "tenant:detail", BURL: "/platform/tenant/detail", Methods: "GET", ParentID: 11, Path: "1-11-31", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 20, CreatedAt: now, UpdatedAt: now},
		{ID: 32, Name: "租户新增", Alias: "tenant:add", BURL: "/platform/tenant/add", Methods: "POST", ParentID: 11, Path: "1-11-32", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 30, CreatedAt: now, UpdatedAt: now},
		{ID: 33, Name: "租户修改", Alias: "tenant:modify", BURL: "/platform/tenant/modify", Methods: "POST", ParentID: 11, Path: "1-11-33", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 40, CreatedAt: now, UpdatedAt: now},
		{ID: 34, Name: "租户启停", Alias: "tenant:status", BURL: "/platform/tenant/status", Methods: "POST", ParentID: 11, Path: "1-11-34", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 50, CreatedAt: now, UpdatedAt: now},
		{ID: 35, Name: "租户绑定", Alias: "tenant:bind", BURL: "/platform/tenant/bind", Methods: "POST", ParentID: 11, Path: "1-11-35", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 60, CreatedAt: now, UpdatedAt: now},

		{ID: 5, Name: "角色管理", Alias: "system_role", Desc: "角色管理", FURL: "/system/role", Icon: "mdi:account-group", ParentID: 1, Path: "1-5", ResourceType: vars.ResourceTypeMenu, HideInMenu: show, Status: 1, SortOrder: 20, CreatedAt: now, UpdatedAt: now},
		{ID: 6, Name: "可分配角色", Alias: "role:assignable", BURL: "/platform/role/assignable", Methods: "GET", ParentID: 5, Path: "1-5-6", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 10, CreatedAt: now, UpdatedAt: now},
		{ID: 7, Name: "角色授权", Alias: "role:auth", BURL: "/platform/role/auth", Methods: "GET,POST", ParentID: 5, Path: "1-5-7", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 20, CreatedAt: now, UpdatedAt: now},
		{ID: 8, Name: "角色树", Alias: "role:tree", BURL: "/platform/role/tree", Methods: "GET", ParentID: 5, Path: "1-5-8", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 30, CreatedAt: now, UpdatedAt: now},
		{ID: 36, Name: "角色新增", Alias: "role:add", BURL: "/platform/role/add", Methods: "POST", ParentID: 5, Path: "1-5-36", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 40, CreatedAt: now, UpdatedAt: now},
		{ID: 37, Name: "角色修改", Alias: "role:modify", BURL: "/platform/role/modify", Methods: "POST", ParentID: 5, Path: "1-5-37", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 50, CreatedAt: now, UpdatedAt: now},
		{ID: 38, Name: "角色删除", Alias: "role:delete", BURL: "/platform/role/delete", Methods: "POST", ParentID: 5, Path: "1-5-38", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 60, CreatedAt: now, UpdatedAt: now},

		{ID: 9, Name: "菜单管理", Alias: "system_menu", Desc: "菜单管理", FURL: "/system/menu", Icon: "mdi:menu", ParentID: 1, Path: "1-9", ResourceType: vars.ResourceTypeMenu, HideInMenu: show, Status: 1, SortOrder: 30, CreatedAt: now, UpdatedAt: now},
		{ID: 39, Name: "菜单树", Alias: "menu:tree", BURL: "/platform/menu/tree", Methods: "GET", ParentID: 9, Path: "1-9-39", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 10, CreatedAt: now, UpdatedAt: now},
		{ID: 40, Name: "菜单新增", Alias: "menu:add", BURL: "/platform/menu/add", Methods: "POST", ParentID: 9, Path: "1-9-40", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 20, CreatedAt: now, UpdatedAt: now},
		{ID: 41, Name: "菜单修改", Alias: "menu:modify", BURL: "/platform/menu/modify", Methods: "POST", ParentID: 9, Path: "1-9-41", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 30, CreatedAt: now, UpdatedAt: now},
		{ID: 42, Name: "菜单删除", Alias: "menu:delete", BURL: "/platform/menu/delete", Methods: "POST", ParentID: 9, Path: "1-9-42", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 40, CreatedAt: now, UpdatedAt: now},

		{ID: 18, Name: "权限管理", Alias: "system_permission", Desc: "权限运维", FURL: "/system/permission", Icon: "mdi:shield-check", ParentID: 1, Path: "1-18", ResourceType: vars.ResourceTypeMenu, HideInMenu: show, Status: 1, SortOrder: 40, CreatedAt: now, UpdatedAt: now},
		{ID: 43, Name: "API列表", Alias: "permission:apis", BURL: "/platform/permission/apis", Methods: "GET", ParentID: 18, Path: "1-18-43", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 10, CreatedAt: now, UpdatedAt: now},
		{ID: 44, Name: "角色策略", Alias: "permission:role-policies", BURL: "/platform/permission/role-policies", Methods: "GET", ParentID: 18, Path: "1-18-44", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 20, CreatedAt: now, UpdatedAt: now},
		{ID: 45, Name: "重载权限", Alias: "permission:reload", BURL: "/platform/permission/reload", Methods: "POST", ParentID: 18, Path: "1-18-45", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 30, CreatedAt: now, UpdatedAt: now},

		{ID: 16, Name: "鉴权", Alias: "system_auth", Desc: "鉴权相关", Icon: "mdi:lock", ParentID: 1, Path: "1-16", ResourceType: vars.ResourceTypeMenu, HideInMenu: hide, Status: 1, SortOrder: 50, CreatedAt: now, UpdatedAt: now},
		{ID: 46, Name: "路由列表", Alias: "auth:routes", BURL: "/auth/routes", Methods: "GET", ParentID: 16, Path: "1-16-46", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 10, CreatedAt: now, UpdatedAt: now},
		{ID: 47, Name: "用户信息", Alias: "auth:info", BURL: "/auth/info", Methods: "GET", ParentID: 16, Path: "1-16-47", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 20, CreatedAt: now, UpdatedAt: now},
		{ID: 48, Name: "动态路由", Alias: "auth:user-routes", BURL: "/auth/user-routes", Methods: "GET", ParentID: 16, Path: "1-16-48", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 30, CreatedAt: now, UpdatedAt: now},
		{ID: 49, Name: "退出登录", Alias: "auth:logout", BURL: "/auth/logout", Methods: "POST", ParentID: 16, Path: "1-16-49", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 40, CreatedAt: now, UpdatedAt: now},
		{ID: 50, Name: "修改密码", Alias: "auth:password", BURL: "/auth/password", Methods: "POST", ParentID: 16, Path: "1-16-50", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 50, CreatedAt: now, UpdatedAt: now},
	}

	for _, row := range resources {
		var existing model.SysResource
		err := db.Where("id = ?", row.ID).First(&existing).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := db.Create(&row).Error; err != nil {
					return err
				}
				continue
			}
			return err
		}
		if err := db.Model(&existing).Updates(map[string]interface{}{
			"name":          row.Name,
			"alias":         row.Alias,
			"desc":          row.Desc,
			"f_url":         row.FURL,
			"b_url":         row.BURL,
			"methods":       row.Methods,
			"icon":          row.Icon,
			"parent_id":     row.ParentID,
			"path":          row.Path,
			"resource_type": row.ResourceType,
			"hide_in_menu":  row.HideInMenu,
			"status":        row.Status,
			"sort_order":    row.SortOrder,
			"updated_at":    now,
		}).Error; err != nil {
			return err
		}
	}

	// 删除旧通配按钮
	obsolete := []uint64{3, 10, 12, 15, 17, 19}
	if err := db.Where("id IN ?", obsolete).Delete(&model.SysResource{}).Error; err != nil {
		return err
	}
	return nil
}

func seedRoleAuth(db *gorm.DB) error {
	authBtns := "16,46,47,48,49,50"
	adminFull := "2,4,21,22,23,13"
	adminList := "2,4"
	attachFull := "14,24,25,26,27,28,29"
	tenantFull := "11,30,31,32,33,34,35"
	roleRead := "5,6"

	auths := []model.SysRoleAuth{
		{ID: 1, RoleID: 2, ResourceIds: "1,20," + tenantFull + "," + adminFull + "," + attachFull + "," + roleRead + "," + authBtns},
		{ID: 2, RoleID: 3, ResourceIds: "20," + adminFull + "," + attachFull + "," + roleRead + "," + authBtns},
		{ID: 3, RoleID: 4, ResourceIds: "20," + adminList + "," + roleRead + "," + authBtns},
	}

	for _, row := range auths {
		var existing model.SysRoleAuth
		err := db.Where("role_id = ?", row.RoleID).First(&existing).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := db.Create(&row).Error; err != nil {
					return err
				}
				continue
			}
			return err
		}
		if err := db.Model(&existing).Updates(map[string]interface{}{
			"resource_ids": row.ResourceIds,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

func syncSeededRoleCasbin(db *gorm.DB) error {
	// 清理历史 v3=resource_sync（Casbin p 仅支持 3 字段，LoadPolicy 会挂）
	if err := rbac.CleanupLegacySyncMarkers(db); err != nil {
		return err
	}
	for _, roleID := range []uint64{2, 3, 4} {
		if err := rbac.SyncRoleCasbin(db, roleID); err != nil {
			return fmt.Errorf("sync casbin role %d: %w", roleID, err)
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
