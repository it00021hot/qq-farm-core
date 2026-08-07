package vars

import (
	"log/slog"
	"path"
	"path/filepath"
	"runtime"

	"github.com/MQEnergy/go-skeleton/pkg/config"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var (
	BasePath string // resource root (bundled assets: configs/resource)
	DataPath string // writable runtime root (sqlite/logs/tsdk); defaults to BasePath
	DB       *gorm.DB
	MDB      map[string]*gorm.DB
	Redis    *redis.Client
	Router   fiber.Router
	Routes   []fiber.Route
	Config   config.Config
	Logger   *slog.Logger

	// DesktopMode disables swagger and other browser-oriented extras.
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
