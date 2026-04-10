package runner

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

// PodStore is the pod persistence interface consumed by PodCoordinator.
// Defined here (runner package) per Interface Segregation Principle.
// Implemented by agentpod.PodService.
type PodStore interface {
	GetByKey(ctx context.Context, podKey string) (*agentpod.Pod, error)
	GetByKeyAndRunner(ctx context.Context, podKey string, runnerID int64) (*agentpod.Pod, error)
	ListActiveByRunner(ctx context.Context, runnerID int64) ([]*agentpod.Pod, error)
	ListInitializingByRunner(ctx context.Context, runnerID int64) ([]*agentpod.Pod, error)
	CountActiveByKeys(ctx context.Context, podKeys []string) (int, error)

	UpdateByKey(ctx context.Context, podKey string, updates map[string]interface{}) (int64, error)
	UpdateByKeyAndStatus(ctx context.Context, podKey, status string, updates map[string]interface{}) error
	UpdateByKeyAndActiveStatus(ctx context.Context, podKey string, updates map[string]interface{}) (int64, error)
	UpdateByKeyAndStatusCounted(ctx context.Context, podKey, status string, updates map[string]interface{}) (int64, error)
	UpdateTerminatedIfActive(ctx context.Context, podKey string, updates map[string]interface{}, fallbackErrorCode string) (int64, error)
	MarkOrphaned(ctx context.Context, pod *agentpod.Pod, finishedAt time.Time) error

	UpdateField(ctx context.Context, podKey, field string, value interface{}) error
	UpdateAgentStatus(ctx context.Context, podKey string, updates map[string]interface{}) error
}
