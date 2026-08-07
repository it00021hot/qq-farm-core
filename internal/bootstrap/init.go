package bootstrap

import (
	"fmt"
	"log/slog"

	"github.com/MQEnergy/go-skeleton/internal/bootstrap/boots"
	"github.com/MQEnergy/go-skeleton/pkg/helper"
)

// Define service list
const (
	PgsqlService  = `Pgsql`
	SQLiteService = `SQLite`
	RedisService  = `Redis`
)

type bootServiceMap map[string]func() error

var (
	BootedService []string
	serviceMap    = bootServiceMap{
		SQLiteService: boots.InitSQLite,
		PgsqlService:  boots.InitMultiPgsql,
		RedisService:  boots.InitRedis,
	}
)

// BootService Load service
func BootService(services ...string) {
	// init config
	if err := boots.InitConfig(); err != nil {
		panic("Failed to load config：" + err.Error())
	}
	boots.NormalizePaths()
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
