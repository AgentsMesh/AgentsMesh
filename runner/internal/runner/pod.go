package runner

import (
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/clipboard"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/relay"
	"github.com/anthropics/agentsmesh/runner/internal/terminal"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/aggregator"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/detector"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/vt"
)

// Pod represents an active terminal pod
type Pod struct {
	ID               string
	PodKey           string
	AgentType        string
	RepositoryURL    string
	Branch           string
	SandboxPath      string
	Clipboard        clipboard.Backend          // Clipboard backend for image paste support
	Terminal         *terminal.Terminal
	VirtualTerminal  *vt.VirtualTerminal        // Virtual terminal for state management and snapshots
	Aggregator       *aggregator.SmartAggregator // Output aggregator for adaptive frame rate
	RelayClient      relay.RelayClient          // WebSocket client for Relay connection (interface for testability)
	relayMu          sync.RWMutex               // Protects RelayClient field
	StartedAt        time.Time
	Status           string              // Pod status - use statusMu for thread-safe access
	statusMu         sync.RWMutex        // Protects Status field
	TicketSlug       string              // Ticket slug for worktree-based pods (e.g., "TBD-123")
	PTYLogger        *aggregator.PTYLogger // PTY logger for debugging (optional)

	// StateDetector for multi-signal state detection.
	// This is a foundational service that can be used by Autopilot, Monitor, or other components.
	stateDetector   *ManagedStateDetector
	stateDetectorMu sync.RWMutex

	// Token refresh channel - used when relay token expires and needs to be refreshed
	tokenRefreshCh   chan string
	tokenRefreshMu   sync.Mutex

	// PTY error message stored when a fatal PTY read error occurs.
	// Used by the exit handler to include the error reason in the termination event.
	ptyErrorMsg   string
	ptyErrorMu    sync.Mutex
}

// PodStatus constants
const (
	PodStatusInitializing = "initializing"
	PodStatusRunning      = "running"
	PodStatusStopped      = "stopped"
	PodStatusFailed       = "failed"
)

// SetPTYError stores a PTY error message for the exit handler to pick up.
func (p *Pod) SetPTYError(msg string) {
	p.ptyErrorMu.Lock()
	defer p.ptyErrorMu.Unlock()
	p.ptyErrorMsg = msg
}

// GetPTYError returns the stored PTY error message, if any.
func (p *Pod) GetPTYError() string {
	p.ptyErrorMu.Lock()
	defer p.ptyErrorMu.Unlock()
	return p.ptyErrorMsg
}

// SetStatus sets the pod status in a thread-safe manner
func (p *Pod) SetStatus(status string) {
	p.statusMu.Lock()
	oldStatus := p.Status
	p.Status = status
	p.statusMu.Unlock()

	if oldStatus != status {
		logger.Pod().Debug("Pod status changed", "pod_key", p.PodKey, "from", oldStatus, "to", status)
	}
}

// GetStatus returns the pod status in a thread-safe manner
func (p *Pod) GetStatus() string {
	p.statusMu.RLock()
	defer p.statusMu.RUnlock()
	return p.Status
}

// SetRelayClient sets the relay client in a thread-safe manner
func (p *Pod) SetRelayClient(client relay.RelayClient) {
	p.relayMu.Lock()
	defer p.relayMu.Unlock()
	p.RelayClient = client
}

// GetRelayClient returns the relay client in a thread-safe manner
func (p *Pod) GetRelayClient() relay.RelayClient {
	p.relayMu.RLock()
	defer p.relayMu.RUnlock()
	return p.RelayClient
}

// LockRelay acquires the relay write lock for atomic check-and-set operations
// (e.g., OnSubscribeTerminal). Caller must call UnlockRelay when done.
func (p *Pod) LockRelay()   { p.relayMu.Lock() }
func (p *Pod) UnlockRelay() { p.relayMu.Unlock() }

// HasRelayClient returns whether a relay client is connected
func (p *Pod) HasRelayClient() bool {
	p.relayMu.RLock()
	defer p.relayMu.RUnlock()
	return p.RelayClient != nil && p.RelayClient.IsConnected()
}

// DisconnectRelay disconnects and clears the relay client
func (p *Pod) DisconnectRelay() {
	p.relayMu.Lock()
	defer p.relayMu.Unlock()
	if p.RelayClient != nil {
		logger.Pod().Debug("Disconnecting relay client", "pod_key", p.PodKey)
		p.RelayClient.Stop()
		p.RelayClient = nil
	}
	// Clear aggregator relay output - will fall back to gRPC
	if p.Aggregator != nil {
		p.Aggregator.SetRelayOutput(nil)
	}
}

// GetOrCreateStateDetector returns the state detector for this pod, creating one if needed.
// Returns the detector.StateDetector interface for use by any component.
// Delegates to getOrCreateStateDetectorInternal to avoid duplicating DCL logic.
func (p *Pod) GetOrCreateStateDetector() detector.StateDetector {
	d := p.getOrCreateStateDetectorInternal()
	if d == nil {
		return nil // Explicit nil to avoid non-nil interface wrapping nil pointer
	}
	return d
}

// SubscribeStateChange subscribes to state change events.
// This is the preferred way to receive state updates (event-driven, single-direction data flow).
// The subscriber ID must be unique; duplicate IDs will replace existing subscriptions.
// Returns false if VirtualTerminal is not available.
func (p *Pod) SubscribeStateChange(id string, cb func(detector.StateChangeEvent)) bool {
	d := p.getOrCreateStateDetectorInternal()
	if d == nil {
		return false
	}
	d.Subscribe(id, cb)
	return true
}

// UnsubscribeStateChange removes a state change subscription by ID.
func (p *Pod) UnsubscribeStateChange(id string) {
	p.stateDetectorMu.RLock()
	d := p.stateDetector
	p.stateDetectorMu.RUnlock()

	if d != nil {
		d.Unsubscribe(id)
	}
}

// getOrCreateStateDetectorInternal returns the internal ManagedStateDetector, creating one if needed.
// This is an internal method that returns the concrete type.
func (p *Pod) getOrCreateStateDetectorInternal() *ManagedStateDetector {
	p.stateDetectorMu.RLock()
	if p.stateDetector != nil {
		defer p.stateDetectorMu.RUnlock()
		return p.stateDetector
	}
	p.stateDetectorMu.RUnlock()

	// Need to create - acquire write lock
	p.stateDetectorMu.Lock()
	defer p.stateDetectorMu.Unlock()

	// Double check after acquiring write lock
	if p.stateDetector != nil {
		return p.stateDetector
	}

	// Create new detector if VirtualTerminal is available
	if p.VirtualTerminal != nil {
		p.stateDetector = NewManagedStateDetector(p.VirtualTerminal)
	}
	return p.stateDetector
}

// NotifyStateDetectorWithScreen notifies the state detector about new output
// and provides the current screen lines for state analysis.
func (p *Pod) NotifyStateDetectorWithScreen(bytes int, screenLines []string) {
	p.stateDetectorMu.RLock()
	detector := p.stateDetector
	p.stateDetectorMu.RUnlock()

	if detector != nil {
		detector.OnOutput(bytes)
		if screenLines != nil {
			detector.OnScreenUpdate(screenLines)
		}
	}
}

// StopStateDetector stops the state detector if running.
func (p *Pod) StopStateDetector() {
	p.stateDetectorMu.Lock()
	defer p.stateDetectorMu.Unlock()

	if p.stateDetector != nil {
		p.stateDetector.Stop()
		p.stateDetector = nil
	}
}

// WaitForNewToken waits for a new token to be delivered via tokenRefreshCh.
func (p *Pod) WaitForNewToken(timeout time.Duration) string {
	p.tokenRefreshMu.Lock()
	if p.tokenRefreshCh == nil {
		p.tokenRefreshCh = make(chan string, 1)
	}
	ch := p.tokenRefreshCh
	p.tokenRefreshMu.Unlock()

	select {
	case token := <-ch:
		return token
	case <-time.After(timeout):
		return ""
	}
}

// DeliverNewToken delivers a new token to the waiting goroutine.
func (p *Pod) DeliverNewToken(token string) {
	p.tokenRefreshMu.Lock()
	defer p.tokenRefreshMu.Unlock()

	if p.tokenRefreshCh == nil {
		p.tokenRefreshCh = make(chan string, 1)
	}

	// Non-blocking send
	select {
	case p.tokenRefreshCh <- token:
	default:
	}
}
