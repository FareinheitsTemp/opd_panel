package http

import "syscall"

type syscallStatfs = syscall.Statfs_t

func statfsCall(path string, stat *syscall.Statfs_t) error {
	return syscall.Statfs(path, stat)
}
