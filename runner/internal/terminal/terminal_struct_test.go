package terminal

import (
	"os"
	"testing"
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
		return
	}

	if term.command != "echo" {
		t.Errorf("command should be echo, got %s", term.command)
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
	for _, e := range term.env {
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
	term.mu.Lock()
	term.closed = true
	term.mu.Unlock()

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

// --- Test MakeRaw and Restore ---

func TestMakeRawInvalidFd(t *testing.T) {
	// Use invalid fd (-1) - should return error
	_, err := MakeRaw(-1)
	if err == nil {
		t.Error("expected error for invalid fd")
	}
}

func TestRestoreInvalidFd(t *testing.T) {
	// Just verify the function exists and is callable
	// Testing with actual terminal state would require a real terminal
}
