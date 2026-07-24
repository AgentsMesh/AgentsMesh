package terminal

import "sync"

// PTYFactory is a function that creates a PtyProcess.
// When set in Options, it replaces the default platform-specific startPTY.
// This enables dependency injection for Pod Daemon mode.
type PTYFactory func(command string, args []string, workDir string, env []string, cols, rows int) (PtyProcess, error)

// Options for creating a new terminal.
type Options struct {
	Command  string
	Args     []string
	WorkDir  string
	Env      map[string]string
	Rows     int
	Cols     int
	Label    string // Identifier for log correlation (e.g., pod_key)
	OnOutput func([]byte)
	OnExit   func(int)

	// PTYFactory overrides the default platform PTY creation.
	// Used by Pod Daemon to inject daemonPTY instead of direct PTY.
	PTYFactory PTYFactory
}

// Terminal represents a PTY terminal session.
type Terminal struct {
	// Command configuration (set in New, consumed in Start)
	command string
	args    []string
	workDir string
	env     []string
	label   string // Identifier for log correlation (e.g., pod_key)

	// PTY process handle (set in Start)
	proc ptyProcess

	// Custom PTY factory (nil = use default platform startPTY)
	ptyFactory PTYFactory

	mu          sync.Mutex
	lifecycleMu sync.Mutex
	closed      bool
	stopping    bool
	onOutput    func([]byte)
	onExit      func(int)
	// readOrder is injected before Start and remains immutable while running.
	// It orders each PTY Read + onOutput callback against external resize/teardown.
	readOrder sync.Locker

	// onPTYError is called when readOutput encounters a fatal I/O error
	// (not timeout, not EOF, not normal close). This allows the runner to
	// send an error message to the frontend before the process is killed.
	onPTYError func(error)

	// Terminal size (set at creation, used when starting PTY)
	rows int
	cols int

	// Lifecycle synchronization
	doneCh         chan struct{} // Closed when process exits (signaled by waitExit)
	ptyCloseOnce   sync.Once     // Ensures PTY file descriptor is closed exactly once
	readDoneCh     chan struct{} // Closed after readOutput and its last callback exit
	readDoneOnce   sync.Once
	readActiveCh   chan struct{} // Closed when the reader starts its first PTY read
	readActiveOnce sync.Once
	readProgressCh chan struct{} // Edge-triggered notification after a dispatched read
	readProgress   uint64
	readStarted    bool
	processExited  bool

	// Backpressure control (ttyd-style flow control)
	// When paused, readOutput() blocks to prevent unbounded memory growth
	readPaused  bool          // Whether PTY reading is paused
	readPauseMu sync.RWMutex  // Protects readPaused flag
	resumeCh    chan struct{} // Signal to resume reading
}
