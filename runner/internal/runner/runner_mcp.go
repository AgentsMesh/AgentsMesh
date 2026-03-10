package runner

import (
	"fmt"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/detector"
)

// GetPodStatus returns the agent status for a given pod key.
// Implements mcp.PodStatusProvider interface.
func (r *Runner) GetPodStatus(podKey string) (agentStatus string, podStatus string, shellPid int, found bool) {
	pod, exists := r.podStore.Get(podKey)
	if !exists || pod == nil {
		return "idle", "not_found", 0, false
	}

	podStatus = pod.GetStatus()
	shellPid = 0
	if pod.Terminal != nil {
		shellPid = pod.Terminal.PID()
	}

	// Get agent status from StateDetector (preferred, unified 3-value: executing/waiting/idle)
	if pod.VirtualTerminal != nil {
		d := pod.GetOrCreateStateDetector()
		if d != nil {
			switch d.GetState() {
			case detector.StateExecuting:
				return "executing", podStatus, shellPid, true
			case detector.StateWaiting:
				return "waiting", podStatus, shellPid, true
			default:
				return "idle", podStatus, shellPid, true
			}
		}
	}

	// Fallback: no state detector available
	return "idle", podStatus, shellPid, true
}

// GetTerminalOutput returns the terminal output for a local pod.
// Implements mcp.LocalTerminalProvider interface.
func (r *Runner) GetTerminalOutput(podKey string, lines int) (string, error) {
	pod, exists := r.podStore.Get(podKey)
	if !exists || pod == nil {
		return "", fmt.Errorf("pod not found: %s", podKey)
	}

	if pod.VirtualTerminal == nil {
		return "", fmt.Errorf("virtual terminal not available for pod: %s", podKey)
	}

	output := pod.VirtualTerminal.GetOutput(lines)
	return output, nil
}

// SendTerminalText sends text to a local pod's terminal.
// Implements mcp.LocalTerminalProvider interface.
func (r *Runner) SendTerminalText(podKey string, text string) error {
	pod, exists := r.podStore.Get(podKey)
	if !exists || pod == nil {
		return fmt.Errorf("pod not found: %s", podKey)
	}

	if pod.Terminal == nil {
		return fmt.Errorf("terminal not available for pod: %s", podKey)
	}

	err := pod.Terminal.Write([]byte(text))
	if err != nil {
		return fmt.Errorf("failed to write to terminal: %w", err)
	}

	logger.Runner().Debug("Sent text to local terminal", "pod_key", podKey, "text_length", len(text))
	return nil
}

// SendTerminalKey sends special keys to a local pod's terminal.
// Implements mcp.LocalTerminalProvider interface.
func (r *Runner) SendTerminalKey(podKey string, keys []string) error {
	pod, exists := r.podStore.Get(podKey)
	if !exists || pod == nil {
		return fmt.Errorf("pod not found: %s", podKey)
	}

	if pod.Terminal == nil {
		return fmt.Errorf("terminal not available for pod: %s", podKey)
	}

	// Map key names to escape sequences
	keyMap := map[string]string{
		"enter":     "\r",
		"escape":    "\x1b",
		"tab":       "\t",
		"backspace": "\x7f",
		"delete":    "\x1b[3~",
		"ctrl+c":    "\x03",
		"ctrl+d":    "\x04",
		"ctrl+u":    "\x15",
		"ctrl+l":    "\x0c",
		"ctrl+z":    "\x1a",
		"ctrl+a":    "\x01",
		"ctrl+e":    "\x05",
		"ctrl+k":    "\x0b",
		"ctrl+w":    "\x17",
		"up":        "\x1b[A",
		"down":      "\x1b[B",
		"right":     "\x1b[C",
		"left":      "\x1b[D",
		"home":      "\x1b[H",
		"end":       "\x1b[F",
		"pageup":    "\x1b[5~",
		"pagedown":  "\x1b[6~",
		"shift+tab": "\x1b[Z",
	}

	for _, key := range keys {
		seq, ok := keyMap[key]
		if !ok {
			return fmt.Errorf("unknown key: %s", key)
		}
		err := pod.Terminal.Write([]byte(seq))
		if err != nil {
			return fmt.Errorf("failed to send key %s: %w", key, err)
		}
	}

	logger.Runner().Debug("Sent keys to local terminal", "pod_key", podKey, "keys", keys)
	return nil
}
