//go:build !windows

package processmgr

import (
	"errors"
	"syscall"
	"testing"
)

func assertReaped(t *testing.T, pid int) {
	t.Helper()
	var ws syscall.WaitStatus
	gotPID, err := syscall.Wait4(pid, &ws, syscall.WNOHANG, nil)
	if gotPID > 0 {
		t.Fatalf("pid %d still reapable (zombie): wait4 returned pid=%d ws=%v", pid, gotPID, ws)
	}
	if err != nil && !errors.Is(err, syscall.ECHILD) {
		t.Fatalf("unexpected wait4 error for pid %d: %v (want ECHILD or 0,nil)", pid, err)
	}
}
