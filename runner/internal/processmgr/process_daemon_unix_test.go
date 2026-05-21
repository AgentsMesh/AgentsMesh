//go:build !windows

package processmgr

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestModeDaemon_DetachesFromRunner(t *testing.T) {
	mgr := newTestManager(t)
	daemonBin := writeDaemonStub(t)

	p, err := mgr.Start(context.Background(), Spec{
		Owner: "test:detach", Command: daemonBin, Mode: ModeDaemon,
		StopTimeout: 1 * time.Second,
	})
	if err != nil {
		t.Fatalf("Start daemon: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = p.Stop(ctx)
	})

	daemonPID := p.PID()
	if daemonPID <= 0 {
		t.Fatalf("daemon PID not reported: %d", daemonPID)
	}

	// Wait briefly for the launcher to exit and the kernel to reparent the
	// daemon to init. 500ms is generous — in practice this is microseconds.
	time.Sleep(500 * time.Millisecond)

	ppid := readPPID(t, daemonPID)
	if ppid == os.Getpid() {
		t.Fatalf("daemon ppid is still the runner (%d) — detachment failed", ppid)
	}
	if ppid != 1 {
		// macOS launchd takes over orphans the same way as Linux init.
		// Some sandboxed environments may use a different reaper PID, so we
		// only require that ppid is NOT the runner — that is the property
		// the zombie fix depends on.
		t.Logf("daemon ppid=%d (not 1, but not runner — acceptable)", ppid)
	}
}

func TestModeDaemon_StopActuallyKills(t *testing.T) {
	mgr := newTestManager(t)
	daemonBin := writeDaemonStub(t)

	p, err := mgr.Start(context.Background(), Spec{
		Owner: "test:stop-daemon", Command: daemonBin, Mode: ModeDaemon,
		StopTimeout: 1 * time.Second,
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	daemonPID := p.PID()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := p.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	if syscall.Kill(daemonPID, 0) == nil {
		t.Fatalf("daemon pid %d still alive after Stop", daemonPID)
	}
}

// TestModeDaemon_NaturalDeath_ClosesDone guards the LSP fix from Phase 3.9:
// when a daemon dies of natural causes (not via Stop), the monitor goroutine
// must observe the death and close Done so callers blocked on <-handle.Done()
// wake up. Without the monitor, Done would only close on explicit Stop and
// callers would deadlock waiting for a self-terminating daemon.
func TestModeDaemon_NaturalDeath_ClosesDone(t *testing.T) {
	mgr := New(context.Background(), Options{
		DaemonAlivePoll: 100 * time.Millisecond, // tighten for test latency
	})
	daemonBin := writeDaemonStub(t)

	p, err := mgr.Start(context.Background(), Spec{
		Owner: "test:natural-death", Command: daemonBin, Mode: ModeDaemon,
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Externally kill the daemon, bypassing processmgr's Stop path entirely.
	// This simulates the daemon crashing or being killed by an operator.
	if err := syscall.Kill(p.PID(), syscall.SIGKILL); err != nil {
		t.Fatalf("external SIGKILL: %v", err)
	}

	// Done must close within a few poll intervals.
	select {
	case <-p.Done():
	case <-time.After(2 * time.Second):
		t.Fatalf("Done() did not close within 2s after daemon was externally killed")
	}

	if p.Alive() {
		t.Fatal("Alive() should be false after Done closes")
	}
	if _, ok := p.ExitInfo(); !ok {
		t.Fatal("ExitInfo should be set after Done closes")
	}
}

func TestReaper_DoesNotInterfereWithProcessmgrChildren(t *testing.T) {
	mgr := newTestManager(t)
	resetMetricsForTest()

	cmd, args := trueCommand()
	p, err := mgr.Start(context.Background(), Spec{
		Owner: "test:reaper", Command: cmd, Args: args, Mode: ModeNormal,
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitForExit(t, p, 5*time.Second)

	// The reaper runs every 30 seconds, but reapOrphans is also exported
	// so we can call it directly. It should find no orphans because the
	// child was reaped by reapLoop.
	if n := reapOrphans(); n != 0 {
		t.Fatalf("reapOrphans found %d untracked zombies after well-managed child", n)
	}
	if zombieReaped.Load() != 0 {
		t.Fatalf("zombie counter is %d, expected 0", zombieReaped.Load())
	}
}

func readPPID(t *testing.T, pid int) int {
	t.Helper()
	// /proc on Linux exposes ppid directly; on macOS we shell out to ps.
	// Both fall back to the other approach if the first is unavailable.
	if data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat"); err == nil {
		return parseStatPPID(t, string(data))
	}

	cmd, args := psPPIDCommand(pid)
	out, err := runCmd(cmd, args)
	if err != nil {
		t.Fatalf("read ppid via ps: %v", err)
	}
	v, err := strconv.Atoi(out)
	if err != nil {
		t.Fatalf("parse ps ppid output %q: %v", out, err)
	}
	return v
}

func parseStatPPID(t *testing.T, stat string) int {
	t.Helper()
	// /proc/<pid>/stat: "pid (comm) state ppid ..." — comm can contain
	// spaces or parens, so split on the closing paren.
	idx := -1
	for i := len(stat) - 1; i >= 0; i-- {
		if stat[i] == ')' {
			idx = i
			break
		}
	}
	if idx < 0 {
		t.Fatalf("malformed /proc/stat: %s", stat)
	}
	fields := splitFields(stat[idx+1:])
	if len(fields) < 3 {
		t.Fatalf("not enough fields in /proc/stat: %s", stat)
	}
	v, err := strconv.Atoi(fields[1])
	if err != nil {
		t.Fatalf("parse ppid: %v", err)
	}
	return v
}

func runCmd(name string, args []string) (string, error) {
	cmd := exec.Command(name, args...) //nolint:gosec // test helper
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

func psPPIDCommand(pid int) (string, []string) {
	return "ps", []string{"-o", "ppid=", "-p", strconv.Itoa(pid)}
}

func splitFields(s string) []string {
	var out []string
	cur := ""
	for _, ch := range s {
		if ch == ' ' || ch == '\t' || ch == '\n' {
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
		} else {
			cur += string(ch)
		}
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}
