package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/MQEnergy/go-skeleton/internal/app/model"
	"github.com/MQEnergy/go-skeleton/internal/app/pkg/redis"
	"github.com/MQEnergy/go-skeleton/internal/app/service"
	authtypes "github.com/MQEnergy/go-skeleton/internal/types/user/auth"
	"github.com/MQEnergy/go-skeleton/internal/vars"
	"github.com/MQEnergy/go-skeleton/pkg/helper"
	"github.com/MQEnergy/go-skeleton/pkg/jwtauth"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

type AuthService struct {
	service.Service
}

var Auth = &AuthService{}

const refreshKeyFmt = "refresh:%s" // jti

func (s *AuthService) issueTokens(admin model.SysAdmin) (*authtypes.LoginTokenResp, error) {
	sub := jwt.MapClaims{
		"id":       admin.ID,
		"uuid":     admin.UUID,
		"role_ids": admin.RoleIds,
	}
	jwtHelper := jwtauth.New(&vars.Config)
	access, refresh, err := jwtHelper.GenerateTokenPair(sub)
	if err != nil {
		return nil, errors.New("登录失败")
	}
	claims, err := jwtHelper.ParseToken(refresh)
	if err != nil {
		return nil, errors.New("登录失败")
	}
	jti := cast.ToString(claims["jti"])
	_ = redis.SetEx(context.Background(), fmt.Sprintf(refreshKeyFmt, jti), admin.UUID, jwtHelper.RefreshTTL()).Err()
	_ = redis.SetEx(context.Background(), fmt.Sprintf(redis.AuthFmt, admin.UUID), jti, jwtHelper.RefreshTTL()).Err()

	return &authtypes.LoginTokenResp{
		Token:        access,
		RefreshToken: refresh,
	}, nil
}

// Login 用户登录
func (s *AuthService) Login(reqParams authtypes.LoginReq) (*authtypes.LoginTokenResp, error) {
	var adminInfo model.SysAdmin
	err := vars.DB.
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
	return s.issueTokens(adminInfo)
}

// Refresh 刷新 access token
func (s *AuthService) Refresh(req authtypes.RefreshReq) (*authtypes.LoginTokenResp, error) {
	jwtHelper := jwtauth.New(&vars.Config)
	claims, err := jwtHelper.ParseToken(req.RefreshToken)
	if err != nil {
		return nil, errors.New("refreshToken 无效或已过期")
	}
	if cast.ToString(claims["typ"]) != jwtauth.TokenTypeRefresh {
		return nil, errors.New("refreshToken 类型错误")
	}
	jti := cast.ToString(claims["jti"])
	storedUUID, err := redis.Get(context.Background(), fmt.Sprintf(refreshKeyFmt, jti)).Result()
	if err != nil || storedUUID == "" {
		return nil, errors.New("refreshToken 已失效")
	}
	sub, ok := claims["sub"].(map[string]interface{})
	if !ok {
		return nil, errors.New("refreshToken 无效")
	}
	uid := cast.ToUint64(sub["id"])
	var admin model.SysAdmin
	if err := vars.DB.Where("id = ?", uid).First(&admin).Error; err != nil {
		return nil, errors.New("用户不存在")
	}
	if admin.Status != vars.StatusNormal {
		return nil, errors.New("用户已锁定")
	}
	// 轮换：删除旧 refresh
	_ = redis.Del(context.Background(), fmt.Sprintf(refreshKeyFmt, jti)).Err()
	return s.issueTokens(admin)
}

// Logout 退出登录
func (s *AuthService) Logout(uuid string) error {
	ctx := context.Background()
	if jti, err := redis.Get(ctx, fmt.Sprintf(redis.AuthFmt, uuid)).Result(); err == nil && jti != "" {
		_ = redis.Del(ctx, fmt.Sprintf(refreshKeyFmt, jti)).Err()
	}
	_ = redis.Del(ctx, fmt.Sprintf(redis.AuthFmt, uuid)).Err()
	_ = redis.Del(ctx, fmt.Sprintf(redis.PermsFmt, uuid)).Err()
	return nil
}

// ChangePassword 当前登录用户修改密码（成功后使会话失效）
func (s *AuthService) ChangePassword(ctx fiber.Ctx, req authtypes.ChangePasswordReq) error {
	uid := cast.ToUint64(ctx.Locals("uid"))
	uuid := cast.ToString(ctx.Locals("uuid"))
	if uid == 0 {
		return errors.New("未登录")
	}
	if req.OldPassword == req.NewPassword {
		return errors.New("新密码不能与旧密码相同")
	}

	db := vars.DB
	var admin model.SysAdmin
	if err := db.Where("id = ?", uid).First(&admin).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("用户不存在")
		}
		return err
	}
	if admin.Password != helper.GeneratePasswordHash(req.OldPassword, admin.Salt) {
		return errors.New("旧密码不正确")
	}

	salt := helper.GenerateUuid(32)
	now := uint(time.Now().Unix())
	if err := db.Model(&admin).Updates(map[string]any{
		"salt":       salt,
		"password":   helper.GeneratePasswordHash(req.NewPassword, salt),
		"updated_at": now,
	}).Error; err != nil {
		return err
	}

	if uuid != "" {
		_ = s.Logout(uuid)
	}
	return nil
}

// Info 当前用户菜单/按钮权限
func (s *AuthService) Info(ctx fiber.Ctx) (*authtypes.InfoResp, error) {
	uuid := cast.ToString(ctx.Locals("uuid"))

	if uuid != "" {
		if cached, err := redis.Get(context.Background(), fmt.Sprintf(redis.PermsFmt, uuid)).Result(); err == nil && cached != "" {
			var resp authtypes.InfoResp
			if json.Unmarshal([]byte(cached), &resp) == nil {
				return &resp, nil
			}
		}
	}

	resp, err := s.infoFromDB(ctx)
	if err != nil {
		return nil, err
	}

	if uuid != "" {
		if b, err := json.Marshal(resp); err == nil {
			expire := vars.Config.GetDuration("jwt.accessExpire") * time.Second
			if expire <= 0 {
				expire = vars.Config.GetDuration("jwt.expire") * time.Second
			}
			_ = redis.SetEx(context.Background(), fmt.Sprintf(redis.PermsFmt, uuid), string(b), expire).Err()
		}
	}
	return resp, nil
}

func (s *AuthService) infoFromDB(ctx fiber.Ctx) (*authtypes.InfoResp, error) {
	uid := cast.ToUint64(ctx.Locals("uid"))

	var admin model.SysAdmin
	if err := vars.DB.Where("id = ?", uid).First(&admin).Error; err != nil {
		return nil, errors.New("用户不存在")
	}

	roleIDStrs := parseRoleIDStrings(admin.RoleIds)

	// 单机：登录即全权限；静态前端路由不依赖 menus
	return &authtypes.InfoResp{
		UserID:   cast.ToString(admin.ID),
		UserName: admin.Account,
		NickName: admin.NickName,
		UUID:     admin.UUID,
		RoleIDs:  roleIDStrs,
		Roles:    []string{"role_superadmin"},
		Menus:    nil,
		Buttons:  []string{"*"},
	}, nil
}

func parseRoleIDStrings(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
