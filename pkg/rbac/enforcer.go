package rbac

import (
	"fmt"
	"strings"
	"sync"

	"github.com/MQEnergy/go-skeleton/configs"
	"github.com/MQEnergy/go-skeleton/pkg/helper"
	"github.com/casbin/casbin/v3"
	"github.com/casbin/casbin/v3/model"
	"github.com/casbin/casbin/v3/util"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

var (
	enforcerMu sync.RWMutex
	enforcer   *casbin.Enforcer
)

// InitEnforcer 进程内单例 Casbin Enforcer（启动时调用一次）
func InitEnforcer(db *gorm.DB, tablePrefix, tableName string) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if tableName == "" {
		tableName = "casbin_rule"
	}
	adapter, err := gormadapter.NewFilteredAdapterByDB(db, tablePrefix, tableName)
	if err != nil {
		return fmt.Errorf("casbin adapter: %w", err)
	}
	rc, err := model.NewModelFromString(configs.RbacModelConf)
	if err != nil {
		return fmt.Errorf("casbin model: %w", err)
	}
	e, err := casbin.NewEnforcer(rc, adapter)
	if err != nil {
		return fmt.Errorf("casbin enforcer: %w", err)
	}
	e.AddFunction("ParamsObjMatch", ParamsObjMatchFunc)
	e.AddFunction("ParamsActMatch", ParamsActMatchFunc)
	if err := e.LoadPolicy(); err != nil {
		return fmt.Errorf("casbin load policy: %w", err)
	}

	enforcerMu.Lock()
	enforcer = e
	enforcerMu.Unlock()
	return nil
}

// GetEnforcer 获取单例（未初始化返回 nil）
func GetEnforcer() *casbin.Enforcer {
	enforcerMu.RLock()
	defer enforcerMu.RUnlock()
	return enforcer
}

// ReloadPolicy 从 DB 重新加载策略
func ReloadPolicy() error {
	e := GetEnforcer()
	if e == nil {
		return fmt.Errorf("casbin enforcer not initialized")
	}
	enforcerMu.Lock()
	defer enforcerMu.Unlock()
	return e.LoadPolicy()
}

// Enforce 对单个角色做权限判断
func Enforce(sub, obj, act string) (bool, error) {
	e := GetEnforcer()
	if e == nil {
		return false, fmt.Errorf("casbin enforcer not initialized")
	}
	enforcerMu.RLock()
	defer enforcerMu.RUnlock()
	return e.Enforce(sub, obj, act)
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
// KeyMatch2 下 /backend/admin/* 不能匹配集合根路径 /backend/admin，这里补齐该语义。
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
	path := rObjArr[0]
	if util.KeyMatch2(path, pObj) {
		return true, nil
	}
	if strings.HasSuffix(pObj, "/*") {
		prefix := strings.TrimSuffix(pObj, "/*")
		if path == prefix {
			return true, nil
		}
	}
	return false, nil
}
