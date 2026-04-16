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
//   - agents/claude: Claude stream-json NDJSON protocol
//   - agents/codex: Codex app-server JSON-RPC protocol
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
	// updatedInput is optional; when non-nil, it replaces the tool's original input.
	RespondToPermission(requestID string, approved bool, updatedInput map[string]any) error

	// CancelSession cancels the active session's processing.
	CancelSession(sessionID string) error

	// SendControlRequest sends an outgoing control_request to the agent CLI
	// and blocks until a control_response is received (or timeout).
	// Only supported by transports that implement bidirectional control protocol
	// (e.g., Claude stream-json). Others return ErrControlNotSupported.
	SendControlRequest(sessionID string, subtype string, payload map[string]any) (map[string]any, error)

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
	commandMap = map[string]string{} // command name → transport type
)

// RegisterTransport registers a named transport factory.
// Typically called from init() in agent subpackages.
// Panics if the name is already registered (detects accidental duplicates).
func RegisterTransport(name string, factory TransportFactory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := registry[name]; exists {
		panic("acp: duplicate transport registration: " + name)
	}
	registry[name] = factory
}

// RegisterCommandMapping maps an agent command name to a transport type.
// Typically called from init() alongside RegisterTransport.
// Panics if the command is already mapped (detects accidental duplicates).
func RegisterCommandMapping(commandName, transportType string) {
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := commandMap[commandName]; exists {
		panic("acp: duplicate command mapping: " + commandName)
	}
	commandMap[commandName] = transportType
}

// RegisterAgent registers a transport factory and command→transport mapping atomically.
func RegisterAgent(commandName string, transportType string, factory TransportFactory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := registry[transportType]; exists {
		panic("acp: duplicate transport registration: " + transportType)
	}
	if _, exists := commandMap[commandName]; exists {
		panic("acp: duplicate command mapping: " + commandName)
	}
	registry[transportType] = factory
	commandMap[commandName] = transportType
}

// TransportTypeForCommand returns the transport type for a given command name.
// Returns TransportTypeACP if no mapping is registered.
func TransportTypeForCommand(command string) string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	if tt, ok := commandMap[command]; ok {
		return tt
	}
	return TransportTypeACP
}

// NewTransport creates a transport by name from the registry.
// Falls back to standard ACP if the name is not registered.
func NewTransport(name string, callbacks EventCallbacks, logger *slog.Logger) Transport {
	registryMu.RLock()
	factory, ok := registry[name]
	registryMu.RUnlock()
	if ok {
		return factory(callbacks, logger)
	}
	logger.Warn("unknown transport type, falling back to ACP", "type", name)
	return NewACPTransport(callbacks, logger)
}
