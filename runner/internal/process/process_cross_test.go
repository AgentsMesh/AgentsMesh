package process

import (
	"os"
	"testing"
)

// Cross-platform tests for the Inspector interface.
// Platform-specific implementation tests are in process_test.go (darwin only).

func TestDefaultInspectorReturnsNonNil(t *testing.T) {
	inspector := DefaultInspector()
	if inspector == nil {
		t.Fatal("DefaultInspector returned nil")
	}
}

func TestInspectorIsRunningCurrentProcess(t *testing.T) {
	inspector := DefaultInspector()
	pid := os.Getpid()
	if !inspector.IsRunning(pid) {
		t.Error("IsRunning should return true for current process")
	}
}

func TestInspectorIsRunningInvalidPid(t *testing.T) {
	inspector := DefaultInspector()
	if inspector.IsRunning(999999998) {
		t.Error("IsRunning should return false for non-existent PID")
	}
}

func TestInspectorGetProcessNameCurrentProcess(t *testing.T) {
	inspector := DefaultInspector()
	pid := os.Getpid()
	name := inspector.GetProcessName(pid)
	if name == "" {
		t.Error("GetProcessName for current process should not be empty")
	}
}

func TestInspectorGetProcessNameInvalidPid(t *testing.T) {
	inspector := DefaultInspector()
	name := inspector.GetProcessName(-1)
	if name != "" {
		t.Errorf("GetProcessName for invalid PID: got %q, want empty", name)
	}
}

func TestInspectorGetStateCurrentProcess(t *testing.T) {
	inspector := DefaultInspector()
	pid := os.Getpid()
	state := inspector.GetState(pid)
	// On Unix: returns "S" (sleeping) or "R" (running).
	// On Windows: returns "R" for any running process.
	// Either way, should not be empty for a live process.
	if state == "" {
		t.Error("GetState for current process should not be empty")
	}
}

func TestInspectorGetChildProcesses(t *testing.T) {
	inspector := DefaultInspector()
	pid := os.Getpid()
	// Should not panic; result may be nil or empty.
	_ = inspector.GetChildProcesses(pid)
}

func TestInspectorHasOpenFilesInvalidPid(t *testing.T) {
	inspector := DefaultInspector()
	if inspector.HasOpenFiles(-1) {
		t.Error("HasOpenFiles should return false for invalid PID")
	}
}
