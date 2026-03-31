package acp

import (
	"context"
	"io"
	"log/slog"
	"sync"
)

// Transport abstracts the wire protocol between ACPClient and an agent subprocess.
// Implementations register via RegisterTransport from their init() functions:
//   - ACPTransport: JSON-RPC 2.0 (Gemini CLI --acp, OpenCode acp) — built-in fallback
//   - claude.Transport: Claude stream-json NDJSON protocol (acp/claude subpackage)
//   - codex.Transport: Codex app-server JSON-RPC protocol (acp/codex subpackage)
type Transport interface {
	// Initialize wires the transport's I/O pipes.
	// Called BEFORE ReadLoop. Must not block on protocol messages.
	Initialize(ctx context.Context, stdin io.Writer, stdout io.Reader, stderr io.Reader) error

	// Handshake performs the protocol-specific initialization.
	// Called AFTER ReadLoop has been started in a goroutine.
	// Returns an auto-discovered sessionID (Claude) or empty string (ACP).
	Handshake(ctx context.Context) (string, error)

	// NewSession creates a new session, returning the sessionID.
	// cwd is the working directory for the session (required by standard ACP).
	// Claude transport returns the cached ID from Handshake.
	NewSession(cwd string, mcpServers map[string]any) (string, error)

	// SendPrompt delivers a prompt to the active session.
	SendPrompt(sessionID, prompt string) error

	// RespondToPermission responds to a permission request.
	RespondToPermission(requestID string, approved bool) error

	// CancelSession cancels the active session's processing.
	CancelSession(sessionID string) error

	// ReadLoop continuously reads messages from stdout and dispatches via callbacks.
	// Blocks until EOF or ctx cancellation.
	ReadLoop(ctx context.Context)

	// Close releases transport-internal resources.
	Close()
}

// --- Transport Registry ---

// TransportFactory creates a Transport with the given callbacks and logger.
type TransportFactory func(callbacks EventCallbacks, logger *slog.Logger) Transport

var (
	registryMu sync.RWMutex
	registry   = map[string]TransportFactory{}
)

// RegisterTransport registers a named transport factory.
// Typically called from init() in transport subpackages.
func RegisterTransport(name string, factory TransportFactory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = factory
}

// NewTransport creates a transport by name from the registry.
// Returns an error if the name is not registered.
func NewTransport(name string, callbacks EventCallbacks, logger *slog.Logger) Transport {
	registryMu.RLock()
	factory, ok := registry[name]
	registryMu.RUnlock()
	if ok {
		return factory(callbacks, logger)
	}
	// Fallback: unknown transport names default to standard ACP.
	logger.Warn("unknown transport type, falling back to ACP", "type", name)
	return NewACPTransport(callbacks, logger)
}
