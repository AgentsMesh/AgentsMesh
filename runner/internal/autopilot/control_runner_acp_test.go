package autopilot

import "testing"

// Compile-time interface compliance checks.
var (
	_ ControlProcess = (*ControlRunner)(nil)
	_ ControlProcess = (*AcpControlProcess)(nil)
)

func TestAcpControlProcess_StopBeforeStart(t *testing.T) {
	// Stop on unstarted process should not panic.
	p := &AcpControlProcess{}
	p.Stop() // should be safe
}

func TestAcpControlProcess_GetSessionIDBeforeStart(t *testing.T) {
	p := &AcpControlProcess{}
	if id := p.GetSessionID(); id != "" {
		t.Errorf("GetSessionID() on unstarted process = %q, want empty", id)
	}
}

func TestAcpControlProcess_SetSessionIDIsNoop(t *testing.T) {
	p := &AcpControlProcess{}
	p.SetSessionID("test-id") // should not panic or change state
	if id := p.GetSessionID(); id != "" {
		t.Errorf("GetSessionID() after SetSessionID = %q, want empty (no-op)", id)
	}
}
