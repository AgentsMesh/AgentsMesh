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
	kernel32DLL                = syscall.NewLazyDLL("kernel32.dll")
	procCreateToolhelp32Snapshot = kernel32DLL.NewProc("CreateToolhelp32Snapshot")
	procProcess32First         = kernel32DLL.NewProc("Process32FirstW")
	procProcess32Next          = kernel32DLL.NewProc("Process32NextW")
	procOpenProcessInsp        = kernel32DLL.NewProc("OpenProcess")
	procGetExitCodeProcessInsp = kernel32DLL.NewProc("GetExitCodeProcess")
)

// processEntry32W mirrors the Windows PROCESSENTRY32W struct.
type processEntry32W struct {
	Size              uint32
	CntUsage          uint32
	ProcessID         uint32
	DefaultHeapID     uintptr
	ModuleID          uint32
	CntThreads        uint32
	ParentProcessID   uint32
	PriClassBase      int32
	Flags             uint32
	ExeFile           [maxPath]uint16
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
	handle, _, _ := procOpenProcessInsp.Call(
		uintptr(processQueryLimitedInfo),
		0,
		uintptr(pid),
	)
	if handle == 0 {
		return false
	}
	defer syscall.CloseHandle(syscall.Handle(handle))

	var exitCode uint32
	ret, _, _ := procGetExitCodeProcessInsp.Call(handle, uintptr(unsafe.Pointer(&exitCode)))
	if ret == 0 {
		return false
	}
	return exitCode == processExitCodeStillActive
}

// GetState returns a Unix-compatible process state string.
// Windows does not have direct equivalents to Unix process states (R, S, D, etc.).
// Returns "R" for running processes, empty string otherwise.
func (i *windowsInspector) GetState(pid int) string {
	if i.IsRunning(pid) {
		return "R"
	}
	return ""
}

// HasOpenFiles checks if a process likely has active I/O.
// On Windows, enumerating open handles requires elevated privileges (NtQuerySystemInformation).
// As a heuristic, we check if the process has child threads beyond its initial thread.
// This correlates with active I/O operations in practice.
func (i *windowsInspector) HasOpenFiles(pid int) bool {
	entries, err := snapshotProcesses()
	if err != nil {
		return false
	}

	for idx := range entries {
		if entries[idx].ProcessID == uint32(pid) {
			// A process doing I/O typically has more threads than its initial thread.
			// The threshold of 4 is a reasonable heuristic: main thread + I/O threads.
			return entries[idx].CntThreads > 4
		}
	}
	return false
}
