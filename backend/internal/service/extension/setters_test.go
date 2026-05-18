package extension

import (
	"testing"
	"time"
)

// Verifies that the Phase-1 setters honor their non-positive guard contracts:
// negative/zero inputs are ignored, valid inputs override the default.

func TestSetSyncConcurrency_IgnoresNonPositive(t *testing.T) {
	w := NewMarketplaceWorker(nil, nil, nil, time.Hour)
	original := w.syncConcurrency

	w.SetSyncConcurrency(-1)
	if w.syncConcurrency != original {
		t.Errorf("negative input changed value: got %d, want %d", w.syncConcurrency, original)
	}

	w.SetSyncConcurrency(0)
	if w.syncConcurrency != original {
		t.Errorf("zero input changed value: got %d, want %d", w.syncConcurrency, original)
	}

	w.SetSyncConcurrency(8)
	if w.syncConcurrency != 8 {
		t.Errorf("positive input not applied: got %d, want 8", w.syncConcurrency)
	}
}

func TestSetStaleSyncTimeout_IgnoresNonPositive(t *testing.T) {
	imp := NewSkillImporter(nil, nil)
	original := imp.staleSyncTimeout

	imp.SetStaleSyncTimeout(-1 * time.Minute)
	if imp.staleSyncTimeout != original {
		t.Errorf("negative input changed value: got %v, want %v", imp.staleSyncTimeout, original)
	}

	imp.SetStaleSyncTimeout(0)
	if imp.staleSyncTimeout != original {
		t.Errorf("zero input changed value: got %v, want %v", imp.staleSyncTimeout, original)
	}

	imp.SetStaleSyncTimeout(5 * time.Minute)
	if imp.staleSyncTimeout != 5*time.Minute {
		t.Errorf("positive input not applied: got %v, want 5m", imp.staleSyncTimeout)
	}
}

func TestSetInitialSyncTimeout_IgnoresNonPositive(t *testing.T) {
	svc := NewService(nil, nil, nil)
	original := svc.initialSyncTimeout

	svc.SetInitialSyncTimeout(-1 * time.Minute)
	if svc.initialSyncTimeout != original {
		t.Errorf("negative input changed value: got %v, want %v", svc.initialSyncTimeout, original)
	}

	svc.SetInitialSyncTimeout(0)
	if svc.initialSyncTimeout != original {
		t.Errorf("zero input changed value: got %v, want %v", svc.initialSyncTimeout, original)
	}

	svc.SetInitialSyncTimeout(3 * time.Minute)
	if svc.initialSyncTimeout != 3*time.Minute {
		t.Errorf("positive input not applied: got %v, want 3m", svc.initialSyncTimeout)
	}
}
