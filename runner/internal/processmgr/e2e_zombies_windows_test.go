//go:build windows

package processmgr

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// writeDaemonStubWindows produces a .cmd script that loops until terminated,
// giving Windows daemon tests something to keep alive across Stop semantics.
// Windows has no "zombie" state — kernel objects are refcounted — so the
// Windows e2e suite is narrower: we verify start → detach (DETACHED_PROCESS
// path in launcher_windows.go) → Stop actually kills the process.
func writeDaemonStubWindows(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "daemon-stub.cmd")
	script := "@echo off\r\n:loop\r\nping -n 60 127.0.0.1 >nul\r\ngoto loop\r\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write daemon stub: %v", err)
	}
	return path
}

// TestModeDaemon_Windows_StartStop verifies the Windows launcher path
// (CREATE_NEW_PROCESS_GROUP + DETACHED_PROCESS) is wired correctly. Before
// the processmgr refactor, Windows daemons used os.StartProcess + Release
// the same way Unix did; even though Windows doesn't leak zombies, the
// detach mechanism was untested in CI.
func TestModeDaemon_Windows_StartStop(t *testing.T) {
	mgr := New(context.Background(), Options{
		DaemonAlivePoll: 100 * time.Millisecond,
	})
	stub := writeDaemonStubWindows(t)

	p, err := mgr.Start(context.Background(), Spec{
		Owner: "test:win-daemon", Command: "cmd.exe",
		Args: []string{"/c", stub}, Mode: ModeDaemon,
	})
	if err != nil {
		t.Fatalf("Start daemon on Windows: %v", err)
	}
	pid := p.PID()
	if pid <= 0 {
		t.Fatalf("daemon PID not reported: %d", pid)
	}

	// Verify the daemon is running by opening a process handle.
	if !windowsProcessAlive(pid) {
		t.Fatalf("daemon pid=%d not alive immediately after Start", pid)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := p.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	// Give Windows a beat to actually tear the process down.
	time.Sleep(300 * time.Millisecond)
	if windowsProcessAlive(pid) {
		t.Fatalf("daemon pid=%d still alive after Stop", pid)
	}
}

// TestE2E_Windows_LauncherSubcommandReachable mirrors the Unix smoke test:
// re-execing the test binary with LauncherSubcommand must complete without
// error, proving TestMain wires the launcher branch.
func TestE2E_Windows_LauncherSubcommandReachable(t *testing.T) {
	selfPath, err := os.Executable()
	if err != nil {
		t.Fatalf("self path: %v", err)
	}

	// cmd.exe /c exit 0 is the Windows equivalent of /bin/true.
	stopwatch := time.Now()
	cmd := exec.Command(selfPath, LauncherSubcommand, "cmd.exe", "/c", "exit 0")
	if err := cmd.Run(); err != nil {
		t.Fatalf("launcher invocation failed after %v: %v", time.Since(stopwatch), err)
	}
}

func windowsProcessAlive(pid int) bool {
	const PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
	h, err := syscall.OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer syscall.CloseHandle(h)
	var code uint32
	if err := syscall.GetExitCodeProcess(h, &code); err != nil {
		return false
	}
	// STILL_ACTIVE = 259. Any other code means the process exited.
	return code == 259
}
