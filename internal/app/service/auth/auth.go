package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/MQEnergy/go-skeleton/internal/app/model"
	"github.com/MQEnergy/go-skeleton/internal/app/pkg/redis"
	"github.com/MQEnergy/go-skeleton/internal/app/service"
	authtypes "github.com/MQEnergy/go-skeleton/internal/types/user/auth"
	"github.com/MQEnergy/go-skeleton/internal/vars"
	"github.com/MQEnergy/go-skeleton/pkg/helper"
	"github.com/MQEnergy/go-skeleton/pkg/jwtauth"
	"github.com/MQEnergy/go-skeleton/pkg/rbac"
	"github.com/MQEnergy/go-skeleton/pkg/tenant"
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
		"id":        admin.ID,
		"uuid":      admin.UUID,
		"role_ids":  admin.RoleIds,
		"tenant_id": admin.TenantID,
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
		TenantID:     admin.TenantID,
	}, nil
}

// Login 用户登录
func (s *AuthService) Login(reqParams authtypes.LoginReq) (*authtypes.LoginTokenResp, error) {
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
	if err := tenant.Global(vars.DB, context.Background()).Where("id = ?", uid).First(&admin).Error; err != nil {
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
	uid := cast.ToUint64(ctx.Locals(tenant.LocalUID))
	uuid := cast.ToString(ctx.Locals(tenant.LocalUUID))
	if uid == 0 {
		return errors.New("未登录")
	}
	if req.OldPassword == req.NewPassword {
		return errors.New("新密码不能与旧密码相同")
	}

	db := tenant.Global(vars.DB, context.Background())
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
	uuid := cast.ToString(ctx.Locals(tenant.LocalUUID))

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
	uid := cast.ToUint64(ctx.Locals(tenant.LocalUID))
	roleIDsStr := cast.ToString(ctx.Locals(tenant.LocalRoleIDs))
	isSuper, _ := ctx.Locals(tenant.LocalIsSuper).(bool)

	db := tenant.Global(vars.DB, context.Background())
	var admin model.SysAdmin
	if err := db.Where("id = ?", uid).First(&admin).Error; err != nil {
		return nil, errors.New("用户不存在")
	}

	roleIDList := rbac.ParseRoleIDs(roleIDsStr)
	roleIDStrs := make([]string, 0, len(roleIDList))
	for _, id := range roleIDList {
		roleIDStrs = append(roleIDStrs, cast.ToString(id))
	}

	var roles []model.SysRole
	roleCodes := make([]string, 0)
	if len(roleIDList) > 0 {
		if err := db.Where("id IN ?", roleIDList).Find(&roles).Error; err != nil {
			return nil, err
		}
		for _, r := range roles {
			roleCodes = append(roleCodes, r.Code)
		}
	}

	resources, err := s.loadResources(db, roleIDList, isSuper)
	if err != nil {
		return nil, err
	}

	buttons := make([]string, 0)
	menuResources := make([]model.SysResource, 0)
	for _, r := range resources {
		switch r.ResourceType {
		case vars.ResourceTypeButton:
			if r.Alias != "" {
				buttons = append(buttons, r.Alias)
			}
		case vars.ResourceTypeDir, vars.ResourceTypeMenu:
			menuResources = append(menuResources, r)
		}
	}
	if isSuper {
		buttons = []string{"*"}
	} else {
		buttons = uniqueSorted(buttons)
	}

	return &authtypes.InfoResp{
		UserID:   cast.ToString(admin.ID),
		UserName: admin.Account,
		NickName: admin.NickName,
		UUID:     admin.UUID,
		TenantID: admin.TenantID,
		RoleIDs:  roleIDStrs,
		Roles:    roleCodes,
		Menus:    buildMenuTree(menuResources, 0),
		Buttons:  buttons,
	}, nil
}

// UserRoutes 返回 soybean Elegant 动态路由（不走 Info Redis 缓存，避免菜单变更后侧栏滞后）
func (s *AuthService) UserRoutes(ctx fiber.Ctx) (*authtypes.UserRouteResp, error) {
	info, err := s.infoFromDB(ctx)
	if err != nil {
		return nil, err
	}
	routes := make([]authtypes.ElegantRoute, 0)
	for _, m := range info.Menus {
		if r, ok := menuNodeToElegant(m); ok {
			routes = append(routes, r)
		}
	}
	return &authtypes.UserRouteResp{Routes: routes, Home: "home"}, nil
}

func (s *AuthService) loadResources(db *gorm.DB, roleIDList []uint64, isSuper bool) ([]model.SysResource, error) {
	var resources []model.SysResource
	if isSuper {
		if err := db.Where("status = ?", vars.StatusNormal).
			Order("sort_order ASC, id ASC").
			Find(&resources).Error; err != nil {
			return nil, err
		}
		return resources, nil
	}
	idSet := make(map[uint64]struct{})
	if len(roleIDList) > 0 {
		var auths []model.SysRoleAuth
		if err := db.Where("role_id IN ?", roleIDList).Find(&auths).Error; err != nil {
			return nil, err
		}
		for _, a := range auths {
			for _, id := range rbac.ParseRoleIDs(a.ResourceIds) {
				idSet[id] = struct{}{}
			}
		}
	}
	ids := make([]uint64, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return resources, nil
	}
	if err := db.Where("id IN ? AND status = ?", ids, vars.StatusNormal).
		Order("sort_order ASC, id ASC").
		Find(&resources).Error; err != nil {
		return nil, err
	}
	return resources, nil
}

func menuNodeToElegant(n authtypes.MenuNode) (authtypes.ElegantRoute, bool) {
	name := n.Alias
	if name == "" {
		return authtypes.ElegantRoute{}, false
	}
	meta := authtypes.ElegantMeta{
		Title:      n.Name,
		Icon:       n.Icon,
		Order:      int(n.SortOrder),
		HideInMenu: n.HideInMenu == 2,
	}

	switch n.ResourceType {
	case vars.ResourceTypeDir:
		children := make([]authtypes.ElegantRoute, 0)
		for _, c := range n.Children {
			if cr, ok := menuNodeToElegant(c); ok {
				children = append(children, cr)
			}
		}
		if len(children) == 0 {
			return authtypes.ElegantRoute{}, false
		}
		path := "/" + name
		if strings.Contains(name, "_") {
			parts := strings.SplitN(name, "_", 2)
			path = "/" + parts[0]
		}
		// 仅顶层目录挂 layout；嵌套目录（如农场下的系统管理）作菜单分组，避免双层 layout
		component := ""
		if n.ParentID == 0 {
			component = "layout.base"
		}
		return authtypes.ElegantRoute{
			Name:      name,
			Path:      path,
			Component: component,
			Meta:      meta,
			Children:  children,
		}, true
	case vars.ResourceTypeMenu:
		// 无前端路径的「菜单」仅作按钮分组，不进 Elegant 路由
		if strings.TrimSpace(n.FURL) == "" {
			return authtypes.ElegantRoute{}, false
		}
		return authtypes.ElegantRoute{
			Name:      name,
			Path:      n.FURL,
			Component: "view." + name,
			Meta:      meta,
		}, true
	default:
		return authtypes.ElegantRoute{}, false
	}
}

func uniqueSorted(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func buildMenuTree(list []model.SysResource, parentID uint64) []authtypes.MenuNode {
	nodes := make([]authtypes.MenuNode, 0)
	for _, r := range list {
		if r.ParentID != parentID {
			continue
		}
		node := authtypes.MenuNode{
			ID:           r.ID,
			Name:         r.Name,
			Alias:        r.Alias,
			FURL:         r.FURL,
			Icon:         r.Icon,
			ParentID:     r.ParentID,
			ResourceType: r.ResourceType,
			HideInMenu:   r.HideInMenu,
			SortOrder:    r.SortOrder,
			Children:     buildMenuTree(list, r.ID),
		}
		nodes = append(nodes, node)
	}
	return nodes
}
