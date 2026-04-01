package runner

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/aggregator"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/vt"
)

// ---------------------------------------------------------------------------
// Test 1: PTYPodIODeps nil defense
// Verify that PTYPodIO methods work safely when deps functions are nil.
// ---------------------------------------------------------------------------

func TestPTYPodIO_NilDeps_GetAgentStatus(t *testing.T) {
	comps := &PTYComponents{}
	io := NewPTYPodIO("test", comps, PTYPodIODeps{})
	// Should return "unknown" without panic
	assert.Equal(t, "unknown", io.GetAgentStatus())
}

func TestPTYPodIO_NilDeps_SubscribeStateChange(t *testing.T) {
	comps := &PTYComponents{}
	io := NewPTYPodIO("test", comps, PTYPodIODeps{})
	// Should not panic when deps.SubscribeState is nil
	io.SubscribeStateChange("id", func(s string) {})
}

func TestPTYPodIO_NilDeps_UnsubscribeStateChange(t *testing.T) {
	comps := &PTYComponents{}
	io := NewPTYPodIO("test", comps, PTYPodIODeps{})
	// Should not panic when deps.UnsubscribeState is nil
	io.UnsubscribeStateChange("id")
}

func TestPTYPodIO_NilDeps_Teardown(t *testing.T) {
	comps := &PTYComponents{}
	io := NewPTYPodIO("test", comps, PTYPodIODeps{})
	// Should return "" without panic when GetPTYError is nil
	assert.Equal(t, "", io.Teardown())
}

// ---------------------------------------------------------------------------
// Test 2: PTYPodIODeps injection works correctly
// Verify that injected functions are actually called.
// ---------------------------------------------------------------------------

func TestPTYPodIO_DepsInjection_GetOrCreateDetector(t *testing.T) {
	vterm := vt.NewVirtualTerminal(80, 24, 100)
	pod := &Pod{
		PodKey:     "inject-test",
		vtProvider: func() *vt.VirtualTerminal { return vterm },
	}
	comps := &PTYComponents{VirtualTerminal: vterm}
	io := NewPTYPodIO("inject-test", comps, PTYPodIODeps{
		GetOrCreateDetector: pod.GetOrCreateStateDetector,
	})
	// Should be able to get state through injected dep
	status := io.GetAgentStatus()
	// New detector defaults to "idle" (StateNotRunning)
	assert.Equal(t, "idle", status)
	pod.StopStateDetector()
}

func TestPTYPodIO_DepsInjection_GetPTYError(t *testing.T) {
	pod := &Pod{PodKey: "err-test"}
	pod.SetPTYError("test error message")
	comps := &PTYComponents{}
	io := NewPTYPodIO("err-test", comps, PTYPodIODeps{
		GetPTYError: pod.GetPTYError,
	})
	result := io.Teardown()
	assert.Equal(t, "test error message", result)
}

// ---------------------------------------------------------------------------
// Test 3: TerminalAccess / SessionAccess type assertion pattern
// Verify that PTY pods satisfy TerminalAccess but ACP pods do not, and
// that ACP pods satisfy SessionAccess but PTY pods do not.
// ---------------------------------------------------------------------------

func TestTerminalAccess_PTYPodSatisfies(t *testing.T) {
	vterm := vt.NewVirtualTerminal(80, 24, 100)
	pod := testNewPTYPod("pty-ta", vterm)
	_, ok := pod.IO.(TerminalAccess)
	assert.True(t, ok, "PTY pod IO should implement TerminalAccess")
}

func TestTerminalAccess_ACPPodDoesNotSatisfy(t *testing.T) {
	var io PodIO = NewACPPodIO(newTestACPClient(), "acp-ta")
	_, ok := io.(TerminalAccess)
	assert.False(t, ok, "ACP pod IO should NOT implement TerminalAccess")
}

func TestSessionAccess_ACPPodSatisfies(t *testing.T) {
	var io PodIO = NewACPPodIO(newTestACPClient(), "acp-sa")
	_, ok := io.(SessionAccess)
	assert.True(t, ok, "ACP pod IO should implement SessionAccess")
}

func TestSessionAccess_PTYPodDoesNotSatisfy(t *testing.T) {
	vterm := vt.NewVirtualTerminal(80, 24, 100)
	pod := testNewPTYPod("pty-sa", vterm)
	_, ok := pod.IO.(SessionAccess)
	assert.False(t, ok, "PTY pod IO should NOT implement SessionAccess")
}

// ---------------------------------------------------------------------------
// Test 4: OnObservePod behavior for PTY vs ACP
// Test that OnObservePod correctly handles both modes via TerminalAccess.
// ---------------------------------------------------------------------------

func TestOnObservePod_PTYMode_ReturnsCursorAndScreen(t *testing.T) {
	runner, mockConn := NewTestRunner(t)
	vterm := vt.NewVirtualTerminal(80, 24, 100)
	pod := testNewPTYPod("observe-pty", vterm)
	runner.podStore.Put("observe-pty", pod)

	handler := runner.messageHandler
	err := handler.OnObservePod(client.ObservePodRequest{
		RequestID: "req-1", PodKey: "observe-pty",
		Lines: 10, IncludeScreen: true,
	})
	require.NoError(t, err)

	// Should have sent result with cursor position and screen
	events := mockConn.GetEvents()
	require.NotEmpty(t, events)
}

func TestOnObservePod_ACPMode_ReturnsZeroCursorNoScreen(t *testing.T) {
	runner, mockConn := NewTestRunner(t)
	pod := &Pod{
		PodKey:          "observe-acp",
		InteractionMode: InteractionModeACP,
		Status:          PodStatusRunning,
	}
	pod.IO = NewACPPodIO(newTestACPClient(), "observe-acp")
	runner.podStore.Put("observe-acp", pod)

	handler := runner.messageHandler
	err := handler.OnObservePod(client.ObservePodRequest{
		RequestID: "req-2", PodKey: "observe-acp",
		Lines: 10, IncludeScreen: true,
	})
	require.NoError(t, err)

	events := mockConn.GetEvents()
	require.NotEmpty(t, events)
}

// ---------------------------------------------------------------------------
// Test 5: WriteOutput through TerminalAccess in error handler
// ---------------------------------------------------------------------------

func TestCreatePTYErrorHandler_WritesViaTerminalAccess(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()
	r := &Runner{cfg: &config.Config{}}
	handler := NewRunnerMessageHandler(r, store, mockConn)

	agg := aggregator.NewSmartAggregator(nil)
	comps := &PTYComponents{Aggregator: agg}
	pod := &Pod{PodKey: "ta-write", Status: PodStatusRunning}
	pod.IO = NewPTYPodIO("ta-write", comps, PTYPodIODeps{})

	errorHandler := handler.createPTYErrorHandler("ta-write", pod)
	errorHandler(fmt.Errorf("test I/O error"))

	// Verify error was written to aggregator via TerminalAccess
	assert.Greater(t, agg.BufferLen(), 0)
}
