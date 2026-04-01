package runner

import (
	"github.com/anthropics/agentsmesh/runner/internal/autopilot"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/updater"
)

// --- UpgradeController delegation (delegates to upgradeController) ---

func (r *Runner) TryStartUpgrade() bool {
	if r.upgradeCoord == nil {
		return false
	}
	ok := r.upgradeCoord.TryStartUpgrade()
	if ok {
		logger.Runner().Info("Upgrade started")
	} else {
		logger.Runner().Warn("Upgrade request rejected (already in progress or pods active)")
	}
	return ok
}

func (r *Runner) FinishUpgrade() {
	if r.upgradeCoord == nil {
		return
	}
	r.upgradeCoord.FinishUpgrade()
	logger.Runner().Info("Upgrade finished")
}

func (r *Runner) GetUpdater() *updater.Updater {
	if r.upgradeCoord == nil {
		return nil
	}
	return r.upgradeCoord.GetUpdater()
}

// SetUpdater sets the updater instance for remote upgrade support.
func (r *Runner) SetUpdater(u *updater.Updater) {
	if r.upgradeCoord == nil {
		return
	}
	r.upgradeCoord.SetUpdater(u)
}

func (r *Runner) GetRestartFunc() func() (int, error) {
	if r.upgradeCoord == nil {
		return nil
	}
	return r.upgradeCoord.GetRestartFunc()
}

// SetRestartFunc sets the restart function for post-upgrade restart.
func (r *Runner) SetRestartFunc(fn func() (int, error)) {
	if r.upgradeCoord == nil {
		return
	}
	r.upgradeCoord.SetRestartFunc(fn)
}

// --- AutopilotRegistry delegation (delegates to AutopilotStore) ---

func (r *Runner) GetAutopilot(key string) *autopilot.AutopilotController {
	if r.autopilotStore == nil {
		return nil
	}
	return r.autopilotStore.GetAutopilot(key)
}

func (r *Runner) AddAutopilot(ac *autopilot.AutopilotController) {
	if r.autopilotStore == nil {
		return
	}
	r.autopilotStore.AddAutopilot(ac)
}

func (r *Runner) RemoveAutopilot(key string) {
	if r.autopilotStore == nil {
		return
	}
	r.autopilotStore.RemoveAutopilot(key)
}

func (r *Runner) GetAutopilotByPodKey(podKey string) *autopilot.AutopilotController {
	if r.autopilotStore == nil {
		return nil
	}
	return r.autopilotStore.GetAutopilotByPodKey(podKey)
}
