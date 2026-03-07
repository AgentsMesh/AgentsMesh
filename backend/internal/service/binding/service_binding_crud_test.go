package binding

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
)

func TestValidateScopes(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db, nil)

	t.Run("valid scopes", func(t *testing.T) {
		err := service.validateScopes([]string{channel.BindingScopeTerminalRead})
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("valid multiple scopes", func(t *testing.T) {
		err := service.validateScopes([]string{channel.BindingScopeTerminalRead, channel.BindingScopeTerminalWrite})
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("invalid scope", func(t *testing.T) {
		err := service.validateScopes([]string{"invalid:scope"})
		if err != ErrInvalidScope {
			t.Errorf("expected ErrInvalidScope, got %v", err)
		}
	})
}

func TestRequestBinding(t *testing.T) {
	db := setupTestDB(t)
	querier := NewMockPodQuerier()
	service := newTestService(db, querier)
	ctx := context.Background()

	t.Run("creates pending binding", func(t *testing.T) {
		binding, err := service.RequestBinding(ctx, 1, "pod-1", "pod-2",
			[]string{channel.BindingScopeTerminalRead}, "")
		if err != nil {
			t.Fatalf("failed to request binding: %v", err)
		}
		if binding.Status != channel.BindingStatusPending {
			t.Errorf("expected status pending, got %s", binding.Status)
		}
		if binding.InitiatorPod != "pod-1" {
			t.Errorf("expected initiator pod-1, got %s", binding.InitiatorPod)
		}
	})

	t.Run("self-binding returns error", func(t *testing.T) {
		_, err := service.RequestBinding(ctx, 1, "pod-1", "pod-1",
			[]string{channel.BindingScopeTerminalRead}, "")
		if err != ErrSelfBinding {
			t.Errorf("expected ErrSelfBinding, got %v", err)
		}
	})

	t.Run("invalid scope returns error", func(t *testing.T) {
		_, err := service.RequestBinding(ctx, 1, "pod-a", "pod-b",
			[]string{"invalid:scope"}, "")
		if err != ErrInvalidScope {
			t.Errorf("expected ErrInvalidScope, got %v", err)
		}
	})

	t.Run("same user auto approves", func(t *testing.T) {
		querier.AddPod("user-pod-1", map[string]interface{}{"user_id": int64(1)})
		querier.AddPod("user-pod-2", map[string]interface{}{"user_id": int64(1)})

		binding, err := service.RequestBinding(ctx, 1, "user-pod-1", "user-pod-2",
			[]string{channel.BindingScopeTerminalRead}, "")
		if err != nil {
			t.Fatalf("failed to request binding: %v", err)
		}
		if binding.Status != channel.BindingStatusActive {
			t.Errorf("expected status active for same user, got %s", binding.Status)
		}
	})
}

func TestCreateAutoBinding(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db, nil)
	ctx := context.Background()

	t.Run("creates active binding", func(t *testing.T) {
		binding, err := service.CreateAutoBinding(ctx, 1, "auto-1", "auto-2",
			[]string{channel.BindingScopeTerminalRead, channel.BindingScopeTerminalWrite})
		if err != nil {
			t.Fatalf("failed to create auto binding: %v", err)
		}
		if binding.Status != channel.BindingStatusActive {
			t.Errorf("expected status active, got %s", binding.Status)
		}
		if len(binding.GrantedScopes) != 2 {
			t.Errorf("expected 2 granted scopes, got %d", len(binding.GrantedScopes))
		}
	})

	t.Run("self-binding returns error", func(t *testing.T) {
		_, err := service.CreateAutoBinding(ctx, 1, "auto-same", "auto-same",
			[]string{channel.BindingScopeTerminalRead})
		if err != ErrSelfBinding {
			t.Errorf("expected ErrSelfBinding, got %v", err)
		}
	})

	t.Run("returns existing binding", func(t *testing.T) {
		binding1, _ := service.CreateAutoBinding(ctx, 1, "exist-1", "exist-2",
			[]string{channel.BindingScopeTerminalRead})
		binding2, err := service.CreateAutoBinding(ctx, 1, "exist-1", "exist-2",
			[]string{channel.BindingScopeTerminalWrite})
		if err != nil {
			t.Fatalf("failed: %v", err)
		}
		if binding2.ID != binding1.ID {
			t.Error("expected same binding to be returned")
		}
	})
}

func TestGetBinding(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db, nil)
	ctx := context.Background()

	t.Run("returns binding by ID", func(t *testing.T) {
		created, _ := service.CreateAutoBinding(ctx, 1, "get-1", "get-2",
			[]string{channel.BindingScopeTerminalRead})

		binding, err := service.GetBinding(ctx, created.ID)
		if err != nil {
			t.Fatalf("failed to get binding: %v", err)
		}
		if binding.ID != created.ID {
			t.Errorf("expected ID %d, got %d", created.ID, binding.ID)
		}
	})

	t.Run("returns error for non-existent binding", func(t *testing.T) {
		_, err := service.GetBinding(ctx, 99999)
		if err != ErrBindingNotFound {
			t.Errorf("expected ErrBindingNotFound, got %v", err)
		}
	})
}

func TestGetActiveBinding(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db, nil)
	ctx := context.Background()

	t.Run("returns active binding", func(t *testing.T) {
		service.CreateAutoBinding(ctx, 1, "active-1", "active-2",
			[]string{channel.BindingScopeTerminalRead})

		binding, err := service.GetActiveBinding(ctx, "active-1", "active-2")
		if err != nil {
			t.Fatalf("failed to get active binding: %v", err)
		}
		if !binding.IsActive() {
			t.Error("expected binding to be active")
		}
	})

	t.Run("returns error for pending binding", func(t *testing.T) {
		service.RequestBinding(ctx, 1, "pending-1", "pending-2",
			[]string{channel.BindingScopeTerminalRead}, channel.BindingPolicyExplicitOnly)

		_, err := service.GetActiveBinding(ctx, "pending-1", "pending-2")
		if err != ErrBindingNotFound {
			t.Errorf("expected ErrBindingNotFound for pending binding, got %v", err)
		}
	})
}

func TestGetBindingsForPod(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db, nil)
	ctx := context.Background()

	t.Run("returns all bindings for pod", func(t *testing.T) {
		service.CreateAutoBinding(ctx, 1, "list-main", "list-1",
			[]string{channel.BindingScopeTerminalRead})
		service.CreateAutoBinding(ctx, 1, "list-main", "list-2",
			[]string{channel.BindingScopeTerminalRead})

		bindings, err := service.GetBindingsForPod(ctx, "list-main", nil)
		if err != nil {
			t.Fatalf("failed: %v", err)
		}
		if len(bindings) != 2 {
			t.Errorf("expected 2 bindings, got %d", len(bindings))
		}
	})

	t.Run("filters by status", func(t *testing.T) {
		service.CreateAutoBinding(ctx, 1, "filter-main", "filter-1",
			[]string{channel.BindingScopeTerminalRead})
		service.RequestBinding(ctx, 1, "filter-main", "filter-2",
			[]string{channel.BindingScopeTerminalRead}, channel.BindingPolicyExplicitOnly)

		activeStatus := channel.BindingStatusActive
		bindings, err := service.GetBindingsForPod(ctx, "filter-main", &activeStatus)
		if err != nil {
			t.Fatalf("failed: %v", err)
		}
		if len(bindings) != 1 {
			t.Errorf("expected 1 active binding, got %d", len(bindings))
		}
	})
}

func TestGetBoundPods(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db, nil)
	ctx := context.Background()

	t.Run("returns bound pod keys", func(t *testing.T) {
		service.CreateAutoBinding(ctx, 1, "hub", "spoke-1",
			[]string{channel.BindingScopeTerminalRead})
		service.CreateAutoBinding(ctx, 1, "hub", "spoke-2",
			[]string{channel.BindingScopeTerminalRead})

		pods, err := service.GetBoundPods(ctx, "hub")
		if err != nil {
			t.Fatalf("failed: %v", err)
		}
		if len(pods) != 2 {
			t.Errorf("expected 2 bound pods, got %d", len(pods))
		}
	})

	t.Run("returns empty for unbound pod", func(t *testing.T) {
		pods, err := service.GetBoundPods(ctx, "isolated")
		if err != nil {
			t.Fatalf("failed: %v", err)
		}
		if len(pods) != 0 {
			t.Errorf("expected 0 bound pods, got %d", len(pods))
		}
	})
}

func TestGetPendingRequests(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db, nil)
	ctx := context.Background()

	t.Run("returns pending requests for target", func(t *testing.T) {
		service.RequestBinding(ctx, 1, "req-1", "target",
			[]string{channel.BindingScopeTerminalRead}, channel.BindingPolicyExplicitOnly)
		service.RequestBinding(ctx, 1, "req-2", "target",
			[]string{channel.BindingScopeTerminalRead}, channel.BindingPolicyExplicitOnly)

		pending, err := service.GetPendingRequests(ctx, "target")
		if err != nil {
			t.Fatalf("failed: %v", err)
		}
		if len(pending) != 2 {
			t.Errorf("expected 2 pending requests, got %d", len(pending))
		}
	})
}
