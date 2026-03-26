package runner

import (
	"fmt"

	"github.com/anthropics/agentsmesh/runner/internal/autopilot"
)

// PodControllerImpl implements autopilot.TargetPodController interface.
// It delegates to PodIO for mode-agnostic Pod interaction (PTY and ACP).
type PodControllerImpl struct {
	pod    *Pod
	runner *Runner
}

// NewPodController creates a new PodController.
func NewPodController(pod *Pod, runner *Runner) *PodControllerImpl {
	return &PodControllerImpl{
		pod:    pod,
		runner: runner,
	}
}

// SendInput sends text to the pod via PodIO.
func (c *PodControllerImpl) SendInput(text string) error {
	if c.pod.IO == nil {
		return fmt.Errorf("pod IO not available for pod %s", c.pod.PodKey)
	}
	return c.pod.IO.SendInput(text + "\n")
}

// GetWorkDir returns the pod's working directory.
func (c *PodControllerImpl) GetWorkDir() string {
	return c.pod.SandboxPath
}

// GetPodKey returns the pod's key.
func (c *PodControllerImpl) GetPodKey() string {
	return c.pod.PodKey
}

// GetAgentStatus returns the pod's agent status via PodIO.
func (c *PodControllerImpl) GetAgentStatus() string {
	agentStatus, _, _, _ := c.runner.GetPodStatus(c.pod.PodKey)
	return agentStatus
}

// SubscribeStateChange delegates to PodIO for mode-agnostic state change events.
func (c *PodControllerImpl) SubscribeStateChange(id string, cb func(newStatus string)) {
	if c.pod.IO != nil {
		c.pod.IO.SubscribeStateChange(id, cb)
	}
}

// UnsubscribeStateChange removes a state change subscription.
func (c *PodControllerImpl) UnsubscribeStateChange(id string) {
	if c.pod.IO != nil {
		c.pod.IO.UnsubscribeStateChange(id)
	}
}

// Compile-time interface check
var _ autopilot.TargetPodController = (*PodControllerImpl)(nil)
