package runner

import (
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/relay"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/vt"
)

// Pod represents an active execution environment (PTY or ACP).
// Mode-specific components live inside PodIO and PodRelay implementations;
// Pod itself only holds mode-agnostic state.
type Pod struct {
	ID            string
	PodKey        string
	Agent         string
	RepositoryURL string
	Branch        string
	SandboxPath   string

	// Interaction mode: "pty" (default) or "acp"
	InteractionMode string

	// Unified I/O interface — all consumers should use this.
	IO PodIO

	// Mode-specific relay behavior (OCP: eliminates IsACPMode branches in relay layer)
	Relay PodRelay

	// Launch configuration (used by session recovery after Runner restart)
	LaunchCommand string
	LaunchArgs    []string
	WorkDir       string
	LaunchEnv     []string // Full environment slice for subprocess

	// Relay client (mode-agnostic, protected by relayMu)
	RelayClient relay.RelayClient
	relayMu     sync.RWMutex

	StartedAt  time.Time
	Status     string       // Pod status - use statusMu for thread-safe access
	statusMu   sync.RWMutex // Protects Status field
	TicketSlug string       // Ticket slug for worktree-based pods (e.g., "TBD-123")

	// StateDetector for multi-signal state detection.
	stateDetector   *ManagedStateDetector
	stateDetectorMu sync.RWMutex

	// vtProvider returns the VirtualTerminal for lazy StateDetector creation.
	// Injected by the build path (PTY only); nil for ACP pods.
	vtProvider func() *vt.VirtualTerminal

	// Token refresh channel - used when relay token expires
	tokenRefreshCh chan string
	tokenRefreshMu sync.Mutex

	// PTY error message stored when a fatal PTY read error occurs.
	ptyErrorMsg string
	ptyErrorMu  sync.Mutex
}

// PodStatus constants
const (
	PodStatusInitializing = "initializing"
	PodStatusRunning      = "running"
	PodStatusStopped      = "stopped"
	PodStatusFailed       = "failed"
)

// Interaction mode constants
const (
	InteractionModePTY = "pty"
	InteractionModeACP = "acp"
)

// IsACPMode returns true if the pod uses ACP interaction mode.
func (p *Pod) IsACPMode() bool {
	return p.InteractionMode == InteractionModeACP
}

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

// LockRelay acquires the relay write lock for atomic check-and-set operations.
func (p *Pod) LockRelay()   { p.relayMu.Lock() }
func (p *Pod) UnlockRelay() { p.relayMu.Unlock() }

// HasRelayClient returns whether a relay client is connected
func (p *Pod) HasRelayClient() bool {
	p.relayMu.RLock()
	defer p.relayMu.RUnlock()
	return p.RelayClient != nil && p.RelayClient.IsConnected()
}

// DisconnectRelay disconnects and clears the relay client.
func (p *Pod) DisconnectRelay() {
	p.relayMu.Lock()
	rc := p.RelayClient
	if rc != nil {
		p.RelayClient = nil
	}
	p.relayMu.Unlock()
	if rc != nil {
		logger.Pod().Debug("Disconnecting relay client", "pod_key", p.PodKey)
		rc.Stop()
	}
	if p.Relay != nil {
		p.Relay.OnRelayDisconnected()
	}
}
