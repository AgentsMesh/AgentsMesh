package runner

import (
	"sync"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/updater"
)

// UpgradeCoordinator manages the upgrade/draining state machine.
// It encapsulates all upgrade-related state and synchronization,
// extracted from Runner to satisfy SRP.
type UpgradeCoordinator struct {
	draining   bool
	drainingMu sync.RWMutex

	upgrading bool
	upgradeMu sync.Mutex

	updater   *updater.Updater
	restartFn func() (int, error)

	// podCounter is injected to break reverse dependency on Runner.
	podCounter func() int
}

// Compile-time check: UpgradeCoordinator implements UpgradeController.
var _ UpgradeController = (*UpgradeCoordinator)(nil)

// NewUpgradeCoordinator creates a new UpgradeCoordinator.
// podCounter provides the active pod count without depending on Runner.
func NewUpgradeCoordinator(podCounter func() int) *UpgradeCoordinator {
	return &UpgradeCoordinator{podCounter: podCounter}
}

// TryStartUpgrade atomically checks and sets the upgrading flag.
// Returns true if upgrade can proceed, false if another upgrade is in progress.
func (uc *UpgradeCoordinator) TryStartUpgrade() bool {
	uc.upgradeMu.Lock()
	defer uc.upgradeMu.Unlock()
	if uc.upgrading {
		return false
	}
	uc.upgrading = true
	return true
}

// FinishUpgrade clears the upgrading flag.
func (uc *UpgradeCoordinator) FinishUpgrade() {
	uc.upgradeMu.Lock()
	defer uc.upgradeMu.Unlock()
	uc.upgrading = false
}

// GetUpdater returns the updater instance.
func (uc *UpgradeCoordinator) GetUpdater() *updater.Updater {
	return uc.updater
}

// SetUpdater sets the updater instance for remote upgrade support.
func (uc *UpgradeCoordinator) SetUpdater(u *updater.Updater) {
	uc.updater = u
}

// GetRestartFunc returns the restart function.
func (uc *UpgradeCoordinator) GetRestartFunc() func() (int, error) {
	return uc.restartFn
}

// SetRestartFunc sets the restart function for post-upgrade restart.
func (uc *UpgradeCoordinator) SetRestartFunc(fn func() (int, error)) {
	uc.restartFn = fn
}

// GetActivePodCount returns the number of active pods via injected counter.
func (uc *UpgradeCoordinator) GetActivePodCount() int {
	if uc.podCounter == nil {
		return 0
	}
	return uc.podCounter()
}

// SetDraining sets the draining state.
func (uc *UpgradeCoordinator) SetDraining(draining bool) {
	uc.drainingMu.Lock()
	defer uc.drainingMu.Unlock()
	uc.draining = draining
	if draining {
		logger.Runner().Info("Entering draining mode - no new pods will be accepted")
	} else {
		logger.Runner().Info("Exiting draining mode - accepting pods again")
	}
}

// IsDraining returns true if the runner is waiting for pods to finish before update.
func (uc *UpgradeCoordinator) IsDraining() bool {
	uc.drainingMu.RLock()
	defer uc.drainingMu.RUnlock()
	return uc.draining
}
