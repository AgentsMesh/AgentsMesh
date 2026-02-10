package v1

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	agentpodService "github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/service/billing"
)

// Re-export errors for use in handlers without importing service packages
var (
	// ErrPodTerminated is returned when trying to terminate an already terminated pod
	ErrPodTerminated = agentpodService.ErrPodTerminated
	// ErrQuotaExceeded is returned when quota check fails
	ErrQuotaExceeded = billing.ErrQuotaExceeded
	// ErrSubscriptionFrozen is returned when subscription is frozen and operations are blocked
	ErrSubscriptionFrozen = billing.ErrSubscriptionFrozen
	// ErrSandboxAlreadyResumed is returned when trying to resume a sandbox that's already been resumed
	ErrSandboxAlreadyResumed = agentpodService.ErrSandboxAlreadyResumed
)

// PodServiceForHandler defines the pod service methods needed by PodHandler
// This interface enables dependency inversion and easier testing
type PodServiceForHandler interface {
	ListPods(ctx context.Context, orgID int64, status string, limit, offset int) ([]*agentpod.Pod, int64, error)
	CreatePod(ctx context.Context, req *agentpodService.CreatePodRequest) (*agentpod.Pod, error)
	GetPod(ctx context.Context, podKey string) (*agentpod.Pod, error)
	TerminatePod(ctx context.Context, podKey string) error
	GetPodsByTicket(ctx context.Context, ticketID int64) ([]*agentpod.Pod, error)
	// GetActivePodBySourcePodKey returns an active pod that was resumed from the given source pod key
	// Used to prevent multiple pods from resuming the same sandbox simultaneously
	GetActivePodBySourcePodKey(ctx context.Context, sourcePodKey string) (*agentpod.Pod, error)
}

// NOTE: BillingServiceForHandler, RepositoryServiceForHandler, TicketServiceForHandler,
// AgentServiceForHandler, and UserServiceForPod interfaces have been moved to
// PodOrchestrator's narrower interface definitions in service/agentpod/pod_orchestrator.go.
// They are no longer needed at the handler level since Pod creation is delegated to PodOrchestrator.
