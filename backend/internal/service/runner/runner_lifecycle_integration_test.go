package runner

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
)

// TestRunnerLifecycle_RegisterAndQuery verifies that a runner created via
// the service can be queried back with all fields intact.
func TestRunnerLifecycle_RegisterAndQuery(t *testing.T) {
	db := testkit.SetupTestDB(t)
	svc := NewService(infra.NewRunnerRepository(db))
	ctx := context.Background()

	userID := testkit.CreateUser(t, db, "u@test.io", "user1")
	orgID := testkit.CreateOrg(t, db, "org-reg", userID)

	// Insert runner via GORM (simulates what gRPC registration does)
	r := &runner.Runner{
		OrganizationID:    orgID,
		NodeID:            "node-query-1",
		Description:       "lifecycle test runner",
		Status:            runner.RunnerStatusOffline,
		MaxConcurrentPods: 3,
		IsEnabled:         true,
		Visibility:        runner.VisibilityOrganization,
	}
	if err := db.Create(r).Error; err != nil {
		t.Fatalf("create runner: %v", err)
	}

	// Query by ID
	got, err := svc.GetRunner(ctx, r.ID)
	if err != nil {
		t.Fatalf("GetRunner: %v", err)
	}
	assertRunnerFields(t, got, r)

	// Query by NodeID
	got2, err := svc.GetByNodeID(ctx, "node-query-1")
	if err != nil {
		t.Fatalf("GetByNodeID: %v", err)
	}
	if got2.ID != r.ID {
		t.Errorf("GetByNodeID ID = %d, want %d", got2.ID, r.ID)
	}
}

// TestRunnerLifecycle_HeartbeatUpdate verifies that Heartbeat updates
// last_heartbeat and current_pods fields.
func TestRunnerLifecycle_HeartbeatUpdate(t *testing.T) {
	db := testkit.SetupTestDB(t)
	svc := NewService(infra.NewRunnerRepository(db))
	ctx := context.Background()

	userID := testkit.CreateUser(t, db, "hb@test.io", "hbuser")
	orgID := testkit.CreateOrg(t, db, "org-hb", userID)

	r := &runner.Runner{
		OrganizationID:    orgID,
		NodeID:            "node-hb-1",
		Status:            runner.RunnerStatusOffline,
		MaxConcurrentPods: 5,
		IsEnabled:         true,
	}
	if err := db.Create(r).Error; err != nil {
		t.Fatalf("create runner: %v", err)
	}

	// Before heartbeat, status is offline
	before, _ := svc.GetRunner(ctx, r.ID)
	if before.Status != runner.RunnerStatusOffline {
		t.Fatalf("initial status = %s, want offline", before.Status)
	}

	// Send heartbeat with 2 pods
	if err := svc.Heartbeat(ctx, r.ID, 2); err != nil {
		t.Fatalf("Heartbeat: %v", err)
	}

	after, err := svc.GetRunner(ctx, r.ID)
	if err != nil {
		t.Fatalf("GetRunner after heartbeat: %v", err)
	}
	if after.Status != runner.RunnerStatusOnline {
		t.Errorf("status = %s, want online", after.Status)
	}
	if after.CurrentPods != 2 {
		t.Errorf("current_pods = %d, want 2", after.CurrentPods)
	}
	if after.LastHeartbeat == nil {
		t.Error("last_heartbeat should be set after heartbeat")
	}
}

// TestRunnerLifecycle_DisableEnable verifies disable/enable toggle
// via UpdateRunner and its effect on field values.
func TestRunnerLifecycle_DisableEnable(t *testing.T) {
	db := testkit.SetupTestDB(t)
	svc := NewService(infra.NewRunnerRepository(db))
	ctx := context.Background()

	userID := testkit.CreateUser(t, db, "de@test.io", "deuser")
	orgID := testkit.CreateOrg(t, db, "org-de", userID)

	r := &runner.Runner{
		OrganizationID:    orgID,
		NodeID:            "node-de-1",
		Status:            runner.RunnerStatusOnline,
		MaxConcurrentPods: 5,
		IsEnabled:         true,
	}
	if err := db.Create(r).Error; err != nil {
		t.Fatalf("create runner: %v", err)
	}

	// Disable
	f := false
	updated, err := svc.UpdateRunner(ctx, r.ID, RunnerUpdateInput{IsEnabled: &f})
	if err != nil {
		t.Fatalf("disable: %v", err)
	}
	if updated.IsEnabled {
		t.Error("expected is_enabled=false after disable")
	}

	// Re-enable
	tr := true
	updated, err = svc.UpdateRunner(ctx, r.ID, RunnerUpdateInput{IsEnabled: &tr})
	if err != nil {
		t.Fatalf("enable: %v", err)
	}
	if !updated.IsEnabled {
		t.Error("expected is_enabled=true after re-enable")
	}
}

// TestRunnerLifecycle_DeleteWithLoopCheck verifies that deleting a runner
// referenced by a loop returns ErrRunnerHasLoopRefs.
func TestRunnerLifecycle_DeleteWithLoopCheck(t *testing.T) {
	db := testkit.SetupTestDB(t)
	svc := NewService(infra.NewRunnerRepository(db))
	ctx := context.Background()

	userID := testkit.CreateUser(t, db, "dl@test.io", "dluser")
	orgID := testkit.CreateOrg(t, db, "org-dl", userID)

	r := &runner.Runner{
		OrganizationID:    orgID,
		NodeID:            "node-dl-1",
		Status:            runner.RunnerStatusOffline,
		MaxConcurrentPods: 5,
		IsEnabled:         true,
	}
	if err := db.Create(r).Error; err != nil {
		t.Fatalf("create runner: %v", err)
	}

	// Create a loop referencing this runner
	if err := db.Exec(
		`INSERT INTO loops (organization_id, name, slug, runner_id, created_by_id, prompt_template) VALUES (?, ?, ?, ?, ?, ?)`,
		orgID, "loop1", "loop-1", r.ID, userID, "test",
	).Error; err != nil {
		t.Fatalf("create loop: %v", err)
	}

	// Delete should fail
	err := svc.DeleteRunner(ctx, r.ID)
	if err != ErrRunnerHasLoopRefs {
		t.Errorf("expected ErrRunnerHasLoopRefs, got %v", err)
	}

	// Confirm runner still exists
	if _, err := svc.GetRunner(ctx, r.ID); err != nil {
		t.Errorf("runner should still exist: %v", err)
	}
}

// TestRunnerLifecycle_ListByOrg verifies that ListRunners returns
// only runners belonging to the queried organization.
func TestRunnerLifecycle_ListByOrg(t *testing.T) {
	db := testkit.SetupTestDB(t)
	svc := NewService(infra.NewRunnerRepository(db))
	ctx := context.Background()

	user1 := testkit.CreateUser(t, db, "o1@test.io", "orguser1")
	user2 := testkit.CreateUser(t, db, "o2@test.io", "orguser2")
	org1 := testkit.CreateOrg(t, db, "org-list-1", user1)
	org2 := testkit.CreateOrg(t, db, "org-list-2", user2)

	// Create 2 runners in org1, 1 in org2
	for _, nodeID := range []string{"node-o1-a", "node-o1-b"} {
		r := &runner.Runner{
			OrganizationID:    org1,
			NodeID:            nodeID,
			Status:            runner.RunnerStatusOffline,
			MaxConcurrentPods: 5,
			IsEnabled:         true,
		}
		if err := db.Create(r).Error; err != nil {
			t.Fatalf("create runner %s: %v", nodeID, err)
		}
	}
	r3 := &runner.Runner{
		OrganizationID:    org2,
		NodeID:            "node-o2-a",
		Status:            runner.RunnerStatusOffline,
		MaxConcurrentPods: 5,
		IsEnabled:         true,
	}
	if err := db.Create(r3).Error; err != nil {
		t.Fatalf("create runner: %v", err)
	}

	list1, err := svc.ListRunners(ctx, org1, user1)
	if err != nil {
		t.Fatalf("ListRunners org1: %v", err)
	}
	if len(list1) != 2 {
		t.Errorf("org1 runners = %d, want 2", len(list1))
	}

	list2, err := svc.ListRunners(ctx, org2, user2)
	if err != nil {
		t.Fatalf("ListRunners org2: %v", err)
	}
	if len(list2) != 1 {
		t.Errorf("org2 runners = %d, want 1", len(list2))
	}
}

// --- helpers ---

func assertRunnerFields(t *testing.T, got, want *runner.Runner) {
	t.Helper()
	if got.OrganizationID != want.OrganizationID {
		t.Errorf("OrgID = %d, want %d", got.OrganizationID, want.OrganizationID)
	}
	if got.NodeID != want.NodeID {
		t.Errorf("NodeID = %s, want %s", got.NodeID, want.NodeID)
	}
	if got.Description != want.Description {
		t.Errorf("Description = %s, want %s", got.Description, want.Description)
	}
	if got.MaxConcurrentPods != want.MaxConcurrentPods {
		t.Errorf("MaxConcurrentPods = %d, want %d", got.MaxConcurrentPods, want.MaxConcurrentPods)
	}
	if got.IsEnabled != want.IsEnabled {
		t.Errorf("IsEnabled = %v, want %v", got.IsEnabled, want.IsEnabled)
	}
}
