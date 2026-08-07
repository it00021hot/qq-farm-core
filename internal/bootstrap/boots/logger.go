package boots

import (
	"fmt"
	"os"

	"github.com/it00021hot/qq-farm-core/internal/vars"
	logger2 "github.com/it00021hot/qq-farm-core/pkg/logger"
)

// InitLogger ...
func InitLogger() {
	s := logger2.New(
		vars.Config.GetString("log.fileName"),
		&vars.Config,
	)
	vars.Logger = s
	fileName := fmt.Sprintf("%s/%s.log", vars.Config.Get("log.dirPath"), vars.Config.GetString("log.fileName"))
	s.Info("Loading Logger service successfully")
	_ = os.Chmod(fileName, 0o644)
}
