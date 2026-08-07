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
	sqlitedriver "github.com/it00021hot/qq-farm-core/pkg/database/driver/sqlite"
	logger2 "github.com/it00021hot/qq-farm-core/pkg/logger"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// InitSQLite initializes the default SQLite connection.
func InitSQLite() error {
	if vars.DB != nil {
		return nil
	}
	if !vars.Config.GetBool("database.sqlite.enabled") {
		return nil
	}

	dbPath := vars.Config.GetString("database.sqlite.path")
	if dbPath == "" {
		dbPath = "runtime/data/qq-farm.db"
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return fmt.Errorf("create sqlite dir: %w", err)
	}

	// Pure-Go sqlite (glebarez) DSN with busy timeout; WAL enabled after open.
	dsn := fmt.Sprintf("%s?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)", dbPath)
	logLevel := vars.Config.GetInt("database.sqlite.logLevel")
	if logLevel == 0 {
		logLevel = 1
	}
	fileName := vars.Config.GetString("database.sqlite.fileName")
	if fileName == "" {
		fileName = "sqlite-sql"
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

	maxIdle := vars.Config.GetInt("database.sqlite.maxIdleConn")
	if maxIdle == 0 {
		maxIdle = 1
	}
	maxOpen := vars.Config.GetInt("database.sqlite.maxOpenConn")
	if maxOpen == 0 {
		maxOpen = 1
	}

	d, err := database.New(
		sqlitedriver.New(sqlitedriver.WithDSN(dsn)),
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

	if err := d.DB.Exec("PRAGMA journal_mode=WAL").Error; err != nil {
		slog.Warn("failed to enable WAL mode", "err", err)
	}

	vars.DB = d.DB
	if vars.MDB == nil {
		vars.MDB = make(map[string]*gorm.DB)
	}
	vars.MDB[database.DefaultAlias] = d.DB
	slog.Info("Starting sqlite connection", "path", dbPath)
	return nil
}

// TablePrefix returns the configured table prefix (sqlite preferred, then pgsql).
func TablePrefix() string {
	if vars.Config.GetBool("database.sqlite.enabled") {
		p := vars.Config.GetString("database.sqlite.prefix")
		if p != "" {
			return p
		}
	}
	return vars.Config.GetString("database.pgsql.sources." + database.DefaultAlias + ".prefix")
}

// AutoMigrateEnabled reports whether AutoMigrate should run.
func AutoMigrateEnabled() bool {
	if vars.Config.GetBool("database.sqlite.enabled") {
		if v := vars.Config.Get("database.sqlite.autoMigrate"); v != nil {
			return vars.Config.GetBool("database.sqlite.autoMigrate")
		}
		return true
	}
	if v := vars.Config.Get("database.pgsql.autoMigrate"); v != nil {
		return vars.Config.GetBool("database.pgsql.autoMigrate")
	}
	return true
}
