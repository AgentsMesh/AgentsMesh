package runner

// PodIO abstracts the core interaction for a Pod.
// All interaction modes (PTY, ACP, future modes) must implement this interface.
// Mode-specific capabilities are exposed through optional extension interfaces
// (TerminalAccess, SessionAccess) that consumers discover via type assertion.
type PodIO interface {
	// Mode returns the interaction mode: "pty" or "acp".
	Mode() string

	// --- I/O (both modes implement meaningfully) ---

	// SendInput sends text input to the pod.
	SendInput(text string) error

	// GetSnapshot returns a text snapshot of recent pod output.
	GetSnapshot(lines int) (string, error)

	// GetAgentStatus returns the current agent execution status.
	// Possible values: "executing", "waiting", "idle", "unknown".
	GetAgentStatus() string

	// SubscribeStateChange registers a callback for agent state changes.
	SubscribeStateChange(id string, cb func(newStatus string))

	// UnsubscribeStateChange removes a state change subscription.
	UnsubscribeStateChange(id string)

	// GetPID returns the process ID of the underlying process.
	// Returns 0 if no process is available (e.g., ACP mode).
	GetPID() int

	// --- Lifecycle (both modes implement) ---

	// Start starts the underlying process.
	Start() error

	// Stop shuts down the pod's I/O pipeline.
	Stop()

	// Teardown cleans up mode-specific infrastructure and returns any
	// captured error message for inclusion in the termination event.
	// Must be called BEFORE DisconnectRelay and Stop.
	Teardown() string

	// SetExitHandler registers a callback invoked when the process exits.
	SetExitHandler(handler func(exitCode int))

	// SetIOErrorHandler registers a callback for fatal I/O read errors.
	SetIOErrorHandler(handler func(error))

	// Detach detaches from the underlying process without stopping it.
	Detach()
}

// TerminalAccess provides optional PTY terminal capabilities.
// Consumers discover this via type assertion: if ta, ok := pod.IO.(TerminalAccess); ok { ... }
// Only PTY-mode pods implement this interface.
type TerminalAccess interface {
	// SendKeys sends special key sequences (e.g., "ctrl+c", "enter").
	SendKeys(keys []string) error

	// Resize changes the terminal dimensions.
	// Returns (true, nil) if resize was performed.
	Resize(cols, rows int) (resized bool, err error)

	// CursorPosition returns the terminal cursor position (row, col).
	CursorPosition() (row, col int)

	// GetScreenSnapshot returns the current visible screen content.
	GetScreenSnapshot() string

	// Redraw triggers a terminal redraw (e.g., after reconnection).
	Redraw() error

	// WriteOutput writes raw data to the output pipeline for relay forwarding.
	WriteOutput(data []byte)
}

// SessionAccess provides optional structured session capabilities.
// Consumers discover this via type assertion: if sa, ok := pod.IO.(SessionAccess); ok { ... }
// Only ACP-mode pods implement this interface.
type SessionAccess interface {
	// RespondToPermission responds to a pending permission request.
	RespondToPermission(requestID string, approved bool) error

	// CancelSession cancels the active session.
	CancelSession() error

	// NotifyStateChange propagates state changes to subscribers.
	NotifyStateChange(state string)
}
