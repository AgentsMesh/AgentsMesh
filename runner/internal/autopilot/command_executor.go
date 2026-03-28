// Package autopilot implements the AutopilotController for supervised Pod automation.
package autopilot

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
)

// maxOutputSize is the maximum bytes captured from stdout/stderr of a control
// process. Claude CLI with --output-format json can produce very large output
// (full tool_use chains, code blocks, etc.). Capping prevents Runner OOM.
const maxOutputSize = 10 * 1024 * 1024 // 10 MB

// errOutputTruncated is returned when command output exceeds maxOutputSize.
var errOutputTruncated = errors.New("command output truncated: exceeded maximum size")

// limitedWriter is a bytes.Buffer wrapper that stops writing after a size limit.
// Once the limit is reached, subsequent writes are silently discarded.
type limitedWriter struct {
	buf   bytes.Buffer
	limit int
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	remaining := w.limit - w.buf.Len()
	if remaining <= 0 {
		// Silently discard — report full len to prevent exec from erroring
		return len(p), nil
	}
	if len(p) > remaining {
		p = p[:remaining]
	}
	return w.buf.Write(p)
}

func (w *limitedWriter) Bytes() []byte { return w.buf.Bytes() }
func (w *limitedWriter) Len() int      { return w.buf.Len() }

// CommandExecutor abstracts command execution for testability and extensibility.
// This follows the Dependency Inversion Principle (DIP).
type CommandExecutor interface {
	// Execute runs a command and returns the combined stdout output.
	// Returns error if the command fails or times out.
	Execute(ctx context.Context, name string, args []string, workDir string) ([]byte, []byte, error)
}

// DefaultCommandExecutor is the standard implementation using os/exec.
type DefaultCommandExecutor struct{}

// NewDefaultCommandExecutor creates a new DefaultCommandExecutor.
func NewDefaultCommandExecutor() *DefaultCommandExecutor {
	return &DefaultCommandExecutor{}
}

// Execute runs a command using os/exec.
// stdout and stderr are each capped at maxOutputSize (10 MB) to prevent OOM.
func (e *DefaultCommandExecutor) Execute(ctx context.Context, name string, args []string, workDir string) ([]byte, []byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = workDir

	stdout := &limitedWriter{limit: maxOutputSize}
	stderr := &limitedWriter{limit: maxOutputSize}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()
	if err != nil {
		if ctx.Err() != nil {
			slog.Error("command timed out", "command", name, "work_dir", workDir, "error", ctx.Err())
			return stdout.Bytes(), stderr.Bytes(), fmt.Errorf("command timed out: %w", ctx.Err())
		}
		slog.Error("command failed", "command", name, "work_dir", workDir, "error", err)
		return stdout.Bytes(), stderr.Bytes(), fmt.Errorf("command failed: %w", err)
	}

	// Warn caller if output was truncated
	if stdout.Len() >= maxOutputSize || stderr.Len() >= maxOutputSize {
		slog.Warn("command output truncated", "command", name, "work_dir", workDir, "stdout_len", stdout.Len(), "stderr_len", stderr.Len())
		return stdout.Bytes(), stderr.Bytes(), errOutputTruncated
	}

	return stdout.Bytes(), stderr.Bytes(), nil
}

// MockCommandExecutor is a test double for CommandExecutor.
type MockCommandExecutor struct {
	// ExecuteFunc is called when Execute is invoked.
	// Set this to customize behavior in tests.
	ExecuteFunc func(ctx context.Context, name string, args []string, workDir string) ([]byte, []byte, error)

	// Calls records all Execute calls for verification.
	Calls []CommandExecutorCall
}

// CommandExecutorCall records a single Execute call.
type CommandExecutorCall struct {
	Name    string
	Args    []string
	WorkDir string
}

// NewMockCommandExecutor creates a new MockCommandExecutor with default success behavior.
func NewMockCommandExecutor() *MockCommandExecutor {
	return &MockCommandExecutor{
		ExecuteFunc: func(ctx context.Context, name string, args []string, workDir string) ([]byte, []byte, error) {
			return []byte("mock output"), nil, nil
		},
		Calls: make([]CommandExecutorCall, 0),
	}
}

// Execute records the call and delegates to ExecuteFunc.
func (m *MockCommandExecutor) Execute(ctx context.Context, name string, args []string, workDir string) ([]byte, []byte, error) {
	m.Calls = append(m.Calls, CommandExecutorCall{
		Name:    name,
		Args:    args,
		WorkDir: workDir,
	})
	return m.ExecuteFunc(ctx, name, args, workDir)
}

// SetOutput sets the mock to return the given output.
func (m *MockCommandExecutor) SetOutput(stdout, stderr []byte, err error) {
	m.ExecuteFunc = func(ctx context.Context, name string, args []string, workDir string) ([]byte, []byte, error) {
		return stdout, stderr, err
	}
}
