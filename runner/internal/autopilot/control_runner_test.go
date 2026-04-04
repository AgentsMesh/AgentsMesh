package autopilot

import (
	"context"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestControlRunner_NewControlRunner(t *testing.T) {
	pb := NewPromptBuilder(PromptBuilderConfig{
		InitialPrompt: "Test task",
		MCPPort:       19000,
		PodKey:        "worker-123",
	})

	cr := NewControlRunner(ControlRunnerConfig{
		WorkDir:        os.TempDir(),
		Agent:          "claude",
		PromptBuilder:  pb,
		DecisionParser: nil, // Should use default
		Logger:         nil,
	})

	assert.NotNil(t, cr)
	assert.Equal(t, "claude", cr.agent)
}

func TestControlRunner_DefaultAgent(t *testing.T) {
	cr := NewControlRunner(ControlRunnerConfig{
		WorkDir: os.TempDir(),
		Agent:   "", // Should default to "claude"
	})

	assert.Equal(t, "claude", cr.agent)
}

func TestControlRunner_DefaultDecisionParser(t *testing.T) {
	cr := NewControlRunner(ControlRunnerConfig{
		WorkDir:        os.TempDir(),
		DecisionParser: nil, // Should create default
	})

	assert.NotNil(t, cr.decisionParser)
}

func TestControlRunner_GetSetSessionID(t *testing.T) {
	cr := NewControlRunner(ControlRunnerConfig{
		WorkDir: os.TempDir(),
	})

	// Initially empty
	assert.Empty(t, cr.GetSessionID())

	// Set session ID
	cr.SetSessionID("test-session-123")
	assert.Equal(t, "test-session-123", cr.GetSessionID())
}

func TestControlRunner_RunControlProcess_Start(t *testing.T) {
	// Create temp directory for work dir
	tmpDir, err := os.MkdirTemp("", "control_runner_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	pb := NewPromptBuilder(PromptBuilderConfig{
		InitialPrompt: "Test task",
		MCPPort:       19000,
		PodKey:        "worker-123",
	})

	cr := NewControlRunner(ControlRunnerConfig{
		WorkDir:       tmpDir,
		Agent:         "echo", // Use echo as a simple command
		PromptBuilder: pb,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First run - starts new process
	decision, err := cr.RunControlProcess(ctx, 1)

	// echo command will fail with wrong args, but we test the flow
	if err != nil {
		// Expected for non-existent agent
		assert.Contains(t, err.Error(), "command failed")
	} else {
		assert.NotNil(t, decision)
	}
}

func TestControlRunner_RunControlProcess_Resume(t *testing.T) {
	// Create temp directory for work dir
	tmpDir, err := os.MkdirTemp("", "control_runner_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	pb := NewPromptBuilder(PromptBuilderConfig{
		InitialPrompt:    "Test task",
		MCPPort:          19000,
		PodKey:           "worker-123",
		GetMaxIterations: func() int { return 10 },
	})

	cr := NewControlRunner(ControlRunnerConfig{
		WorkDir:       tmpDir,
		Agent:         "echo", // Use echo as a simple command
		PromptBuilder: pb,
	})

	// Set session ID to trigger resume path
	cr.SetSessionID("existing-session")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// With session ID set, should try to resume
	decision, err := cr.RunControlProcess(ctx, 2)

	// echo command will fail with wrong args, but we test the flow
	if err != nil {
		// Expected for non-existent agent
		assert.Contains(t, err.Error(), "command failed")
	} else {
		assert.NotNil(t, decision)
	}
}

func TestControlRunner_StartControlProcess_Timeout(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping test that requires shell execution in CI environment")
	}
	if runtime.GOOS == "windows" {
		t.Skip("requires Unix shell")
	}
	// Create temp directory with a script that sleeps
	tmpDir, err := os.MkdirTemp("", "control_runner_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a script that sleeps longer than timeout
	scriptPath := testutil.WriteTestScript(t, tmpDir, "slow_agent", "sleep 10")

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

	// Very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = cr.RunControlProcess(ctx, 1)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
}

func TestControlRunner_StartControlProcess_Success(t *testing.T) {
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
	scriptPath := testutil.WriteTestScript(t, tmpDir, "mock_agent",
		`printf '%s\n' '{"result": "TASK_COMPLETED\nAll done.", "session_id": "test-session-abc"}'`)

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
	assert.Equal(t, "test-session-abc", cr.GetSessionID())
}

func TestControlRunner_ResumeControlProcess_Success(t *testing.T) {
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
	scriptPath := testutil.WriteTestScript(t, tmpDir, "mock_agent",
		`printf '%s\n' '{"result": "CONTINUE\nMore work needed."}'`)

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

	// Set session ID to trigger resume
	cr.SetSessionID("existing-session")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	decision, err := cr.RunControlProcess(ctx, 2)

	require.NoError(t, err)
	assert.Equal(t, DecisionContinue, decision.Type)
}
