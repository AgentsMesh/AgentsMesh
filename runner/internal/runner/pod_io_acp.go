package runner

import (
	"sync"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// ACPPodIO wraps an ACPClient to implement PodIO for ACP-mode pods.
// State mapping: processing→executing, idle→idle, waiting_permission→waiting.
type ACPPodIO struct {
	client *acp.ACPClient
	podKey string // for structured logging context

	// State change subscribers for event-driven autopilot integration.
	stateSubsMu sync.RWMutex
	stateSubs   map[string]func(newStatus string)
}

// NewACPPodIO creates a PodIO that delegates to an ACPClient.
func NewACPPodIO(client *acp.ACPClient, podKey string) *ACPPodIO {
	return &ACPPodIO{
		client:    client,
		podKey:    podKey,
		stateSubs: make(map[string]func(newStatus string)),
	}
}

func (a *ACPPodIO) Mode() string { return InteractionModeACP }

func (a *ACPPodIO) SendInput(text string) error {
	logger.Pod().Debug("ACP sending prompt", "pod_key", a.podKey)
	return a.client.SendPrompt(text)
}

func (a *ACPPodIO) GetSnapshot(lines int) (string, error) {
	return a.client.GetRecentMessages(lines), nil
}

func (a *ACPPodIO) GetAgentStatus() string {
	return mapACPState(a.client.State())
}

func (a *ACPPodIO) SubscribeStateChange(id string, cb func(newStatus string)) {
	a.stateSubsMu.Lock()
	a.stateSubs[id] = cb
	a.stateSubsMu.Unlock()
}

func (a *ACPPodIO) UnsubscribeStateChange(id string) {
	a.stateSubsMu.Lock()
	delete(a.stateSubs, id)
	a.stateSubsMu.Unlock()
}

// NotifyStateChange is called by the ACP event wiring (message_handler_acp.go)
// to propagate state changes to all subscribers (e.g. Autopilot).
func (a *ACPPodIO) NotifyStateChange(acpState string) {
	mapped := mapACPState(acpState)
	a.stateSubsMu.RLock()
	defer a.stateSubsMu.RUnlock()
	for _, cb := range a.stateSubs {
		cb(mapped)
	}
}

func (a *ACPPodIO) GetPID() int {
	return 0 // ACP has no shell process
}

func (a *ACPPodIO) Stop() {
	logger.Pod().Info("ACP stopping", "pod_key", a.podKey)
	a.client.Stop()
}

func (a *ACPPodIO) SetExitHandler(handler func(exitCode int)) {
	// ACP exit is handled via the Done() channel and OnExit callback
	// configured at client construction time.
}

func (a *ACPPodIO) Detach() {
	// No-op: ACP has no terminal to detach from
}

func (a *ACPPodIO) RespondToPermission(requestID string, approved bool) error {
	logger.Pod().Info("ACP responding to permission",
		"pod_key", a.podKey, "request_id", requestID, "approved", approved)
	err := a.client.RespondToPermission(requestID, approved)
	if err != nil {
		logger.Pod().Error("ACP permission response failed",
			"pod_key", a.podKey, "request_id", requestID, "error", err)
	} else {
		a.client.RemovePendingPermission(requestID)
	}
	return err
}

func (a *ACPPodIO) CancelSession() error {
	logger.Pod().Info("ACP cancelling session", "pod_key", a.podKey)
	if err := a.client.CancelSession(); err != nil {
		logger.Pod().Error("ACP cancel session failed",
			"pod_key", a.podKey, "error", err)
		return err
	}
	return nil
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

func (a *ACPPodIO) Start() error {
	return nil // ACP start is done externally by wireAndStartACPPod
}

func (a *ACPPodIO) SetIOErrorHandler(_ func(error)) {
	// No-op: ACP has no PTY
}

func (a *ACPPodIO) Teardown() string {
	return "" // ACP has no aggregator or PTY logger to clean up
}

// Compile-time interface checks.
var _ PodIO = (*ACPPodIO)(nil)
var _ SessionAccess = (*ACPPodIO)(nil)
