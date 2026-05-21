package processmgr

import (
	"context"
	"io"
	"runtime"
	"testing"
	"time"

	"github.com/creack/pty"
)

func TestOptions_WithDefaults_FillsZeroValues(t *testing.T) {
	got := Options{}.withDefaults()
	if got.ReaperInterval == 0 || got.DefaultStopTimeout == 0 ||
		got.DaemonAlivePoll == 0 || got.LauncherStartTimeout == 0 {
		t.Fatalf("withDefaults left a zero value: %+v", got)
	}
}

func TestOptions_PreservesCustomValues(t *testing.T) {
	custom := Options{
		ReaperInterval:       7 * time.Second,
		DefaultStopTimeout:   333 * time.Millisecond,
		DaemonAlivePoll:      11 * time.Millisecond,
		LauncherStartTimeout: 99 * time.Millisecond,
	}
	got := custom.withDefaults()
	if got != custom {
		t.Fatalf("withDefaults mutated custom values: got %+v want %+v", got, custom)
	}
}

func TestOptions_DefaultStopTimeoutApplied(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mgr := New(ctx, Options{DefaultStopTimeout: 200 * time.Millisecond})

	cmd, args := sleepCommand(t)
	p, err := mgr.Start(context.Background(), Spec{
		Owner: "test:opts-stop", Command: cmd, Args: args, Mode: ModeNormal,
		// Spec.StopTimeout intentionally unset — manager default must apply.
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer stopCancel()

	start := time.Now()
	_ = p.Stop(stopCtx)
	elapsed := time.Since(start)

	// 200ms SIGTERM grace + brief SIGKILL grace. If the manager's default
	// stop timeout is ignored, this would take the package default (5s)
	// before SIGKILL escalation, far exceeding 2s.
	if elapsed > 2*time.Second {
		t.Fatalf("Stop took %v, expected ≤ 2s — manager DefaultStopTimeout was ignored", elapsed)
	}
}

func TestModePTY_LifecycleAndExitInfo(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("ModePTY not implemented on Windows")
	}
	mgr := newTestManager(t)

	p, err := mgr.Start(context.Background(), Spec{
		Owner:   "test:pty",
		Command: "/bin/sh",
		Args:    []string{"-c", "echo hello; exit 0"},
		Mode:    ModePTY,
		PTYSize: &pty.Winsize{Rows: 24, Cols: 80},
	})
	if err != nil {
		t.Fatalf("Start PTY: %v", err)
	}
	if p.PTY() == nil {
		t.Fatal("PTY() returned nil for ModePTY handle")
	}

	// Read until EOF; the child's exit closes the PTY slave which the master
	// observes as EOF.
	buf, _ := io.ReadAll(p.PTY())
	if len(buf) == 0 {
		t.Fatal("expected output from PTY child, got nothing")
	}

	waitForExit(t, p, 3*time.Second)
	info, ok := p.ExitInfo()
	if !ok {
		t.Fatal("ExitInfo not set after Done")
	}
	if info.Code != 0 {
		t.Errorf("expected exit code 0, got %d", info.Code)
	}
}

func TestPipeStdin_RoundtripsThroughChild(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("cat not portable to Windows; covered indirectly by acp/mcp tests")
	}
	mgr := newTestManager(t)

	p, err := mgr.Start(context.Background(), Spec{
		Owner:      "test:pipe",
		Command:    "/bin/cat",
		Mode:       ModeNormal,
		PipeStdin:  true,
		PipeStdout: true,
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	stdin := p.StdinWriter()
	stdout := p.StdoutReader()
	if stdin == nil || stdout == nil {
		t.Fatalf("Pipe* set but writer/reader is nil")
	}

	payload := []byte("processmgr-pipe-roundtrip\n")
	if _, err := stdin.Write(payload); err != nil {
		t.Fatalf("write stdin: %v", err)
	}

	got := make([]byte, len(payload))
	if _, err := io.ReadFull(stdout, got); err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("roundtrip mismatch: got %q want %q", got, payload)
	}

	// Closing stdin signals EOF to cat, which then exits cleanly.
	_ = stdin.Close()
	waitForExit(t, p, 3*time.Second)
}

func TestPipe_RejectsConflictingStdin(t *testing.T) {
	mgr := newTestManager(t)
	_, err := mgr.Start(context.Background(), Spec{
		Owner:     "test:pipe-conflict",
		Command:   "/bin/sh",
		Mode:      ModeNormal,
		PipeStdin: true,
		Stdin:     readerNop{},
	})
	if err == nil {
		t.Fatal("expected error when both PipeStdin and Stdin are set")
	}
}

type readerNop struct{}

func (readerNop) Read([]byte) (int, error) { return 0, io.EOF }
