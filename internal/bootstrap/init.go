package bootstrap

import (
	"fmt"
	"log/slog"

	"github.com/MQEnergy/go-skeleton/internal/bootstrap/boots"
	"github.com/MQEnergy/go-skeleton/pkg/helper"
)

// Define service list
const (
	PgsqlService = `Pgsql`
	RedisService = `Redis`
	S3Service    = `S3`
)

type bootServiceMap map[string]func() error

var (
	BootedService []string
	serviceMap    = bootServiceMap{
		PgsqlService: boots.InitMultiPgsql,
		RedisService: boots.InitRedis,
		S3Service:    boots.InitS3,
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
	// init dao
	boots.InitDao()
}

// keys ...
func (m bootServiceMap) keys() []string {
	keys := make([]string, 0)
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
