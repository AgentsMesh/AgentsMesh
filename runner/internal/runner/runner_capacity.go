package runner

import "github.com/anthropics/agentsmesh/runner/internal/logger"

// --- Delegation to UpgradeCoordinator ---

// IsDraining returns true if the runner is waiting for pods to finish before update.
func (r *Runner) IsDraining() bool {
	if r.upgradeCoord == nil {
		return false
	}
	return r.upgradeCoord.IsDraining()
}

// SetDraining sets the draining state.
func (r *Runner) SetDraining(draining bool) {
	if r.upgradeCoord == nil {
		return
	}
	r.upgradeCoord.SetDraining(draining)
}

// CanAcceptPod returns true if the runner can accept new pods.
func (r *Runner) CanAcceptPod() bool {
	if r.IsDraining() {
		logger.Runner().Debug("Cannot accept pod: runner is draining")
		return false
	}

	currentCount := r.GetActivePodCount()
	if currentCount >= r.cfg.MaxConcurrentPods {
		logger.Runner().Debug("Cannot accept pod: max capacity reached",
			"current", currentCount, "max", r.cfg.MaxConcurrentPods)
		return false
	}

	return true
}

// GetActivePodCount returns the number of currently active pods.
func (r *Runner) GetActivePodCount() int {
	return r.podStore.Count()
}
