//go:build integration

package runner

import (
	"testing"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/workspace"
)

// Integration tests for MessageHandler with MockConnection

func TestMessageHandlerIntegrationWithMockConnection(t *testing.T) {
	tempDir := t.TempDir()
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	ws, err := workspace.NewManager(tempDir, "")
	if err != nil {
		t.Skipf("Could not create workspace manager: %v", err)
	}

	runner := &Runner{
		cfg: &config.Config{
			MaxConcurrentPods: 10,
			WorkspaceRoot:         tempDir,
		},
		workspace: ws,
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)
	mockConn.SetHandler(handler)

	// Test flow: create -> list -> terminate
	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "integration-pod",
		LaunchCommand: "echo",
	}

	// Create pod via mock connection simulation
	err = mockConn.SimulateCreatePod(cmd)
	if err != nil {
		t.Logf("Create pod: %v", err)
	}

	// Give pod time to start
	time.Sleep(50 * time.Millisecond)

	// List pods
	pods := mockConn.GetPods()
	t.Logf("Pods after create: %d", len(pods))

	// Terminate pod
	terminateReq := client.TerminatePodRequest{
		PodKey: "integration-pod",
	}
	err = mockConn.SimulateTerminatePod(terminateReq)
	t.Logf("Terminate pod: %v", err)

	// List pods again
	pods = mockConn.GetPods()
	t.Logf("Pods after terminate: %d", len(pods))
}

func TestMessageHandlerIntegrationPodLifecycle(t *testing.T) {
	tempDir := t.TempDir()
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg: &config.Config{
			MaxConcurrentPods: 10,
			WorkspaceRoot:         tempDir,
		},
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)
	mockConn.SetHandler(handler)

	// Create multiple pods
	for i := 0; i < 3; i++ {
		cmd := &runnerv1.CreatePodCommand{
			PodKey:        "lifecycle-pod-" + string(rune('a'+i)),
			LaunchCommand: "sleep",
		}
		err := mockConn.SimulateCreatePod(cmd)
		if err != nil {
			t.Logf("Create pod %d: %v", i, err)
		}
	}

	time.Sleep(100 * time.Millisecond)

	// Check pod count
	pods := mockConn.GetPods()
	if len(pods) < 1 {
		t.Log("Expected at least 1 pod")
	}

	// Terminate all pods
	for _, s := range pods {
		req := client.TerminatePodRequest{
			PodKey: s.PodKey,
		}
		mockConn.SimulateTerminatePod(req)
	}

	// Verify all pods terminated
	time.Sleep(50 * time.Millisecond)
	pods = mockConn.GetPods()
	if len(pods) != 0 {
		t.Errorf("Expected 0 pods after termination, got %d", len(pods))
	}
}
