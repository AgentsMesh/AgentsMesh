package runner

import (
	"log/slog"
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
)

// newTestACPClient creates an unstarted ACPClient suitable for unit tests.
// The client is created via NewClient() with a dummy command and no-op callbacks.
// IMPORTANT: Do NOT call Start() — it would attempt to launch a real subprocess.
func newTestACPClient() *acp.ACPClient {
	return acp.NewClient(acp.ClientConfig{
		Command: "echo",
		Logger:  slog.Default(),
	})
}

// --- ACPPodIO method tests ---

func TestACPPodIO_Mode(t *testing.T) {
	client := newTestACPClient()
	io := NewACPPodIO(client, "test-pod")

	if got := io.Mode(); got != "acp" {
		t.Errorf("Mode() = %q, want %q", got, "acp")
	}
}

func TestACPPodIO_GetPID_ReturnsZero(t *testing.T) {
	client := newTestACPClient()
	io := NewACPPodIO(client, "test-pod")

	if got := io.GetPID(); got != 0 {
		t.Errorf("GetPID() = %d, want 0", got)
	}
}

func TestACPPodIO_Detach_NoPanic(t *testing.T) {
	client := newTestACPClient()
	io := NewACPPodIO(client, "test-pod")

	// Should not panic — it's a no-op.
	io.Detach()
}

func TestACPPodIO_Teardown_ReturnsEmpty(t *testing.T) {
	client := newTestACPClient()
	io := NewACPPodIO(client, "test-pod")

	if got := io.Teardown(); got != "" {
		t.Errorf("Teardown() = %q, want empty string", got)
	}
}

func TestACPPodIO_SetExitHandler_NoPanic(t *testing.T) {
	client := newTestACPClient()
	io := NewACPPodIO(client, "test-pod")

	// Should not panic — it's a no-op.
	io.SetExitHandler(func(exitCode int) {})
	io.SetExitHandler(nil)
}

func TestACPPodIO_SubscribeStateChange_DeliveryToMultipleSubscribers(t *testing.T) {
	client := newTestACPClient()
	io := NewACPPodIO(client, "test-pod")

	var received1, received2 []string
	io.SubscribeStateChange("sub-1", func(s string) { received1 = append(received1, s) })
	io.SubscribeStateChange("sub-2", func(s string) { received2 = append(received2, s) })

	io.NotifyStateChange(acp.StateProcessing) // mapped → "executing"

	if len(received1) != 1 || received1[0] != "executing" {
		t.Errorf("sub-1 got %v, want [executing]", received1)
	}
	if len(received2) != 1 || received2[0] != "executing" {
		t.Errorf("sub-2 got %v, want [executing]", received2)
	}
}

func TestACPPodIO_UnsubscribeStateChange_StopsDelivery(t *testing.T) {
	client := newTestACPClient()
	io := NewACPPodIO(client, "test-pod")

	callCount := 0
	io.SubscribeStateChange("x", func(_ string) { callCount++ })
	io.NotifyStateChange(acp.StateIdle) // +1
	if callCount != 1 {
		t.Fatalf("expected 1 call before unsubscribe, got %d", callCount)
	}

	io.UnsubscribeStateChange("x")
	io.NotifyStateChange(acp.StateIdle) // should NOT call
	if callCount != 1 {
		t.Errorf("expected still 1 call after unsubscribe, got %d", callCount)
	}
}

func TestACPPodIO_NotifyStateChange_MapsAllStates(t *testing.T) {
	client := newTestACPClient()
	io := NewACPPodIO(client, "test-pod")

	var results []string
	io.SubscribeStateChange("rec", func(s string) { results = append(results, s) })

	io.NotifyStateChange(acp.StateProcessing)
	io.NotifyStateChange(acp.StateIdle)
	io.NotifyStateChange(acp.StateWaitingPermission)

	want := []string{"executing", "idle", "waiting"}
	if len(results) != len(want) {
		t.Fatalf("got %d results, want %d", len(results), len(want))
	}
	for i, w := range want {
		if results[i] != w {
			t.Errorf("results[%d] = %q, want %q", i, results[i], w)
		}
	}
}

// --- GetAgentStatus delegates to mapACPState(client.State()) ---

func TestACPPodIO_GetAgentStatus_Unstarted(t *testing.T) {
	client := newTestACPClient()
	io := NewACPPodIO(client, "test-pod")

	// An unstarted client has state "uninitialized", which maps to "idle".
	got := io.GetAgentStatus()
	if got != "idle" {
		t.Errorf("GetAgentStatus() on unstarted client = %q, want %q", got, "idle")
	}
}

// --- GetSnapshot delegates to client.GetRecentMessages ---

func TestACPPodIO_GetSnapshot_EmptyClient(t *testing.T) {
	client := newTestACPClient()
	io := NewACPPodIO(client, "test-pod")

	snapshot, err := io.GetSnapshot(10)
	if err != nil {
		t.Errorf("GetSnapshot() error = %v, want nil", err)
	}
	if snapshot != "" {
		t.Errorf("GetSnapshot() = %q, want empty string (no messages)", snapshot)
	}
}

// --- mapACPState tests ---

func TestMapACPState_AllMappings(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{acp.StateProcessing, "executing"},
		{acp.StateIdle, "idle"},
		{acp.StateWaitingPermission, "waiting"},
		{acp.StateInitializing, "executing"},
		{acp.StateUninitialized, "idle"},
		{acp.StateStopped, "idle"},
		{"", "idle"},
		{"some_random_state", "idle"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := mapACPState(tt.input); got != tt.want {
				t.Errorf("mapACPState(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// --- Stop on unstarted client is safe (stopOnce + nil process) ---

func TestACPPodIO_Stop_Unstarted(t *testing.T) {
	client := newTestACPClient()
	io := NewACPPodIO(client, "test-pod")

	// Stop on an unstarted client should not panic.
	// ACPClient.Stop() uses sync.Once and checks cmd.Process == nil.
	io.Stop()
}

// --- Compile-time interface assertion ---

func TestACPPodIO_ImplementsPodIO(t *testing.T) {
	// This is also checked at compile time in pod_io_acp.go,
	// but having it in a test makes the intent explicit.
	var _ PodIO = (*ACPPodIO)(nil)
	var _ SessionAccess = (*ACPPodIO)(nil)
}
