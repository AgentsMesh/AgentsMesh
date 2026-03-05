// Package pidfile manages a PID file for the runner process.
//
// The PID file (~/.agentsmesh/runner.pid) allows the runner to detect and clean up
// stale processes from previous runs that were killed ungracefully (SIGKILL, OOM).
// Without this, ports (MCP :19000, Console :19080) remain occupied and the runner
// cannot restart.
//
// File format: "<PID> <executable-name>\n"
// Example: "12345 agentsmesh-runner\n"
package pidfile

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const pidFileName = "runner.pid"

// GetPath returns the path to the PID file (~/.agentsmesh/runner.pid).
func GetPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".agentsmesh", pidFileName)
}

// Write writes the current process PID and executable name to the PID file.
// Creates ~/.agentsmesh/ directory if needed.
func Write() error {
	pidPath := GetPath()
	if pidPath == "" {
		return fmt.Errorf("cannot determine PID file path")
	}

	// Ensure directory exists
	dir := filepath.Dir(pidPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	execName := filepath.Base(os.Args[0])
	content := fmt.Sprintf("%d %s\n", os.Getpid(), execName)

	return os.WriteFile(pidPath, []byte(content), 0644)
}

// Remove deletes the PID file, but only if it contains the current process's PID.
// This prevents a race where a newer instance wrote its PID and we accidentally delete it.
func Remove() {
	pidPath := GetPath()
	if pidPath == "" {
		return
	}

	data, err := os.ReadFile(pidPath)
	if err != nil {
		return
	}

	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return
	}

	pid, err := strconv.Atoi(fields[0])
	if err != nil {
		return
	}

	// Only remove if it's our PID
	if pid == os.Getpid() {
		os.Remove(pidPath)
	}
}

// parsePIDFile reads and parses the PID file, returning the PID and executable name.
// Returns (0, "", nil) if the file doesn't exist or is corrupt (corrupt files are removed).
func parsePIDFile(pidPath string) (int, string, error) {
	data, err := os.ReadFile(pidPath)
	if os.IsNotExist(err) {
		return 0, "", nil
	}
	if err != nil {
		return 0, "", fmt.Errorf("failed to read PID file: %w", err)
	}

	fields := strings.Fields(string(data))
	if len(fields) < 2 {
		// Corrupt PID file — remove and continue
		os.Remove(pidPath)
		return 0, "", nil
	}

	pid, err := strconv.Atoi(fields[0])
	if err != nil {
		// Corrupt PID file
		os.Remove(pidPath)
		return 0, "", nil
	}

	return pid, fields[1], nil
}
