//go:build !windows

package mcp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// TryReclaimPort attempts to reclaim a port held by a stale runner process.
// It finds the process listening on the port, verifies it's an agentsmesh-runner,
// and kills it (SIGTERM then SIGKILL). Returns true if the port was freed.
//
// Called in two places:
//   - At startup (cmd_run.go) to clean up stale runners missed by pidfile cleanup
//   - In MCP server Start() as a fallback if bind fails
func TryReclaimPort(port int) bool {
	log := logger.MCP()

	pid, err := findListenerPID(port)
	if err != nil || pid == 0 {
		log.Debug("Could not find process holding port", "port", port, "error", err)
		return false
	}

	// Never kill ourselves
	if pid == os.Getpid() {
		return false
	}

	// Verify it's an agentsmesh-runner process
	name := getProcessName(pid)
	if !strings.Contains(name, "agentsmesh-runner") && !strings.Contains(name, "agentsmesh-runn") {
		log.Info("Port held by non-runner process, skipping reclaim",
			"port", port, "pid", pid, "process", name)
		return false
	}

	log.Warn("Reclaiming MCP port from stale runner",
		"port", port, "pid", pid, "process", name)

	// Send SIGTERM for graceful shutdown
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		log.Error("Failed to send SIGTERM to stale runner", "pid", pid, "error", err)
		return false
	}

	// Wait up to 5s for graceful exit
	if waitForProcessExit(pid, 5*time.Second) {
		log.Info("Stale runner terminated gracefully", "pid", pid)
		return true
	}

	// Escalate to SIGKILL
	log.Warn("SIGTERM ignored, sending SIGKILL", "pid", pid)
	if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
		log.Error("Failed to send SIGKILL to stale runner", "pid", pid, "error", err)
		return false
	}

	if waitForProcessExit(pid, 2*time.Second) {
		log.Info("Stale runner force-killed", "pid", pid)
		return true
	}

	log.Error("Failed to kill stale runner", "pid", pid)
	return false
}

// findListenerPID finds the PID of the process listening on the given TCP port.
// Tries ss (Linux), then lsof (macOS), as both provide PID directly.
func findListenerPID(port int) (int, error) {
	// Try ss first (available on most Linux systems, gives PID directly)
	if pid, err := findListenerPIDViaSS(port); err == nil && pid > 0 {
		return pid, nil
	}

	// Fallback to lsof (macOS and other Unix)
	return findListenerPIDViaLsof(port)
}

// findListenerPIDViaSS uses ss to find the PID listening on a port.
// Output format: "users:(("agentsmesh-runn",pid=1436796,fd=7))"
func findListenerPIDViaSS(port int) (int, error) {
	out, err := exec.Command("ss", "-tlnp", fmt.Sprintf("sport = :%d", port)).Output()
	if err != nil {
		return 0, fmt.Errorf("ss failed: %w", err)
	}

	// Parse pid=NNNN from the output
	for _, line := range strings.Split(string(out), "\n") {
		idx := strings.Index(line, "pid=")
		if idx < 0 {
			continue
		}
		rest := line[idx+4:]
		end := strings.IndexAny(rest, ",)")
		if end < 0 {
			continue
		}
		if pid, err := strconv.Atoi(rest[:end]); err == nil {
			return pid, nil
		}
	}

	return 0, nil
}

// findListenerPIDViaLsof uses lsof to find the PID listening on a port (macOS fallback).
func findListenerPIDViaLsof(port int) (int, error) {
	out, err := exec.Command("lsof", "-ti", fmt.Sprintf("tcp:%d", port), "-sTCP:LISTEN").Output()
	if err != nil {
		return 0, fmt.Errorf("lsof failed: %w", err)
	}
	pidStr := strings.TrimSpace(strings.Split(string(out), "\n")[0])
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0, nil
	}
	return pid, nil
}

// getProcessName reads the process name from /proc/<pid>/comm (Linux)
// or falls back to ps (macOS).
func getProcessName(pid int) string {
	data, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "comm"))
	if err != nil {
		// Fallback for macOS
		out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "comm=").Output()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(out))
	}
	return strings.TrimSpace(string(data))
}

// waitForProcessExit polls until the process exits or timeout.
func waitForProcessExit(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(pid, 0); err != nil {
			return true
		}
		time.Sleep(200 * time.Millisecond)
	}
	return false
}
