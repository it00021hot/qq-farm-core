package permission

import (
	"github.com/MQEnergy/go-skeleton/internal/app/model"
)

type RoleIDReq struct {
	ID uint64 `json:"id" params:"id" validate:"required"`
}

type APIItem struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Name   string `json:"name"`
}

type RolePolicyItem struct {
	ID     uint64  `json:"id"`
	Ptype  *string `json:"ptype"`
	V0     *string `json:"v0"`
	V1     *string `json:"v1"`
	V2     *string `json:"v2"`
	V3     *string `json:"v3"`
	Source string  `json:"source"` // resource_sync | seed
}

func PolicySource(rule model.SysCasbinRule) string {
	// 现网策略由 role_auth → SyncRoleCasbin 全量生成；历史 seed 无特殊标记
	if rule.V3 != nil && *rule.V3 != "" {
		return *rule.V3
	}
	return "policy"
}
