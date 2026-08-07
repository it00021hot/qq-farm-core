package boots

import (
	"os"
	"path/filepath"

	"github.com/MQEnergy/go-skeleton/internal/vars"
)

// NormalizePaths turns relative config paths into absolute ones under
// BasePath (read-only resources) or DataPath (writable runtime data).
func NormalizePaths() {
	if vars.DataPath == "" {
		vars.DataPath = vars.BasePath
	}

	absUnder := func(root, key, def string) {
		p := vars.Config.GetString(key)
		if p == "" {
			p = def
		}
		if p != "" && !filepath.IsAbs(p) {
			vars.Config.Set(key, filepath.Join(root, p))
		}
	}

	absUnder(vars.DataPath, "log.dirPath", "runtime/logs")
	absUnder(vars.DataPath, "database.sqlite.path", "runtime/data/qq-farm.db")
	absUnder(vars.DataPath, "farm.tsdkDataDir", "runtime/tsdk")
	absUnder(vars.BasePath, "farm.wasmPath", "resource/farm/tsdk.wasm")
	absUnder(vars.BasePath, "farm.gameConfigDir", "resource/farm/gameConfig")
	absUnder(vars.BasePath, "swagger.filePath", "./docs/swagger.json")

	if vars.DesktopMode {
		vars.Config.Set("swagger.enabled", false)
		vars.Config.Set("server.prefork", false)
	}

	_ = os.MkdirAll(vars.Config.GetString("log.dirPath"), 0o755)
	if dbPath := vars.Config.GetString("database.sqlite.path"); dbPath != "" {
		_ = os.MkdirAll(filepath.Dir(dbPath), 0o755)
	}
	if tsdk := vars.Config.GetString("farm.tsdkDataDir"); tsdk != "" {
		_ = os.MkdirAll(tsdk, 0o755)
	}
}
