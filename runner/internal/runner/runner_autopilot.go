package runner

import (
	"github.com/anthropics/agentsmesh/runner/internal/autopilot"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// Autopilot management methods

// GetAutopilot returns an AutopilotController by key.
func (r *Runner) GetAutopilot(key string) *autopilot.AutopilotController {
	r.autopilotsMu.RLock()
	defer r.autopilotsMu.RUnlock()
	return r.autopilots[key]
}

// AddAutopilot adds an AutopilotController.
func (r *Runner) AddAutopilot(ac *autopilot.AutopilotController) {
	r.autopilotsMu.Lock()
	defer r.autopilotsMu.Unlock()
	r.autopilots[ac.Key()] = ac
	logger.Runner().Debug("Autopilot added", "autopilot_key", ac.Key(), "pod_key", ac.PodKey())
}

// RemoveAutopilot removes an AutopilotController by key.
func (r *Runner) RemoveAutopilot(key string) {
	r.autopilotsMu.Lock()
	defer r.autopilotsMu.Unlock()
	delete(r.autopilots, key)
	logger.Runner().Debug("Autopilot removed", "autopilot_key", key)
}

// GetAutopilotByPodKey returns an AutopilotController by its associated pod key.
func (r *Runner) GetAutopilotByPodKey(podKey string) *autopilot.AutopilotController {
	r.autopilotsMu.RLock()
	defer r.autopilotsMu.RUnlock()
	for _, ac := range r.autopilots {
		if ac.PodKey() == podKey {
			return ac
		}
	}
	return nil
}
