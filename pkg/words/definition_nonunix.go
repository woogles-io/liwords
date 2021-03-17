// +build !darwin,!linux

package words

import (
	"os"
)

func getBlkSize(fileInfo *os.FileInfo, defaultBlkSize int64) int64 {
	return defaultBlkSize
}
