package boots

import (
	"log/slog"
	"path/filepath"

	"github.com/MQEnergy/go-skeleton/internal/farm/hub"
	"github.com/MQEnergy/go-skeleton/internal/farm/logic"
	farmruntime "github.com/MQEnergy/go-skeleton/internal/farm/runtime"
	"github.com/MQEnergy/go-skeleton/internal/vars"
)

// InitFarmRuntime wires the in-process AccountManager and loads game config JSON.
func InitFarmRuntime() {
	cfgDir := vars.Config.GetString("farm.gameConfigDir")
	if cfgDir == "" {
		cfgDir = "resource/farm/gameConfig"
	}
	if !filepath.IsAbs(cfgDir) {
		cfgDir = filepath.Clean(cfgDir)
	}
	if err := logic.LoadGameConfig(cfgDir); err != nil {
		slog.Error("Failed to load farm game config", "dir", cfgDir, "err", err)
	} else {
		slog.Info("Farm game config loaded", "dir", cfgDir)
	}

	m := farmruntime.NewAccountManager(hub.Default)
	farmruntime.SetManager(m)
	farmruntime.ResetPersistedRunStatus()
	slog.Info("Farm runtime manager ready",
		"gateway", vars.Config.GetString("farm.gatewayUrl"),
		"wasm", vars.Config.GetString("farm.wasmPath"),
	)
}
