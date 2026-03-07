package channel

import (
	"context"
	"time"
)

// BindingRepository defines the data access interface for pod binding operations.
type BindingRepository interface {
	// GetByID returns a binding by its primary key.
	GetByID(ctx context.Context, bindingID int64) (*PodBinding, error)

	// GetActive returns an active binding between initiator and target pods.
	// Returns (nil, nil) if not found.
	GetActive(ctx context.Context, initiatorPod, targetPod string) (*PodBinding, error)

	// GetExisting returns any active or pending binding between initiator and target pods.
	// Returns (nil, nil) if not found.
	GetExisting(ctx context.Context, initiatorPod, targetPod string) (*PodBinding, error)

	// ListForPod returns all bindings for a pod (as initiator or target).
	// If status is non-nil, only bindings with that status are returned.
	ListForPod(ctx context.Context, podKey string, status *string) ([]*PodBinding, error)

	// ListPending returns pending binding requests for a target pod.
	ListPending(ctx context.Context, targetPod string) ([]*PodBinding, error)

	// Create inserts a new binding record.
	Create(ctx context.Context, binding *PodBinding) error

	// Save persists all fields of the binding (update).
	Save(ctx context.Context, binding *PodBinding) error

	// MarkExpired marks pending bindings past their expiry as expired.
	// Returns the number of affected rows.
	MarkExpired(ctx context.Context, now time.Time) (int64, error)
}
