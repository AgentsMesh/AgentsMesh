package runner

import (
	"errors"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
)

// ErrKeysNotSupported is returned when special key input is attempted in ACP mode.
var ErrKeysNotSupported = errors.New("special keys not supported in ACP mode")

// ACPPodIO wraps an ACPClient to implement PodIO for ACP-mode pods.
// State mapping: processing→executing, idle→idle, waiting_permission→waiting.
type ACPPodIO struct {
	client *acp.ACPClient
}

// NewACPPodIO creates a PodIO that delegates to an ACPClient.
func NewACPPodIO(client *acp.ACPClient) *ACPPodIO {
	return &ACPPodIO{client: client}
}

func (a *ACPPodIO) Mode() string { return "acp" }

func (a *ACPPodIO) SendInput(text string) error {
	return a.client.SendPrompt(text)
}

func (a *ACPPodIO) GetSnapshot(lines int) (string, error) {
	return a.client.GetRecentMessages(lines), nil
}

func (a *ACPPodIO) GetAgentStatus() string {
	return mapACPState(a.client.State())
}

func (a *ACPPodIO) SubscribeStateChange(id string, cb func(newStatus string)) {
	// ACP state changes are delivered via the OnStateChange callback
	// which is set during ACPClient creation. We store this subscriber
	// and the wiring is done when building the ACP pod.
	// For now, this is a no-op since the OnStateChange callback is
	// configured at client construction time via EventCallbacks.
	// The message_handler_acp.go wireACPPod() sets up the bridge.
}

func (a *ACPPodIO) UnsubscribeStateChange(id string) {
	// See SubscribeStateChange comment.
}

func (a *ACPPodIO) SendKeys(keys []string) error {
	if len(keys) > 0 {
		return ErrKeysNotSupported
	}
	return nil
}

func (a *ACPPodIO) Resize(cols, rows int) (bool, error) {
	return false, nil // No-op: ACP has no terminal to resize
}

func (a *ACPPodIO) GetPID() int {
	return 0 // ACP has no shell process
}

func (a *ACPPodIO) CursorPosition() (row, col int) {
	return 0, 0 // ACP has no terminal cursor
}

func (a *ACPPodIO) GetScreenSnapshot() string {
	return "" // ACP has no terminal screen
}

func (a *ACPPodIO) Start() error {
	return a.client.Start()
}

func (a *ACPPodIO) Stop() {
	a.client.Stop()
}

func (a *ACPPodIO) SetExitHandler(handler func(exitCode int)) {
	// ACP exit is handled via the Done() channel and OnExit callback
	// configured at client construction time.
}

func (a *ACPPodIO) Redraw() error {
	return nil // No-op: ACP has no terminal to redraw
}

func (a *ACPPodIO) Detach() {
	// No-op: ACP has no terminal to detach from
}

func (a *ACPPodIO) WriteOutput(data []byte) {
	// No-op: ACP has no aggregator output pipeline
}

func (a *ACPPodIO) RespondToPermission(requestID string, approved bool) error {
	err := a.client.RespondToPermission(requestID, approved)
	if err == nil {
		a.client.RemovePendingPermission(requestID)
	}
	return err
}

func (a *ACPPodIO) CancelSession() error {
	return a.client.CancelSession()
}

// mapACPState maps ACP client states to backend-compatible status strings.
func mapACPState(acpState string) string {
	switch acpState {
	case acp.StateProcessing:
		return "executing"
	case acp.StateIdle:
		return "idle"
	case acp.StateWaitingPermission:
		return "waiting"
	case acp.StateInitializing:
		return "executing"
	case acp.StateStopped, acp.StateUninitialized:
		return "idle"
	default:
		return "idle"
	}
}

func (a *ACPPodIO) Teardown() string {
	return "" // ACP has no aggregator or PTY logger to clean up
}

// Compile-time interface check.
var _ PodIO = (*ACPPodIO)(nil)
