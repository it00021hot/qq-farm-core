package boots

import (
	"fmt"
	"log"
	"log/slog"
	"net/url"
	"regexp"
	"time"

	"github.com/MQEnergy/go-skeleton/internal/vars"
	"github.com/MQEnergy/go-skeleton/pkg/database"
	pgdriver "github.com/MQEnergy/go-skeleton/pkg/database/driver/postgres"
	"github.com/MQEnergy/go-skeleton/pkg/database/migrate"
	logger2 "github.com/MQEnergy/go-skeleton/pkg/logger"
	"github.com/gogf/gf/v2/util/gconv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

var dbNamePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// InitMultiPgsql initialize one or more postgres connections
func InitMultiPgsql() error {
	if vars.MDB != nil {
		return nil
	}
	if !vars.Config.GetBool("database.pgsql.enabled") {
		return nil
	}
	sources := vars.Config.Get("database.pgsql.sources")
	sourceList, ok := sources.(map[string]interface{})
	if !ok {
		return nil
	}
	if len(sourceList) == 0 {
		return nil
	}
	vars.MDB = make(map[string]*gorm.DB, len(sourceList))
	for alias, m := range sourceList {
		sm := gconv.Map(m)
		d, err := handlePgsql(sm)
		if err != nil {
			slog.Error("Failed to start pgsql connection ", "alias", alias, "err", err.Error())
			if alias == database.DefaultAlias {
				return fmt.Errorf("default pgsql connection failed: %w", err)
			}
			continue
		}
		if alias == database.DefaultAlias {
			vars.DB = d.DB
		}
		vars.MDB[alias] = d.DB
		slog.Info("Starting pgsql connection db:" + alias)
	}
	return nil
}

// InitMigrate AutoMigrate + 幂等初始化数据
func InitMigrate() error {
	if vars.DB == nil {
		return nil
	}
	// 默认开启；仅当显式配置为 false 时跳过
	if v := vars.Config.Get("database.pgsql.autoMigrate"); v != nil && !vars.Config.GetBool("database.pgsql.autoMigrate") {
		slog.Info("Database autoMigrate disabled by config")
		return nil
	}
	return migrate.Run(vars.DB)
}

// handlePgsql ...
func handlePgsql(sourceMaps map[string]interface{}) (*database.Database, error) {
	fileName := sourceMaps["filename"].(string)
	logLevel := sourceMaps["loglevel"].(int)
	masterDsn, _ := url.QueryUnescape(sourceMaps["master"].(string))
	prefix := sourceMaps["prefix"].(string)
	separation := sourceMaps["separation"].(bool)
	slaves := sourceMaps["slave"].([]interface{})

	if err := ensureDatabase(masterDsn); err != nil {
		return nil, err
	}

	dbContainer := func(dsn string) *pgdriver.Postgres {
		return pgdriver.New(func(opts *postgres.Config) {
			opts.DSN = dsn
			opts.PreferSimpleProtocol = true
		})
	}
	newLogger := logger.New(
		log.New(logger2.ApplyWriter(
			fileName,
			&vars.Config,
		), "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.LogLevel(logLevel),
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      true,
			Colorful:                  false,
		},
	)
	// model.TableName 已包含完整表名（如 cn_sys_admin），这里不再叠加 TablePrefix，避免双前缀
	_ = prefix
	d, err := database.New(
		dbContainer(masterDsn),
		&gorm.Config{
			NamingStrategy: schema.NamingStrategy{
				SingularTable: true,
			},
			Logger: newLogger,
		},
		database.WithMaxIdleConn(vars.Config.GetInt("database.pgsql.maxIdleConn")),
		database.WithMaxOpenConn(vars.Config.GetInt("database.pgsql.maxOpenConn")),
		database.WithConnMaxIdleTime(vars.Config.GetDuration("database.pgsql.maxIdleTime")*time.Second),
		database.WithConnMaxLifetime(vars.Config.GetDuration("database.pgsql.maxLifeTime")*time.Minute),
	)
	if err != nil {
		return nil, err
	}
	if separation {
		var replicas []gorm.Dialector
		for _, slave := range slaves {
			slaveDsn, _ := url.QueryUnescape(slave.(string))
			replicas = append(replicas, dbContainer(slaveDsn).Instance())
		}
		if err := d.WithSlaveDB([]gorm.Dialector{dbContainer(masterDsn).Instance()}, replicas); err != nil {
			return nil, err
		}
	}
	return d, nil
}

// ensureDatabase 若不存在则自动创建目标库（通过连接 postgres 系统库）
func ensureDatabase(dsn string) error {
	dbName := migrate.ParseDBName(dsn)
	if dbName == "" {
		return fmt.Errorf("dbname missing in dsn")
	}
	if !dbNamePattern.MatchString(dbName) {
		return fmt.Errorf("invalid dbname: %s", dbName)
	}

	adminDSN := migrate.ReplaceDBName(dsn, "postgres")
	db, err := gorm.Open(postgres.Open(adminDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return fmt.Errorf("connect postgres for ensure database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	var exists bool
	if err := db.Raw("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = ?)", dbName).Scan(&exists).Error; err != nil {
		return fmt.Errorf("check database exists: %w", err)
	}
	if exists {
		return nil
	}
	// CREATE DATABASE 不支持参数绑定
	if err := db.Exec(fmt.Sprintf(`CREATE DATABASE "%s"`, dbName)).Error; err != nil {
		return fmt.Errorf("create database %s: %w", dbName, err)
	}
	slog.Info("Created database", "name", dbName)
	return nil
}
