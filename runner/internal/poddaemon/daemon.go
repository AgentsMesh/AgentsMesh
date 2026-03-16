package poddaemon

import (
	"encoding/binary"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

// RunDaemon is the main entry point for the daemon process.
// It is invoked when the runner binary detects _AGENTSMESH_POD_DAEMON env var.
// configPath is the full path to pod_daemon.json.
func RunDaemon(configPath string) {
	log := slog.Default()
	log.Info("pod daemon starting", "config", configPath)

	// configPath is the full path; LoadState expects the sandbox directory
	sandboxPath := filepath.Dir(configPath)
	state, err := LoadState(sandboxPath)
	if err != nil {
		log.Error("failed to load state", "error", err)
		os.Exit(1)
	}

	// Start the child process with PTY
	proc, err := startDaemonProcess(
		state.Command, state.Args, state.WorkDir, os.Environ(),
		state.Cols, state.Rows,
	)
	if err != nil {
		log.Error("failed to start process", "error", err)
		os.Exit(1)
	}
	defer proc.Close()

	log.Info("child process started", "pid", proc.Pid())

	// Update state with daemon PID
	state.DaemonPID = os.Getpid()
	if err := SaveState(state); err != nil {
		log.Error("failed to save state", "error", err)
	}

	// Listen on IPC
	listener, err := Listen(state.IPCPath)
	if err != nil {
		log.Error("failed to listen on IPC", "error", err, "path", state.IPCPath)
		os.Exit(1)
	}
	defer listener.Close()
	defer os.Remove(state.IPCPath)

	log.Info("IPC listening", "path", state.IPCPath)

	// Accept client connections and forward I/O
	d := &daemonServer{
		proc:     proc,
		listener: listener,
		exitDone: make(chan struct{}),
		orphanCh: make(chan struct{}),
		log:      log,
		state:    state,
	}

	// Allow tests to override the orphan check interval via env var.
	if v := os.Getenv("_AGENTSMESH_ORPHAN_CHECK_INTERVAL_SEC"); v != "" {
		if sec, err := strconv.Atoi(v); err == nil && sec > 0 {
			d.orphanCheckInterval = time.Duration(sec) * time.Second
		}
	}

	// Wait for child process exit in background
	go func() {
		code, err := proc.Wait()
		if err != nil {
			log.Error("process wait error", "error", err)
		}
		log.Info("child process exited", "exit_code", code)
		d.exitCode = code
		close(d.exitDone) // broadcast to all listeners
	}()

	d.run()
}

// daemonServer manages the IPC server and PTY I/O forwarding.
type daemonServer struct {
	proc     daemonProcess
	listener net.Listener
	exitCode int            // set before exitDone is closed
	exitDone chan struct{}   // closed when child process exits (broadcast)
	orphanCh chan struct{}   // closed when state file is deleted (orphan protection)
	log      *slog.Logger
	state    *PodDaemonState

	// orphanCheckInterval controls how often orphanChecker polls.
	// Defaults to 60s in production; tests can inject a shorter value.
	orphanCheckInterval time.Duration

	// clientMu protects the client pointer only. Hold briefly to read/swap
	// the pointer — never hold while doing network I/O.
	clientMu sync.Mutex
	client   net.Conn

	// connWriteMu serializes writes to the IPC connection. This is separate
	// from clientMu so that ptyReader's potentially slow data writes don't
	// block control-plane operations (Pong, Exit notification) from acquiring
	// the client pointer.
	connWriteMu sync.Mutex
}

func (d *daemonServer) run() {
	// PTY reader goroutine: reads from PTY and forwards to current client
	go d.ptyReader()

	// Accept connections until child exits
	go d.acceptLoop()

	// Orphan protection: exit if state file is deleted (Runner cleanup)
	go d.orphanChecker()

	// Wait for child exit or orphan signal
	select {
	case <-d.exitDone:
		d.log.Info("daemon shutting down (child exited)", "exit_code", d.exitCode)

		// Notify connected client about exit
		d.clientMu.Lock()
		client := d.client
		d.clientMu.Unlock()
		if client != nil {
			d.connWriteMu.Lock()
			payload := make([]byte, 4)
			binary.BigEndian.PutUint32(payload, uint32(int32(d.exitCode)))
			if err := WriteMessage(client, MsgExit, payload); err != nil {
				d.log.Debug("failed to send exit notification", "error", err)
			}
			d.connWriteMu.Unlock()
		}

	case <-d.orphanCh:
		d.log.Info("daemon shutting down (state file deleted, orphan protection)")
		// Kill the child process and exit
		d.proc.GracefulStop()
		select {
		case <-d.exitDone:
		case <-time.After(5 * time.Second):
			d.proc.Kill()
		}
	}
}

// orphanChecker periodically checks if the state file still exists.
// If deleted, signals the daemon to exit (orphan protection).
func (d *daemonServer) orphanChecker() {
	interval := d.orphanCheckInterval
	if interval <= 0 {
		interval = 60 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if _, err := os.Stat(StatePath(d.state.SandboxPath)); os.IsNotExist(err) {
				d.log.Info("state file deleted, triggering orphan protection")
				close(d.orphanCh)
				return
			}
		case <-d.exitDone:
			return
		}
	}
}
