package boots

import (
	"log/slog"

	"github.com/it00021hot/qq-farm-core/internal/vars"
	"github.com/it00021hot/qq-farm-core/pkg/config"
)

// InitConfig ...
func InitConfig() error {
	cfg, err := config.New(config.NewViper(), config.Options{
		BasePath: vars.BasePath,
		FileName: "config." + config.ConfEnv,
	})
	if err != nil {
		return err
	}
	vars.Config = *cfg
	slog.Info("Server.mode: " + vars.Config.GetString("server.mode"))
	slog.Info("Loading Configuration successfully")
	return nil
}
