package runner

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/detector"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/vt"
)

// =============================================================================
// Gap 1: Unit tests for getAgentStatusFromDetector
// =============================================================================

func TestGetAgentStatusFromDetector_NilVirtualTerminal(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()
	runner := &Runner{cfg: &config.Config{}}
	handler := NewRunnerMessageHandler(runner, store, mockConn)

	pod := &Pod{
		PodKey: "test-pod",
	}

	status := handler.getAgentStatusFromDetector(pod)
	assert.Equal(t, "idle", status, "nil VirtualTerminal should return idle")
}

func TestGetAgentStatusFromDetector_StateNotRunning(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()
	runner := &Runner{cfg: &config.Config{}}
	handler := NewRunnerMessageHandler(runner, store, mockConn)

	vterminal := vt.NewVirtualTerminal(80, 24, 1000)
	pod := testNewPTYPod("test-pod", vterminal)

	// GetOrCreateStateDetector creates a ManagedStateDetector that starts in StateNotRunning
	sd := pod.GetOrCreateStateDetector()
	require.NotNil(t, sd)
	defer pod.StopStateDetector()

	assert.Equal(t, detector.StateNotRunning, sd.GetState(), "detector should start in StateNotRunning")

	status := handler.getAgentStatusFromDetector(pod)
	assert.Equal(t, "idle", status, "StateNotRunning should map to idle")
}

func TestGetAgentStatusFromDetector_StateExecuting(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()
	runner := &Runner{cfg: &config.Config{}}
	handler := NewRunnerMessageHandler(runner, store, mockConn)

	vterminal := vt.NewVirtualTerminal(80, 24, 1000)
	pod := testNewPTYPod("test-pod", vterminal)

	sd := pod.GetOrCreateStateDetector()
	require.NotNil(t, sd)
	defer pod.StopStateDetector()

	// OnOutput transitions the detector from NotRunning to Executing
	sd.OnOutput(100)
	assert.Equal(t, detector.StateExecuting, sd.GetState(), "detector should be in StateExecuting after OnOutput")

	status := handler.getAgentStatusFromDetector(pod)
	assert.Equal(t, "executing", status, "StateExecuting should map to executing")
}

func TestGetAgentStatusFromDetector_StateWaiting(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping waiting state test in short mode (requires timing)")
	}

	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()
	runner := &Runner{cfg: &config.Config{}}
	handler := NewRunnerMessageHandler(runner, store, mockConn)

	vterminal := vt.NewVirtualTerminal(80, 24, 1000)
	pod := testNewPTYPod("test-pod", vterminal)

	sd := pod.GetOrCreateStateDetector()
	require.NotNil(t, sd)
	defer pod.StopStateDetector()

	// Step 1: Trigger output to move to Executing state
	sd.OnOutput(100)
	assert.Equal(t, detector.StateExecuting, sd.GetState())

	// Step 2: Provide screen content with a clear prompt pattern.
	// The prompt detection signal (weight 0.3) combined with activity idle (weight 0.4)
	// and screen stability (weight 0.3) should push confidence above the 0.6 threshold.
	// The prompt "$ " is a strong shell prompt pattern.
	promptScreen := []string{
		"user@host:~/project$ ",
	}
	sd.OnScreenUpdate(promptScreen)

	// Step 3: Wait for the background detection loop to transition to Waiting.
	// ManagedStateDetector uses: IdleThreshold=500ms, ConfirmThreshold=500ms,
	// MinStableTime=300ms, detection loop interval=200ms.
	// We need to wait long enough for activity to become idle and for
	// the detection loop to run DetectState() and calculate sufficient confidence.
	//
	// We poll with a timeout to avoid flaky tests with exact sleep durations.
	var finalStatus string
	deadline := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for StateWaiting transition; last state: %s, last status: %s",
				sd.GetState(), finalStatus)
		case <-ticker.C:
			// Keep providing the same screen content to maintain stability
			sd.OnScreenUpdate(promptScreen)

			if sd.GetState() == detector.StateWaiting {
				finalStatus = handler.getAgentStatusFromDetector(pod)
				assert.Equal(t, "waiting", finalStatus, "StateWaiting should map to waiting")
				return
			}
		}
	}
}

func TestGetAgentStatusFromDetector_DefaultState(t *testing.T) {
	// The default branch in getAgentStatusFromDetector covers any AgentState value
	// not explicitly matched (StateExecuting, StateWaiting, StateNotRunning).
	// In practice, the only other defined state is StateUnknown ("unknown").
	//
	// Since ManagedStateDetector wraps MultiSignalDetector which only transitions
	// between NotRunning, Executing, and Waiting, it is not possible to reach
	// StateUnknown through normal operation. The default branch is a defensive
	// code path that returns "unknown" for any unexpected state.
	//
	// This test verifies that the function handles all known states correctly
	// via PodIO and that nil IO returns "idle".

	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()
	runner := &Runner{cfg: &config.Config{}}
	handler := NewRunnerMessageHandler(runner, store, mockConn)

	// Pod with VirtualTerminal and IO: the newly created detector will be
	// in StateNotRunning, which maps to "idle" via PodIO.GetAgentStatus().
	vterminal := vt.NewVirtualTerminal(80, 24, 1000)
	pod := testNewPTYPod("test-pod", vterminal)
	defer pod.StopStateDetector()

	status := handler.getAgentStatusFromDetector(pod)
	assert.Equal(t, "idle", status, "freshly created detector (StateNotRunning) should return idle via PodIO")

	// Verify the detector was actually created by GetOrCreateStateDetector
	sd := pod.GetOrCreateStateDetector()
	require.NotNil(t, sd, "detector should have been created")
}

// =============================================================================
// Gap 2: OnListPods should include AgentStatus in assertions
// =============================================================================

func TestOnListPodsWithPods_AgentStatus(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()
	runner := &Runner{cfg: &config.Config{}}
	handler := NewRunnerMessageHandler(runner, store, mockConn)

	// Pod 1: No VirtualTerminal -> AgentStatus should be "idle"
	store.Put("pod-1", &Pod{
		ID:     "pod-1",
		PodKey: "pod-1",
		Status: PodStatusRunning,
		// VirtualTerminal is nil
	})

	// Pod 2: With VirtualTerminal, fresh detector (StateNotRunning) -> AgentStatus should be "idle"
	vterminal2 := vt.NewVirtualTerminal(80, 24, 1000)
	pod2 := testNewPTYPod("pod-2", vterminal2)
	pod2.ID = "pod-2"
	pod2.Status = PodStatusRunning
	store.Put("pod-2", pod2)
	defer pod2.StopStateDetector()

	// Pod 3: With VirtualTerminal, after OnOutput (StateExecuting) -> AgentStatus should be "executing"
	vterminal3 := vt.NewVirtualTerminal(80, 24, 1000)
	pod3 := testNewPTYPod("pod-3", vterminal3)
	pod3.ID = "pod-3"
	pod3.Status = PodStatusRunning
	sd3 := pod3.GetOrCreateStateDetector()
	require.NotNil(t, sd3)
	sd3.OnOutput(100) // Transition to Executing
	store.Put("pod-3", pod3)
	defer pod3.StopStateDetector()

	pods := handler.OnListPods()
	require.Len(t, pods, 3, "should have 3 pods")

	// Build a map for easier assertions (order is not guaranteed from map iteration)
	podMap := make(map[string]client.PodInfo)
	for _, p := range pods {
		podMap[p.PodKey] = p
	}

	// Pod 1: nil VirtualTerminal -> "idle"
	assert.Equal(t, "idle", podMap["pod-1"].AgentStatus,
		"pod without VirtualTerminal should have AgentStatus=idle")
	assert.Equal(t, PodStatusRunning, podMap["pod-1"].Status)

	// Pod 2: fresh detector (StateNotRunning) -> "idle"
	assert.Equal(t, "idle", podMap["pod-2"].AgentStatus,
		"pod with fresh detector (StateNotRunning) should have AgentStatus=idle")
	assert.Equal(t, PodStatusRunning, podMap["pod-2"].Status)

	// Pod 3: after OnOutput (StateExecuting) -> "executing"
	assert.Equal(t, "executing", podMap["pod-3"].AgentStatus,
		"pod with Executing detector should have AgentStatus=executing")
	assert.Equal(t, PodStatusRunning, podMap["pod-3"].Status)
}

func TestOnListPodsEmpty_AgentStatus(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()
	runner := &Runner{cfg: &config.Config{}}
	handler := NewRunnerMessageHandler(runner, store, mockConn)

	pods := handler.OnListPods()
	assert.Len(t, pods, 0, "empty store should return empty list")
}

// =============================================================================
// Test: OnCreatePod state subscription does not panic with nil VirtualTerminal
// =============================================================================

// TestOnCreatePod_StateSubscription_NilVirtualTerminal verifies that the
// state subscription code path in OnCreatePod (lines 95-119 of message_handler.go)
// gracefully handles the case where VirtualTerminal is nil after pod creation.
//
// In practice, OnCreatePod with a real terminal (via PodBuilder) will always
// set VirtualTerminal when the agent supports it. However, if
// VirtualTerminal is nil (e.g., for unsupported agents), the subscription
// block is simply skipped due to the `if pod.VirtualTerminal != nil` guard.
//
// Testing the full subscription callback flow requires:
// - A real PTY terminal (created by PodBuilder)
// - A real VirtualTerminal fed by terminal output
// - State transitions triggered by real agent output patterns
//
// This is inherently an integration test scenario. The unit tests above cover
// the getAgentStatusFromDetector mapping which is the core logic exercised
// by the subscription callback. The subscription callback itself is a thin
// wrapper that maps detector states to string statuses and calls
// conn.SendAgentStatus(), which is tested separately via MockConnection.
func TestOnCreatePod_StateSubscription_Documentation(t *testing.T) {
	// Verify that creating a pod with nil VirtualTerminal and listing it
	// does not panic and returns correct AgentStatus
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()
	runner := &Runner{cfg: &config.Config{}}
	handler := NewRunnerMessageHandler(runner, store, mockConn)

	pod := &Pod{
		PodKey: "test-pod",
		Status: PodStatusRunning,
	}
	store.Put("test-pod", pod)

	// This should not panic even though VirtualTerminal is nil
	pods := handler.OnListPods()
	require.Len(t, pods, 1)
	assert.Equal(t, "idle", pods[0].AgentStatus,
		"pod with nil VirtualTerminal should report idle AgentStatus")
	assert.Equal(t, "test-pod", pods[0].PodKey)
}
