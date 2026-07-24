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
	agentStatus = "idle"
	pod.WithActiveIO(func(io PodIO) {
		shellPid = io.GetPID()
		agentStatus = io.GetAgentStatus()
	})
	return agentStatus, podStatus, shellPid, true
}

// GetPodSnapshot returns the terminal output for a local pod.
// Implements mcp.LocalPodProvider interface.
func (r *Runner) GetPodSnapshot(podKey string, lines int) (string, error) {
	pod, exists := r.podStore.Get(podKey)
	if !exists || pod == nil {
		return "", fmt.Errorf("pod not found: %s", podKey)
	}

	var snapshot string
	var snapshotErr error
	if !pod.WithActiveIO(func(io PodIO) {
		snapshot, snapshotErr = io.GetSnapshot(lines)
	}) {
		return "", fmt.Errorf("pod IO not available for pod: %s", podKey)
	}
	return snapshot, snapshotErr
}

// SendPodInput sends text and/or special keys to a local pod.
// Implements mcp.LocalPodProvider interface.
func (r *Runner) SendPodInput(podKey string, text string, keys []string) error {
	pod, exists := r.podStore.Get(podKey)
	if !exists || pod == nil {
		return fmt.Errorf("pod not found: %s", podKey)
	}
	var sendErr error
	available := pod.WithActiveIO(func(io PodIO) {
		if text != "" {
			if err := io.SendInput(text); err != nil {
				sendErr = fmt.Errorf("failed to send text: %w", err)
				return
			}
		}
		if len(keys) == 0 {
			return
		}
		ta, ok := io.(TerminalAccess)
		if !ok {
			sendErr = fmt.Errorf("special keys not supported in %s mode", io.Mode())
			return
		}
		sendErr = ta.SendKeys(keys)
	})
	if !available {
		return fmt.Errorf("pod IO not available: %s", podKey)
	}
	return sendErr
}
