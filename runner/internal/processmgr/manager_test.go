package processmgr

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestMain wires the test binary so it can act as a launcher subprocess when
// startDaemon invokes os.Executable(). Without this, every ModeDaemon test
// would have to build a separate launcher binary.
func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == LauncherSubcommand {
		RunLauncher()
		return
	}
	os.Exit(m.Run())
}

func newTestManager(t *testing.T) Manager {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return New(ctx, Options{})
}

func sleepCommand(t *testing.T) (string, []string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		return "cmd.exe", []string{"/C", "ping", "127.0.0.1", "-n", "60"}
	}
	return "/bin/sh", []string{"-c", "sleep 30"}
}

func trueCommand() (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd.exe", []string{"/C", "exit", "0"}
	}
	return "/bin/sh", []string{"-c", "exit 0"}
}

// waitForExit polls until p.Done() closes or timeout fires. Returns the exit
// info or fails the test.
func waitForExit(t *testing.T, p Handle, timeout time.Duration) ExitInfo {
	t.Helper()
	select {
	case <-p.Done():
		info, _ := p.ExitInfo()
		return info
	case <-time.After(timeout):
		t.Fatalf("process %s did not exit within %s", p.Owner(), timeout)
		return ExitInfo{}
	}
}

// assertReaped is platform-dispatched: on Unix it calls waitpid(WNOHANG) and
// expects ECHILD (the kernel has already reaped the child); on Windows there
// is no zombie state to assert against, so the helper is a no-op there. See
// assert_reaped_unix.go / assert_reaped_windows.go for the implementations.

func TestStart_RejectsInvalidSpec(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	if _, err := mgr.Start(ctx, Spec{Command: "x"}); err == nil {
		t.Fatal("missing Owner should be rejected")
	}
	if _, err := mgr.Start(ctx, Spec{Owner: "test"}); err == nil {
		t.Fatal("missing Command should be rejected")
	}
}

func TestModeNormal_NaturalExit_NoZombie(t *testing.T) {
	mgr := newTestManager(t)
	cmd, args := trueCommand()

	p, err := mgr.Start(context.Background(), Spec{
		Owner: "test:natural", Command: cmd, Args: args, Mode: ModeNormal,
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	pid := p.PID()
	waitForExit(t, p, 5*time.Second)
	assertReaped(t, pid)

	if list := mgr.List(); len(list) != 0 {
		t.Fatalf("expected registry empty after reap, got %d entries", len(list))
	}
}

func TestModeNormal_Stop_KillsAndReaps(t *testing.T) {
	mgr := newTestManager(t)
	cmd, args := sleepCommand(t)

	p, err := mgr.Start(context.Background(), Spec{
		Owner: "test:stop", Command: cmd, Args: args, Mode: ModeNormal,
		StopTimeout: 500 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	pid := p.PID()
	if !p.Alive() {
		t.Fatal("process should be alive immediately after Start")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := p.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	waitForExit(t, p, 2*time.Second)
	assertReaped(t, pid)
}

func TestStopAll_LeavesDaemonsAlone(t *testing.T) {
	mgr := newTestManager(t)
	cmd, args := sleepCommand(t)

	normalProc, err := mgr.Start(context.Background(), Spec{
		Owner: "test:normal", Command: cmd, Args: args, Mode: ModeNormal,
		StopTimeout: 500 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Start normal: %v", err)
	}

	daemonBin := writeDaemonStub(t)
	daemonProc, err := mgr.Start(context.Background(), Spec{
		Owner: "test:daemon", Command: daemonBin, Mode: ModeDaemon,
		StopTimeout: 1 * time.Second,
	})
	if err != nil {
		t.Fatalf("Start daemon: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := mgr.StopAll(ctx); err != nil {
		t.Fatalf("StopAll: %v", err)
	}

	if normalProc.Alive() {
		t.Fatal("normal process should be dead after StopAll")
	}
	if !daemonProc.Alive() {
		t.Fatal("daemon should still be alive after StopAll (StopAll skips daemons)")
	}

	if err := mgr.StopDaemons(ctx); err != nil {
		t.Fatalf("StopDaemons: %v", err)
	}
	if daemonProc.Alive() {
		t.Fatal("daemon should be dead after explicit StopDaemons")
	}
}

// writeDaemonStub creates a small shell script that sleeps long enough for
// the test to exercise daemon lifecycle. ModeDaemon needs a separate binary
// the launcher can exec; the test binary itself would loop into the launcher
// branch and exit immediately.
func writeDaemonStub(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("daemon stub for Windows not implemented in this test helper")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "daemon-stub.sh")
	script := strings.Join([]string{
		"#!/bin/sh",
		"trap 'exit 0' TERM",
		"sleep 60 &",
		"wait",
	}, "\n")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write daemon stub: %v", err)
	}
	return path
}
