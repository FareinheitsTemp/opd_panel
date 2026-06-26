//go:build windows

package httpapi

import (
	"syscall"
	"unsafe"
)

func getDiskFreeGB(path string) uint64 {
	h := syscall.MustLoadDLL("kernel32.dll")
	proc := h.MustFindProc("GetDiskFreeSpaceExW")
	var freeBytes, totalBytes, totalFreeBytes int64
	p, _ := syscall.UTF16PtrFromString(path)
	proc.Call(
		uintptr(unsafe.Pointer(p)),
		uintptr(unsafe.Pointer(&freeBytes)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFreeBytes)),
	)
	if freeBytes < 0 {
		return 0
	}
	return uint64(freeBytes) / 1024 / 1024 / 1024
}
