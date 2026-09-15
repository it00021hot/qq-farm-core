package boots

import (
	"log/slog"

	"github.com/it00021hot/qq-farm-core/internal/vars"
	"github.com/it00021hot/qq-farm-core/pkg/database/migrate"
)

// InitMigrate AutoMigrate + 幂等初始化数据.
// Kept untagged: desktop builds run this against the embedded Turso database.
func InitMigrate() error {
	if vars.DB == nil {
		return nil
	}
	if !AutoMigrateEnabled() {
		slog.Info("Database autoMigrate disabled by config")
		return nil
	}
	return migrate.Run(vars.DB)
}
