package autopilot

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestControlRunner_StartControlProcess_WithLogger(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping test that requires shell execution in CI environment")
	}
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test that requires printf on Windows")
	}
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "control_runner_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a script that outputs a valid decision with session_id
	// Use printf instead of echo to avoid macOS /bin/sh interpreting \n as newline
	scriptPath := filepath.Join(tmpDir, "mock_agent")
	script := `#!/bin/sh
printf '%s\n' '{"result": "TASK_COMPLETED\nAll done.", "session_id": "test-session-xyz"}'
`
	err = os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	pb := NewPromptBuilder(PromptBuilderConfig{
		Prompt: "Test task",
		MCPPort:       19000,
		PodKey:        "worker-123",
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cr := NewControlRunner(ControlRunnerConfig{
		WorkDir:       tmpDir,
		Agent:         scriptPath,
		PromptBuilder: pb,
		Logger:        logger,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	decision, err := cr.RunControlProcess(ctx, 1)

	require.NoError(t, err)
	assert.Equal(t, DecisionCompleted, decision.Type)
	assert.Equal(t, "test-session-xyz", cr.GetSessionID())
}

func TestControlRunner_ResumeControlProcess_WithLogger(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping test that requires shell execution in CI environment")
	}
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test that requires printf on Windows")
	}
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "control_runner_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a script that outputs a valid decision
	// Use printf instead of echo to avoid macOS /bin/sh interpreting \n as newline
	scriptPath := filepath.Join(tmpDir, "mock_agent")
	script := `#!/bin/sh
printf '%s\n' '{"result": "CONTINUE\nMore work needed."}'
`
	err = os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	pb := NewPromptBuilder(PromptBuilderConfig{
		Prompt:    "Test task",
		MCPPort:          19000,
		PodKey:           "worker-123",
		GetMaxIterations: func() int { return 10 },
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cr := NewControlRunner(ControlRunnerConfig{
		WorkDir:       tmpDir,
		Agent:         scriptPath,
		PromptBuilder: pb,
		Logger:        logger,
	})

	// Set session ID to trigger resume
	cr.SetSessionID("existing-session")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	decision, err := cr.RunControlProcess(ctx, 2)

	require.NoError(t, err)
	assert.Equal(t, DecisionContinue, decision.Type)
}

func TestControlRunner_StartControlProcess_LongOutputTruncation(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping test that requires shell execution in CI environment")
	}
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test that requires shell scripts on Windows")
	}
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "control_runner_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a script that outputs >2000 chars without JSON
	// Use POSIX-compatible while loop instead of bash-only brace expansion {1..300}
	scriptPath := filepath.Join(tmpDir, "mock_agent")
	script := `#!/bin/sh
i=1
while [ $i -le 300 ]; do
  echo "This is a very long line of output number $i that should help us exceed the 2000 character limit"
  i=$((i + 1))
done
echo "TASK_COMPLETED"
echo "Done."
`
	err = os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	pb := NewPromptBuilder(PromptBuilderConfig{
		Prompt: "Test task",
		MCPPort:       19000,
		PodKey:        "worker-123",
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cr := NewControlRunner(ControlRunnerConfig{
		WorkDir:       tmpDir,
		Agent:         scriptPath,
		PromptBuilder: pb,
		Logger:        logger,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	decision, err := cr.RunControlProcess(ctx, 1)

	require.NoError(t, err)
	assert.Equal(t, DecisionCompleted, decision.Type)
}

func TestControlRunner_ResumeControlProcess_LongOutputTruncation(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping test that requires shell execution in CI environment")
	}
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test that requires shell scripts on Windows")
	}
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "control_runner_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a script that outputs >2000 chars without JSON
	// Use POSIX-compatible while loop instead of bash-only brace expansion {1..300}
	scriptPath := filepath.Join(tmpDir, "mock_agent")
	script := `#!/bin/sh
i=1
while [ $i -le 300 ]; do
  echo "This is a very long line of output number $i that should help us exceed the 2000 character limit"
  i=$((i + 1))
done
echo "CONTINUE"
echo "More work."
`
	err = os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	pb := NewPromptBuilder(PromptBuilderConfig{
		Prompt:    "Test task",
		MCPPort:          19000,
		PodKey:           "worker-123",
		GetMaxIterations: func() int { return 10 },
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cr := NewControlRunner(ControlRunnerConfig{
		WorkDir:       tmpDir,
		Agent:         scriptPath,
		PromptBuilder: pb,
		Logger:        logger,
	})

	cr.SetSessionID("existing-session")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	decision, err := cr.RunControlProcess(ctx, 2)

	require.NoError(t, err)
	assert.Equal(t, DecisionContinue, decision.Type)
}

func TestControlRunner_StartControlProcess_ErrorWithLogger(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping test that requires shell execution in CI environment")
	}
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test that requires shell scripts on Windows")
	}
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "control_runner_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a script that exits with error
	scriptPath := filepath.Join(tmpDir, "error_agent")
	err = os.WriteFile(scriptPath, []byte("#!/bin/sh\necho 'error' >&2\nexit 1"), 0755)
	require.NoError(t, err)

	pb := NewPromptBuilder(PromptBuilderConfig{
		Prompt: "Test task",
		MCPPort:       19000,
		PodKey:        "worker-123",
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cr := NewControlRunner(ControlRunnerConfig{
		WorkDir:       tmpDir,
		Agent:         scriptPath,
		PromptBuilder: pb,
		Logger:        logger,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = cr.RunControlProcess(ctx, 1)

	assert.Error(t, err)
	// CommandExecutor returns "command failed: ..." format
	assert.Contains(t, err.Error(), "command failed")
}

func TestControlRunner_ResumeControlProcess_ErrorWithLogger(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping test that requires shell execution in CI environment")
	}
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test that requires shell scripts on Windows")
	}
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "control_runner_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a script that exits with error
	scriptPath := filepath.Join(tmpDir, "error_agent")
	err = os.WriteFile(scriptPath, []byte("#!/bin/sh\necho 'error' >&2\nexit 1"), 0755)
	require.NoError(t, err)

	pb := NewPromptBuilder(PromptBuilderConfig{
		Prompt:    "Test task",
		MCPPort:          19000,
		PodKey:           "worker-123",
		GetMaxIterations: func() int { return 10 },
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cr := NewControlRunner(ControlRunnerConfig{
		WorkDir:       tmpDir,
		Agent:         scriptPath,
		PromptBuilder: pb,
		Logger:        logger,
	})

	cr.SetSessionID("existing-session")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = cr.RunControlProcess(ctx, 2)

	assert.Error(t, err)
	// CommandExecutor returns "command failed: ..." format
	assert.Contains(t, err.Error(), "command failed")
}
