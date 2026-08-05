package rbac

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/MQEnergy/go-skeleton/internal/app/model"
	"github.com/MQEnergy/go-skeleton/internal/vars"
	"gorm.io/gorm"
)

// AggregatePolicies 将资源列表中带 b_url 的项按 path 聚合 methods
func AggregatePolicies(resources []model.SysResource) map[string]string {
	out := make(map[string]map[string]struct{})
	for _, r := range resources {
		path := strings.TrimSpace(r.BURL)
		if path == "" {
			continue
		}
		methods := NormalizeMethods(r.Methods)
		if methods == "" {
			continue
		}
		if out[path] == nil {
			out[path] = make(map[string]struct{})
		}
		for _, m := range strings.Split(methods, ",") {
			m = strings.TrimSpace(m)
			if m != "" {
				out[path][m] = struct{}{}
			}
		}
	}
	result := make(map[string]string, len(out))
	for path, set := range out {
		list := make([]string, 0, len(set))
		for m := range set {
			list = append(list, m)
		}
		sort.Strings(list)
		result[path] = strings.Join(list, ",")
	}
	return result
}

// NormalizeMethods 规范化方法列表（去空格、大写、去重）
func NormalizeMethods(s string) string {
	if s == "" {
		return ""
	}
	parts := strings.Split(s, ",")
	seen := make(map[string]struct{}, len(parts))
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.ToUpper(strings.TrimSpace(p))
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	sort.Strings(out)
	return strings.Join(out, ",")
}

// SyncRoleCasbin 按角色已授权资源全量重写该角色的 Casbin p 策略（3 字段：sub/obj/act）
// 说明：rbac_model 的 p = sub, obj, act，不能写入 v3 标记，否则 LoadPolicy 会失败。
func SyncRoleCasbin(db *gorm.DB, roleID uint64) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if roleID == 0 {
		return fmt.Errorf("invalid role id")
	}

	var auth model.SysRoleAuth
	err := db.Where("role_id = ?", roleID).First(&auth).Error
	var resourceIDs []uint64
	if err == nil {
		resourceIDs = ParseRoleIDs(auth.ResourceIds)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	var resources []model.SysResource
	if len(resourceIDs) > 0 {
		if err := db.Where("id IN ? AND status = ?", resourceIDs, vars.StatusNormal).Find(&resources).Error; err != nil {
			return err
		}
	}
	policies := AggregatePolicies(resources)

	roleStr := fmt.Sprintf("%d", roleID)
	strPtr := func(s string) *string { return &s }

	return db.Transaction(func(tx *gorm.DB) error {
		// 全量替换该角色策略，避免与 seed 双源并存；也不写 v3（Casbin 模型仅 3 字段）
		if err := tx.Where("ptype = ? AND v0 = ?", "p", roleStr).
			Delete(&model.SysCasbinRule{}).Error; err != nil {
			return err
		}
		for path, methods := range policies {
			rule := model.SysCasbinRule{
				Ptype: strPtr("p"),
				V0:    strPtr(roleStr),
				V1:    strPtr(path),
				V2:    strPtr(methods),
				V6:    "",
				V7:    "",
			}
			if err := tx.Create(&rule).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// CleanupLegacySyncMarkers 清理历史错误写入的 v3=resource_sync 策略（会导致 Casbin LoadPolicy 失败）
func CleanupLegacySyncMarkers(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	return db.Where("ptype = ? AND v3 = ?", "p", vars.CasbinSyncMarker).
		Delete(&model.SysCasbinRule{}).Error
}

// SyncRolesCasbin 批量同步多个角色
func SyncRolesCasbin(db *gorm.DB, roleIDs []uint64) error {
	for _, id := range roleIDs {
		if err := SyncRoleCasbin(db, id); err != nil {
			return err
		}
	}
	return nil
}

// FindRoleIDsByResource 找出授权了指定资源的角色 ID
func FindRoleIDsByResource(db *gorm.DB, resourceID uint64) ([]uint64, error) {
	var auths []model.SysRoleAuth
	if err := db.Find(&auths).Error; err != nil {
		return nil, err
	}
	out := make([]uint64, 0)
	for _, a := range auths {
		ids := ParseRoleIDs(a.ResourceIds)
		for _, id := range ids {
			if id == resourceID {
				out = append(out, a.RoleID)
				break
			}
		}
	}
	return out, nil
}

// ListRolePolicies 列出某角色全部 Casbin p 策略
func ListRolePolicies(db *gorm.DB, roleID uint64) ([]model.SysCasbinRule, error) {
	var list []model.SysCasbinRule
	err := db.Where("ptype = ? AND v0 = ?", "p", fmt.Sprintf("%d", roleID)).
		Order("id ASC").
		Find(&list).Error
	return list, err
}
