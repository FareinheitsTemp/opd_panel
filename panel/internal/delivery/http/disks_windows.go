package http

import (
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

type diskInfo struct {
	Path   string  `json:"path"`
	Label  string  `json:"label"`
	FreeGB float64 `json:"free_gb"`
}

func getDiskList() []diskInfo {
	var disks []diskInfo
	for _, letter := range "CDEFGHIJKLMNOPQRSTUVWXYZ" {
		path := string(letter) + `:\`
		if _, err := os.Stat(path); err != nil {
			continue
		}
		pathPtr, _ := windows.UTF16PtrFromString(path)
		var freeBytes, totalBytes, totalFree uint64
		err := windows.GetDiskFreeSpaceEx(
			pathPtr,
			(*uint64)(unsafe.Pointer(&freeBytes)),
			(*uint64)(unsafe.Pointer(&totalBytes)),
			(*uint64)(unsafe.Pointer(&totalFree)),
		)
		if err != nil {
			continue
		}
		disks = append(disks, diskInfo{
			Path:   path,
			Label:  string(letter) + ":",
			FreeGB: float64(freeBytes) / 1024 / 1024 / 1024,
		})
	}
	return disks
}
