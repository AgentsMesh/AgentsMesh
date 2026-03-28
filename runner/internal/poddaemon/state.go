package poddaemon

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

const stateFileName = "pod_daemon.json"

// PodDaemonState holds the persistent state of a pod daemon process.
type PodDaemonState struct {
	PodKey         string    `json:"pod_key"`
	Agent          string    `json:"agent"`
	IPCAddr        string    `json:"ipc_addr"`   // TCP loopback address (e.g. "127.0.0.1:12345")
	AuthToken      string    `json:"auth_token"` // hex-encoded 32-byte random token for IPC authentication
	DaemonPID      int       `json:"daemon_pid"`
	SandboxPath    string    `json:"sandbox_path"`
	WorkDir        string    `json:"work_dir"`
	RepositoryURL  string    `json:"repository_url,omitempty"`
	Branch         string    `json:"branch,omitempty"`
	TicketSlug     string    `json:"ticket_slug,omitempty"`
	Command        string    `json:"command"`
	Args           []string  `json:"args"`
	Cols           int       `json:"cols"`
	Rows           int       `json:"rows"`
	StartedAt      time.Time `json:"started_at"`
	VTHistoryLimit int       `json:"vt_history_limit"`
}

// SaveState atomically writes the daemon state to disk.
func SaveState(state *PodDaemonState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		slog.Error("Failed to marshal pod daemon state", "pod_key", state.PodKey, "error", err)
		return fmt.Errorf("marshal state: %w", err)
	}

	path := StatePath(state.SandboxPath)
	tmpPath := path + ".tmp"

	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		slog.Error("Failed to write temp pod daemon state", "path", tmpPath, "error", err)
		return fmt.Errorf("write temp state: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		slog.Error("Failed to rename pod daemon state file", "path", path, "error", err)
		return fmt.Errorf("rename state file: %w", err)
	}
	return nil
}

// LoadState reads the daemon state from the given sandbox path.
func LoadState(sandboxPath string) (*PodDaemonState, error) {
	data, err := os.ReadFile(StatePath(sandboxPath))
	if err != nil {
		slog.Error("Failed to read pod daemon state", "sandbox", sandboxPath, "error", err)
		return nil, fmt.Errorf("read state: %w", err)
	}

	var state PodDaemonState
	if err := json.Unmarshal(data, &state); err != nil {
		slog.Error("Failed to unmarshal pod daemon state", "sandbox", sandboxPath, "error", err)
		return nil, fmt.Errorf("unmarshal state: %w", err)
	}
	return &state, nil
}

// DeleteState removes the daemon state file.
func DeleteState(sandboxPath string) error {
	if err := os.Remove(StatePath(sandboxPath)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete state: %w", err)
	}
	return nil
}

// StatePath returns the file path for the daemon state.
func StatePath(sandboxPath string) string {
	return filepath.Join(sandboxPath, stateFileName)
}
