package http

import "syscall"

type diskInfo struct {
	Path   string  `json:"path"`
	Label  string  `json:"label"`
	FreeGB float64 `json:"free_gb"`
}

func getDiskList() []diskInfo {
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err != nil {
		return []diskInfo{}
	}
	freeBytes := stat.Bavail * uint64(stat.Bsize)
	return []diskInfo{{
		Path:   "/",
		Label:  "root",
		FreeGB: float64(freeBytes) / 1024 / 1024 / 1024,
	}}
}
