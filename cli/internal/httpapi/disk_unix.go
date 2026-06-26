//go:build !windows

package httpapi

import "syscall"

func getDiskFreeGB(path string) uint64 {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0
	}
	return stat.Bavail * uint64(stat.Bsize) / 1024 / 1024 / 1024
}
