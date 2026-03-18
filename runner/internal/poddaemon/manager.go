package poddaemon

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// PodDaemonManager manages the lifecycle of pod daemon sessions.
type PodDaemonManager struct {
	workspaceRoot string
	runnerBinPath string
}

// CreateOpts holds options for creating a new daemon session.
type CreateOpts struct {
	PodKey    string
	AgentType string
	Command   string
	Args      []string
	WorkDir   string
	Env       []string
	Cols      int
	Rows      int

	SandboxPath    string
	RepositoryURL  string
	Branch         string
	TicketSlug     string
	VTHistoryLimit int
}

// NewPodDaemonManager creates a new manager.
// workspaceRoot is the base directory for sandbox directories.
func NewPodDaemonManager(workspaceRoot string) (*PodDaemonManager, error) {
	binPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("get executable path: %w", err)
	}

	return &PodDaemonManager{
		workspaceRoot: workspaceRoot,
		runnerBinPath: binPath,
	}, nil
}

// CreateSession spawns a new daemon process and returns a connected daemonPTY.
func (m *PodDaemonManager) CreateSession(opts CreateOpts) (*daemonPTY, *PodDaemonState, error) {
	log := slog.Default()

	if opts.SandboxPath == "" {
		return nil, nil, fmt.Errorf("sandbox path is required")
	}

	ipcPath := IPCPath(opts.SandboxPath, opts.PodKey)

	state := &PodDaemonState{
		PodKey:         opts.PodKey,
		AgentType:      opts.AgentType,
		IPCPath:        ipcPath,
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

	// Save state before starting daemon (daemon reads it on startup)
	if err := SaveState(state); err != nil {
		return nil, nil, fmt.Errorf("save state: %w", err)
	}

	configPath := StatePath(opts.SandboxPath)
	pid, err := startDaemon(m.runnerBinPath, configPath, opts.SandboxPath, opts.Env)
	if err != nil {
		_ = DeleteState(opts.SandboxPath)
		return nil, nil, fmt.Errorf("start daemon: %w", err)
	}

	state.DaemonPID = pid
	if err := SaveState(state); err != nil {
		log.Error("failed to update state with daemon PID", "error", err)
	}

	log.Info("daemon started, waiting for IPC", "pid", pid, "ipc", ipcPath)

	// Wait for daemon to start listening on IPC
	dpty, err := m.waitForDaemon(ipcPath)
	if err != nil {
		_ = DeleteState(opts.SandboxPath)
		return nil, nil, fmt.Errorf("connect to daemon: %w", err)
	}

	return dpty, state, nil
}

// waitForDaemon polls the IPC path until the daemon is ready.
func (m *PodDaemonManager) waitForDaemon(ipcPath string) (*daemonPTY, error) {
	const maxAttempts = 50
	const retryDelay = 100 * time.Millisecond

	var lastErr error
	for i := 0; i < maxAttempts; i++ {
		dpty, err := connectDaemon(ipcPath)
		if err == nil {
			return dpty, nil
		}
		lastErr = err
		time.Sleep(retryDelay)
	}
	return nil, fmt.Errorf("daemon did not become ready within %v: %w", time.Duration(maxAttempts)*retryDelay, lastErr)
}

// AttachSession connects to an existing daemon via IPC.
func (m *PodDaemonManager) AttachSession(state *PodDaemonState) (*daemonPTY, error) {
	return connectDaemon(state.IPCPath)
}

// RecoverSessions scans the workspace root for existing daemon state files.
func (m *PodDaemonManager) RecoverSessions() ([]*PodDaemonState, error) {
	entries, err := os.ReadDir(m.workspaceRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read workspace root: %w", err)
	}

	var sessions []*PodDaemonState
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		sandboxPath := filepath.Join(m.workspaceRoot, entry.Name())
		state, err := LoadState(sandboxPath)
		if err != nil {
			continue // No state file or corrupt
		}
		sessions = append(sessions, state)
	}
	return sessions, nil
}

// CleanupSession removes the state file for a session.
func (m *PodDaemonManager) CleanupSession(sandboxPath string) error {
	return DeleteState(sandboxPath)
}
