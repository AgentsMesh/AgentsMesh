package runner

// PodIO abstracts the interaction mode for a Pod.
// MCP tools, Autopilot, and other high-level consumers use this interface
// instead of directly accessing Terminal or ACPClient, making them
// mode-agnostic (PTY vs ACP).
type PodIO interface {
	// Mode returns the interaction mode: "pty" or "acp".
	Mode() string

	// SendInput sends text input to the pod.
	// PTY: writes bytes to terminal stdin.
	// ACP: sends session/prompt to the agent.
	SendInput(text string) error

	// GetSnapshot returns a text snapshot of recent pod output.
	// PTY: returns VirtualTerminal.GetOutput(lines).
	// ACP: returns recent messages formatted as text.
	GetSnapshot(lines int) (string, error)

	// GetAgentStatus returns the current agent execution status as a string.
	// Possible values: "executing", "waiting", "idle", "unknown".
	GetAgentStatus() string

	// SubscribeStateChange registers a callback for agent state changes.
	// The callback receives the new status string.
	SubscribeStateChange(id string, cb func(newStatus string))

	// UnsubscribeStateChange removes a state change subscription.
	UnsubscribeStateChange(id string)

	// Start begins the pod's I/O pipeline.
	// PTY: starts the terminal process.
	// ACP: launches subprocess, performs initialize handshake, creates session.
	Start() error

	// SendKeys sends special key sequences (e.g., "ctrl+c", "enter", "up").
	// PTY: maps key names to escape sequences and writes to stdin.
	// ACP: returns ErrKeysNotSupported.
	SendKeys(keys []string) error

	// Resize changes the terminal dimensions.
	// Returns (true, nil) if a real terminal resize was performed (PTY),
	// or (false, nil) if resize is not applicable (ACP).
	// Callers use the bool to decide whether to send protocol confirmations.
	Resize(cols, rows int) (resized bool, err error)

	// GetPID returns the process ID of the underlying shell.
	// PTY: returns Terminal.PID(). ACP: returns 0.
	GetPID() int

	// CursorPosition returns the terminal cursor position (row, col).
	// PTY: returns VirtualTerminal.CursorPosition(). ACP: returns (0, 0).
	CursorPosition() (row, col int)

	// GetScreenSnapshot returns the current visible screen content.
	// PTY: returns VirtualTerminal.GetScreenSnapshot(). ACP: returns "".
	GetScreenSnapshot() string

	// Stop shuts down the pod's I/O pipeline.
	// PTY: stops the terminal.
	// ACP: stops the ACPClient and subprocess.
	Stop()

	// Teardown cleans up mode-specific infrastructure resources (aggregator,
	// loggers, etc.) and returns any captured early output or error message
	// for inclusion in the pod termination event.
	// Must be called BEFORE DisconnectRelay and Stop — the aggregator's
	// final flush may still need the relay connection.
	// PTY: stops Aggregator, closes PTYLogger, returns early buffer or PTY error.
	// ACP: no-op, returns "".
	Teardown() string

	// SetExitHandler registers a callback invoked when the underlying
	// process exits.
	SetExitHandler(handler func(exitCode int))

	// Redraw triggers a terminal redraw (e.g., after reconnection).
	// PTY: delegates to Terminal.Redraw(). ACP: returns nil (no-op).
	Redraw() error

	// Detach detaches from the underlying process without stopping it.
	// PTY: delegates to Terminal.Detach(). ACP: no-op.
	Detach()

	// WriteOutput writes raw data to the output pipeline for relay forwarding.
	// PTY: writes to the aggregator so it appears in the browser terminal.
	// ACP: no-op.
	WriteOutput(data []byte)

	// RespondToPermission responds to a pending permission request.
	// PTY: returns ErrNotSupported. ACP: delegates to ACPClient.
	RespondToPermission(requestID string, approved bool) error

	// CancelSession cancels the active session.
	// PTY: returns ErrNotSupported. ACP: delegates to ACPClient.
	CancelSession() error
}
