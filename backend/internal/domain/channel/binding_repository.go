package channel

import (
	"context"
	"time"
)

type BindingRepository interface {
	GetByID(ctx context.Context, bindingID int64) (*PodBinding, error)

	GetActive(ctx context.Context, initiatorPod, targetPod string) (*PodBinding, error)

	GetExisting(ctx context.Context, initiatorPod, targetPod string) (*PodBinding, error)

	ListForPod(ctx context.Context, podKey string, status *string) ([]*PodBinding, error)

	ListPending(ctx context.Context, targetPod string) ([]*PodBinding, error)

	Create(ctx context.Context, binding *PodBinding) error

	Save(ctx context.Context, binding *PodBinding) error

	MarkExpired(ctx context.Context, now time.Time) (int64, error)
}
