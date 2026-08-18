package boots

import (
	"log/slog"
	"path/filepath"

	"github.com/it00021hot/qq-farm-core/internal/farm/hub"
	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
	farmruntime "github.com/it00021hot/qq-farm-core/internal/farm/runtime"
	"github.com/it00021hot/qq-farm-core/internal/vars"
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
	farmruntime.ScheduleWxAuthorizedStart()
	farmruntime.StartWxKeepalive()
	slog.Info("Farm runtime manager ready",
		"gateway", vars.Config.GetString("farm.gatewayUrl"),
		"wasm", vars.Config.GetString("farm.wasmPath"),
	)
}
