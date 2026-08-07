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
		&farmAccount{},
		&farmAccountConfig{},
		&farmCard{},
		&farmCardClaim{},
		&farmFriendGid{},
		&farmStats{},
		&farmInteractLog{},
		&farmSystemConfig{},
		&farmGameConfig{},
		&farmActivityState{},
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
	if db == nil || db.Dialector == nil {
		return nil
	}
	name := strings.ToLower(db.Dialector.Name())
	if name != "postgres" && name != "postgresql" {
		return nil
	}
	tables := []string{
		"cn_sys_admin",
		"cn_sys_role",
		"cn_sys_tenant",
		"cn_sys_admin_tenant",
		"cn_sys_resource",
		"cn_sys_role_auth",
		"cn_sys_casbin_rule",
		"cn_attachment",
		"cn_farm_account",
		"cn_farm_account_config",
		"cn_farm_card",
		"cn_farm_card_claim",
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
		{ID: 1, Name: "平台管理", Alias: "platform", Desc: "平台配置目录", Icon: "mdi:office-building-cog", ParentID: 0, Path: "1", ResourceType: vars.ResourceTypeDir, HideInMenu: show, Status: 1, SortOrder: 20, CreatedAt: now, UpdatedAt: now},
		// 系统管理挂到农场管理下（ParentID=60）
		{ID: 20, Name: "系统管理", Alias: "system", Desc: "系统业务目录", Icon: "mdi:cog-outline", ParentID: 60, Path: "60-20", ResourceType: vars.ResourceTypeDir, HideInMenu: show, Status: 1, SortOrder: 90, CreatedAt: now, UpdatedAt: now},

		{ID: 2, Name: "用户管理", Alias: "system_admin", Desc: "用户管理", FURL: "/system/admin", Icon: "mdi:account", ParentID: 20, Path: "60-20-2", ResourceType: vars.ResourceTypeMenu, HideInMenu: show, Status: 1, SortOrder: 10, CreatedAt: now, UpdatedAt: now},
		{ID: 4, Name: "用户列表", Alias: "admin:list", Desc: "用户列表", BURL: "/system/admin/list", Methods: "GET", ParentID: 2, Path: "60-20-2-4", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 10, CreatedAt: now, UpdatedAt: now},
		{ID: 21, Name: "用户新增", Alias: "admin:add", Desc: "创建用户", BURL: "/system/admin/add", Methods: "POST", ParentID: 2, Path: "60-20-2-21", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 20, CreatedAt: now, UpdatedAt: now},
		{ID: 22, Name: "用户修改", Alias: "admin:modify", Desc: "更新用户", BURL: "/system/admin/modify", Methods: "POST", ParentID: 2, Path: "60-20-2-22", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 30, CreatedAt: now, UpdatedAt: now},
		{ID: 23, Name: "用户启停", Alias: "admin:status", Desc: "启停用户", BURL: "/system/admin/status", Methods: "POST", ParentID: 2, Path: "60-20-2-23", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 40, CreatedAt: now, UpdatedAt: now},
		{ID: 13, Name: "平台用户新增", Alias: "platform-user:add", Desc: "创建平台用户", BURL: "/system/platform-user/add", Methods: "POST", ParentID: 2, Path: "60-20-2-13", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 50, CreatedAt: now, UpdatedAt: now},

		{ID: 14, Name: "附件管理", Alias: "system_attachment", Desc: "附件", FURL: "/system/attachment", Icon: "mdi:paperclip", ParentID: 20, Path: "60-20-14", ResourceType: vars.ResourceTypeMenu, HideInMenu: show, Status: 1, SortOrder: 20, CreatedAt: now, UpdatedAt: now},
		{ID: 24, Name: "附件列表", Alias: "attachment:list", BURL: "/system/attachment/list", Methods: "GET", ParentID: 14, Path: "60-20-14-24", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 10, CreatedAt: now, UpdatedAt: now},
		{ID: 25, Name: "附件详情", Alias: "attachment:detail", BURL: "/system/attachment/detail", Methods: "GET", ParentID: 14, Path: "60-20-14-25", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 20, CreatedAt: now, UpdatedAt: now},
		{ID: 26, Name: "附件上传", Alias: "attachment:upload", BURL: "/system/attachment/upload", Methods: "POST", ParentID: 14, Path: "60-20-14-26", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 30, CreatedAt: now, UpdatedAt: now},
		{ID: 27, Name: "附件访问址", Alias: "attachment:access-url", BURL: "/system/attachment/access-url", Methods: "POST", ParentID: 14, Path: "60-20-14-27", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 40, CreatedAt: now, UpdatedAt: now},
		{ID: 28, Name: "附件启停", Alias: "attachment:status", BURL: "/system/attachment/status", Methods: "POST", ParentID: 14, Path: "60-20-14-28", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 50, CreatedAt: now, UpdatedAt: now},
		{ID: 29, Name: "附件删除", Alias: "attachment:delete", BURL: "/system/attachment/delete", Methods: "POST", ParentID: 14, Path: "60-20-14-29", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 60, CreatedAt: now, UpdatedAt: now},

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

		// 农场业务（FURL 对齐 vue-framework Elegant Router）
		{ID: 60, Name: "农场管理", Alias: "farm", Desc: "农场业务目录", Icon: "mdi:barn", ParentID: 0, Path: "60", ResourceType: vars.ResourceTypeDir, HideInMenu: show, Status: 1, SortOrder: 10, CreatedAt: now, UpdatedAt: now},

		// 概览已迁到首页 /home，菜单隐藏；权限按钮仍挂在此节点
		{ID: 69, Name: "农场概览", Alias: "farm_dashboard", Desc: "农场概览（已迁至首页）", FURL: "/farm/dashboard", Icon: "mdi:view-dashboard", ParentID: 60, Path: "60-69", ResourceType: vars.ResourceTypeMenu, HideInMenu: hide, Status: 1, SortOrder: 1, CreatedAt: now, UpdatedAt: now},
		{ID: 76, Name: "状态详情", Alias: "farm-status:detail", BURL: "/farm/status/detail", Methods: "GET", ParentID: 69, Path: "60-69-76", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 10, CreatedAt: now, UpdatedAt: now},
		{ID: 77, Name: "状态列表", Alias: "farm-status:list", BURL: "/farm/status/list", Methods: "GET", ParentID: 69, Path: "60-69-77", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 20, CreatedAt: now, UpdatedAt: now},
		{ID: 108, Name: "实时通道", Alias: "farm-ws:get", BURL: "/farm/ws", Methods: "GET", ParentID: 69, Path: "60-69-108", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 30, CreatedAt: now, UpdatedAt: now},
		{ID: 109, Name: "运行日志", Alias: "farm-logs:list", BURL: "/farm/logs", Methods: "GET", ParentID: 69, Path: "60-69-109", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 40, CreatedAt: now, UpdatedAt: now},
		{ID: 110, Name: "清空日志", Alias: "farm-logs:clear", BURL: "/farm/logs", Methods: "DELETE", ParentID: 69, Path: "60-69-110", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 50, CreatedAt: now, UpdatedAt: now},

		{ID: 61, Name: "农场账号", Alias: "farm_account", Desc: "农场账号管理", FURL: "/farm/account", Icon: "mdi:account-cowboy-hat", ParentID: 60, Path: "60-61", ResourceType: vars.ResourceTypeMenu, HideInMenu: show, Status: 1, SortOrder: 10, CreatedAt: now, UpdatedAt: now},
		{ID: 62, Name: "账号列表", Alias: "farm-account:list", BURL: "/farm/account/list", Methods: "GET", ParentID: 61, Path: "60-61-62", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 10, CreatedAt: now, UpdatedAt: now},
		{ID: 63, Name: "账号详情", Alias: "farm-account:detail", BURL: "/farm/account/detail", Methods: "GET", ParentID: 61, Path: "60-61-63", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 20, CreatedAt: now, UpdatedAt: now},
		{ID: 64, Name: "账号新增", Alias: "farm-account:add", BURL: "/farm/account/add", Methods: "POST", ParentID: 61, Path: "60-61-64", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 30, CreatedAt: now, UpdatedAt: now},
		{ID: 65, Name: "账号修改", Alias: "farm-account:modify", BURL: "/farm/account/modify", Methods: "POST", ParentID: 61, Path: "60-61-65", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 40, CreatedAt: now, UpdatedAt: now},
		{ID: 66, Name: "账号删除", Alias: "farm-account:delete", BURL: "/farm/account/delete", Methods: "POST", ParentID: 61, Path: "60-61-66", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 50, CreatedAt: now, UpdatedAt: now},
		{ID: 67, Name: "账号启动", Alias: "farm-account:start", BURL: "/farm/account/start", Methods: "POST", ParentID: 61, Path: "60-61-67", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 60, CreatedAt: now, UpdatedAt: now},
		{ID: 68, Name: "账号停止", Alias: "farm-account:stop", BURL: "/farm/account/stop", Methods: "POST", ParentID: 61, Path: "60-61-68", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 70, CreatedAt: now, UpdatedAt: now},
		{ID: 112, Name: "微信扫码任务", Alias: "farm-wx-login:tasks", BURL: "/farm/wx-login/tasks", Methods: "POST", ParentID: 61, Path: "60-61-112", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 80, CreatedAt: now, UpdatedAt: now},
		{ID: 113, Name: "微信扫码二维码", Alias: "farm-wx-login:qr", BURL: "/farm/wx-login/tasks/:taskId/qr", Methods: "GET", ParentID: 61, Path: "60-61-113", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 81, CreatedAt: now, UpdatedAt: now},
		{ID: 114, Name: "微信扫码状态", Alias: "farm-wx-login:status", BURL: "/farm/wx-login/tasks/:taskId/status", Methods: "GET", ParentID: 61, Path: "60-61-114", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 82, CreatedAt: now, UpdatedAt: now},
		{ID: 115, Name: "微信扫码确认", Alias: "farm-wx-login:confirm", BURL: "/farm/wx-login/tasks/:taskId/confirm", Methods: "POST", ParentID: 61, Path: "60-61-115", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 83, CreatedAt: now, UpdatedAt: now},
		{ID: 116, Name: "微信扫码取码", Alias: "farm-wx-login:code", BURL: "/farm/wx-login/tasks/:taskId/code", Methods: "POST", ParentID: 61, Path: "60-61-116", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 84, CreatedAt: now, UpdatedAt: now},

		{ID: 85, Name: "好友", Alias: "farm_friends", Desc: "好友互动", FURL: "/farm/friends", Icon: "mdi:account-group", ParentID: 60, Path: "60-85", ResourceType: vars.ResourceTypeMenu, HideInMenu: show, Status: 1, SortOrder: 15, CreatedAt: now, UpdatedAt: now},
		{ID: 86, Name: "好友列表", Alias: "farm-friend:list", BURL: "/farm/friend/list", Methods: "GET", ParentID: 85, Path: "60-85-86", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 10, CreatedAt: now, UpdatedAt: now},
		{ID: 87, Name: "互动记录", Alias: "farm-friend:interact-logs", BURL: "/farm/friend/interact-logs", Methods: "GET", ParentID: 85, Path: "60-85-87", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 20, CreatedAt: now, UpdatedAt: now},
		{ID: 111, Name: "最近访客", Alias: "farm-friend:interact-records", BURL: "/farm/friend/interact-records", Methods: "GET", ParentID: 85, Path: "60-85-111", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 25, CreatedAt: now, UpdatedAt: now},
		{ID: 104, Name: "同步好友", Alias: "farm-friend:sync", BURL: "/farm/friend/sync", Methods: "POST", ParentID: 85, Path: "60-85-104", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 30, CreatedAt: now, UpdatedAt: now},
		{ID: 105, Name: "好友互动", Alias: "farm-friend:op", BURL: "/farm/friend/op", Methods: "POST", ParentID: 85, Path: "60-85-105", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 40, CreatedAt: now, UpdatedAt: now},
		{ID: 106, Name: "好友土地", Alias: "farm-friend:lands", BURL: "/farm/friend/lands", Methods: "GET", ParentID: 85, Path: "60-85-106", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 50, CreatedAt: now, UpdatedAt: now},

		{ID: 88, Name: "活动中心", Alias: "farm_activity", Desc: "活动中心", FURL: "/farm/activity", Icon: "mdi:star-circle", ParentID: 60, Path: "60-88", ResourceType: vars.ResourceTypeMenu, HideInMenu: show, Status: 1, SortOrder: 16, CreatedAt: now, UpdatedAt: now},
		{ID: 89, Name: "活动快照", Alias: "farm-activity:snapshot", BURL: "/farm/activity/snapshot", Methods: "GET", ParentID: 88, Path: "60-88-89", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 10, CreatedAt: now, UpdatedAt: now},
		{ID: 90, Name: "领取通行证", Alias: "farm-activity:pass-claim", BURL: "/farm/activity/pass/claim", Methods: "POST", ParentID: 88, Path: "60-88-90", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 20, CreatedAt: now, UpdatedAt: now},
		{ID: 91, Name: "点亮观星", Alias: "farm-activity:constellation", BURL: "/farm/activity/constellation/light", Methods: "POST", ParentID: 88, Path: "60-88-91", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 30, CreatedAt: now, UpdatedAt: now},
		{ID: 92, Name: "星砂兑换", Alias: "farm-activity:shop", BURL: "/farm/activity/shop/exchange", Methods: "POST", ParentID: 88, Path: "60-88-92", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 40, CreatedAt: now, UpdatedAt: now},
		{ID: 93, Name: "领取节令", Alias: "farm-activity:solar", BURL: "/farm/activity/solar-terms/claim", Methods: "POST", ParentID: 88, Path: "60-88-93", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 50, CreatedAt: now, UpdatedAt: now},

		{ID: 94, Name: "分析", Alias: "farm_analytics", Desc: "数据分析", FURL: "/farm/analytics", Icon: "mdi:chart-line", ParentID: 60, Path: "60-94", ResourceType: vars.ResourceTypeMenu, HideInMenu: show, Status: 1, SortOrder: 17, CreatedAt: now, UpdatedAt: now},
		{ID: 95, Name: "分析详情", Alias: "farm-analytics:detail", BURL: "/farm/analytics/detail", Methods: "GET", ParentID: 94, Path: "60-94-95", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 10, CreatedAt: now, UpdatedAt: now},

		{ID: 70, Name: "自动化设置", Alias: "farm_settings", Desc: "自动化与间隔设置", FURL: "/farm/settings", Icon: "mdi:cog", ParentID: 60, Path: "60-70", ResourceType: vars.ResourceTypeMenu, HideInMenu: show, Status: 1, SortOrder: 20, CreatedAt: now, UpdatedAt: now},
		{ID: 71, Name: "配置详情", Alias: "farm-automation:detail", BURL: "/farm/automation/detail", Methods: "GET", ParentID: 70, Path: "60-70-71", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 10, CreatedAt: now, UpdatedAt: now},
		{ID: 72, Name: "配置修改", Alias: "farm-automation:modify", BURL: "/farm/automation/modify", Methods: "POST", ParentID: 70, Path: "60-70-72", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 20, CreatedAt: now, UpdatedAt: now},

		{ID: 96, Name: "游戏配置", Alias: "farm_game-config", Desc: "游戏静态配置", FURL: "/farm/game-config", Icon: "mdi:seed", ParentID: 60, Path: "60-96", ResourceType: vars.ResourceTypeMenu, HideInMenu: show, Status: 1, SortOrder: 25, CreatedAt: now, UpdatedAt: now},
		// KeyMatch2：/farm/game-config/* 覆盖 list/seeds/fruits/items/plants/item-types 等只读目录 API
		{ID: 97, Name: "配置列表", Alias: "farm-game-config:list", BURL: "/farm/game-config/*", Methods: "GET", ParentID: 96, Path: "60-96-97", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 10, CreatedAt: now, UpdatedAt: now},
		// 覆盖 modify 与 seed/fruit/item 的 add|modify|delete
		{ID: 98, Name: "配置修改", Alias: "farm-game-config:modify", BURL: "/farm/game-config/*", Methods: "POST", ParentID: 96, Path: "60-96-98", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 20, CreatedAt: now, UpdatedAt: now},

		{ID: 80, Name: "卡密管理", Alias: "farm_card", Desc: "卡密生成与兑换", FURL: "/farm/card", Icon: "mdi:ticket-percent", ParentID: 60, Path: "60-80", ResourceType: vars.ResourceTypeMenu, HideInMenu: show, Status: 1, SortOrder: 40, CreatedAt: now, UpdatedAt: now},
		{ID: 81, Name: "卡密列表", Alias: "farm-card:list", BURL: "/farm/card/list", Methods: "GET", ParentID: 80, Path: "60-80-81", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 10, CreatedAt: now, UpdatedAt: now},
		{ID: 82, Name: "卡密生成", Alias: "farm-card:add", BURL: "/farm/card/add", Methods: "POST", ParentID: 80, Path: "60-80-82", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 20, CreatedAt: now, UpdatedAt: now},
		{ID: 83, Name: "卡密兑换", Alias: "farm-card:redeem", BURL: "/farm/card/redeem", Methods: "POST", ParentID: 80, Path: "60-80-83", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 30, CreatedAt: now, UpdatedAt: now},
		{ID: 84, Name: "卡密作废", Alias: "farm-card:status", BURL: "/farm/card/status", Methods: "POST", ParentID: 80, Path: "60-80-84", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 40, CreatedAt: now, UpdatedAt: now},

		{ID: 99, Name: "个人农场", Alias: "farm_personal", Desc: "个人农场面板", FURL: "/farm/personal", Icon: "mdi:sprout", ParentID: 60, Path: "60-99", ResourceType: vars.ResourceTypeMenu, HideInMenu: show, Status: 1, SortOrder: 8, CreatedAt: now, UpdatedAt: now},
		{ID: 107, Name: "可用种子", Alias: "farm-seeds:get", BURL: "/farm/seeds", Methods: "GET", ParentID: 70, Path: "60-70-107", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 30, CreatedAt: now, UpdatedAt: now},
		{ID: 100, Name: "土地列表", Alias: "farm-lands:get", BURL: "/farm/lands", Methods: "GET", ParentID: 99, Path: "60-99-100", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 10, CreatedAt: now, UpdatedAt: now},
		{ID: 101, Name: "农场操作", Alias: "farm-operate:post", BURL: "/farm/operate", Methods: "POST", ParentID: 99, Path: "60-99-101", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 20, CreatedAt: now, UpdatedAt: now},
		{ID: 102, Name: "背包列表", Alias: "farm-bag:get", BURL: "/farm/bag", Methods: "GET", ParentID: 99, Path: "60-99-102", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 30, CreatedAt: now, UpdatedAt: now},
		{ID: 103, Name: "背包出售", Alias: "farm-bag:sell", BURL: "/farm/bag/sell", Methods: "POST", ParentID: 99, Path: "60-99-103", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 40, CreatedAt: now, UpdatedAt: now},
		{ID: 123, Name: "每日礼包任务", Alias: "farm-daily-gifts:get", BURL: "/farm/daily-gifts", Methods: "GET", ParentID: 99, Path: "60-99-123", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 50, CreatedAt: now, UpdatedAt: now},
		{ID: 124, Name: "背包使用", Alias: "farm-bag:use", BURL: "/farm/bag/use", Methods: "POST", ParentID: 99, Path: "60-99-124", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 45, CreatedAt: now, UpdatedAt: now},

		{ID: 117, Name: "游戏商城", Alias: "farm_game-mall", Desc: "游戏商城商品与购买", FURL: "/farm/game-mall", Icon: "mdi:storefront", ParentID: 60, Path: "60-117", ResourceType: vars.ResourceTypeMenu, HideInMenu: show, Status: 1, SortOrder: 18, CreatedAt: now, UpdatedAt: now},
		{ID: 118, Name: "商城商品", Alias: "farm-game-mall:get", BURL: "/farm/game-mall", Methods: "GET", ParentID: 117, Path: "60-117-118", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 10, CreatedAt: now, UpdatedAt: now},
		{ID: 119, Name: "购买商品", Alias: "farm-game-mall:purchase", BURL: "/farm/game-mall/purchase", Methods: "POST", ParentID: 117, Path: "60-117-119", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 20, CreatedAt: now, UpdatedAt: now},
		{ID: 120, Name: "钻石余额", Alias: "farm-diamond:get", BURL: "/farm/diamond", Methods: "GET", ParentID: 117, Path: "60-117-120", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 30, CreatedAt: now, UpdatedAt: now},

		{ID: 121, Name: "神秘商人", Alias: "farm_mystery-shop", Desc: "神秘商人限时商品", FURL: "/farm/mystery-shop", Icon: "mdi:account-question", ParentID: 60, Path: "60-121", ResourceType: vars.ResourceTypeMenu, HideInMenu: show, Status: 1, SortOrder: 19, CreatedAt: now, UpdatedAt: now},
		{ID: 122, Name: "神秘商人商品", Alias: "farm-mystery-shop:get", BURL: "/farm/mystery-shop", Methods: "GET", ParentID: 121, Path: "60-121-122", ResourceType: vars.ResourceTypeButton, HideInMenu: show, Status: 1, SortOrder: 10, CreatedAt: now, UpdatedAt: now},
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

	// 删除旧通配按钮与过时农场菜单
	obsolete := []uint64{3, 10, 12, 15, 17, 19, 75}
	if err := db.Where("id IN ?", obsolete).Delete(&model.SysResource{}).Error; err != nil {
		return err
	}
	return nil
}

func seedRoleAuth(db *gorm.DB) error {
	authBtns := "16,46,47,48,49,50"
	// 租户用户管理：不含 13(platform-user:add)，该按钮仅平台侧
	adminTenant := "2,4,21,22,23"
	adminPlatform := adminTenant + ",13"
	adminList := "2,4"
	attachFull := "14,24,25,26,27,28,29"
	tenantFull := "11,30,31,32,33,34,35"
	roleRead := "5,6"
	farmDash := "69,76,77,108,109,110"
	farmAccount := "61,62,63,64,65,66,67,68,112,113,114,115,116"
	farmFriends := "85,86,87,111,104,105,106"
	farmActivity := "88,89,90,91,92,93"
	farmAnalytics := "94,95"
	farmSettings := "70,71,72,107"
	farmGameCfg := "96,97,98"
	farmCardAll := "80,81,82,83,84"
	farmPersonal := "99,100,101,102,103,123,124"
	farmCommerce := "117,118,119,120,121,122"
	// 租户仅保留兑换按钮权限，不授卡密管理菜单(80)与列表(81)
	farmCardRedeem := "83"
	farmCommon := "60," + farmDash + "," + farmAccount + "," + farmFriends + "," + farmActivity + "," + farmAnalytics + "," + farmSettings + "," + farmGameCfg + "," + farmPersonal + "," + farmCommerce
	farmTenant := farmCommon + "," + farmCardRedeem
	farmPlatform := farmCommon + "," + farmCardAll

	auths := []model.SysRoleAuth{
		{ID: 1, RoleID: 2, ResourceIds: "1,20," + tenantFull + "," + adminPlatform + "," + attachFull + "," + roleRead + "," + farmPlatform + "," + authBtns},
		{ID: 2, RoleID: 3, ResourceIds: "20," + adminTenant + "," + attachFull + "," + roleRead + "," + farmTenant + "," + authBtns},
		{ID: 3, RoleID: 4, ResourceIds: "20," + adminList + "," + roleRead + "," + farmTenant + "," + authBtns},
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
	// Enforcer 可能已在上一轮启动中加载；热迁移后尽量刷新内存策略。
	_ = rbac.ReloadPolicy()
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
