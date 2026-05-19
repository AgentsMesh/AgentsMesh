package v1

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	agentpodService "github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/service/billing"
)

var (
	ErrQuotaExceeded = billing.ErrQuotaExceeded
	ErrSubscriptionFrozen = billing.ErrSubscriptionFrozen
	ErrSandboxAlreadyResumed = agentpodService.ErrSandboxAlreadyResumed
)

type PodServiceForHandler interface {
	ListPods(ctx context.Context, orgID int64, q agentpod.PodListQuery) ([]*agentpod.Pod, int64, error)
	CreatePod(ctx context.Context, req *agentpodService.CreatePodRequest) (*agentpod.Pod, error)
	GetPod(ctx context.Context, podKey string) (*agentpod.Pod, error)
	GetPodsByTicket(ctx context.Context, ticketID int64) ([]*agentpod.Pod, error)
	UpdateAlias(ctx context.Context, podKey string, alias *string) error
	UpdatePerpetual(ctx context.Context, podKey string, perpetual bool) error
	GetActivePodBySourcePodKey(ctx context.Context, sourcePodKey string) (*agentpod.Pod, error)
}
