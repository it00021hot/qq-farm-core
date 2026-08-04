package middleware

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/MQEnergy/go-skeleton/configs"
	"github.com/MQEnergy/go-skeleton/internal/vars"
	"github.com/MQEnergy/go-skeleton/pkg/helper"
	"github.com/MQEnergy/go-skeleton/pkg/response"
	"github.com/casbin/casbin/v3"
	"github.com/casbin/casbin/v3/model"
	"github.com/casbin/casbin/v3/util"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/gofiber/fiber/v3"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

// 定义不需要权限校验的路径
var excludePaths = []string{
	"/backend/auth/login",
	"/backend/auth/loginForTest",
	"/backend/auth/shop-login",
	"/backend/auth/wx-login",
	"/backend/auth/wx-qr-register",
	"/backend/auth/forget-pass",
}

// CasbinMiddleware casbin middleware
func CasbinMiddleware(db *gorm.DB, prefix, tableName string) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		if helper.InAnySlice[string](excludePaths, ctx.Path()) {
			return ctx.Next()
		}
		if db == nil {
			return ctx.Next()
		}

		roleIds := ctx.GetRespHeader("role_ids")
		if roleIds == "" {
			return response.UnauthorizedException(ctx, "该用户还未分配角色权限")
		}
		roleList := strings.Split(roleIds, ",")
		if helper.InAnySlice[string](roleList, vars.Config.GetString("server.superRoleId")) {
			return ctx.Next()
		}
		if tableName == "" {
			tableName = "casbin_rule"
		}
		adapter, err := gormadapter.NewFilteredAdapterByDB(db, prefix, tableName)
		if err != nil {
			return response.InternalServerException(ctx, "casbin adapter: "+err.Error())
		}
		rc, err := model.NewModelFromString(configs.RbacModelConf)
		if err != nil {
			return response.InternalServerException(ctx, "casbin model: "+err.Error())
		}
		e, err := casbin.NewEnforcer(rc, adapter)
		if err != nil {
			return response.InternalServerException(ctx, "casbin enforcer: "+err.Error())
		}
		e.AddFunction("ParamsObjMatch", ParamsObjMatchFunc)
		e.AddFunction("ParamsActMatch", ParamsActMatchFunc)
		if err := e.LoadPolicy(); err != nil {
			return response.InternalServerException(ctx, "casbin load policy: "+err.Error())
		}
		obj := ctx.Path()
		act := ctx.Method()
		flag := false
		for _, sub := range roleList {
			if ok, _ := e.Enforce(sub, obj, act); ok {
				flag = true
				break
			}
		}
		if !flag {
			return response.ForbiddenException(ctx, "该用户未授权访问权限")
		}
		return ctx.Next()
	}
}

// ParamsActMatchFunc 自定义规则函数 method
func ParamsActMatchFunc(args ...interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("must be 2 arguments")
	}
	rAct := cast.ToString(args[0])
	pAct := cast.ToString(args[1])
	pActArr := strings.Split(pAct, ",")
	if len(pActArr) == 1 {
		return pActArr[0] == rAct, nil
	}
	if len(pActArr) > 1 {
		return helper.InAnySlice[string](pActArr, rAct), nil
	}
	return false, nil
}

// ParamsObjMatchFunc 自定义规则函数 path
func ParamsObjMatchFunc(args ...interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("must be 2 arguments")
	}
	rObj := cast.ToString(args[0])
	pObj := cast.ToString(args[1])
	rObjArr := strings.Split(rObj, "?")
	if len(rObjArr) == 0 {
		return false, nil
	}
	return util.KeyMatch2(rObjArr[0], pObj), nil
}
