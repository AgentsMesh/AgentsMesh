package runner

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
)

// --- Runner Status Tests ---

func TestUpdateRunnerStatus(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	plain, _ := service.CreateRegistrationToken(ctx, 1, 1, "Test Token", nil, nil)
	r, _ := service.RegisterRunner(ctx, plain, "test-runner", "Test", 5)

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
	service := NewService(db)
	ctx := context.Background()

	plain, _ := service.CreateRegistrationToken(ctx, 1, 1, "Test Token", nil, nil)
	r, _ := service.RegisterRunner(ctx, plain, "test-runner", "Test", 5)

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
	service := NewService(db)

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
	service := NewService(db)
	ctx := context.Background()

	plain, _ := service.CreateRegistrationToken(ctx, 1, 1, "Test Token", nil, nil)
	r, _ := service.RegisterRunner(ctx, plain, "test-runner", "Test", 5)

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
	service := NewService(db)
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
	service := NewService(db)
	ctx := context.Background()

	// Create a runner
	plain, _ := service.CreateRegistrationToken(ctx, 1, 1, "Test Token", nil, nil)
	r, _ := service.RegisterRunner(ctx, plain, "test-runner", "Test", 5)

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
