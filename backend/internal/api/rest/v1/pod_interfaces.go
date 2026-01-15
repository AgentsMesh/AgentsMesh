package v1

import (
	"context"

	"github.com/anthropics/agentmesh/backend/internal/domain/agent"
	"github.com/anthropics/agentmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentmesh/backend/internal/domain/gitprovider"
	"github.com/anthropics/agentmesh/backend/internal/domain/ticket"
	agentpodService "github.com/anthropics/agentmesh/backend/internal/service/agentpod"
	"github.com/anthropics/agentmesh/backend/internal/service/billing"
)

// Re-export errors for use in handlers without importing service packages
var (
	// ErrPodTerminated is returned when trying to terminate an already terminated pod
	ErrPodTerminated = agentpodService.ErrPodTerminated
	// ErrQuotaExceeded is returned when quota check fails
	ErrQuotaExceeded = billing.ErrQuotaExceeded
)

// PodServiceForHandler defines the pod service methods needed by PodHandler
// This interface enables dependency inversion and easier testing
type PodServiceForHandler interface {
	ListPods(ctx context.Context, orgID int64, teamID *int64, status string, limit, offset int) ([]*agentpod.Pod, int64, error)
	CreatePod(ctx context.Context, req *agentpodService.CreatePodRequest) (*agentpod.Pod, error)
	GetPod(ctx context.Context, podKey string) (*agentpod.Pod, error)
	TerminatePod(ctx context.Context, podKey string) error
	GetPodsByTicket(ctx context.Context, ticketID int64) ([]*agentpod.Pod, error)
}

// RepositoryServiceForHandler defines the repository service methods needed by PodHandler
type RepositoryServiceForHandler interface {
	GetByID(ctx context.Context, id int64) (*gitprovider.Repository, error)
}

// TicketServiceForHandler defines the ticket service methods needed by PodHandler
type TicketServiceForHandler interface {
	GetTicket(ctx context.Context, ticketID int64) (*ticket.Ticket, error)
}

// AgentServiceForHandler defines the agent service methods needed by PodHandler
type AgentServiceForHandler interface {
	GetUserEffectiveConfig(ctx context.Context, userID, agentTypeID int64, overrides agent.ConfigValues) agent.ConfigValues
	GetEffectiveCredentialsForPod(ctx context.Context, userID, agentTypeID int64, profileID *int64) (agent.EncryptedCredentials, bool, error)
	GetAgentType(ctx context.Context, id int64) (*agent.AgentType, error)
}

// BillingServiceForHandler defines the billing service methods needed by PodHandler
type BillingServiceForHandler interface {
	CheckQuota(ctx context.Context, orgID int64, quotaType string, amount int) error
}
