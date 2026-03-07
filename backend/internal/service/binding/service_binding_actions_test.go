package binding

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
)

func TestAcceptBinding(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db, nil)
	ctx := context.Background()

	t.Run("accepts pending binding", func(t *testing.T) {
		pending, _ := service.RequestBinding(ctx, 1, "accept-1", "accept-2",
			[]string{channel.BindingScopeTerminalRead}, channel.BindingPolicyExplicitOnly)

		accepted, err := service.AcceptBinding(ctx, pending.ID, "accept-2")
		if err != nil {
			t.Fatalf("failed to accept binding: %v", err)
		}
		if accepted.Status != channel.BindingStatusActive {
			t.Errorf("expected status active, got %s", accepted.Status)
		}
		if len(accepted.GrantedScopes) != 1 {
			t.Errorf("expected 1 granted scope, got %d", len(accepted.GrantedScopes))
		}
	})

	t.Run("wrong pod returns error", func(t *testing.T) {
		pending, _ := service.RequestBinding(ctx, 1, "wrong-1", "wrong-2",
			[]string{channel.BindingScopeTerminalRead}, channel.BindingPolicyExplicitOnly)

		_, err := service.AcceptBinding(ctx, pending.ID, "wrong-1") // Should be wrong-2
		if err != ErrNotAuthorized {
			t.Errorf("expected ErrNotAuthorized, got %v", err)
		}
	})

	t.Run("accepting non-pending returns error", func(t *testing.T) {
		active, _ := service.CreateAutoBinding(ctx, 1, "not-pending-1", "not-pending-2",
			[]string{channel.BindingScopeTerminalRead})

		_, err := service.AcceptBinding(ctx, active.ID, "not-pending-2")
		if err != ErrBindingNotPending {
			t.Errorf("expected ErrBindingNotPending, got %v", err)
		}
	})
}

func TestRejectBinding(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db, nil)
	ctx := context.Background()

	t.Run("rejects pending binding", func(t *testing.T) {
		pending, _ := service.RequestBinding(ctx, 1, "reject-1", "reject-2",
			[]string{channel.BindingScopeTerminalRead}, channel.BindingPolicyExplicitOnly)

		rejected, err := service.RejectBinding(ctx, pending.ID, "reject-2", "not interested")
		if err != nil {
			t.Fatalf("failed to reject binding: %v", err)
		}
		if rejected.Status != channel.BindingStatusRejected {
			t.Errorf("expected status rejected, got %s", rejected.Status)
		}
		if rejected.RejectionReason == nil || *rejected.RejectionReason != "not interested" {
			t.Error("expected rejection reason to be set")
		}
	})

	t.Run("wrong pod returns error", func(t *testing.T) {
		pending, _ := service.RequestBinding(ctx, 1, "reject-wrong-1", "reject-wrong-2",
			[]string{channel.BindingScopeTerminalRead}, channel.BindingPolicyExplicitOnly)

		_, err := service.RejectBinding(ctx, pending.ID, "reject-wrong-1", "")
		if err != ErrNotAuthorized {
			t.Errorf("expected ErrNotAuthorized, got %v", err)
		}
	})
}

func TestUnbind(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db, nil)
	ctx := context.Background()

	t.Run("unbinds active binding", func(t *testing.T) {
		service.CreateAutoBinding(ctx, 1, "unbind-1", "unbind-2",
			[]string{channel.BindingScopeTerminalRead})

		success, err := service.Unbind(ctx, "unbind-1", "unbind-2")
		if err != nil {
			t.Fatalf("failed to unbind: %v", err)
		}
		if !success {
			t.Error("expected unbind to succeed")
		}

		// Verify it's no longer active
		_, err = service.GetActiveBinding(ctx, "unbind-1", "unbind-2")
		if err != ErrBindingNotFound {
			t.Error("expected binding to be inactive")
		}
	})

	t.Run("unbinds in reverse direction", func(t *testing.T) {
		service.CreateAutoBinding(ctx, 1, "unbind-rev-1", "unbind-rev-2",
			[]string{channel.BindingScopeTerminalRead})

		success, err := service.Unbind(ctx, "unbind-rev-2", "unbind-rev-1")
		if err != nil {
			t.Fatalf("failed to unbind: %v", err)
		}
		if !success {
			t.Error("expected unbind to succeed")
		}
	})

	t.Run("returns false for non-existent binding", func(t *testing.T) {
		success, err := service.Unbind(ctx, "nonexistent-1", "nonexistent-2")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if success {
			t.Error("expected unbind to return false for non-existent binding")
		}
	})
}

func TestIsBound(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db, nil)
	ctx := context.Background()

	t.Run("returns true for bound pods", func(t *testing.T) {
		service.CreateAutoBinding(ctx, 1, "bound-1", "bound-2",
			[]string{channel.BindingScopeTerminalRead})

		bound, err := service.IsBound(ctx, "bound-1", "bound-2")
		if err != nil {
			t.Fatalf("failed: %v", err)
		}
		if !bound {
			t.Error("expected pods to be bound")
		}
	})

	t.Run("returns true in reverse direction", func(t *testing.T) {
		service.CreateAutoBinding(ctx, 1, "bound-rev-1", "bound-rev-2",
			[]string{channel.BindingScopeTerminalRead})

		bound, err := service.IsBound(ctx, "bound-rev-2", "bound-rev-1")
		if err != nil {
			t.Fatalf("failed: %v", err)
		}
		if !bound {
			t.Error("expected pods to be bound in reverse")
		}
	})

	t.Run("returns false for unbound pods", func(t *testing.T) {
		bound, err := service.IsBound(ctx, "unbound-1", "unbound-2")
		if err != nil {
			t.Fatalf("failed: %v", err)
		}
		if bound {
			t.Error("expected pods to not be bound")
		}
	})
}

func TestCleanupExpiredBindings(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db, nil)
	ctx := context.Background()

	t.Run("cleans up expired bindings", func(t *testing.T) {
		// Create a pending binding and manually set expires_at to past
		service.RequestBinding(ctx, 1, "expired-1", "expired-2",
			[]string{channel.BindingScopeTerminalRead}, channel.BindingPolicyExplicitOnly)

		db.Exec("UPDATE pod_bindings SET expires_at = datetime('now', '-1 day') WHERE initiator_pod = ?", "expired-1")

		count, err := service.CleanupExpiredBindings(ctx)
		if err != nil {
			t.Fatalf("failed: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 expired binding cleaned, got %d", count)
		}
	})
}
