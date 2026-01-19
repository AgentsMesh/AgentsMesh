package terminal

import (
	"os"
	"testing"
	"time"
)

// --- Test Options and Terminal Struct ---

func TestOptionsStruct(t *testing.T) {
	opts := Options{
		Command:  "echo",
		Args:     []string{"hello"},
		WorkDir:  "/tmp",
		Env:      map[string]string{"KEY": "VALUE"},
		Rows:     24,
		Cols:     80,
		OnOutput: func([]byte) {},
		OnExit:   func(int) {},
	}

	if opts.Command != "echo" {
		t.Errorf("Command: got %v, want echo", opts.Command)
	}

	if opts.Rows != 24 {
		t.Errorf("Rows: got %v, want 24", opts.Rows)
	}

	if opts.Cols != 80 {
		t.Errorf("Cols: got %v, want 80", opts.Cols)
	}
}

func TestNewTerminal(t *testing.T) {
	opts := Options{
		Command: "echo",
		Args:    []string{"hello"},
	}

	term, err := New(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if term == nil {
		t.Fatal("New returned nil")
	}

	if term.cmd == nil {
		t.Error("cmd should not be nil")
	}
}

func TestNewTerminalEmptyCommand(t *testing.T) {
	opts := Options{
		Command: "",
	}

	_, err := New(opts)
	if err == nil {
		t.Error("expected error for empty command")
	}
}

func TestNewTerminalWithEnv(t *testing.T) {
	opts := Options{
		Command: "echo",
		Env:     map[string]string{"TEST_VAR": "test_value"},
	}

	term, err := New(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that environment is set
	envFound := false
	for _, e := range term.cmd.Env {
		if e == "TEST_VAR=test_value" {
			envFound = true
			break
		}
	}

	if !envFound {
		t.Error("environment variable should be set")
	}
}

func TestTerminalPIDNotStarted(t *testing.T) {
	opts := Options{
		Command: "echo",
	}

	term, _ := New(opts)

	// PID should be 0 before start
	if term.PID() != 0 {
		t.Errorf("PID before start: got %v, want 0", term.PID())
	}
}

func TestTerminalWriteNotStarted(t *testing.T) {
	opts := Options{
		Command: "echo",
	}

	term, _ := New(opts)

	err := term.Write([]byte("test"))
	if err == nil {
		t.Error("expected error when writing to not started terminal")
	}
}

func TestTerminalResizeNotStarted(t *testing.T) {
	opts := Options{
		Command: "echo",
	}

	term, _ := New(opts)

	err := term.Resize(24, 80)
	if err == nil {
		t.Error("expected error when resizing not started terminal")
	}
}

func TestTerminalStopNotStarted(t *testing.T) {
	opts := Options{
		Command: "echo",
	}

	term, _ := New(opts)

	// Should not panic when stopping not started terminal
	term.Stop()

	// Second stop should also not panic
	term.Stop()
}

func TestTerminalStartClosed(t *testing.T) {
	opts := Options{
		Command: "echo",
	}

	term, _ := New(opts)
	term.closed = true

	err := term.Start()
	if err == nil {
		t.Error("expected error when starting closed terminal")
	}
}

// --- Test IsRaw ---

func TestIsRaw(t *testing.T) {
	// Test with stdin fd
	result := IsRaw(int(os.Stdin.Fd()))
	// Result depends on whether running in a terminal
	_ = result
}

// --- Test Terminal Start and PTY operations ---

func TestTerminalStartSuccess(t *testing.T) {
	outputReceived := make(chan bool, 1)
	exitReceived := make(chan int, 1)

	opts := Options{
		Command: "echo",
		Args:    []string{"hello"},
		WorkDir: "/tmp",
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
	opts := Options{
		Command:  "cat",
		WorkDir:  "/tmp",
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
	opts := Options{
		Command: "sleep",
		Args:    []string{"5"},
		WorkDir: "/tmp",
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
	opts := Options{
		Command: "sleep",
		Args:    []string{"5"},
		WorkDir: "/tmp",
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
	opts := Options{
		Command: "sleep",
		Args:    []string{"60"},
		WorkDir: "/tmp",
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
	opts := Options{
		Command: "sleep",
		Args:    []string{"60"},
		WorkDir: "/tmp",
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
	opts := Options{
		Command: "sleep",
		Args:    []string{"60"},
		WorkDir: "/tmp",
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

	opts := Options{
		Command: "false", // returns exit code 1
		WorkDir: "/tmp",
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

// --- Test SetOutputHandler and SetExitHandler ---

func TestSetOutputHandler(t *testing.T) {
	opts := Options{
		Command: "echo",
		Args:    []string{"test"},
	}

	term, err := New(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var received []byte
	term.SetOutputHandler(func(data []byte) {
		received = append(received, data...)
	})

	err = term.Start()
	if err != nil {
		t.Fatalf("failed to start terminal: %v", err)
	}

	// Wait for output
	time.Sleep(500 * time.Millisecond)
	term.Stop()

	// Should have received some output
	if len(received) == 0 {
		t.Error("expected to receive output from echo command")
	}
}

func TestSetExitHandler(t *testing.T) {
	opts := Options{
		Command: "true", // Command that exits immediately with code 0
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
	opts := Options{
		Command: "echo",
		Args:    []string{"hello"},
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
	opts := Options{
		Command: "true",
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
	// Use sleep instead of echo to ensure we have time to capture output
	// echo may complete too fast in CI environments with race detector
	opts := Options{
		Command:  "sh",
		Args:     []string{"-c", "echo hello && sleep 0.1"},
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
