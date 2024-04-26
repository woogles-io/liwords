package common

import (
	"time"

	"gorm.io/gorm/logger"
	"gorm.io/plugin/opentelemetry/logging/logrus"
)

// we're going to remove Gorm. for now let's put some common
// Gorm-related things here.

var GormLogger logger.Interface

func init() {
	GormLogger = logger.New(
		logrus.NewWriter(),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond, // Slow SQL threshold
			LogLevel:                  logger.Warn,            // Log level
			IgnoreRecordNotFoundError: true,                   // Ignore ErrRecordNotFound error for logger
			Colorful:                  false,                  // Disable color
		},
	)
}
