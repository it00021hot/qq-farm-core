package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/MQEnergy/go-skeleton/internal/app/dao"
	"github.com/MQEnergy/go-skeleton/internal/app/model"
	"github.com/MQEnergy/go-skeleton/internal/app/pkg/redis"
	"github.com/MQEnergy/go-skeleton/internal/app/service"
	"github.com/MQEnergy/go-skeleton/internal/types/user/auth"
	"github.com/MQEnergy/go-skeleton/internal/vars"
	"github.com/MQEnergy/go-skeleton/pkg/helper"
	"github.com/MQEnergy/go-skeleton/pkg/jwtauth"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

type AuthService struct {
	service.Service
}

var Auth = &AuthService{}

// Login 用户登录
func (s *AuthService) Login(reqParams auth.LoginReq) (fiber.Map, error) {
	var (
		err       error
		adminInfo *model.SysAdmin
	)
	adminInfo, err = dao.SysAdmin.Where(dao.SysAdmin.Account.Eq(reqParams.Account)).First()
	if err != nil {
		return nil, errors.New("账号或密码不正确")
	}
	if adminInfo.Status != 1 {
		return nil, errors.New("用户已锁定，无法登录")
	}
	if adminInfo.Password != helper.GeneratePasswordHash(reqParams.Password, adminInfo.Salt) {
		return nil, errors.New("账号或密码不正确")
	}
	token, err := jwtauth.New(&vars.Config).WithClaims(jwt.MapClaims{
		"id":       adminInfo.ID,
		"uuid":     adminInfo.UUID,
		"role_ids": adminInfo.RoleIds,
	}).GenerateToken()
	if err != nil {
		return nil, errors.New("登录失败")
	}
	return fiber.Map{
		"token": token,
	}, nil
}

// Logout 退出登录
func (s *AuthService) Logout(uuid string) error {
	// 删除auth
	redis.Del(context.Background(), fmt.Sprintf(redis.AuthFmt, uuid))
	// 删除用户信息
	redis.Del(context.Background(), fmt.Sprintf(redis.PermsFmt, uuid))
	return nil
}
