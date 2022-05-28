package common

import (
	"log"
	"os"
	"time"

	"gorm.io/gorm/logger"
)

// we're going to remove Gorm. for now let's put some common
// Gorm-related things here.

var GormLogger logger.Interface

func init() {
	GormLogger = logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             200 * time.Millisecond, // Slow SQL threshold
			LogLevel:                  logger.Warn,            // Log level
			IgnoreRecordNotFoundError: true,                   // Ignore ErrRecordNotFound error for logger
			Colorful:                  false,                  // Disable color
		},
	)
}
