//go:build windows

package process

import (
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

const (
	thSnapshotProcess          = 0x00000002
	processQueryLimitedInfo    = 0x1000
	processExitCodeStillActive = 259
	maxPath                    = 260
)

var (
	kernel32DLL                  = syscall.NewLazyDLL("kernel32.dll")
	procCreateToolhelp32Snapshot = kernel32DLL.NewProc("CreateToolhelp32Snapshot")
	procProcess32First           = kernel32DLL.NewProc("Process32FirstW")
	procProcess32Next            = kernel32DLL.NewProc("Process32NextW")
	procOpenProcess              = kernel32DLL.NewProc("OpenProcess")
	procGetExitCodeProcess       = kernel32DLL.NewProc("GetExitCodeProcess")
	procGetProcessHandleCount    = kernel32DLL.NewProc("GetProcessHandleCount")
)

// processEntry32W mirrors the Windows PROCESSENTRY32W struct.
type processEntry32W struct {
	Size            uint32
	CntUsage        uint32
	ProcessID       uint32
	DefaultHeapID   uintptr
	ModuleID        uint32
	CntThreads      uint32
	ParentProcessID uint32
	PriClassBase    int32
	Flags           uint32
	ExeFile         [maxPath]uint16
}

// windowsInspector implements Inspector for Windows
// using the Toolhelp32 snapshot API.
type windowsInspector struct{}

// DefaultInspector returns the default inspector for Windows.
func DefaultInspector() Inspector {
	return &windowsInspector{}
}

// snapshotProcesses takes a Toolhelp32 snapshot and returns all process entries.
func snapshotProcesses() ([]processEntry32W, error) {
	handle, _, err := procCreateToolhelp32Snapshot.Call(uintptr(thSnapshotProcess), 0)
	if handle == uintptr(syscall.InvalidHandle) {
		return nil, err
	}
	defer syscall.CloseHandle(syscall.Handle(handle))

	var entry processEntry32W
	entry.Size = uint32(unsafe.Sizeof(entry))

	ret, _, err := procProcess32First.Call(handle, uintptr(unsafe.Pointer(&entry)))
	if ret == 0 {
		return nil, err
	}

	var entries []processEntry32W
	entries = append(entries, entry)

	for {
		entry.Size = uint32(unsafe.Sizeof(entry))
		ret, _, _ = procProcess32Next.Call(handle, uintptr(unsafe.Pointer(&entry)))
		if ret == 0 {
			break
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// exeName extracts the base executable name (without .exe suffix) from a processEntry.
func exeName(entry *processEntry32W) string {
	name := syscall.UTF16ToString(entry.ExeFile[:])
	name = filepath.Base(name)
	// Strip .exe suffix for matching (e.g. "claude.exe" → "claude")
	name = strings.TrimSuffix(name, ".exe")
	return strings.ToLower(name)
}

// GetChildProcesses returns PIDs of direct child processes.
func (i *windowsInspector) GetChildProcesses(pid int) []int {
	entries, err := snapshotProcesses()
	if err != nil {
		return nil
	}

	var children []int
	for idx := range entries {
		if entries[idx].ParentProcessID == uint32(pid) && entries[idx].ProcessID != uint32(pid) {
			children = append(children, int(entries[idx].ProcessID))
		}
	}
	return children
}

// GetProcessName returns the base executable name (without .exe) of a process.
func (i *windowsInspector) GetProcessName(pid int) string {
	entries, err := snapshotProcesses()
	if err != nil {
		return ""
	}

	for idx := range entries {
		if entries[idx].ProcessID == uint32(pid) {
			return exeName(&entries[idx])
		}
	}
	return ""
}

// IsRunning checks if a process is still alive.
func (i *windowsInspector) IsRunning(pid int) bool {
	handle, _, _ := procOpenProcess.Call(
		uintptr(processQueryLimitedInfo),
		0,
		uintptr(pid),
	)
	if handle == 0 {
		return false
	}
	defer syscall.CloseHandle(syscall.Handle(handle))

	var exitCode uint32
	ret, _, _ := procGetExitCodeProcess.Call(handle, uintptr(unsafe.Pointer(&exitCode)))
	if ret == 0 {
		return false
	}
	return exitCode == processExitCodeStillActive
}

// GetState returns a Unix-compatible process state string.
// Windows does not expose per-process scheduling states (R, S, D, T, Z) like
// /proc/<pid>/stat on Linux. The Win32 API only lets us check if a process
// is running or exited. As a result:
//   - "R" is returned for any running process (cannot distinguish sleeping vs CPU-running).
//   - "" is returned for exited or inaccessible processes.
//
// Callers (e.g., monitor_check.go) treat "R" as "active" which is correct —
// the only false-positive scenario would be a truly zombie process, which
// Windows handles by auto-reaping via the kernel.
func (i *windowsInspector) GetState(pid int) string {
	if i.IsRunning(pid) {
		return "R"
	}
	return ""
}

// HasOpenFiles checks if a process likely has active I/O.
// On Windows, uses GetProcessHandleCount to approximate file descriptor usage.
// A process with many open handles (files, pipes, sockets, registry keys) is
// likely performing active I/O. The threshold of 100 is a heuristic: a typical
// idle process has ~30-60 handles, while one doing file I/O has significantly more.
func (i *windowsInspector) HasOpenFiles(pid int) bool {
	handle, _, _ := procOpenProcess.Call(
		uintptr(processQueryLimitedInfo),
		0,
		uintptr(pid),
	)
	if handle == 0 {
		return false
	}
	defer syscall.CloseHandle(syscall.Handle(handle))

	var handleCount uint32
	ret, _, _ := procGetProcessHandleCount.Call(handle, uintptr(unsafe.Pointer(&handleCount)))
	if ret == 0 {
		return false
	}
	return handleCount > 100
}
