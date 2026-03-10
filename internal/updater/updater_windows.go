//go:build windows

package updater

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	shell32           = syscall.NewLazyDLL("shell32.dll")
	procShellExecuteW = shell32.NewProc("ShellExecuteW")
)

func utf16Ptr(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

// launchElevated starts exePath with UAC elevation prompt.
func launchElevated(exePath string, args string, workingDir string) error {
	verb := utf16Ptr("runas") // request elevation
	file := utf16Ptr(exePath)

	var params *uint16
	if args != "" {
		params = utf16Ptr(args)
	}

	var dir *uint16
	if workingDir != "" {
		dir = utf16Ptr(workingDir)
	}

	// HINSTANCE > 32 means success. <= 32 is an error code.
	ret, _, _ := procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(file)),
		uintptr(unsafe.Pointer(params)),
		uintptr(unsafe.Pointer(dir)),
		1, // SW_SHOWNORMAL
	)

	if ret <= 32 {
		return fmt.Errorf("ShellExecuteW failed with code %d", ret)
	}
	return nil
}
