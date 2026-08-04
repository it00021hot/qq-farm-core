package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/MQEnergy/go-skeleton/internal/app/model"
	"github.com/MQEnergy/go-skeleton/internal/app/pkg/redis"
	"github.com/MQEnergy/go-skeleton/internal/app/service"
	"github.com/MQEnergy/go-skeleton/internal/types/user/auth"
	"github.com/MQEnergy/go-skeleton/internal/vars"
	"github.com/MQEnergy/go-skeleton/pkg/helper"
	"github.com/MQEnergy/go-skeleton/pkg/jwtauth"
	"github.com/MQEnergy/go-skeleton/pkg/tenant"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type AuthService struct {
	service.Service
}

var Auth = &AuthService{}

// Login 用户登录
func (s *AuthService) Login(reqParams auth.LoginReq) (fiber.Map, error) {
	var adminInfo model.SysAdmin
	err := tenant.Global(vars.DB, context.Background()).
		Where("account = ?", reqParams.Account).
		First(&adminInfo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("账号或密码不正确")
		}
		return nil, err
	}
	if adminInfo.Status != vars.StatusNormal {
		return nil, errors.New("用户已锁定，无法登录")
	}
	if adminInfo.Password != helper.GeneratePasswordHash(reqParams.Password, adminInfo.Salt) {
		return nil, errors.New("账号或密码不正确")
	}
	// 租户用户：校验租户状态与过期
	if adminInfo.TenantID > 0 {
		var t model.SysTenant
		if err := tenant.Global(vars.DB, context.Background()).Where("id = ?", adminInfo.TenantID).First(&t).Error; err != nil {
			return nil, errors.New("所属租户不存在")
		}
		if t.Status != vars.StatusNormal {
			return nil, errors.New("所属租户已禁用")
		}
		if t.ExpireAt > 0 && uint64(t.ExpireAt) < uint64(time.Now().Unix()) {
			return nil, errors.New("所属租户已过期")
		}
	}
	token, err := jwtauth.New(&vars.Config).WithClaims(jwt.MapClaims{
		"id":        adminInfo.ID,
		"uuid":      adminInfo.UUID,
		"role_ids":  adminInfo.RoleIds,
		"tenant_id": adminInfo.TenantID,
	}).GenerateToken()
	if err != nil {
		return nil, errors.New("登录失败")
	}
	return fiber.Map{
		"token":     token,
		"tenant_id": adminInfo.TenantID,
	}, nil
}

// Logout 退出登录
func (s *AuthService) Logout(uuid string) error {
	redis.Del(context.Background(), fmt.Sprintf(redis.AuthFmt, uuid))
	redis.Del(context.Background(), fmt.Sprintf(redis.PermsFmt, uuid))
	return nil
}
