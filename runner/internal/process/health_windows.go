//go:build windows

package process

import (
	"fmt"
	"syscall"
	"unsafe"
)

const (
	processQueryLimitedInformation = 0x1000
	stillActive                    = 259
)

var (
	kernel32              = syscall.NewLazyDLL("kernel32.dll")
	procOpenProcess       = kernel32.NewProc("OpenProcess")
	procGetExitCodeProcess = kernel32.NewProc("GetExitCodeProcess")
)

// IsAlive checks whether the process with the given PID is still running.
// On Windows, it opens a handle to the process and checks its exit code.
func IsAlive(pid int) error {
	handle, _, err := procOpenProcess.Call(
		uintptr(processQueryLimitedInformation),
		0,
		uintptr(pid),
	)
	if handle == 0 {
		return fmt.Errorf("process not found (pid %d): %w", pid, err)
	}
	defer syscall.CloseHandle(syscall.Handle(handle))

	var exitCode uint32
	ret, _, err := procGetExitCodeProcess.Call(handle, uintptr(unsafe.Pointer(&exitCode)))
	if ret == 0 {
		return fmt.Errorf("failed to get exit code for pid %d: %w", pid, err)
	}

	if exitCode != stillActive {
		return fmt.Errorf("process not running (pid %d, exit code %d)", pid, exitCode)
	}

	return nil
}
