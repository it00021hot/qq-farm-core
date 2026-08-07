package bootstrap

import (
	"fmt"
	"log/slog"

	"github.com/MQEnergy/go-skeleton/internal/bootstrap/boots"
	"github.com/MQEnergy/go-skeleton/internal/vars"
	"github.com/MQEnergy/go-skeleton/pkg/helper"
	"github.com/MQEnergy/go-skeleton/pkg/rbac"
)

// Define service list
const (
	PgsqlService  = `Pgsql`
	SQLiteService = `SQLite`
	RedisService  = `Redis`
	S3Service     = `S3`
)

type bootServiceMap map[string]func() error

var (
	BootedService []string
	serviceMap    = bootServiceMap{
		SQLiteService: boots.InitSQLite,
		PgsqlService:  boots.InitMultiPgsql,
		RedisService:  boots.InitRedis,
		S3Service:     boots.InitS3,
	}
)

// BootService Load service
func BootService(services ...string) {
	// init config
	if err := boots.InitConfig(); err != nil {
		panic("Failed to load config：" + err.Error())
	}
	// init logger
	boots.InitLogger()

	// init service
	if len(services) == 0 {
		services = serviceMap.keys()
	}
	BootedService = make([]string, 0)
	for k, val := range serviceMap {
		if helper.InAnySlice[string](services, k) {
			if err := val(); err != nil {
				panic(fmt.Sprintf("Failed to load service %s err: %s", k, err.Error()))
			}
			slog.Info("Loading " + k + " service successfully")
			BootedService = append(BootedService, k)
		}
	}
	// auto migrate + seed
	if err := boots.InitMigrate(); err != nil {
		panic("Failed to migrate database：" + err.Error())
	}
	// casbin singleton（依赖 migrate 后的策略表）
	if vars.DB != nil {
		prefix := boots.TablePrefix()
		if err := rbac.InitEnforcer(vars.DB, prefix, "sys_casbin_rule"); err != nil {
			panic("Failed to init casbin enforcer：" + err.Error())
		}
		slog.Info("Loading Casbin enforcer successfully")
	}
	// tenant plugin
	if err := boots.InitTenantPlugin(); err != nil {
		panic("Failed to register tenant plugin：" + err.Error())
	}
	// init dao
	boots.InitDao()
	// farm runtime manager (in-process bot sessions)
	boots.InitFarmRuntime()
}

// keys ...
func (m bootServiceMap) keys() []string {
	keys := make([]string, 0)
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
