package terminal

import (
	"os"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/testutil"
)

// --- Test Terminal Start and PTY operations ---

func TestTerminalStartSuccess(t *testing.T) {
	outputReceived := make(chan bool, 1)
	exitReceived := make(chan int, 1)

	cmd, args := testutil.EchoCommand("hello")
	opts := Options{
		Command: cmd,
		Args:    args,
		WorkDir: os.TempDir(),
		OnOutput: func(data []byte) {
			select {
			case outputReceived <- true:
			default:
			}
		},
		OnExit: func(code int) {
			exitReceived <- code
		},
	}

	term, err := New(opts)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	err = term.Start()
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}

	// Wait for output or timeout
	select {
	case <-outputReceived:
		// Good - got output
	case <-time.After(2 * time.Second):
		// May timeout if output is too fast
	}

	// Wait for exit
	select {
	case code := <-exitReceived:
		if code != 0 {
			t.Logf("Exit code: %d", code)
		}
	case <-time.After(3 * time.Second):
		t.Log("Timeout waiting for exit")
	}

	term.Stop()
}

func TestTerminalWriteSuccess(t *testing.T) {
	cmd, args := testutil.CatCommand()
	opts := Options{
		Command:  cmd,
		Args:     args,
		WorkDir:  os.TempDir(),
		OnOutput: func(data []byte) {},
	}

	term, err := New(opts)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	err = term.Start()
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}
	defer term.Stop()

	// Write data
	err = term.Write([]byte("test input\n"))
	if err != nil {
		t.Errorf("Write error: %v", err)
	}
}

func TestTerminalResizeSuccess(t *testing.T) {
	cmd, args := testutil.SleepCommand(5)
	opts := Options{
		Command: cmd,
		Args:    args,
		WorkDir: os.TempDir(),
	}

	term, err := New(opts)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	err = term.Start()
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}
	defer term.Stop()

	// Resize
	err = term.Resize(40, 120)
	if err != nil {
		t.Errorf("Resize error: %v", err)
	}
}

func TestTerminalPIDRunning(t *testing.T) {
	cmd, args := testutil.SleepCommand(5)
	opts := Options{
		Command: cmd,
		Args:    args,
		WorkDir: os.TempDir(),
	}

	term, err := New(opts)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	err = term.Start()
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}
	defer term.Stop()

	pid := term.PID()
	if pid <= 0 {
		t.Errorf("PID should be positive, got %d", pid)
	}
}

func TestTerminalStopRunning(t *testing.T) {
	cmd, args := testutil.SleepCommand(60)
	opts := Options{
		Command: cmd,
		Args:    args,
		WorkDir: os.TempDir(),
	}

	term, err := New(opts)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	err = term.Start()
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}

	// Stop should work
	term.Stop()

	// Wait a bit for waitExit goroutine to complete
	time.Sleep(100 * time.Millisecond)

	// Verify closed flag using thread-safe method
	if !term.IsClosed() {
		t.Error("closed flag should be true")
	}
}

func TestTerminalWriteClosed(t *testing.T) {
	cmd, args := testutil.SleepCommand(60)
	opts := Options{
		Command: cmd,
		Args:    args,
		WorkDir: os.TempDir(),
	}

	term, err := New(opts)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	err = term.Start()
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}

	term.Stop()

	// Write should fail after close
	err = term.Write([]byte("test"))
	if err == nil {
		t.Error("Write after close should error")
	}
}

func TestTerminalResizeClosed(t *testing.T) {
	cmd, args := testutil.SleepCommand(60)
	opts := Options{
		Command: cmd,
		Args:    args,
		WorkDir: os.TempDir(),
	}

	term, err := New(opts)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	err = term.Start()
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}

	term.Stop()

	// Resize should fail after close
	err = term.Resize(40, 120)
	if err == nil {
		t.Error("Resize after close should error")
	}
}

// --- Test Terminal with exit code ---

func TestTerminalExitCode(t *testing.T) {
	exitCode := -1
	exitReceived := make(chan bool, 1)

	cmd, args := testutil.FalseCommand()
	opts := Options{
		Command: cmd, // returns exit code 1
		Args:    args,
		WorkDir: os.TempDir(),
		OnExit: func(code int) {
			exitCode = code
			exitReceived <- true
		},
	}

	term, err := New(opts)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	err = term.Start()
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}

	// Wait for exit
	select {
	case <-exitReceived:
		if exitCode != 1 {
			t.Errorf("exit code: got %v, want 1", exitCode)
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for exit")
	}
}
