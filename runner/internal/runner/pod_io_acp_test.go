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
	io := NewACPPodIO(client)

	if got := io.Mode(); got != "acp" {
		t.Errorf("Mode() = %q, want %q", got, "acp")
	}
}

func TestACPPodIO_SendKeys_WithKeys_ReturnsError(t *testing.T) {
	client := newTestACPClient()
	io := NewACPPodIO(client)

	err := io.SendKeys([]string{"ctrl+c"})
	if err == nil {
		t.Fatal("SendKeys with keys should return an error")
	}
	if err != ErrKeysNotSupported {
		t.Errorf("SendKeys error = %v, want ErrKeysNotSupported", err)
	}
}

func TestACPPodIO_SendKeys_Empty_NoError(t *testing.T) {
	client := newTestACPClient()
	io := NewACPPodIO(client)

	if err := io.SendKeys(nil); err != nil {
		t.Errorf("SendKeys(nil) = %v, want nil", err)
	}
	if err := io.SendKeys([]string{}); err != nil {
		t.Errorf("SendKeys([]) = %v, want nil", err)
	}
}

func TestACPPodIO_Resize_NopReturnsFalse(t *testing.T) {
	client := newTestACPClient()
	io := NewACPPodIO(client)

	resized, err := io.Resize(120, 40)
	if err != nil {
		t.Errorf("Resize() error = %v, want nil", err)
	}
	if resized {
		t.Error("Resize() resized = true, want false (ACP has no terminal)")
	}
}

func TestACPPodIO_GetPID_ReturnsZero(t *testing.T) {
	client := newTestACPClient()
	io := NewACPPodIO(client)

	if got := io.GetPID(); got != 0 {
		t.Errorf("GetPID() = %d, want 0", got)
	}
}

func TestACPPodIO_CursorPosition_ReturnsZeroZero(t *testing.T) {
	client := newTestACPClient()
	io := NewACPPodIO(client)

	row, col := io.CursorPosition()
	if row != 0 || col != 0 {
		t.Errorf("CursorPosition() = (%d, %d), want (0, 0)", row, col)
	}
}

func TestACPPodIO_GetScreenSnapshot_ReturnsEmpty(t *testing.T) {
	client := newTestACPClient()
	io := NewACPPodIO(client)

	if got := io.GetScreenSnapshot(); got != "" {
		t.Errorf("GetScreenSnapshot() = %q, want empty string", got)
	}
}

func TestACPPodIO_Redraw_NopReturnsNil(t *testing.T) {
	client := newTestACPClient()
	io := NewACPPodIO(client)

	if err := io.Redraw(); err != nil {
		t.Errorf("Redraw() = %v, want nil", err)
	}
}

func TestACPPodIO_Detach_NoPanic(t *testing.T) {
	client := newTestACPClient()
	io := NewACPPodIO(client)

	// Should not panic — it's a no-op.
	io.Detach()
}

func TestACPPodIO_WriteOutput_NoPanic(t *testing.T) {
	client := newTestACPClient()
	io := NewACPPodIO(client)

	// Should not panic — it's a no-op.
	io.WriteOutput([]byte("some data"))
	io.WriteOutput(nil)
	io.WriteOutput([]byte{})
}

func TestACPPodIO_Teardown_ReturnsEmpty(t *testing.T) {
	client := newTestACPClient()
	io := NewACPPodIO(client)

	if got := io.Teardown(); got != "" {
		t.Errorf("Teardown() = %q, want empty string", got)
	}
}

func TestACPPodIO_SetExitHandler_NoPanic(t *testing.T) {
	client := newTestACPClient()
	io := NewACPPodIO(client)

	// Should not panic — it's a no-op.
	io.SetExitHandler(func(exitCode int) {})
	io.SetExitHandler(nil)
}

func TestACPPodIO_SubscribeStateChange_NoPanic(t *testing.T) {
	client := newTestACPClient()
	io := NewACPPodIO(client)

	// Should not panic — currently a no-op per implementation comment.
	io.SubscribeStateChange("test-sub", func(newStatus string) {})
}

func TestACPPodIO_UnsubscribeStateChange_NoPanic(t *testing.T) {
	client := newTestACPClient()
	io := NewACPPodIO(client)

	// Should not panic — currently a no-op per implementation comment.
	io.UnsubscribeStateChange("test-sub")
}

// --- GetAgentStatus delegates to mapACPState(client.State()) ---

func TestACPPodIO_GetAgentStatus_Unstarted(t *testing.T) {
	client := newTestACPClient()
	io := NewACPPodIO(client)

	// An unstarted client has state "uninitialized", which maps to "unknown".
	got := io.GetAgentStatus()
	if got != "unknown" {
		t.Errorf("GetAgentStatus() on unstarted client = %q, want %q", got, "unknown")
	}
}

// --- GetSnapshot delegates to client.GetRecentMessages ---

func TestACPPodIO_GetSnapshot_EmptyClient(t *testing.T) {
	client := newTestACPClient()
	io := NewACPPodIO(client)

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
		{acp.StateUninitialized, "unknown"},
		{acp.StateStopped, "unknown"},
		{"", "unknown"},
		{"some_random_state", "unknown"},
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
	io := NewACPPodIO(client)

	// Stop on an unstarted client should not panic.
	// ACPClient.Stop() uses sync.Once and checks cmd.Process == nil.
	io.Stop()
}

// --- Compile-time interface assertion ---

func TestACPPodIO_ImplementsPodIO(t *testing.T) {
	// This is also checked at compile time in pod_io_acp.go,
	// but having it in a test makes the intent explicit.
	var _ PodIO = (*ACPPodIO)(nil)
}
