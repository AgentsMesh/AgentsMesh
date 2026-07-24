package terminal

import (
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/testutil"
)

// --- Test SetOutputHandler and SetExitHandler ---

func TestSetOutputHandler(t *testing.T) {
	// Use shell script with a short delay to ensure output is captured reliably in CI.
	// Windows has no 'sleep' command; use 'ping -n 2 127.0.0.1 >nul' as a ~1s delay.
	var script string
	if runtime.GOOS == "windows" {
		script = "echo test & ping -n 2 127.0.0.1 >nul"
	} else {
		script = "echo test && sleep 0.1"
	}
	cmd, args := testutil.ShellScript(script)
	opts := Options{
		Command: cmd,
		Args:    args,
	}

	term, err := New(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	outputReceived := make(chan struct{})
	var received []byte
	var mu sync.Mutex

	term.SetOutputHandler(func(data []byte) {
		mu.Lock()
		received = append(received, data...)
		mu.Unlock()
		select {
		case <-outputReceived:
		default:
			close(outputReceived)
		}
	})

	err = term.Start()
	if err != nil {
		t.Fatalf("failed to start terminal: %v", err)
	}

	// Wait for output with timeout
	select {
	case <-outputReceived:
		// Good - received output
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for output")
	}

	term.Stop()

	// Should have received some output
	mu.Lock()
	receivedLen := len(received)
	mu.Unlock()
	if receivedLen == 0 {
		t.Error("expected to receive output from echo command")
	}
}

func TestSetExitHandler(t *testing.T) {
	cmd, args := testutil.TrueCommand()
	opts := Options{
		Command: cmd, // Command that exits immediately with code 0
		Args:    args,
	}

	term, err := New(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	exitCode := -1
	exitCalled := make(chan struct{})
	term.SetExitHandler(func(code int) {
		exitCode = code
		close(exitCalled)
	})

	err = term.Start()
	if err != nil {
		t.Fatalf("failed to start terminal: %v", err)
	}

	// Wait for exit handler to be called
	select {
	case <-exitCalled:
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for exit handler")
		term.Stop()
	}
}

func TestSetOutputHandlerNil(t *testing.T) {
	cmd, args := testutil.EchoCommand("hello")
	opts := Options{
		Command: cmd,
		Args:    args,
	}

	term, err := New(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should not panic when setting nil handler
	term.SetOutputHandler(nil)

	// Terminal should still work
	err = term.Start()
	if err != nil {
		t.Fatalf("failed to start terminal: %v", err)
	}

	time.Sleep(200 * time.Millisecond)
	term.Stop()
}

func TestSetExitHandlerNil(t *testing.T) {
	cmd, args := testutil.TrueCommand()
	opts := Options{
		Command: cmd,
		Args:    args,
	}

	term, err := New(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should not panic when setting nil handler
	term.SetExitHandler(nil)

	// Terminal should still work
	err = term.Start()
	if err != nil {
		t.Fatalf("failed to start terminal: %v", err)
	}

	time.Sleep(200 * time.Millisecond)
	term.Stop()
}

func TestSetHandlersBeforeStart(t *testing.T) {
	// Use shell script instead of echo to ensure we have time to capture output.
	// echo may complete too fast in CI environments with race detector.
	var script string
	if runtime.GOOS == "windows" {
		script = "echo hello & ping -n 2 127.0.0.1 >nul"
	} else {
		script = "echo hello && sleep 0.1"
	}
	cmd, args := testutil.ShellScript(script)
	opts := Options{
		Command:  cmd,
		Args:     args,
		OnOutput: func([]byte) { /* initial handler */ },
		OnExit:   func(int) { /* initial handler */ },
	}

	term, err := New(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Override handlers before Start
	outputReceived := make(chan struct{})
	exitReceived := make(chan struct{})

	term.SetOutputHandler(func(data []byte) {
		select {
		case <-outputReceived:
		default:
			close(outputReceived)
		}
	})
	term.SetExitHandler(func(code int) {
		close(exitReceived)
	})

	err = term.Start()
	if err != nil {
		t.Fatalf("failed to start terminal: %v", err)
	}

	// Wait for both handlers to be called with longer timeout for CI
	select {
	case <-outputReceived:
		// Good
	case <-time.After(5 * time.Second):
		t.Error("timeout waiting for output handler")
	}

	select {
	case <-exitReceived:
		// Good
	case <-time.After(5 * time.Second):
		t.Error("timeout waiting for exit handler")
		term.Stop()
	}
}
