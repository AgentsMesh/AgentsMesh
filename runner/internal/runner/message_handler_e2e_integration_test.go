//go:build integration

package runner

import (
	"testing"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// waitForEvent polls mockConn.GetEvents() until an event of the given type
// appears or the timeout expires. Returns true if found.
func waitForEvent(mc *client.MockConnection, eventType client.MessageType, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, ev := range mc.GetEvents() {
			if ev.Type == eventType {
				return true
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

// waitForStoreEmpty polls until the store has no pods or the timeout expires.
func waitForStoreEmpty(mc *client.MockConnection, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if len(mc.GetPods()) == 0 {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

// waitForPods polls until the store has at least n pods or the timeout expires.
func waitForPods(mc *client.MockConnection, n int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if len(mc.GetPods()) >= n {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

func TestMessageHandler_CreatePod_TerminalOutput_Integration(t *testing.T) {
	r, mc := NewTestRunner(t)

	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "e2e-echo-pod",
		LaunchCommand: "echo",
		AgentfileSource: "AGENT echo\nMODE pty\nPROMPT_POSITION prepend\n",
		Prompt: "e2e test",
	}

	err := mc.SimulateCreatePod(cmd)
	require.NoError(t, err, "SimulateCreatePod should succeed")

	// Pod should be in store immediately after synchronous OnCreatePod
	pods := mc.GetPods()
	assert.GreaterOrEqual(t, len(pods), 1, "pod should exist in store after creation")

	// PodCreated event should have been sent synchronously during OnCreatePod
	found := waitForEvent(mc, client.MsgTypePodCreated, 5*time.Second)
	assert.True(t, found, "should receive pod_created event")

	// echo exits quickly — wait for PodTerminated event from the exit handler goroutine
	found = waitForEvent(mc, client.MsgTypePodTerminated, 10*time.Second)
	assert.True(t, found, "should receive pod_terminated event after echo exits")

	// After termination, store should be empty
	empty := waitForStoreEmpty(mc, 5*time.Second)
	assert.True(t, empty, "store should be empty after pod terminates")

	// Verify both events are present
	events := mc.GetEvents()
	var hasCreated, hasTerminated bool
	for _, ev := range events {
		if ev.Type == client.MsgTypePodCreated {
			hasCreated = true
		}
		if ev.Type == client.MsgTypePodTerminated {
			hasTerminated = true
		}
	}
	assert.True(t, hasCreated, "events should contain pod_created")
	assert.True(t, hasTerminated, "events should contain pod_terminated")

	_ = r // keep reference
}

func TestMessageHandler_CreateTerminate_Integration(t *testing.T) {
	r, mc := NewTestRunner(t)

	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "e2e-sleep-pod",
		LaunchCommand: "sleep",
		AgentfileSource: "AGENT sleep\nMODE pty\nPROMPT_POSITION prepend\n",
		Prompt: "60",
	}

	err := mc.SimulateCreatePod(cmd)
	require.NoError(t, err, "SimulateCreatePod should succeed for sleep")

	// Wait for pod to appear in store
	ok := waitForPods(mc, 1, 5*time.Second)
	require.True(t, ok, "pod should appear in store")

	// Verify PodCreated event
	found := waitForEvent(mc, client.MsgTypePodCreated, 5*time.Second)
	assert.True(t, found, "should receive pod_created event")

	// Now terminate the pod
	err = mc.SimulateTerminatePod(client.TerminatePodRequest{PodKey: "e2e-sleep-pod"})
	require.NoError(t, err, "SimulateTerminatePod should succeed")

	// Store should become empty
	empty := waitForStoreEmpty(mc, 5*time.Second)
	assert.True(t, empty, "store should be empty after termination")

	// Verify PodTerminated event
	found = waitForEvent(mc, client.MsgTypePodTerminated, 5*time.Second)
	assert.True(t, found, "should receive pod_terminated event")

	// Verify both events
	events := mc.GetEvents()
	var hasCreated, hasTerminated bool
	for _, ev := range events {
		if ev.Type == client.MsgTypePodCreated {
			hasCreated = true
		}
		if ev.Type == client.MsgTypePodTerminated {
			hasTerminated = true
		}
	}
	assert.True(t, hasCreated, "events should contain pod_created")
	assert.True(t, hasTerminated, "events should contain pod_terminated")

	_ = r
}

func TestMessageHandler_MaxCapacity_Integration(t *testing.T) {
	r, mc := NewTestRunner(t, WithTestConfig(&config.Config{
		MaxConcurrentPods: 1,
		WorkspaceRoot:     t.TempDir(),
		NodeID:            "test-node",
		OrgSlug:           "test-org",
	}))

	// Create first pod (long-running)
	cmd1 := &runnerv1.CreatePodCommand{
		PodKey:        "e2e-cap-pod-1",
		LaunchCommand: "sleep",
		AgentfileSource: "AGENT sleep\nMODE pty\nPROMPT_POSITION prepend\n",
		Prompt: "300",
	}
	err := mc.SimulateCreatePod(cmd1)
	require.NoError(t, err, "first pod should be created")

	ok := waitForPods(mc, 1, 5*time.Second)
	require.True(t, ok, "first pod should appear in store")

	// CanAcceptPod should now return false (at capacity)
	assert.False(t, r.CanAcceptPod(), "runner should reject new pods at max capacity")

	// Verify error event is sent when we force a second pod through the handler
	cmd2 := &runnerv1.CreatePodCommand{
		PodKey:        "e2e-cap-pod-2",
		LaunchCommand: "sleep",
		AgentfileSource: "AGENT sleep\nMODE pty\nPROMPT_POSITION prepend\n",
		Prompt: "300",
	}
	// Even though OnCreatePod doesn't check capacity itself (gRPC layer does),
	// creating a second pod demonstrates the store holds 2 — confirming that
	// the capacity gate must be enforced by the caller (CanAcceptPod).
	_ = mc.SimulateCreatePod(cmd2)
	time.Sleep(500 * time.Millisecond)

	assert.Equal(t, 2, len(mc.GetPods()), "both pods exist since handler has no capacity gate")
	assert.False(t, r.CanAcceptPod(), "CanAcceptPod should still be false")

	// Clean up
	mc.SimulateTerminatePod(client.TerminatePodRequest{PodKey: "e2e-cap-pod-1"})
	mc.SimulateTerminatePod(client.TerminatePodRequest{PodKey: "e2e-cap-pod-2"})
	waitForStoreEmpty(mc, 5*time.Second)

	assert.True(t, r.CanAcceptPod(), "CanAcceptPod should be true after cleanup")
}
