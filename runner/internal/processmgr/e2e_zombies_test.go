//go:build !windows

package processmgr

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestE2E_ManyDaemonsLeakNoZombies is the headline guard for the zombie fix.
// It exercises the full ModeDaemon path end-to-end: real fork via the
// __processmgr_launcher__ subcommand (the test binary itself, wired up in
// TestMain), real SIGKILL, real ppid reassignment to init(1). The assertion
// is the property that motivated this whole package: after N create/destroy
// cycles, the runner's process table contains zero <defunct> entries.
//
// Before the processmgr refactor, this scenario leaked ~1 zombie per cycle
// (the old daemon_start_unix.go path called proc.Release without ever Wait'ing
// the spawned process). The test fails fast if anyone reintroduces that
// pattern.
func TestE2E_ManyDaemonsLeakNoZombies(t *testing.T) {
	const cycles = 20

	mgr := New(context.Background(), Options{
		DaemonAlivePoll: 50 * time.Millisecond,
	})
	daemonBin := writeDaemonStub(t)

	for i := 0; i < cycles; i++ {
		owner := "e2e:zombies-" + strconv.Itoa(i)
		p, err := mgr.Start(context.Background(), Spec{
			Owner: owner, Command: daemonBin, Mode: ModeDaemon,
		})
		if err != nil {
			t.Fatalf("cycle %d Start: %v", i, err)
		}

		if ppid := readPPID(t, p.PID()); ppid == os.Getpid() {
			t.Fatalf("cycle %d daemon ppid is the test process (%d) — detachment failed", i, ppid)
		}

		stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		if err := p.Stop(stopCtx); err != nil {
			cancel()
			t.Fatalf("cycle %d Stop: %v", i, err)
		}
		cancel()
		<-p.Done()
	}

	// Give the kernel a beat to settle reparented exits.
	time.Sleep(200 * time.Millisecond)

	if zombies := countOwnZombies(t); zombies > 0 {
		t.Fatalf("after %d daemon cycles, found %d zombie(s) under the test process — processmgr is leaking", cycles, zombies)
	}

	if z := zombieReaped.Load(); z > 0 {
		t.Fatalf("reaper caught %d untracked zombies — some Start path bypassed processmgr", z)
	}
}

// TestE2E_LauncherSubcommandReachable verifies the test binary really wires
// LauncherSubcommand in TestMain. Without this, every ModeDaemon test would
// silently fall back to launching the test binary as a normal subprocess
// (which would then run TestMain → m.Run again → spawn an explosion of
// subtests). A green test here proves TestMain's launcher branch fires when
// invoked with the magic argv.
func TestE2E_LauncherSubcommandReachable(t *testing.T) {
	selfPath, err := os.Executable()
	if err != nil {
		t.Fatalf("self path: %v", err)
	}

	// `true` lives in different absolute paths across Unix flavors
	// (/bin/true on Linux, /usr/bin/true on macOS). Resolve at runtime.
	truePath, err := exec.LookPath("true")
	if err != nil {
		t.Fatalf("locate true binary: %v", err)
	}

	cmd := exec.Command(selfPath, LauncherSubcommand, truePath)
	var stderr strings.Builder
	cmd.Stderr = &stderr

	done := make(chan error, 1)
	go func() { done <- cmd.Run() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("launcher invocation failed: %v\nstderr: %s", err, stderr.String())
		}
	case <-time.After(3 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("launcher did not exit within 3s — TestMain's launcher branch is not wired")
	}
}

// countOwnZombies returns the number of <defunct> processes whose ppid is
// the current test process. Uses ps which is portable across Linux + macOS;
// /proc parsing would be Linux-only.
func countOwnZombies(t *testing.T) int {
	t.Helper()
	pid := os.Getpid()

	out, err := runCmd("ps", []string{"-A", "-o", "ppid=,stat="})
	if err != nil {
		t.Fatalf("ps: %v", err)
	}
	count := 0
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		ppid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		if ppid == pid && strings.HasPrefix(fields[1], "Z") {
			count++
		}
	}
	return count
}
