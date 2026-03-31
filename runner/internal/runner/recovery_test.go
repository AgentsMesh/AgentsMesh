package runner

import (
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/stretchr/testify/assert"
)

// --- recoverDaemonSessions guard tests ---

func TestRecoverDaemonSessions_NilManager(t *testing.T) {
	r, _ := NewTestRunner(t)

	// Ensure podDaemonManager is nil (default from NewTestRunner).
	r.podDaemonManager = nil

	// Must return immediately without panic.
	r.recoverDaemonSessions()

	assert.Equal(t, 0, r.podStore.Count(), "no pods should be added when manager is nil")
}

// --- IsDraining / SetDraining with nil upgradeCoord ---

func TestIsDraining_NilUpgradeCoord(t *testing.T) {
	store := NewInMemoryPodStore()
	r := &Runner{
		cfg:      &config.Config{WorkspaceRoot: t.TempDir()},
		podStore: store,
		// upgradeCoord intentionally nil
	}

	assert.False(t, r.IsDraining(), "IsDraining should return false when upgradeCoord is nil")
}

func TestSetDraining_NilUpgradeCoord(t *testing.T) {
	store := NewInMemoryPodStore()
	r := &Runner{
		cfg:      &config.Config{WorkspaceRoot: t.TempDir()},
		podStore: store,
		// upgradeCoord intentionally nil
	}

	// Must not panic.
	r.SetDraining(true)
	assert.False(t, r.IsDraining(), "IsDraining should still return false after SetDraining on nil coord")
}

// --- CanAcceptPod with nil upgradeCoord ---

func TestCanAcceptPod_NilUpgradeCoord(t *testing.T) {
	store := NewInMemoryPodStore()
	r := &Runner{
		cfg: &config.Config{
			WorkspaceRoot:     t.TempDir(),
			MaxConcurrentPods: 5,
		},
		podStore: store,
		// upgradeCoord intentionally nil — IsDraining() returns false
	}

	assert.True(t, r.CanAcceptPod(), "should accept pod when upgradeCoord is nil and below limit")
}

// --- Delegation methods with nil upgradeCoord ---

func TestTryStartUpgrade_NilUpgradeCoord(t *testing.T) {
	r := &Runner{
		cfg:      &config.Config{WorkspaceRoot: t.TempDir()},
		podStore: NewInMemoryPodStore(),
	}

	assert.False(t, r.TryStartUpgrade(), "TryStartUpgrade should return false with nil upgradeCoord")
}

func TestFinishUpgrade_NilUpgradeCoord(t *testing.T) {
	r := &Runner{
		cfg:      &config.Config{WorkspaceRoot: t.TempDir()},
		podStore: NewInMemoryPodStore(),
	}

	// Must not panic.
	r.FinishUpgrade()
}

func TestGetUpdater_NilUpgradeCoord(t *testing.T) {
	r := &Runner{
		cfg:      &config.Config{WorkspaceRoot: t.TempDir()},
		podStore: NewInMemoryPodStore(),
	}

	assert.Nil(t, r.GetUpdater(), "GetUpdater should return nil with nil upgradeCoord")
}

func TestGetRestartFunc_NilUpgradeCoord(t *testing.T) {
	r := &Runner{
		cfg:      &config.Config{WorkspaceRoot: t.TempDir()},
		podStore: NewInMemoryPodStore(),
	}

	assert.Nil(t, r.GetRestartFunc(), "GetRestartFunc should return nil with nil upgradeCoord")
}
