// +build darwin linux

package words

import (
	"os"
	"syscall"
)

func getBlkSize(fileInfo *os.FileInfo, defaultBlkSize int64) int64 {
	if fileInfoSys, ok := (*fileInfo).Sys().(*syscall.Stat_t); ok && fileInfoSys != nil && fileInfoSys.Blksize > 0 {
		// this thing is int32 on mac, but int64 on linux, so might as well always cast it
		return int64(fileInfoSys.Blksize)
	}
	return defaultBlkSize
}
