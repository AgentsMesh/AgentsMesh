package agentpod

import (
	"context"
	"errors"

	agentDomain "github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	podDomain "github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/gitprovider"
	runnerDomain "github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"github.com/anthropics/agentsmesh/backend/internal/service/agent"
	userService "github.com/anthropics/agentsmesh/backend/internal/service/user"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

var (
	ErrMissingRunnerID            = errors.New("runner_id is required")
	ErrMissingAgentSlug           = errors.New("agent_slug is required")
	ErrSourcePodNotFound          = errors.New("source pod not found")
	ErrSourcePodAccessDenied      = errors.New("source pod belongs to different organization")
	ErrSourcePodNotTerminated     = errors.New("source pod is not terminated")
	ErrSourcePodAlreadyResumed    = errors.New("source pod already resumed")
	ErrResumeAgentMismatch        = errors.New("resume requires same agent as source pod")
	ErrResumeRunnerMismatch       = errors.New("resume requires same runner")
	ErrConfigBuildFailed          = errors.New("failed to build pod configuration")
	ErrInvalidAgentfileLayer        = errors.New("invalid agentfile layer")
	ErrRunnerDispatchFailed       = errors.New("failed to dispatch pod to runner")
	ErrUnsupportedInteractionMode = errors.New("agent type does not support the requested interaction mode")
)

const errCodeRunnerUnreachable = "RUNNER_UNREACHABLE"

// OrchestrateCreatePodRequest is the unified Pod creation request (protocol-agnostic).
// Pod configuration flows exclusively through AgentfileLayer (SSOT).
type OrchestrateCreatePodRequest struct {
	OrganizationID int64
	UserID         int64

	RunnerID            int64
	AgentSlug           string
	RepositoryID        *int64  // Platform-level ID (from AgentFile REPO slug resolution or resume inheritance)
	TicketID            *int64
	TicketSlug          *string
	Alias               *string
	CredentialProfileID *int64
	AgentfileLayer      *string // SSOT for all CONFIG, MODE, PROMPT, REPO, BRANCH, CREDENTIAL
	Cols                int32
	Rows                int32

	SourcePodKey       string
	ResumeAgentSession *bool

	Perpetual bool

	BranchName *string
}

type OrchestrateCreatePodResult struct {
	Pod     *podDomain.Pod
	Warning string
}

type PodCoordinatorForOrchestrator interface {
	CreatePod(ctx context.Context, runnerID int64, cmd *runnerv1.CreatePodCommand) error
}

type BillingServiceForOrchestrator interface {
	CheckQuota(ctx context.Context, orgID int64, quotaType string, amount int) error
}

type UserServiceForOrchestrator interface {
	GetDefaultGitCredential(ctx context.Context, userID int64) (*user.GitCredential, error)
	GetDecryptedCredentialToken(ctx context.Context, userID, credentialID int64) (*userService.DecryptedCredential, error)
}

type RepositoryServiceForOrchestrator interface {
	GetByID(ctx context.Context, id int64) (*gitprovider.Repository, error)
	FindByOrgSlug(ctx context.Context, orgID int64, slug string) (*gitprovider.Repository, error)
}

type TicketServiceForOrchestrator interface {
	GetTicket(ctx context.Context, ticketID int64) (*ticket.Ticket, error)
	GetTicketBySlug(ctx context.Context, organizationID int64, slug string) (*ticket.Ticket, error)
}

type RunnerSelectorForOrchestrator interface {
	SelectRunnerWithAffinity(ctx context.Context, orgID int64, userID int64, agentSlug string, hints *runnerDomain.AffinityHints, repoHistory map[int64]int) (*runnerDomain.Runner, error)
}

type RunnerQueryForOrchestrator interface {
	GetRunner(ctx context.Context, runnerID int64) (*runnerDomain.Runner, error)
}

type AgentResolverForOrchestrator interface {
	GetAgent(ctx context.Context, slug string) (*agentDomain.Agent, error)
}

type UserConfigQueryForOrchestrator interface {
	GetUserConfigPrefs(ctx context.Context, userID int64, agentSlug string) map[string]interface{}
}

type PodOrchestratorDeps struct {
	PodService      *PodService
	ConfigBuilder   *agent.ConfigBuilder
	PodCoordinator  PodCoordinatorForOrchestrator
	BillingService  BillingServiceForOrchestrator
	UserService     UserServiceForOrchestrator
	RepoService     RepositoryServiceForOrchestrator
	TicketService   TicketServiceForOrchestrator
	RunnerSelector  RunnerSelectorForOrchestrator
	AgentResolver   AgentResolverForOrchestrator
	RunnerQuery     RunnerQueryForOrchestrator
	UserConfigQuery UserConfigQueryForOrchestrator
	PodRepo         podDomain.PodRepository
}

type PodOrchestrator struct {
	podService      *PodService
	configBuilder   *agent.ConfigBuilder
	podCoordinator  PodCoordinatorForOrchestrator
	billingService  BillingServiceForOrchestrator
	userService     UserServiceForOrchestrator
	repoService     RepositoryServiceForOrchestrator
	ticketService   TicketServiceForOrchestrator
	runnerSelector  RunnerSelectorForOrchestrator
	agentResolver   AgentResolverForOrchestrator
	runnerQuery     RunnerQueryForOrchestrator
	userConfigQuery UserConfigQueryForOrchestrator
	podRepo         podDomain.PodRepository
}

type agentfileResolved struct {
	InteractionMode      string
	BranchName           string
	RepositoryID         *int64
	Prompt               string
	MergedAgentfileSource string
	CredentialProfile    string
	ConfigValues         agentDomain.ConfigValues
}

func NewPodOrchestrator(deps *PodOrchestratorDeps) *PodOrchestrator {
	return &PodOrchestrator{
		podService:      deps.PodService,
		configBuilder:   deps.ConfigBuilder,
		podCoordinator:  deps.PodCoordinator,
		billingService:  deps.BillingService,
		userService:     deps.UserService,
		repoService:     deps.RepoService,
		ticketService:   deps.TicketService,
		runnerSelector:  deps.RunnerSelector,
		agentResolver:   deps.AgentResolver,
		runnerQuery:     deps.RunnerQuery,
		userConfigQuery: deps.UserConfigQuery,
		podRepo:         deps.PodRepo,
	}
}
