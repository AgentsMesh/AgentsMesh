package runner

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
)

// --- Runner Registration Tests ---

func TestRegisterRunner(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	// Create a registration token first
	plain, err := service.CreateRegistrationToken(ctx, 1, 1, "Test Token", nil, nil)
	if err != nil {
		t.Fatalf("failed to create registration token: %v", err)
	}

	// Register a runner
	r, err := service.RegisterRunner(ctx, plain, "test-runner-1", "Test Runner", 5)
	if err != nil {
		t.Fatalf("failed to register runner: %v", err)
	}

	if r == nil {
		t.Fatal("expected non-nil runner")
	}
	if r.NodeID != "test-runner-1" {
		t.Errorf("expected NodeID 'test-runner-1', got %s", r.NodeID)
	}
	if r.OrganizationID != 1 {
		t.Errorf("expected OrganizationID 1, got %d", r.OrganizationID)
	}
	if r.Status != runner.RunnerStatusOffline {
		t.Errorf("expected Status '%s', got %s", runner.RunnerStatusOffline, r.Status)
	}
	if r.MaxConcurrentPods != 5 {
		t.Errorf("expected MaxConcurrentPods 5, got %d", r.MaxConcurrentPods)
	}

	// Check that token usage count was incremented
	var updatedToken runner.RegistrationToken
	db.First(&updatedToken)
	if updatedToken.UsedCount != 1 {
		t.Errorf("expected UsedCount 1, got %d", updatedToken.UsedCount)
	}
}

func TestRegisterRunnerInvalidToken(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	_, err := service.RegisterRunner(ctx, "invalid-token", "test-runner", "Test", 5)
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestDeleteRunner(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	// Create runner
	plain, _ := service.CreateRegistrationToken(ctx, 1, 1, "Test Token", nil, nil)
	r, _ := service.RegisterRunner(ctx, plain, "test-runner", "Test", 5)

	// Delete runner
	err := service.DeleteRunner(ctx, r.ID)
	if err != nil {
		t.Fatalf("failed to delete runner: %v", err)
	}

	// Verify deletion
	_, err = service.GetRunner(ctx, r.ID)
	if err != ErrRunnerNotFound {
		t.Errorf("expected ErrRunnerNotFound, got %v", err)
	}
}

func TestListRunners(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	// Create multiple runners
	plain, _ := service.CreateRegistrationToken(ctx, 1, 1, "Test Token", nil, nil)
	service.RegisterRunner(ctx, plain, "runner-1", "Runner 1", 5)
	service.RegisterRunner(ctx, plain, "runner-2", "Runner 2", 5)
	service.RegisterRunner(ctx, plain, "runner-3", "Runner 3", 5)

	// List all runners
	runners, err := service.ListRunners(ctx, 1)
	if err != nil {
		t.Fatalf("failed to list runners: %v", err)
	}
	if len(runners) != 3 {
		t.Errorf("expected 3 runners, got %d", len(runners))
	}
}

func TestListAvailableRunners(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	plain, _ := service.CreateRegistrationToken(ctx, 1, 1, "Test Token", nil, nil)

	// Register multiple runners
	r1, _ := service.RegisterRunner(ctx, plain, "runner-1", "Runner 1", 5)
	r2, _ := service.RegisterRunner(ctx, plain, "runner-2", "Runner 2", 5)

	// Set one runner online
	service.Heartbeat(ctx, r1.ID, 0)

	// List available runners
	runners, err := service.ListAvailableRunners(ctx, 1)
	if err != nil {
		t.Fatalf("failed to list available runners: %v", err)
	}

	if len(runners) != 1 {
		t.Errorf("expected 1 available runner, got %d", len(runners))
	}
	_ = r2
}

func TestSelectAvailableRunner(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	plain, _ := service.CreateRegistrationToken(ctx, 1, 1, "Test Token", nil, nil)
	r1, _ := service.RegisterRunner(ctx, plain, "runner-1", "Runner 1", 5)
	r2, _ := service.RegisterRunner(ctx, plain, "runner-2", "Runner 2", 5)

	// Make both online
	service.Heartbeat(ctx, r1.ID, 3)
	service.Heartbeat(ctx, r2.ID, 1)

	// Should select r2 (least pods)
	selected, err := service.SelectAvailableRunner(ctx, 1)
	if err != nil {
		t.Fatalf("failed to select available runner: %v", err)
	}
	if selected.ID != r2.ID {
		t.Errorf("expected runner with least pods (r2), got ID %d", selected.ID)
	}
}

func TestSelectAvailableRunnerNoneAvailable(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	// No runners at all
	_, err := service.SelectAvailableRunner(ctx, 1)
	if err != ErrRunnerOffline {
		t.Errorf("expected ErrRunnerOffline, got %v", err)
	}
}

func TestSelectAvailableRunnerFromCache(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	plain, _ := service.CreateRegistrationToken(ctx, 1, 1, "Test Token", nil, nil)
	r1, _ := service.RegisterRunner(ctx, plain, "runner-1", "Runner 1", 5)
	r2, _ := service.RegisterRunner(ctx, plain, "runner-2", "Runner 2", 5)

	// Update runners status to online
	service.SetRunnerStatus(ctx, r1.ID, "online")
	service.SetRunnerStatus(ctx, r2.ID, "online")

	// Add to cache with different pod counts
	// r1 has 3 pods, r2 has 1 pod
	r1Updated, _ := service.GetRunner(ctx, r1.ID)
	r2Updated, _ := service.GetRunner(ctx, r2.ID)

	service.activeRunners.Store(r1.ID, &ActiveRunner{
		Runner:   r1Updated,
		LastPing: time.Now(),
		PodCount: 3,
	})
	service.activeRunners.Store(r2.ID, &ActiveRunner{
		Runner:   r2Updated,
		LastPing: time.Now(),
		PodCount: 1,
	})

	// Should select r2 from cache (least pods)
	selected, err := service.SelectAvailableRunner(ctx, 1)
	if err != nil {
		t.Fatalf("failed to select available runner: %v", err)
	}
	if selected.ID != r2.ID {
		t.Errorf("expected runner with least pods (r2=%d), got ID %d", r2.ID, selected.ID)
	}
}

func TestSelectAvailableRunnerSkipsInactiveInCache(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	plain, _ := service.CreateRegistrationToken(ctx, 1, 1, "Test Token", nil, nil)
	r1, _ := service.RegisterRunner(ctx, plain, "runner-1", "Runner 1", 5)

	// Update runner status to online
	service.SetRunnerStatus(ctx, r1.ID, "online")
	r1Updated, _ := service.GetRunner(ctx, r1.ID)

	// Add to cache with old last ping (inactive)
	service.activeRunners.Store(r1.ID, &ActiveRunner{
		Runner:   r1Updated,
		LastPing: time.Now().Add(-2 * time.Minute), // 2 minutes ago - inactive
		PodCount: 1,
	})

	// Should fall back to DB query
	selected, err := service.SelectAvailableRunner(ctx, 1)
	if err != nil {
		t.Fatalf("failed to select available runner: %v", err)
	}
	if selected.ID != r1.ID {
		t.Errorf("expected runner r1=%d, got ID %d", r1.ID, selected.ID)
	}
}

func TestSelectAvailableRunnerSkipsDisabledInCache(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	plain, _ := service.CreateRegistrationToken(ctx, 1, 1, "Test Token", nil, nil)
	r1, _ := service.RegisterRunner(ctx, plain, "runner-1", "Runner 1", 5)

	// Update runner to online but disabled
	service.SetRunnerStatus(ctx, r1.ID, "online")
	isEnabled := false
	service.UpdateRunner(ctx, r1.ID, RunnerUpdateInput{IsEnabled: &isEnabled})

	r1Updated, _ := service.GetRunner(ctx, r1.ID)

	// Add to cache
	service.activeRunners.Store(r1.ID, &ActiveRunner{
		Runner:   r1Updated,
		LastPing: time.Now(),
		PodCount: 1,
	})

	// Should return ErrRunnerOffline because disabled runner is filtered
	_, err := service.SelectAvailableRunner(ctx, 1)
	if err != ErrRunnerOffline {
		t.Errorf("expected ErrRunnerOffline for disabled runner, got %v", err)
	}
}

func TestSelectAvailableRunnerSkipsFullInCache(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	plain, _ := service.CreateRegistrationToken(ctx, 1, 1, "Test Token", nil, nil)
	r1, _ := service.RegisterRunner(ctx, plain, "runner-1", "Runner 1", 2) // max 2 pods

	// Update runner status to online and update current_pods to max (so DB query also skips)
	service.SetRunnerStatus(ctx, r1.ID, "online")
	db.Model(&runner.Runner{}).Where("id = ?", r1.ID).Update("current_pods", 2)
	r1Updated, _ := service.GetRunner(ctx, r1.ID)

	// Add to cache with max pods
	service.activeRunners.Store(r1.ID, &ActiveRunner{
		Runner:   r1Updated,
		LastPing: time.Now(),
		PodCount: 2, // already at max
	})

	// Should return ErrRunnerOffline because runner is at capacity in both cache and DB
	_, err := service.SelectAvailableRunner(ctx, 1)
	if err != ErrRunnerOffline {
		t.Errorf("expected ErrRunnerOffline for full runner, got %v", err)
	}
}

func TestRegisterRunnerDuplicate(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	plain, _ := service.CreateRegistrationToken(ctx, 1, 1, "Test Token", nil, nil)

	// Register first runner
	_, err := service.RegisterRunner(ctx, plain, "test-runner", "Test", 5)
	if err != nil {
		t.Fatalf("failed to register first runner: %v", err)
	}

	// Try to register with same node_id
	_, err = service.RegisterRunner(ctx, plain, "test-runner", "Test Duplicate", 5)
	if err != ErrRunnerAlreadyExists {
		t.Errorf("expected ErrRunnerAlreadyExists, got %v", err)
	}
}
