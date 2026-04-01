package runner

import (
	"fmt"
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
	if pod.IO != nil {
		shellPid = pod.IO.GetPID()
		return pod.IO.GetAgentStatus(), podStatus, shellPid, true
	}

	return "idle", podStatus, shellPid, true
}

// GetPodSnapshot returns the terminal output for a local pod.
// Implements mcp.LocalPodProvider interface.
func (r *Runner) GetPodSnapshot(podKey string, lines int) (string, error) {
	pod, exists := r.podStore.Get(podKey)
	if !exists || pod == nil {
		return "", fmt.Errorf("pod not found: %s", podKey)
	}

	if pod.IO == nil {
		return "", fmt.Errorf("pod IO not available for pod: %s", podKey)
	}

	return pod.IO.GetSnapshot(lines)
}

// SendPodInput sends text and/or special keys to a local pod.
// Implements mcp.LocalPodProvider interface.
func (r *Runner) SendPodInput(podKey string, text string, keys []string) error {
	pod, exists := r.podStore.Get(podKey)
	if !exists || pod == nil {
		return fmt.Errorf("pod not found: %s", podKey)
	}
	if pod.IO == nil {
		return fmt.Errorf("pod IO not available: %s", podKey)
	}

	if text != "" {
		if err := pod.IO.SendInput(text); err != nil {
			return fmt.Errorf("failed to send text: %w", err)
		}
	}
	if len(keys) > 0 {
		ta, ok := pod.IO.(TerminalAccess)
		if !ok {
			return fmt.Errorf("special keys not supported in %s mode", pod.IO.Mode())
		}
		if err := ta.SendKeys(keys); err != nil {
			return err
		}
	}
	return nil
}
