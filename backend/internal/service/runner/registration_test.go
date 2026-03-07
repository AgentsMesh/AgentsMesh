package runner

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	// Use the GORM db from test helper
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

	// Create runner directly
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
	service := newTestService(db)
	ctx := context.Background()

	// Create multiple runners directly
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

	// List all runners
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

	// Create two runners
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

	// Set one runner online
	service.Heartbeat(ctx, r1.ID, 0)

	// List available runners
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

	// Create two runners
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

	// Make both online with different pod counts
	service.Heartbeat(ctx, r1.ID, 3)
	service.Heartbeat(ctx, r2.ID, 1)

	// Should select r2 (least pods)
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

	// No runners at all
	_, err := service.SelectAvailableRunner(ctx, 1, 1)
	if err != ErrRunnerOffline {
		t.Errorf("expected ErrRunnerOffline, got %v", err)
	}
}

func TestSelectAvailableRunnerFromCache(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	// Create two runners
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

	// Create runner
	r1 := &runner.Runner{
		OrganizationID:    1,
		NodeID:            "runner-1",
		Description:       "Runner 1",
		Status:            runner.RunnerStatusOffline,
		MaxConcurrentPods: 5,
		IsEnabled:         true,
	}
	db.Create(r1)

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

	// Create runner
	r1 := &runner.Runner{
		OrganizationID:    1,
		NodeID:            "runner-1",
		Description:       "Runner 1",
		Status:            runner.RunnerStatusOffline,
		MaxConcurrentPods: 5,
		IsEnabled:         true,
	}
	db.Create(r1)

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
	_, err := service.SelectAvailableRunner(ctx, 1, 1)
	if err != ErrRunnerOffline {
		t.Errorf("expected ErrRunnerOffline for disabled runner, got %v", err)
	}
}

func TestListRunnersVisibility(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	registrantUserID := int64(10)
	otherUserID := int64(20)

	// Create an organization-visible runner
	orgRunner := &runner.Runner{
		OrganizationID:     1,
		NodeID:             "runner-org-vis",
		Status:             runner.RunnerStatusOffline,
		MaxConcurrentPods:  5,
		IsEnabled:          true,
		Visibility:         runner.VisibilityOrganization,
		RegisteredByUserID: &registrantUserID,
	}
	require.NoError(t, db.Create(orgRunner).Error)

	// Create a private runner owned by registrantUserID
	privateRunner := &runner.Runner{
		OrganizationID:     1,
		NodeID:             "runner-private-vis",
		Status:             runner.RunnerStatusOffline,
		MaxConcurrentPods:  5,
		IsEnabled:          true,
		Visibility:         runner.VisibilityPrivate,
		RegisteredByUserID: &registrantUserID,
	}
	require.NoError(t, db.Create(privateRunner).Error)

	t.Run("registrant sees both org and private runners", func(t *testing.T) {
		runners, err := service.ListRunners(ctx, 1, registrantUserID)
		require.NoError(t, err)
		assert.Len(t, runners, 2)
	})

	t.Run("other user sees only org runner", func(t *testing.T) {
		runners, err := service.ListRunners(ctx, 1, otherUserID)
		require.NoError(t, err)
		assert.Len(t, runners, 1)
		assert.Equal(t, runner.VisibilityOrganization, runners[0].Visibility)
	})
}

func TestAuthorizeRunnerSetsRegisteredByUserID(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	// Create org
	org := createTestOrg(t, db, "test-org-regby")

	// Create pending auth
	authKey := generateTestAuthKey()
	pendingAuth := &runner.PendingAuth{
		AuthKey:    authKey,
		MachineKey: "test-machine",
		ExpiresAt:  time.Now().Add(15 * time.Minute),
		Authorized: false,
	}
	require.NoError(t, db.Create(pendingAuth).Error)

	userID := int64(42)
	r, err := service.AuthorizeRunner(ctx, authKey, org.ID, userID, "my-registered-runner")
	require.NoError(t, err)
	assert.NotZero(t, r.ID)

	// Verify RegisteredByUserID was set
	assert.NotNil(t, r.RegisteredByUserID)
	assert.Equal(t, userID, *r.RegisteredByUserID)

	// Verify Visibility defaults to organization
	assert.Equal(t, runner.VisibilityOrganization, r.Visibility)

	// Double-check by reading from DB
	var dbRunner runner.Runner
	require.NoError(t, db.First(&dbRunner, r.ID).Error)
	assert.NotNil(t, dbRunner.RegisteredByUserID)
	assert.Equal(t, userID, *dbRunner.RegisteredByUserID)
	assert.Equal(t, runner.VisibilityOrganization, dbRunner.Visibility)
}

func TestSelectAvailableRunnerSkipsFullInCache(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	// Create runner with max 2 pods
	r1 := &runner.Runner{
		OrganizationID:    1,
		NodeID:            "runner-1",
		Description:       "Runner 1",
		Status:            runner.RunnerStatusOffline,
		MaxConcurrentPods: 2, // max 2 pods
		IsEnabled:         true,
	}
	db.Create(r1)

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
	_, err := service.SelectAvailableRunner(ctx, 1, 1)
	if err != ErrRunnerOffline {
		t.Errorf("expected ErrRunnerOffline for full runner, got %v", err)
	}
}
