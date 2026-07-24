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
	var sendErr error
	if !c.pod.WithActiveIO(func(io PodIO) {
		sendErr = io.SendInput(text + "\n")
	}) {
		return fmt.Errorf("pod IO not available for pod %s", c.pod.PodKey)
	}
	return sendErr
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
	c.pod.WithActiveIO(func(io PodIO) { io.SubscribeStateChange(id, cb) })
}

// UnsubscribeStateChange removes a state change subscription.
func (c *PodController) UnsubscribeStateChange(id string) {
	c.pod.WithActiveIO(func(io PodIO) { io.UnsubscribeStateChange(id) })
}

// Compile-time interface check
var _ autopilot.TargetPodController = (*PodController)(nil)
