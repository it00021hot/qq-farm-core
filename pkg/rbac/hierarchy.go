package rbac

import (
	"strconv"
	"strings"

	"github.com/MQEnergy/go-skeleton/internal/app/model"
)

// ParseRoleIDs 解析逗号分隔的 role_ids
func ParseRoleIDs(s string) []uint64 {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]uint64, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := strconv.ParseUint(p, 10, 64)
		if err != nil {
			continue
		}
		out = append(out, id)
	}
	return out
}

// BuildChildrenMap parent_id -> children ids
func BuildChildrenMap(roles []model.SysRole) map[uint64][]uint64 {
	m := make(map[uint64][]uint64, len(roles))
	for _, r := range roles {
		m[r.ParentID] = append(m[r.ParentID], r.ID)
	}
	return m
}

// CollectSubtree 收集 rootIDs 各自子树（含自身）
func CollectSubtree(children map[uint64][]uint64, rootIDs []uint64) map[uint64]struct{} {
	allowed := make(map[uint64]struct{})
	var walk func(uint64)
	walk = func(id uint64) {
		if _, ok := allowed[id]; ok {
			return
		}
		allowed[id] = struct{}{}
		for _, c := range children[id] {
			walk(c)
		}
	}
	for _, id := range rootIDs {
		walk(id)
	}
	return allowed
}

// CanAssign 校验 targetRoleIDs 是否均在操作者角色子树内（含自身）
// roles 为全量角色列表；operatorRoleIDs 为操作者拥有的角色
func CanAssign(roles []model.SysRole, operatorRoleIDs []uint64, targetRoleIDs []uint64) bool {
	if len(targetRoleIDs) == 0 {
		return true
	}
	children := BuildChildrenMap(roles)
	allowed := CollectSubtree(children, operatorRoleIDs)
	for _, tid := range targetRoleIDs {
		if _, ok := allowed[tid]; !ok {
			return false
		}
	}
	return true
}

// FilterAssignable 返回操作者可分配的角色列表（子树 ∩ roleType 过滤）
func FilterAssignable(roles []model.SysRole, operatorRoleIDs []uint64, roleType uint8) []model.SysRole {
	children := BuildChildrenMap(roles)
	allowed := CollectSubtree(children, operatorRoleIDs)
	out := make([]model.SysRole, 0)
	for _, r := range roles {
		if _, ok := allowed[r.ID]; !ok {
			continue
		}
		if roleType > 0 && r.RoleType != roleType {
			continue
		}
		if r.Status != 1 {
			continue
		}
		out = append(out, r)
	}
	return out
}
