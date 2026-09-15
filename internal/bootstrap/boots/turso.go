package boots

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/vars"
	"github.com/it00021hot/qq-farm-core/pkg/database"
	tursodriver "github.com/it00021hot/qq-farm-core/pkg/database/driver/turso"
	logger2 "github.com/it00021hot/qq-farm-core/pkg/logger"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// databaseDriver 适配 pkg/database.DriverInterface（dialector → Instance）。
type databaseDriver struct{ dialector gorm.Dialector }

func (d databaseDriver) Instance() gorm.Dialector { return d.dialector }

// InitTurso initializes the default Turso (SQLite-compatible embedded) connection.
func InitTurso() error {
	if vars.DB != nil {
		return nil
	}
	if !vars.Config.GetBool("database.turso.enabled") {
		return nil
	}

	dbPath := vars.Config.GetString("database.turso.path")
	if dbPath == "" {
		dbPath = "runtime/data/qq-farm.db"
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return fmt.Errorf("create turso data dir: %w", err)
	}

	// turso 驱动的 DSN 就是文件路径；PRAGMA 在连接建立后单独执行。
	logLevel := vars.Config.GetInt("database.turso.logLevel")
	if logLevel == 0 {
		logLevel = 1
	}
	fileName := vars.Config.GetString("database.turso.fileName")
	if fileName == "" {
		fileName = "turso-sql"
	}

	newLogger := logger.New(
		log.New(logger2.ApplyWriter(fileName, &vars.Config), "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.LogLevel(logLevel),
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      true,
			Colorful:                  false,
		},
	)

	maxIdle := vars.Config.GetInt("database.turso.maxIdleConn")
	if maxIdle == 0 {
		maxIdle = 1
	}
	maxOpen := vars.Config.GetInt("database.turso.maxOpenConn")
	if maxOpen == 0 {
		maxOpen = 1
	}

	d, err := database.New(
		databaseDriver{dialector: &tursodriver.Dialector{DriverName: tursodriver.DriverName, DSN: dbPath}},
		&gorm.Config{
			NamingStrategy: schema.NamingStrategy{
				SingularTable: true,
			},
			Logger: newLogger,
		},
		database.WithMaxIdleConn(maxIdle),
		database.WithMaxOpenConn(maxOpen),
	)
	if err != nil {
		return err
	}

	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA foreign_keys=1",
	} {
		if err := d.DB.Exec(pragma).Error; err != nil {
			slog.Warn("failed to apply pragma", "pragma", pragma, "err", err)
		}
	}

	vars.DB = d.DB
	if vars.MDB == nil {
		vars.MDB = make(map[string]*gorm.DB)
	}
	vars.MDB[database.DefaultAlias] = d.DB
	slog.Info("Starting turso connection", "path", dbPath)
	return nil
}

// TablePrefix returns the configured table prefix.
func TablePrefix() string {
	return vars.Config.GetString("database.turso.prefix")
}

// AutoMigrateEnabled reports whether AutoMigrate should run.
func AutoMigrateEnabled() bool {
	if v := vars.Config.Get("database.turso.autoMigrate"); v != nil {
		return vars.Config.GetBool("database.turso.autoMigrate")
	}
	return true
}
