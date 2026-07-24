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

	// Perpetual mode: auto-restart on clean exit
	Perpetual    bool
	RestartCount int
	lifecycleMu  sync.Mutex // Serializes exit, restart, terminate, and shutdown.

	// Relay client (mode-agnostic, protected by relayMu)
	RelayClient relay.RelayClient
	relayMu     sync.RWMutex
	// relayTransitionMu serializes pointer ownership with mode-specific
	// publisher attach/detach and runtime replacement.
	relayTransitionMu   sync.Mutex
	relayEpoch          uint64
	relayRuntimeEpoch   uint64
	relayRuntimeBlocked bool
	relayLocalEpoch     uint64
	relayLocalActive    bool
	relayTransportEpoch uint64
	relayTransportBlock bool
	runtimeActivity     *podActivity
	cloudActivity       *podActivity
	localActivity       *podActivity

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
