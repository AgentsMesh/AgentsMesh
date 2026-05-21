package mockagent

import (
	"sync"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
)

// runtimeState is the per-process composition root: it owns the writer
// (single source of outbound JSON-RPC), a WaitGroup so EOF can drain
// in-flight scenario goroutines, and two narrow sub-aggregates that
// carry the actually-stateful pieces (pendingRegistry for IPC round-trips,
// configState for AgentsMesh control plane).
//
// Keeping the sub-aggregates separate means new scenarios reach for
// exactly the surface they need (state.pending vs state.config) and the
// runtime entrypoint stays thin.
type runtimeState struct {
	writer  *acp.Writer
	wg      sync.WaitGroup
	pending *pendingRegistry
	config  *configState
}

func newRuntimeState(writer *acp.Writer) *runtimeState {
	return &runtimeState{
		writer:  writer,
		pending: newPendingRegistry(),
		config:  newConfigState(),
	}
}

func (s *runtimeState) deliverResponse(msg *acp.JSONRPCMessage) { s.pending.deliver(msg) }

func (s *runtimeState) setPermissionMode(mode string) { s.config.setPermissionMode(mode) }
func (s *runtimeState) setModel(model string)         { s.config.setModel(model) }
func (s *runtimeState) setThinkingLevel(level string) { s.config.setThinkingLevel(level) }
