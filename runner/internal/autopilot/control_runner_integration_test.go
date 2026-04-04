//go:build integration

package autopilot

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestControlRunner_StartControlProcess_NonJSONOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires Unix shell")
	}
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "control_runner_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a script that outputs non-JSON
	scriptPath := filepath.Join(tmpDir, "mock_agent")
	script := `#!/bin/sh
echo "Working on task..."
echo "TASK_COMPLETED"
echo "All done successfully."
`
	err = os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	pb := NewPromptBuilder(PromptBuilderConfig{
		InitialPrompt: "Test task",
		MCPPort:       19000,
		PodKey:        "worker-123",
	})

	cr := NewControlRunner(ControlRunnerConfig{
		WorkDir:       tmpDir,
		Agent:         scriptPath,
		PromptBuilder: pb,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	decision, err := cr.RunControlProcess(ctx, 1)

	require.NoError(t, err)
	assert.Equal(t, DecisionCompleted, decision.Type)
}

func TestControlRunner_StartControlProcess_LongOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires Unix shell")
	}
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "control_runner_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a script that outputs a lot of text
	scriptPath := filepath.Join(tmpDir, "mock_agent")
	script := `#!/bin/sh
for i in {1..500}; do
  echo "Line $i of output"
done
echo "TASK_COMPLETED"
echo "Done."
`
	err = os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	pb := NewPromptBuilder(PromptBuilderConfig{
		InitialPrompt: "Test task",
		MCPPort:       19000,
		PodKey:        "worker-123",
	})

	cr := NewControlRunner(ControlRunnerConfig{
		WorkDir:       tmpDir,
		Agent:         scriptPath,
		PromptBuilder: pb,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	decision, err := cr.RunControlProcess(ctx, 1)

	require.NoError(t, err)
	assert.Equal(t, DecisionCompleted, decision.Type)
}

func TestControlRunner_ResumeControlProcess_Timeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires Unix shell")
	}
	// Create temp directory with a script that sleeps
	tmpDir, err := os.MkdirTemp("", "control_runner_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	scriptPath := filepath.Join(tmpDir, "slow_agent")
	err = os.WriteFile(scriptPath, []byte("#!/bin/sh\nsleep 10"), 0755)
	require.NoError(t, err)

	pb := NewPromptBuilder(PromptBuilderConfig{
		InitialPrompt:    "Test task",
		MCPPort:          19000,
		PodKey:           "worker-123",
		GetMaxIterations: func() int { return 10 },
	})

	cr := NewControlRunner(ControlRunnerConfig{
		WorkDir:       tmpDir,
		Agent:         scriptPath,
		PromptBuilder: pb,
	})

	cr.SetSessionID("existing-session")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = cr.RunControlProcess(ctx, 2)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
}

func TestControlRunner_StartControlProcess_ProcessError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires Unix shell")
	}
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "control_runner_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a script that exits with error
	scriptPath := filepath.Join(tmpDir, "error_agent")
	err = os.WriteFile(scriptPath, []byte("#!/bin/sh\nexit 1"), 0755)
	require.NoError(t, err)

	pb := NewPromptBuilder(PromptBuilderConfig{
		InitialPrompt: "Test task",
		MCPPort:       19000,
		PodKey:        "worker-123",
	})

	cr := NewControlRunner(ControlRunnerConfig{
		WorkDir:       tmpDir,
		Agent:         scriptPath,
		PromptBuilder: pb,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = cr.RunControlProcess(ctx, 1)

	assert.Error(t, err)
	// CommandExecutor returns "command failed: ..." format
	assert.Contains(t, err.Error(), "command failed")
}

func TestControlRunner_ResumeControlProcess_ProcessError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires Unix shell")
	}
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "control_runner_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a script that exits with error
	scriptPath := filepath.Join(tmpDir, "error_agent")
	err = os.WriteFile(scriptPath, []byte("#!/bin/sh\nexit 1"), 0755)
	require.NoError(t, err)

	pb := NewPromptBuilder(PromptBuilderConfig{
		InitialPrompt:    "Test task",
		MCPPort:          19000,
		PodKey:           "worker-123",
		GetMaxIterations: func() int { return 10 },
	})

	cr := NewControlRunner(ControlRunnerConfig{
		WorkDir:       tmpDir,
		Agent:         scriptPath,
		PromptBuilder: pb,
	})

	cr.SetSessionID("existing-session")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = cr.RunControlProcess(ctx, 2)

	assert.Error(t, err)
	// CommandExecutor returns "command failed: ..." format
	assert.Contains(t, err.Error(), "command failed")
}
