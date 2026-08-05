package permission

import (
	"context"
	"errors"
	"strings"

	"github.com/MQEnergy/go-skeleton/internal/app/service"
	permtypes "github.com/MQEnergy/go-skeleton/internal/types/permission"
	"github.com/MQEnergy/go-skeleton/internal/vars"
	"github.com/MQEnergy/go-skeleton/pkg/rbac"
	"github.com/MQEnergy/go-skeleton/pkg/tenant"
	"github.com/gofiber/fiber/v3"
)

type Service struct {
	service.Service
}

var Permission = &Service{}

func (s *Service) assertPlatform(ctx fiber.Ctx) error {
	isPlatform, _ := ctx.Locals(tenant.LocalIsPlatform).(bool)
	if !isPlatform {
		return errors.New("仅平台用户可查看权限配置")
	}
	return nil
}

// ListAPIs 返回可绑定的后端路由（path + method）
func (s *Service) ListAPIs(ctx fiber.Ctx) ([]permtypes.APIItem, error) {
	if err := s.assertPlatform(ctx); err != nil {
		return nil, err
	}
	items := make([]permtypes.APIItem, 0)
	for _, route := range vars.Routes {
		method := strings.ToUpper(route.Method)
		if method != "GET" && method != "POST" {
			continue
		}
		path := route.Path
		if !strings.HasPrefix(path, "/platform") &&
			!strings.HasPrefix(path, "/system") &&
			!strings.HasPrefix(path, "/auth") {
			continue
		}
		items = append(items, permtypes.APIItem{
			Method: method,
			Path:   path,
			Name:   route.Name,
		})
	}
	return items, nil
}

// RolePolicies 查看角色当前 Casbin 策略
func (s *Service) RolePolicies(ctx fiber.Ctx, roleID uint64) ([]permtypes.RolePolicyItem, error) {
	if err := s.assertPlatform(ctx); err != nil {
		return nil, err
	}
	db := tenant.Global(vars.DB, context.Background())
	rules, err := rbac.ListRolePolicies(db, roleID)
	if err != nil {
		return nil, err
	}
	out := make([]permtypes.RolePolicyItem, 0, len(rules))
	for _, r := range rules {
		out = append(out, permtypes.RolePolicyItem{
			ID:     r.ID,
			Ptype:  r.Ptype,
			V0:     r.V0,
			V1:     r.V1,
			V2:     r.V2,
			V3:     r.V3,
			Source: permtypes.PolicySource(r),
		})
	}
	return out, nil
}

// Reload 强制重载 Casbin
func (s *Service) Reload(ctx fiber.Ctx) error {
	if err := s.assertPlatform(ctx); err != nil {
		return err
	}
	return rbac.ReloadPolicy()
}
