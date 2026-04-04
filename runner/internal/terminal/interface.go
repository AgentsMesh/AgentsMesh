package terminal

// TerminalInterface defines the interface for PTY terminal operations.
// This interface abstracts Terminal for testing and decoupling.
type TerminalInterface interface {
	// Start starts the terminal process.
	Start() error

	// Stop stops the terminal and releases resources.
	Stop()

	// Write writes data to the terminal stdin.
	Write(data []byte) error

	// Resize changes the terminal window size.
	Resize(cols, rows int) error

	// PID returns the process ID of the terminal process.
	PID() int

	// IsClosed returns whether the terminal is closed.
	IsClosed() bool

	// SetOutputHandler sets the callback for terminal output.
	// Must be called before Start().
	SetOutputHandler(handler func([]byte))

	// SetExitHandler sets the callback for process exit.
	// Must be called before Start().
	SetExitHandler(handler func(int))
}

// TerminalBackpressure defines the interface for terminal backpressure control.
// Use this interface when you need to manage PTY read flow control.
type TerminalBackpressure interface {
	// PauseRead pauses reading from the PTY to apply backpressure.
	PauseRead()

	// ResumeRead resumes reading from the PTY.
	ResumeRead()

	// IsReadPaused returns whether PTY reading is paused.
	IsReadPaused() bool
}

// Ensure Terminal implements both interfaces.
var (
	_ TerminalInterface    = (*Terminal)(nil)
	_ TerminalBackpressure = (*Terminal)(nil)
)
