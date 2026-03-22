package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
)

func TestNewGRPCCommandSender(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr, nil)
	sender := NewGRPCCommandSender(adapter)

	assert.NotNil(t, sender)
	assert.Equal(t, adapter, sender.adapter)
}

func TestGRPCCommandSender_SendCreatePod(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr, nil)
	sender := NewGRPCCommandSender(adapter)

	ctx := context.Background()

	t.Run("returns error when runner not connected", func(t *testing.T) {
		cmd := &runnerv1.CreatePodCommand{
			PodKey:        "test-pod",
			LaunchCommand: "claude",
		}
		err := sender.SendCreatePod(ctx, 999, cmd)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not connected")
	})

	t.Run("with files_to_create", func(t *testing.T) {
		cmd := &runnerv1.CreatePodCommand{
			PodKey:        "test-pod",
			LaunchCommand: "claude",
			FilesToCreate: []*runnerv1.FileToCreate{
				{
					Path:    "/tmp/test.txt",
					Content: "hello world",
					Mode:    0644,
				},
			},
		}
		err := sender.SendCreatePod(ctx, 999, cmd)
		require.Error(t, err) // Runner not connected
		assert.Contains(t, err.Error(), "not connected")
	})

	t.Run("with sandbox_config", func(t *testing.T) {
		cmd := &runnerv1.CreatePodCommand{
			PodKey:        "test-pod",
			LaunchCommand: "claude",
			SandboxConfig: &runnerv1.SandboxConfig{
				RepositoryUrl:  "https://github.com/org/repo.git",
				SourceBranch:   "feature-branch",
				CredentialType: "runner_local",
			},
		}
		err := sender.SendCreatePod(ctx, 999, cmd)
		require.Error(t, err) // Runner not connected
		assert.Contains(t, err.Error(), "not connected")
	})
}

func TestGRPCCommandSender_SendTerminatePod(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr, nil)
	sender := NewGRPCCommandSender(adapter)

	ctx := context.Background()
	err := sender.SendTerminatePod(ctx, 999, "test-pod")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestGRPCCommandSender_SendPodInput(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr, nil)
	sender := NewGRPCCommandSender(adapter)

	ctx := context.Background()
	err := sender.SendPodInput(ctx, 999, "test-pod", []byte("hello"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestGRPCCommandSender_SendPrompt(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr, nil)
	sender := NewGRPCCommandSender(adapter)

	ctx := context.Background()
	err := sender.SendPrompt(ctx, 999, "test-pod", "Hello, Claude!")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestGRPCCommandSender_ImplementsInterface(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr, nil)
	sender := NewGRPCCommandSender(adapter)

	// Verify it implements the interface
	var _ runner.RunnerCommandSender = sender
}
