package runner

import (
	"fmt"

	"github.com/anthropics/agentsmesh/runner/internal/autopilot"
)

// PodController implements autopilot.TargetPodController interface.
// It delegates to PodIO for mode-agnostic Pod interaction (PTY and ACP).
type PodController struct {
	pod    *Pod
	runner *Runner
}

// NewPodController creates a new PodController.
func NewPodController(pod *Pod, runner *Runner) *PodController {
	return &PodController{
		pod:    pod,
		runner: runner,
	}
}

// SendInput sends text to the pod via PodIO.
func (c *PodController) SendInput(text string) error {
	if c.pod.IO == nil {
		return fmt.Errorf("pod IO not available for pod %s", c.pod.PodKey)
	}
	return c.pod.IO.SendInput(text + "\n")
}

// GetWorkDir returns the pod's working directory.
func (c *PodController) GetWorkDir() string {
	return c.pod.SandboxPath
}

// GetPodKey returns the pod's key.
func (c *PodController) GetPodKey() string {
	return c.pod.PodKey
}

// GetAgentStatus returns the pod's agent status via PodIO.
func (c *PodController) GetAgentStatus() string {
	agentStatus, _, _, _ := c.runner.GetPodStatus(c.pod.PodKey)
	return agentStatus
}

// SubscribeStateChange delegates to PodIO for mode-agnostic state change events.
func (c *PodController) SubscribeStateChange(id string, cb func(newStatus string)) {
	if c.pod.IO != nil {
		c.pod.IO.SubscribeStateChange(id, cb)
	}
}

// UnsubscribeStateChange removes a state change subscription.
func (c *PodController) UnsubscribeStateChange(id string) {
	if c.pod.IO != nil {
		c.pod.IO.UnsubscribeStateChange(id)
	}
}

// Compile-time interface check
var _ autopilot.TargetPodController = (*PodController)(nil)
