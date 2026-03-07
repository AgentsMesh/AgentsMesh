package binding

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
)

func TestRequestScopes(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db, nil)
	ctx := context.Background()

	t.Run("requests additional scopes", func(t *testing.T) {
		binding, _ := service.CreateAutoBinding(ctx, 1, "scope-req-1", "scope-req-2",
			[]string{channel.BindingScopeTerminalRead})

		updated, err := service.RequestScopes(ctx, binding.ID, "scope-req-1",
			[]string{channel.BindingScopeTerminalWrite})
		if err != nil {
			t.Fatalf("failed: %v", err)
		}
		// Since same user is not set, scope should be pending
		if len(updated.PendingScopes) != 1 {
			t.Errorf("expected 1 pending scope, got %d", len(updated.PendingScopes))
		}
	})

	t.Run("wrong pod returns error", func(t *testing.T) {
		binding, _ := service.CreateAutoBinding(ctx, 1, "scope-wrong-1", "scope-wrong-2",
			[]string{channel.BindingScopeTerminalRead})

		_, err := service.RequestScopes(ctx, binding.ID, "scope-wrong-2",
			[]string{channel.BindingScopeTerminalWrite})
		if err != ErrNotAuthorized {
			t.Errorf("expected ErrNotAuthorized, got %v", err)
		}
	})

	t.Run("invalid scope returns error", func(t *testing.T) {
		binding, _ := service.CreateAutoBinding(ctx, 1, "scope-inv-1", "scope-inv-2",
			[]string{channel.BindingScopeTerminalRead})

		_, err := service.RequestScopes(ctx, binding.ID, "scope-inv-1",
			[]string{"invalid:scope"})
		if err != ErrInvalidScope {
			t.Errorf("expected ErrInvalidScope, got %v", err)
		}
	})
}

func TestApproveScopes(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db, nil)
	ctx := context.Background()

	t.Run("approves pending scopes", func(t *testing.T) {
		binding, _ := service.CreateAutoBinding(ctx, 1, "approve-1", "approve-2",
			[]string{channel.BindingScopeTerminalRead})
		binding, _ = service.RequestScopes(ctx, binding.ID, "approve-1",
			[]string{channel.BindingScopeTerminalWrite})

		approved, err := service.ApproveScopes(ctx, binding.ID, "approve-2",
			[]string{channel.BindingScopeTerminalWrite})
		if err != nil {
			t.Fatalf("failed: %v", err)
		}
		if !approved.HasScope(channel.BindingScopeTerminalWrite) {
			t.Error("expected write scope to be granted")
		}
	})

	t.Run("wrong pod returns error", func(t *testing.T) {
		binding, _ := service.CreateAutoBinding(ctx, 1, "approve-wrong-1", "approve-wrong-2",
			[]string{channel.BindingScopeTerminalRead})
		binding, _ = service.RequestScopes(ctx, binding.ID, "approve-wrong-1",
			[]string{channel.BindingScopeTerminalWrite})

		_, err := service.ApproveScopes(ctx, binding.ID, "approve-wrong-1",
			[]string{channel.BindingScopeTerminalWrite})
		if err != ErrNotAuthorized {
			t.Errorf("expected ErrNotAuthorized, got %v", err)
		}
	})

	t.Run("no valid pending scopes returns error", func(t *testing.T) {
		binding, _ := service.CreateAutoBinding(ctx, 1, "approve-none-1", "approve-none-2",
			[]string{channel.BindingScopeTerminalRead})

		_, err := service.ApproveScopes(ctx, binding.ID, "approve-none-2",
			[]string{channel.BindingScopeTerminalWrite})
		if err != ErrNoValidPendingScopes {
			t.Errorf("expected ErrNoValidPendingScopes, got %v", err)
		}
	})
}

func TestHasScope(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db, nil)
	ctx := context.Background()

	t.Run("returns true for granted scope", func(t *testing.T) {
		service.CreateAutoBinding(ctx, 1, "has-1", "has-2",
			[]string{channel.BindingScopeTerminalRead})

		hasScope, err := service.HasScope(ctx, "has-1", "has-2", channel.BindingScopeTerminalRead)
		if err != nil {
			t.Fatalf("failed: %v", err)
		}
		if !hasScope {
			t.Error("expected to have scope")
		}
	})

	t.Run("returns false for missing scope", func(t *testing.T) {
		service.CreateAutoBinding(ctx, 1, "miss-1", "miss-2",
			[]string{channel.BindingScopeTerminalRead})

		hasScope, err := service.HasScope(ctx, "miss-1", "miss-2", channel.BindingScopeTerminalWrite)
		if err != nil {
			t.Fatalf("failed: %v", err)
		}
		if hasScope {
			t.Error("expected to not have write scope")
		}
	})

	t.Run("returns false for no binding", func(t *testing.T) {
		hasScope, err := service.HasScope(ctx, "no-bind-1", "no-bind-2", channel.BindingScopeTerminalRead)
		if err != nil {
			t.Fatalf("failed: %v", err)
		}
		if hasScope {
			t.Error("expected false for no binding")
		}
	})
}

func TestEvaluatePolicy(t *testing.T) {
	db := setupTestDB(t)
	querier := NewMockPodQuerier()
	service := newTestService(db, querier)
	ctx := context.Background()

	t.Run("explicit only policy returns pending", func(t *testing.T) {
		autoApprove, status := service.evaluatePolicy(ctx, "s1", "s2", channel.BindingPolicyExplicitOnly)
		if autoApprove {
			t.Error("expected no auto-approve for explicit only")
		}
		if status != channel.BindingStatusPending {
			t.Errorf("expected pending status, got %s", status)
		}
	})

	t.Run("same user auto approves", func(t *testing.T) {
		querier.AddPod("same-user-1", map[string]interface{}{"user_id": int64(100)})
		querier.AddPod("same-user-2", map[string]interface{}{"user_id": int64(100)})

		autoApprove, status := service.evaluatePolicy(ctx, "same-user-1", "same-user-2", "")
		if !autoApprove {
			t.Error("expected auto-approve for same user")
		}
		if status != channel.BindingStatusActive {
			t.Errorf("expected active status, got %s", status)
		}
	})

	t.Run("same project auto approves with policy", func(t *testing.T) {
		querier.AddPod("proj-1", map[string]interface{}{"user_id": int64(1), "project_id": int64(10)})
		querier.AddPod("proj-2", map[string]interface{}{"user_id": int64(2), "project_id": int64(10)})

		autoApprove, status := service.evaluatePolicy(ctx, "proj-1", "proj-2", channel.BindingPolicySameProjectAuto)
		if !autoApprove {
			t.Error("expected auto-approve for same project")
		}
		if status != channel.BindingStatusActive {
			t.Errorf("expected active status, got %s", status)
		}
	})
}

func TestErrorVariables(t *testing.T) {
	if ErrBindingNotFound.Error() != "binding not found" {
		t.Errorf("unexpected error message: %s", ErrBindingNotFound.Error())
	}
	if ErrBindingExists.Error() != "binding already exists" {
		t.Errorf("unexpected error message: %s", ErrBindingExists.Error())
	}
	if ErrSelfBinding.Error() != "cannot bind a pod to itself" {
		t.Errorf("unexpected error message: %s", ErrSelfBinding.Error())
	}
	if ErrInvalidScope.Error() != "invalid scope" {
		t.Errorf("unexpected error message: %s", ErrInvalidScope.Error())
	}
	if ErrNotAuthorized.Error() != "not authorized for this operation" {
		t.Errorf("unexpected error message: %s", ErrNotAuthorized.Error())
	}
	if ErrBindingNotPending.Error() != "binding is not pending" {
		t.Errorf("unexpected error message: %s", ErrBindingNotPending.Error())
	}
	if ErrBindingNotActive.Error() != "binding is not active" {
		t.Errorf("unexpected error message: %s", ErrBindingNotActive.Error())
	}
	if ErrNoValidPendingScopes.Error() != "no valid pending scopes to approve" {
		t.Errorf("unexpected error message: %s", ErrNoValidPendingScopes.Error())
	}
}
