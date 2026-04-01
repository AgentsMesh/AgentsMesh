package runner

import (
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/updater"
)

func TestUpgradeControllerTryStartUpgrade(t *testing.T) {
	uc := newUpgradeController(func() int { return 0 })

	// First call should succeed
	if !uc.TryStartUpgrade() {
		t.Error("first TryStartUpgrade should return true")
	}

	// Second call should fail (already upgrading)
	if uc.TryStartUpgrade() {
		t.Error("second TryStartUpgrade should return false")
	}

	// After finish, should succeed again
	uc.FinishUpgrade()
	if !uc.TryStartUpgrade() {
		t.Error("TryStartUpgrade after FinishUpgrade should return true")
	}
}

func TestUpgradeControllerDraining(t *testing.T) {
	uc := newUpgradeController(func() int { return 0 })

	if uc.IsDraining() {
		t.Error("should not be draining initially")
	}

	uc.SetDraining(true)
	if !uc.IsDraining() {
		t.Error("should be draining after SetDraining(true)")
	}

	uc.SetDraining(false)
	if uc.IsDraining() {
		t.Error("should not be draining after SetDraining(false)")
	}
}

func TestUpgradeControllerGetActivePodCount(t *testing.T) {
	count := 0
	uc := newUpgradeController(func() int { return count })

	if uc.GetActivePodCount() != 0 {
		t.Errorf("GetActivePodCount = %d, want 0", uc.GetActivePodCount())
	}

	count = 5
	if uc.GetActivePodCount() != 5 {
		t.Errorf("GetActivePodCount = %d, want 5", uc.GetActivePodCount())
	}
}

func TestUpgradeControllerGetActivePodCountNilCounter(t *testing.T) {
	uc := newUpgradeController(nil)
	if uc.GetActivePodCount() != 0 {
		t.Errorf("GetActivePodCount with nil counter = %d, want 0", uc.GetActivePodCount())
	}
}

func TestUpgradeControllerUpdater(t *testing.T) {
	uc := newUpgradeController(func() int { return 0 })

	if uc.GetUpdater() != nil {
		t.Error("updater should be nil initially")
	}

	u := updater.New("1.0.0")
	uc.SetUpdater(u)
	if uc.GetUpdater() != u {
		t.Error("GetUpdater should return the set updater")
	}
}

func TestUpgradeControllerRestartFunc(t *testing.T) {
	uc := newUpgradeController(func() int { return 0 })

	if uc.GetRestartFunc() != nil {
		t.Error("restart func should be nil initially")
	}

	called := false
	fn := func() (int, error) { called = true; return 0, nil }
	uc.SetRestartFunc(fn)

	got := uc.GetRestartFunc()
	if got == nil {
		t.Fatal("GetRestartFunc should return the set function")
	}
	got()
	if !called {
		t.Error("restart func should have been called")
	}
}

func TestUpgradeControllerInterfaceCompliance(t *testing.T) {
	var _ UpgradeController = (*upgradeController)(nil)
}
