package http

import (
	"golang.org/x/sys/windows"
)

func getDiskFreeSpace(path string, freeBytes *uint64) error {
	var totalBytes, totalFreeBytes uint64
	err := windows.GetDiskFreeSpaceEx(
		windows.StringToUTF16Ptr(path),
		freeBytes,
		&totalBytes,
		&totalFreeBytes,
	)
	return err
}
