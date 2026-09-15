package vars

import (
	"log/slog"
	"path"
	"path/filepath"
	"runtime"

	"github.com/it00021hot/qq-farm-core/pkg/config"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

var (
	BasePath string // resource root (bundled assets: configs/resource)
	DataPath string // writable runtime root (turso db/logs/tsdk); defaults to BasePath
	DB       *gorm.DB
	MDB      map[string]*gorm.DB
	Router   fiber.Router
	Routes   []fiber.Route
	Config   config.Config
	Logger   *slog.Logger

	// DesktopMode tunes behavior for the embedded desktop shell.
	DesktopMode bool
)

func init() {
	_, filename, _, _ := runtime.Caller(0)
	root := path.Dir(path.Dir(path.Dir(filename)))
	BasePath = root
	DataPath = root
}

// SetPaths overrides resource and data roots (used by the Wails desktop shell).
func SetPaths(resourceRoot, dataRoot string) {
	if resourceRoot != "" {
		BasePath = filepath.Clean(resourceRoot)
	}
	if dataRoot != "" {
		DataPath = filepath.Clean(dataRoot)
	} else if DataPath == "" {
		DataPath = BasePath
	}
}
