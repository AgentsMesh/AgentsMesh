package runner

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
)

// --- Runner Status Tests ---

func TestUpdateRunnerStatus(t *testing.T) {
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
	db.Create(r)

	err := service.UpdateRunnerStatus(ctx, r.ID, runner.RunnerStatusOnline)
	if err != nil {
		t.Fatalf("failed to update runner status: %v", err)
	}

	updated, _ := service.GetRunner(ctx, r.ID)
	if updated.Status != runner.RunnerStatusOnline {
		t.Errorf("expected status online, got %s", updated.Status)
	}
}

func TestSetRunnerStatus(t *testing.T) {
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
	db.Create(r)

	// Test SetRunnerStatus (alias for UpdateRunnerStatus)
	err := service.SetRunnerStatus(ctx, r.ID, runner.RunnerStatusOnline)
	if err != nil {
		t.Fatalf("failed to set runner status: %v", err)
	}

	updated, _ := service.GetRunner(ctx, r.ID)
	if updated.Status != runner.RunnerStatusOnline {
		t.Errorf("expected status online, got %s", updated.Status)
	}

	// Set back to offline
	err = service.SetRunnerStatus(ctx, r.ID, runner.RunnerStatusOffline)
	if err != nil {
		t.Fatalf("failed to set runner status: %v", err)
	}

	updated, _ = service.GetRunner(ctx, r.ID)
	if updated.Status != runner.RunnerStatusOffline {
		t.Errorf("expected status offline, got %s", updated.Status)
	}
}

func TestIsConnected(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)

	// Not connected initially
	if service.IsConnected(1) {
		t.Error("expected runner to not be connected initially")
	}

	// Mark connected
	service.activeRunners.Store(int64(1), &ActiveRunner{})

	if !service.IsConnected(1) {
		t.Error("expected runner to be connected after storing")
	}
}

func TestMarkConnectedDisconnected(t *testing.T) {
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
	db.Create(r)

	// Mark connected
	err := service.MarkConnected(ctx, r.ID)
	if err != nil {
		t.Fatalf("failed to mark connected: %v", err)
	}

	if !service.IsConnected(r.ID) {
		t.Error("expected runner to be connected")
	}

	updated, _ := service.GetRunner(ctx, r.ID)
	if updated.Status != runner.RunnerStatusOnline {
		t.Errorf("expected status online, got %s", updated.Status)
	}

	// Mark disconnected
	err = service.MarkDisconnected(ctx, r.ID)
	if err != nil {
		t.Fatalf("failed to mark disconnected: %v", err)
	}

	if service.IsConnected(r.ID) {
		t.Error("expected runner to be disconnected")
	}

	updated, _ = service.GetRunner(ctx, r.ID)
	if updated.Status != runner.RunnerStatusOffline {
		t.Errorf("expected status offline, got %s", updated.Status)
	}
}

func TestSubscribeStatusChanges(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	unsubscribe, err := service.SubscribeStatusChanges(ctx, func(r *runner.Runner) {})
	if err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}

	if unsubscribe == nil {
		t.Error("expected non-nil unsubscribe function")
	}

	// Calling unsubscribe should not panic
	unsubscribe()
}

func TestUpdateAvailableAgents(t *testing.T) {
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
	db.Create(r)

	t.Run("updates available agents", func(t *testing.T) {
		agents := []string{"claude-code", "aider", "gemini-cli"}
		err := service.UpdateAvailableAgents(ctx, r.ID, agents)
		if err != nil {
			t.Fatalf("failed to update available agents: %v", err)
		}

		// Verify the agents were saved
		updated, _ := service.GetRunner(ctx, r.ID)
		if len(updated.AvailableAgents) != 3 {
			t.Errorf("expected 3 agents, got %d", len(updated.AvailableAgents))
		}
		for i, agent := range agents {
			if updated.AvailableAgents[i] != agent {
				t.Errorf("expected agent %s at index %d, got %s", agent, i, updated.AvailableAgents[i])
			}
		}
	})

	t.Run("updates with empty list", func(t *testing.T) {
		err := service.UpdateAvailableAgents(ctx, r.ID, []string{})
		if err != nil {
			t.Fatalf("failed to update available agents: %v", err)
		}

		updated, _ := service.GetRunner(ctx, r.ID)
		if len(updated.AvailableAgents) != 0 {
			t.Errorf("expected 0 agents, got %d", len(updated.AvailableAgents))
		}
	})

	t.Run("updates with nil list", func(t *testing.T) {
		err := service.UpdateAvailableAgents(ctx, r.ID, nil)
		if err != nil {
			t.Fatalf("failed to update available agents: %v", err)
		}

		updated, _ := service.GetRunner(ctx, r.ID)
		if updated.AvailableAgents != nil && len(updated.AvailableAgents) != 0 {
			t.Errorf("expected nil or empty agents, got %v", updated.AvailableAgents)
		}
	})

	t.Run("runner supports agent check", func(t *testing.T) {
		// Set some agents
		agents := []string{"claude-code", "aider"}
		service.UpdateAvailableAgents(ctx, r.ID, agents)

		updated, _ := service.GetRunner(ctx, r.ID)

		// Test SupportsAgent helper
		if !updated.SupportsAgent("claude-code") {
			t.Error("expected runner to support claude-code")
		}
		if !updated.SupportsAgent("aider") {
			t.Error("expected runner to support aider")
		}
		if updated.SupportsAgent("unknown-agent") {
			t.Error("expected runner to not support unknown-agent")
		}
	})
}

func TestMergeAgentVersions(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	r := &runner.Runner{
		OrganizationID:    1,
		NodeID:            "merge-test-runner",
		Status:            runner.RunnerStatusOnline,
		MaxConcurrentPods: 5,
		IsEnabled:         true,
	}
	db.Create(r)

	// Set initial versions
	initial := []runner.AgentVersion{
		{Slug: "claude-code", Version: "1.0.0", Path: "/usr/bin/claude"},
		{Slug: "codex-cli", Version: "0.1.2025040100", Path: "/usr/bin/codex"},
	}
	err := service.UpdateAgentVersions(ctx, r.ID, initial)
	if err != nil {
		t.Fatalf("failed to set initial versions: %v", err)
	}

	t.Run("merge updates existing version", func(t *testing.T) {
		changes := map[string]runner.AgentVersion{
			"claude-code": {Slug: "claude-code", Version: "1.1.0", Path: "/usr/bin/claude"},
		}
		err := service.MergeAgentVersions(ctx, r.ID, changes)
		if err != nil {
			t.Fatalf("merge failed: %v", err)
		}

		updated, _ := service.GetRunner(ctx, r.ID)
		v := updated.AgentVersions.GetAgentVersion("claude-code")
		if v == nil || v.Version != "1.1.0" {
			t.Errorf("expected claude-code v1.1.0, got %+v", v)
		}
		// codex should still be there
		v2 := updated.AgentVersions.GetAgentVersion("codex-cli")
		if v2 == nil || v2.Version != "0.1.2025040100" {
			t.Errorf("expected codex-cli unchanged, got %+v", v2)
		}
	})

	t.Run("merge adds new agent", func(t *testing.T) {
		changes := map[string]runner.AgentVersion{
			"aider": {Slug: "aider", Version: "0.50.1", Path: "/usr/bin/aider"},
		}
		err := service.MergeAgentVersions(ctx, r.ID, changes)
		if err != nil {
			t.Fatalf("merge failed: %v", err)
		}

		updated, _ := service.GetRunner(ctx, r.ID)
		if len(updated.AgentVersions) != 3 {
			t.Errorf("expected 3 agents, got %d", len(updated.AgentVersions))
		}
		v := updated.AgentVersions.GetAgentVersion("aider")
		if v == nil || v.Version != "0.50.1" {
			t.Errorf("expected aider v0.50.1, got %+v", v)
		}
	})

	t.Run("merge removes agent with empty version", func(t *testing.T) {
		changes := map[string]runner.AgentVersion{
			"aider": {Slug: "aider", Version: "", Path: ""},
		}
		err := service.MergeAgentVersions(ctx, r.ID, changes)
		if err != nil {
			t.Fatalf("merge failed: %v", err)
		}

		updated, _ := service.GetRunner(ctx, r.ID)
		v := updated.AgentVersions.GetAgentVersion("aider")
		if v != nil {
			t.Errorf("expected aider to be removed, got %+v", v)
		}
		if len(updated.AgentVersions) != 2 {
			t.Errorf("expected 2 agents after removal, got %d", len(updated.AgentVersions))
		}
	})
}
