package runner

import (
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/relay"
	"github.com/anthropics/agentsmesh/runner/internal/terminal"
)

// Pod represents an active terminal pod
type Pod struct {
	ID               string
	PodKey           string
	AgentType        string
	RepositoryURL    string
	Branch           string
	SandboxPath      string
	Terminal         *terminal.Terminal
	VirtualTerminal  *terminal.VirtualTerminal  // Virtual terminal for state management and snapshots
	Aggregator       *terminal.SmartAggregator  // Output aggregator for adaptive frame rate
	RelayClient      *relay.Client              // WebSocket client for Relay connection
	relayMu          sync.RWMutex               // Protects RelayClient field
	StartedAt        time.Time
	Status           string              // Pod status - use statusMu for thread-safe access
	statusMu         sync.RWMutex        // Protects Status field
	TicketIdentifier string              // Ticket ID for worktree-based pods
	OnOutput         func([]byte)        // Output callback
	OnExit           func(int)           // Exit callback
	PTYLogger        *terminal.PTYLogger // PTY logger for debugging (optional)

	// StateDetector for multi-signal state detection (used by Autopilot)
	stateDetector    *multiSignalDetectorAdapter
	stateDetectorMu  sync.RWMutex

	// Token refresh channel - used when relay token expires and needs to be refreshed
	tokenRefreshCh   chan string
	tokenRefreshMu   sync.Mutex
}

// PodStatus constants
const (
	PodStatusInitializing = "initializing"
	PodStatusRunning      = "running"
	PodStatusStopped      = "stopped"
	PodStatusFailed       = "failed"
)

// NewVirtualTerminal creates a new VirtualTerminal.
// This is a wrapper for terminal.NewVirtualTerminal to avoid importing terminal package in message_handler.
func NewVirtualTerminal(cols, rows, historyLimit int) *terminal.VirtualTerminal {
	return terminal.NewVirtualTerminal(cols, rows, historyLimit)
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
func (p *Pod) SetRelayClient(client *relay.Client) {
	p.relayMu.Lock()
	defer p.relayMu.Unlock()
	p.RelayClient = client
}

// GetRelayClient returns the relay client in a thread-safe manner
func (p *Pod) GetRelayClient() *relay.Client {
	p.relayMu.RLock()
	defer p.relayMu.RUnlock()
	return p.RelayClient
}

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
func (p *Pod) GetOrCreateStateDetector() *multiSignalDetectorAdapter {
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
		p.stateDetector = newMultiSignalDetectorAdapter(p.VirtualTerminal)
	}
	return p.stateDetector
}

// NotifyStateDetectorOutput notifies the state detector about new output.
// Deprecated: Use NotifyStateDetectorWithScreen for single-direction data flow.
func (p *Pod) NotifyStateDetectorOutput(bytes int) {
	p.stateDetectorMu.RLock()
	detector := p.stateDetector
	p.stateDetectorMu.RUnlock()

	if detector != nil {
		detector.OnOutput(bytes)
	}
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
