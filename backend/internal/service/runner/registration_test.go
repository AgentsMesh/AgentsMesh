package runner

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
)

// --- Helper Functions ---

// createTestRunner creates a runner directly in the database for testing
func createTestRunner(t *testing.T, db interface{ Exec(string, ...any) interface{ Error() error } }, orgID int64, nodeID, description string, maxPods int) *runner.Runner {
	t.Helper()

	r := &runner.Runner{
		OrganizationID:    orgID,
		NodeID:            nodeID,
		Description:       description,
		Status:            runner.RunnerStatusOffline,
		MaxConcurrentPods: maxPods,
		IsEnabled:         true,
	}

	if gormDB, ok := db.(interface {
		Create(value any) interface{ Error() error }
	}); ok {
		if err := gormDB.Create(r).Error(); err != nil {
			t.Fatalf("failed to create test runner: %v", err)
		}
	}

	return r
}

// --- Runner Tests ---

func TestDeleteRunner(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	r := &runner.Runner{
		OrganizationID:    1,
		NodeID:            "test-runner",
		Description:       "Test",
		Status:            runner.RunnerStatusOffline,
		MaxConcurrentPods: 5,
		IsEnabled:         true,
	}
	if err := db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	err := service.DeleteRunner(ctx, r.ID)
	if err != nil {
		t.Fatalf("failed to delete runner: %v", err)
	}

	_, err = service.GetRunner(ctx, r.ID)
	if err != ErrRunnerNotFound {
		t.Errorf("expected ErrRunnerNotFound, got %v", err)
	}
}

func TestListRunners(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	for i, nodeID := range []string{"runner-1", "runner-2", "runner-3"} {
		r := &runner.Runner{
			OrganizationID:    1,
			NodeID:            nodeID,
			Description:       "Runner " + string(rune('1'+i)),
			Status:            runner.RunnerStatusOffline,
			MaxConcurrentPods: 5,
			IsEnabled:         true,
		}
		if err := db.Create(r).Error; err != nil {
			t.Fatalf("failed to create runner: %v", err)
		}
	}

	runners, err := service.ListRunners(ctx, 1, 1)
	if err != nil {
		t.Fatalf("failed to list runners: %v", err)
	}
	if len(runners) != 3 {
		t.Errorf("expected 3 runners, got %d", len(runners))
	}
}

func TestListAvailableRunners(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	r1 := &runner.Runner{
		OrganizationID:    1,
		NodeID:            "runner-1",
		Description:       "Runner 1",
		Status:            runner.RunnerStatusOffline,
		MaxConcurrentPods: 5,
		IsEnabled:         true,
	}
	r2 := &runner.Runner{
		OrganizationID:    1,
		NodeID:            "runner-2",
		Description:       "Runner 2",
		Status:            runner.RunnerStatusOffline,
		MaxConcurrentPods: 5,
		IsEnabled:         true,
	}
	db.Create(r1)
	db.Create(r2)

	service.Heartbeat(ctx, r1.ID, 0)

	runners, err := service.ListAvailableRunners(ctx, 1, 1)
	if err != nil {
		t.Fatalf("failed to list available runners: %v", err)
	}
	if len(runners) != 1 {
		t.Errorf("expected 1 available runner, got %d", len(runners))
	}
}

func TestSelectAvailableRunner(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	r1 := &runner.Runner{
		OrganizationID:    1,
		NodeID:            "runner-1",
		Description:       "Runner 1",
		Status:            runner.RunnerStatusOffline,
		MaxConcurrentPods: 5,
		IsEnabled:         true,
	}
	r2 := &runner.Runner{
		OrganizationID:    1,
		NodeID:            "runner-2",
		Description:       "Runner 2",
		Status:            runner.RunnerStatusOffline,
		MaxConcurrentPods: 5,
		IsEnabled:         true,
	}
	db.Create(r1)
	db.Create(r2)

	service.Heartbeat(ctx, r1.ID, 3)
	service.Heartbeat(ctx, r2.ID, 1)

	selected, err := service.SelectAvailableRunner(ctx, 1, 1)
	if err != nil {
		t.Fatalf("failed to select available runner: %v", err)
	}
	if selected.ID != r2.ID {
		t.Errorf("expected runner with least pods (r2), got ID %d", selected.ID)
	}
}

func TestSelectAvailableRunnerNoneAvailable(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	_, err := service.SelectAvailableRunner(ctx, 1, 1)
	if err != ErrRunnerOffline {
		t.Errorf("expected ErrRunnerOffline, got %v", err)
	}
}

func TestSelectAvailableRunnerFromCache(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	r1 := &runner.Runner{
		OrganizationID:    1,
		NodeID:            "runner-1",
		Description:       "Runner 1",
		Status:            runner.RunnerStatusOffline,
		MaxConcurrentPods: 5,
		IsEnabled:         true,
	}
	r2 := &runner.Runner{
		OrganizationID:    1,
		NodeID:            "runner-2",
		Description:       "Runner 2",
		Status:            runner.RunnerStatusOffline,
		MaxConcurrentPods: 5,
		IsEnabled:         true,
	}
	db.Create(r1)
	db.Create(r2)

	service.SetRunnerStatus(ctx, r1.ID, "online")
	service.SetRunnerStatus(ctx, r2.ID, "online")

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

	selected, err := service.SelectAvailableRunner(ctx, 1, 1)
	if err != nil {
		t.Fatalf("failed to select available runner: %v", err)
	}
	if selected.ID != r2.ID {
		t.Errorf("expected runner with least pods (r2=%d), got ID %d", r2.ID, selected.ID)
	}
}

func TestSelectAvailableRunnerSkipsInactiveInCache(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	r1 := &runner.Runner{
		OrganizationID:    1,
		NodeID:            "runner-1",
		Description:       "Runner 1",
		Status:            runner.RunnerStatusOffline,
		MaxConcurrentPods: 5,
		IsEnabled:         true,
	}
	db.Create(r1)

	service.SetRunnerStatus(ctx, r1.ID, "online")
	r1Updated, _ := service.GetRunner(ctx, r1.ID)

	service.activeRunners.Store(r1.ID, &ActiveRunner{
		Runner:   r1Updated,
		LastPing: time.Now().Add(-2 * time.Minute),
		PodCount: 1,
	})

	selected, err := service.SelectAvailableRunner(ctx, 1, 1)
	if err != nil {
		t.Fatalf("failed to select available runner: %v", err)
	}
	if selected.ID != r1.ID {
		t.Errorf("expected runner r1=%d, got ID %d", r1.ID, selected.ID)
	}
}

func TestSelectAvailableRunnerSkipsDisabledInCache(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	r1 := &runner.Runner{
		OrganizationID:    1,
		NodeID:            "runner-1",
		Description:       "Runner 1",
		Status:            runner.RunnerStatusOffline,
		MaxConcurrentPods: 5,
		IsEnabled:         true,
	}
	db.Create(r1)

	service.SetRunnerStatus(ctx, r1.ID, "online")
	isEnabled := false
	service.UpdateRunner(ctx, r1.ID, RunnerUpdateInput{IsEnabled: &isEnabled})

	r1Updated, _ := service.GetRunner(ctx, r1.ID)

	service.activeRunners.Store(r1.ID, &ActiveRunner{
		Runner:   r1Updated,
		LastPing: time.Now(),
		PodCount: 1,
	})

	_, err := service.SelectAvailableRunner(ctx, 1, 1)
	if err != ErrRunnerOffline {
		t.Errorf("expected ErrRunnerOffline for disabled runner, got %v", err)
	}
}
