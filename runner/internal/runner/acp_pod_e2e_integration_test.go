//go:build integration

package runner

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/acp"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
)

// TestACPPod_FullLifecycle_Integration builds an ACP pod via AgentFile,
// manually wires an ACPClient with the mock agent, then verifies:
// Start -> NewSession -> SendPrompt -> content chunks -> state transitions -> Stop.
func TestACPPod_FullLifecycle_Integration(t *testing.T) {
	// 1. Build the ACP pod (no ACPClient yet -- just like buildACPPod does).
	agentfile := "AGENT echo\nMODE acp\nPROMPT_POSITION prepend\n"
	runner := &Runner{cfg: &config.Config{WorkspaceRoot: t.TempDir()}}
	cmd := &runnerv1.CreatePodCommand{
		PodKey:          "acp-lifecycle-test",
		AgentfileSource: agentfile,
	}
	pod, err := NewPodBuilderFromRunner(runner).
		WithCommand(cmd).WithPtySize(80, 24).
		Build(context.Background())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if pod.InteractionMode != InteractionModeACP {
		t.Fatalf("InteractionMode = %q, want acp", pod.InteractionMode)
	}

	// 2. Wire ACPClient using the mock agent (test binary itself).
	var mu sync.Mutex
	var chunks []acp.ContentChunk
	var states []string

	acpClient := acp.NewClient(acp.ClientConfig{
		Command: mockAgentCmd(),
		Args:    mockAgentArgs(),
		Env:     mockAgentEnv(),
		WorkDir: t.TempDir(),
		Logger:  slog.Default(),
		Callbacks: acp.EventCallbacks{
			OnContentChunk: func(_ string, chunk acp.ContentChunk) {
				mu.Lock()
				chunks = append(chunks, chunk)
				mu.Unlock()
			},
			OnStateChange: func(newState string) {
				mu.Lock()
				states = append(states, newState)
				mu.Unlock()
			},
		},
	})
	pod.IO = NewACPPodIO(acpClient, pod.PodKey)

	// 3. Start → Handshake → state should be idle.
	if err := acpClient.Start(); err != nil {
		t.Fatalf("ACPClient.Start: %v", err)
	}
	defer acpClient.Stop()

	if acpClient.State() != acp.StateIdle {
		t.Fatalf("state after Start = %q, want idle", acpClient.State())
	}

	// 4. NewSession.
	if err := acpClient.NewSession(nil); err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if acpClient.SessionID() != "mock-session-001" {
		t.Errorf("SessionID = %q, want mock-session-001", acpClient.SessionID())
	}

	// 5. SendPrompt and wait for content chunks.
	if err := acpClient.SendPrompt("hello"); err != nil {
		t.Fatalf("SendPrompt: %v", err)
	}

	deadline := time.After(5 * time.Second)
	for {
		mu.Lock()
		gotChunks := len(chunks)
		mu.Unlock()
		if gotChunks > 0 && acpClient.State() == acp.StateIdle {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("timeout: chunks=%d state=%s", gotChunks, acpClient.State())
		case <-time.After(50 * time.Millisecond):
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if chunks[0].Text != "Hello from runner mock" {
		t.Errorf("chunk text = %q, want 'Hello from runner mock'", chunks[0].Text)
	}

	// 6. Verify PodIO mapping works.
	if pod.IO.GetAgentStatus() != "idle" {
		t.Errorf("PodIO.GetAgentStatus = %q, want idle", pod.IO.GetAgentStatus())
	}

	// 7. Verify state transitions included processing → idle.
	hasProcessing := false
	for _, s := range states {
		if s == acp.StateProcessing {
			hasProcessing = true
		}
	}
	if !hasProcessing {
		t.Errorf("state transitions = %v, want processing in sequence", states)
	}
}

// TestACPPod_MessageHandler_CreatePod_Integration uses the full OnCreatePod
// path via MockConnection to verify end-to-end ACP pod creation and events.
func TestACPPod_MessageHandler_CreatePod_Integration(t *testing.T) {
	_, mc := NewTestRunner(t, WithTestConfig(&config.Config{
		MaxConcurrentPods: 10,
		WorkspaceRoot:     t.TempDir(),
		NodeID:            "test-node",
		OrgSlug:           "test-org",
	}))

	// Use the mock agent binary as the ACP launch command.
	// The path must be quoted in AgentFile because it contains slashes.
	cmd := &runnerv1.CreatePodCommand{
		PodKey:          "acp-e2e-handler-pod",
		LaunchCommand:   mockAgentCmd(),
		AgentfileSource: "AGENT \"" + mockAgentCmd() + "\"\nMODE acp\nPROMPT_POSITION prepend\n",
		EnvVars:       map[string]string{"ACP_MOCK_AGENT": "1"},
		LaunchArgs:    mockAgentArgs(),
		InitialPrompt: "hello from e2e",
	}

	err := mc.SimulateCreatePod(cmd)
	if err != nil {
		t.Fatalf("SimulateCreatePod: %v", err)
	}

	// Wait for PodCreated event.
	ok := waitForEvent(mc, client.MsgTypePodCreated, 10*time.Second)
	if !ok {
		t.Fatal("timeout waiting for pod_created event")
	}

	// Verify pod is ACP mode.
	pods := mc.GetPods()
	if len(pods) == 0 {
		t.Fatal("expected at least 1 pod in store")
	}

	// Verify AgentStatus event was sent (initial prompt triggers processing→idle).
	ok = waitForAgentStatusEvent(mc, "idle", 10*time.Second)
	if !ok {
		dumpEvents(t, mc)
		t.Fatal("timeout waiting for agent_status=idle event")
	}

	// Cleanup.
	mc.SimulateTerminatePod(client.TerminatePodRequest{PodKey: cmd.PodKey})
	waitForStoreEmpty(mc, 5*time.Second)
}

// waitForAgentStatusEvent polls for an agent_status event with the given status.
func waitForAgentStatusEvent(mc *client.MockConnection, status string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, ev := range mc.GetEvents() {
			if ev.Type == "agent_status" {
				if m, ok := ev.Data.(map[string]string); ok && m["status"] == status {
					return true
				}
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

// dumpEvents logs all events for debugging test failures.
func dumpEvents(t *testing.T, mc *client.MockConnection) {
	t.Helper()
	for i, ev := range mc.GetEvents() {
		t.Logf("  event[%d]: type=%s data=%v", i, ev.Type, ev.Data)
	}
}
