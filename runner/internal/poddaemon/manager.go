package poddaemon

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/process"
)

// PodDaemonManager manages the lifecycle of pod daemon sessions.
type PodDaemonManager struct {
	sandboxesDir  string // Base directory containing per-pod sandbox directories
	runnerBinPath string
}

// CreateOpts holds options for creating a new daemon session.
type CreateOpts struct {
	PodKey  string
	Agent   string
	Command string
	Args    []string
	WorkDir string
	Env     []string
	Cols    int
	Rows    int

	SandboxPath    string
	RepositoryURL  string
	Branch         string
	TicketSlug     string
	VTHistoryLimit int
}

// authTokenBytes is the number of random bytes for IPC authentication tokens.
const authTokenBytes = 32

// NewPodDaemonManager creates a new manager.
// sandboxesDir is the base directory containing per-pod sandbox directories (each with pod_daemon.json).
func NewPodDaemonManager(sandboxesDir string) (*PodDaemonManager, error) {
	binPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("get executable path: %w", err)
	}

	return &PodDaemonManager{
		sandboxesDir:  sandboxesDir,
		runnerBinPath: binPath,
	}, nil
}

// generateAuthToken creates a cryptographically random hex-encoded token.
func generateAuthToken() (string, error) {
	b := make([]byte, authTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate auth token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// CreateSession spawns a new daemon process and returns a connected daemonPTY.
func (m *PodDaemonManager) CreateSession(opts CreateOpts) (*daemonPTY, *PodDaemonState, error) {
	log := slog.Default()

	if opts.SandboxPath == "" {
		return nil, nil, fmt.Errorf("sandbox path is required")
	}

	authToken, err := generateAuthToken()
	if err != nil {
		return nil, nil, err
	}

	state := &PodDaemonState{
		PodKey:         opts.PodKey,
		Agent:          opts.Agent,
		AuthToken:      authToken,
		SandboxPath:    opts.SandboxPath,
		WorkDir:        opts.WorkDir,
		RepositoryURL:  opts.RepositoryURL,
		Branch:         opts.Branch,
		TicketSlug:     opts.TicketSlug,
		Command:        opts.Command,
		Args:           opts.Args,
		Cols:           opts.Cols,
		Rows:           opts.Rows,
		StartedAt:      time.Now(),
		VTHistoryLimit: opts.VTHistoryLimit,
	}

	// Save state before starting daemon (daemon reads it on startup).
	// IPCAddr is empty — the daemon will fill it after binding a port.
	if err := SaveState(state); err != nil {
		return nil, nil, fmt.Errorf("save state: %w", err)
	}

	configPath := StatePath(opts.SandboxPath)
	pid, err := startDaemon(m.runnerBinPath, configPath, opts.SandboxPath, opts.Env)
	if err != nil {
		_ = DeleteState(opts.SandboxPath)
		return nil, nil, fmt.Errorf("start daemon: %w", err)
	}

	log.Info("daemon started, waiting for IPC", "pid", pid)

	// Wait for daemon to bind a port and write it to state file
	dpty, updatedState, err := m.waitForDaemon(opts.SandboxPath, authToken, pid)
	if err != nil {
		status := daemonProcessStatus(pid)
		log.Error("daemon failed to become ready",
			"pod_key", opts.PodKey, "pid", pid, "process_status", status, "error", err)
		captureDaemonLog(log, opts.SandboxPath, opts.PodKey)
		_ = DeleteState(opts.SandboxPath)
		return nil, nil, fmt.Errorf("connect to daemon (pid %d, %s): %w", pid, status, err)
	}

	return dpty, updatedState, nil
}

// waitForDaemon polls the state file until the daemon writes its IPC address,
// then connects. It also checks if the daemon process is still alive to fail fast.
func (m *PodDaemonManager) waitForDaemon(sandboxPath, authToken string, pid int) (*daemonPTY, *PodDaemonState, error) {
	const maxAttempts = 50
	const retryDelay = 100 * time.Millisecond

	var lastErr error
	for i := 0; i < maxAttempts; i++ {
		// Read state file to check if daemon has written its address
		state, err := LoadState(sandboxPath)
		if err == nil && state.IPCAddr != "" {
			// Defensive check: verify the auth token hasn't been tampered with
			if state.AuthToken != authToken {
				return nil, nil, fmt.Errorf("auth token mismatch in state file (possible tampering)")
			}
			dpty, err := connectDaemon(connectOpts{Addr: state.IPCAddr, AuthToken: authToken})
			if err == nil {
				return dpty, state, nil
			}
			lastErr = err
		} else if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("daemon has not written IPC address yet")
		}

		// Fail fast if daemon process is no longer alive
		if pid > 0 && process.IsAlive(pid) != nil {
			return nil, nil, fmt.Errorf("daemon process (pid %d) exited before IPC ready: %w", pid, lastErr)
		}

		time.Sleep(retryDelay)
	}
	return nil, nil, fmt.Errorf("daemon did not become ready within %v: %w", time.Duration(maxAttempts)*retryDelay, lastErr)
}

// AttachSession connects to an existing daemon via IPC.
func (m *PodDaemonManager) AttachSession(state *PodDaemonState) (*daemonPTY, error) {
	return connectDaemon(connectOpts{Addr: state.IPCAddr, AuthToken: state.AuthToken})
}

// RecoverSessions scans the sandboxes directory for existing daemon state files.
func (m *PodDaemonManager) RecoverSessions() ([]*PodDaemonState, error) {
	entries, err := os.ReadDir(m.sandboxesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read sandboxes dir: %w", err)
	}

	var sessions []*PodDaemonState
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		sandboxPath := filepath.Join(m.sandboxesDir, entry.Name())
		state, err := LoadState(sandboxPath)
		if err != nil {
			continue // No state file or corrupt
		}
		sessions = append(sessions, state)
	}
	return sessions, nil
}

// CleanupSession removes the state file for a session.
// TCP loopback connections have no file artifacts to clean up.
func (m *PodDaemonManager) CleanupSession(sandboxPath string) error {
	return DeleteState(sandboxPath)
}

const daemonLogFile = "pod_daemon.log"

// captureDaemonLog reads the daemon log and outputs to runner log for diagnostics.
// Called when daemon fails to become ready, before sandbox cleanup destroys the log.
func captureDaemonLog(log *slog.Logger, sandboxPath, podKey string) {
	logPath := filepath.Join(sandboxPath, daemonLogFile)
	data, err := os.ReadFile(logPath)
	if err != nil {
		log.Error("pod daemon log unavailable",
			"pod_key", podKey, "path", logPath, "error", err)
		return
	}
	if len(data) == 0 {
		log.Error("pod daemon log is empty (daemon likely crashed before any Go code executed)",
			"pod_key", podKey, "path", logPath)
		return
	}
	const maxLen = 2048
	if len(data) > maxLen {
		data = data[len(data)-maxLen:]
	}
	log.Error("pod daemon log (process exited before IPC ready)",
		"pod_key", podKey, "log", strings.TrimSpace(string(data)))
}

// daemonProcessStatus returns a human-readable status of the daemon process.
func daemonProcessStatus(pid int) string {
	if pid <= 0 {
		return "unknown"
	}
	if process.IsAlive(pid) == nil {
		return "alive"
	}
	return "dead"
}
