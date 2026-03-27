package agentpod

import (
	"context"
	"errors"
	"log"

	"github.com/google/uuid"

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

// Typed errors returned by PodOrchestrator.
var (
	ErrMissingRunnerID            = errors.New("runner_id is required")
	ErrMissingAgentSlug           = errors.New("agent_slug is required")
	ErrSourcePodNotFound          = errors.New("source pod not found")
	ErrSourcePodAccessDenied      = errors.New("source pod belongs to different organization")
	ErrSourcePodNotTerminated     = errors.New("source pod is not terminated")
	ErrSourcePodAlreadyResumed    = errors.New("source pod already resumed")
	ErrResumeRunnerMismatch       = errors.New("resume requires same runner")
	ErrConfigBuildFailed          = errors.New("failed to build pod configuration")
	ErrRunnerDispatchFailed       = errors.New("failed to dispatch pod to runner")
	ErrUnsupportedInteractionMode = errors.New("agent type does not support the requested interaction mode")
)

const errCodeRunnerUnreachable = "RUNNER_UNREACHABLE"

// OrchestrateCreatePodRequest is the unified Pod creation request (protocol-agnostic).
type OrchestrateCreatePodRequest struct {
	OrganizationID int64
	UserID         int64

	RunnerID            int64
	AgentSlug           string
	RepositoryID        *int64
	RepositoryURL       *string
	TicketID            *int64
	TicketSlug          *string
	InitialPrompt       string
	Alias               *string
	BranchName          *string
	PermissionMode      *string
	InteractionMode     *string
	CredentialProfileID *int64
	ConfigOverrides     map[string]interface{}
	Cols                int32
	Rows                int32

	SourcePodKey       string
	ResumeAgentSession *bool
}

// OrchestrateCreatePodResult is the result of a successful Pod creation.
type OrchestrateCreatePodResult struct {
	Pod     *podDomain.Pod
	Warning string
}

// --- Narrow interfaces for PodOrchestrator dependencies ---

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
}

type TicketServiceForOrchestrator interface {
	GetTicket(ctx context.Context, ticketID int64) (*ticket.Ticket, error)
	GetTicketBySlug(ctx context.Context, organizationID int64, slug string) (*ticket.Ticket, error)
}

type RunnerSelectorForOrchestrator interface {
	SelectAvailableRunnerForAgent(ctx context.Context, orgID int64, userID int64, agentSlug string) (*runnerDomain.Runner, error)
}

type RunnerQueryForOrchestrator interface {
	GetRunner(ctx context.Context, runnerID int64) (*runnerDomain.Runner, error)
}

type AgentResolverForOrchestrator interface {
	GetAgent(ctx context.Context, slug string) (*agentDomain.Agent, error)
}

// PodOrchestratorDeps holds all dependencies for PodOrchestrator.
type PodOrchestratorDeps struct {
	PodService     *PodService
	ConfigBuilder  *agent.ConfigBuilder
	PodCoordinator PodCoordinatorForOrchestrator
	BillingService BillingServiceForOrchestrator
	UserService    UserServiceForOrchestrator
	RepoService    RepositoryServiceForOrchestrator
	TicketService  TicketServiceForOrchestrator
	RunnerSelector RunnerSelectorForOrchestrator
	AgentResolver  AgentResolverForOrchestrator
	RunnerQuery    RunnerQueryForOrchestrator
}

// PodOrchestrator encapsulates the complete Pod creation workflow.
type PodOrchestrator struct {
	podService     *PodService
	configBuilder  *agent.ConfigBuilder
	podCoordinator PodCoordinatorForOrchestrator
	billingService BillingServiceForOrchestrator
	userService    UserServiceForOrchestrator
	repoService    RepositoryServiceForOrchestrator
	ticketService  TicketServiceForOrchestrator
	runnerSelector RunnerSelectorForOrchestrator
	agentResolver  AgentResolverForOrchestrator
	runnerQuery    RunnerQueryForOrchestrator
}

func NewPodOrchestrator(deps *PodOrchestratorDeps) *PodOrchestrator {
	return &PodOrchestrator{
		podService:     deps.PodService,
		configBuilder:  deps.ConfigBuilder,
		podCoordinator: deps.PodCoordinator,
		billingService: deps.BillingService,
		userService:    deps.UserService,
		repoService:    deps.RepoService,
		ticketService:  deps.TicketService,
		runnerSelector: deps.RunnerSelector,
		agentResolver:  deps.AgentResolver,
		runnerQuery:    deps.RunnerQuery,
	}
}

// CreatePod orchestrates the full Pod creation flow:
// resume handling -> validation -> quota -> DB record -> config build -> dispatch to Runner.
func (o *PodOrchestrator) CreatePod(ctx context.Context, req *OrchestrateCreatePodRequest) (*OrchestrateCreatePodResult, error) {
	var sourcePod *podDomain.Pod
	var sessionID string
	isResumeMode := req.SourcePodKey != ""

	if isResumeMode {
		var err error
		sourcePod, sessionID, err = o.handleResumeMode(ctx, req)
		if err != nil {
			return nil, err
		}
	} else {
		if req.AgentSlug == "" {
			return nil, ErrMissingAgentSlug
		}
		if req.RunnerID == 0 {
			if o.runnerSelector == nil || o.agentResolver == nil {
				return nil, ErrMissingRunnerID
			}
			agentDef, err := o.agentResolver.GetAgent(ctx, req.AgentSlug)
			if err != nil {
				return nil, ErrMissingAgentSlug
			}
			selectedRunner, err := o.runnerSelector.SelectAvailableRunnerForAgent(ctx, req.OrganizationID, req.UserID, agentDef.Slug)
			if err != nil {
				return nil, ErrNoAvailableRunner
			}
			req.RunnerID = selectedRunner.ID
		}
		sessionID = uuid.New().String()
	}

	if req.ConfigOverrides == nil {
		req.ConfigOverrides = make(map[string]interface{})
	}
	if !isResumeMode {
		req.ConfigOverrides["session_id"] = sessionID
	}

	// Validate interaction mode against agent capabilities
	interactionMode := podDomain.InteractionModePTY
	if req.InteractionMode != nil && *req.InteractionMode != "" {
		interactionMode = *req.InteractionMode
	}
	if req.AgentSlug != "" && o.agentResolver != nil {
		agentDef, err := o.agentResolver.GetAgent(ctx, req.AgentSlug)
		if err == nil && !agentDef.SupportsMode(interactionMode) {
			return nil, ErrUnsupportedInteractionMode
		}
	}

	// Quota check
	if o.billingService != nil {
		if err := o.billingService.CheckQuota(ctx, req.OrganizationID, "concurrent_pods", 1); err != nil {
			return nil, err
		}
	}

	// Resolve TicketSlug -> TicketID
	if req.TicketID == nil && req.TicketSlug != nil && *req.TicketSlug != "" && o.ticketService != nil {
		t, err := o.ticketService.GetTicketBySlug(ctx, req.OrganizationID, *req.TicketSlug)
		if err == nil && t != nil {
			req.TicketID = &t.ID
		}
	}

	// Convert credential_profile_id: 0 (explicit RunnerHost) -> nil (FK constraint)
	var dbCredProfileID *int64
	if req.CredentialProfileID != nil && *req.CredentialProfileID > 0 {
		dbCredProfileID = req.CredentialProfileID
	}

	pod, err := o.podService.CreatePod(ctx, &CreatePodRequest{
		OrganizationID:      req.OrganizationID,
		RunnerID:            req.RunnerID,
		AgentSlug:           req.AgentSlug,
		RepositoryID:        req.RepositoryID,
		TicketID:            req.TicketID,
		CreatedByID:         req.UserID,
		InitialPrompt:       req.InitialPrompt,
		Alias:               req.Alias,
		BranchName:          req.BranchName,
		SessionID:           sessionID,
		SourcePodKey:        req.SourcePodKey,
		CredentialProfileID: dbCredProfileID,
		InteractionMode:     interactionMode,
	})
	if err != nil {
		return nil, err
	}

	podCmd, err := o.buildPodCommand(ctx, req, pod, sourcePod, isResumeMode)
	if err != nil {
		log.Printf("[pod-orchestrator] Failed to build pod command: %v", err)
		return nil, errors.Join(ErrConfigBuildFailed, err)
	}

	// Dispatch to Runner
	if o.podCoordinator != nil {
		log.Printf("[pod-orchestrator] Sending create_pod to runner %d for pod %s (resume=%v)", req.RunnerID, pod.PodKey, isResumeMode)
		if err := o.podCoordinator.CreatePod(ctx, req.RunnerID, podCmd); err != nil {
			log.Printf("[pod-orchestrator] Failed to send create_pod: %v", err)
			if markErr := o.podService.MarkInitFailed(ctx, pod.PodKey, errCodeRunnerUnreachable,
				"Failed to dispatch pod to runner: "+err.Error()); markErr != nil {
				log.Printf("[pod-orchestrator] Failed to mark pod as init failed: %v", markErr)
			}
			return nil, ErrRunnerDispatchFailed
		}
		log.Printf("[pod-orchestrator] create_pod sent successfully for pod %s", pod.PodKey)
	} else {
		log.Printf("[pod-orchestrator] PodCoordinator is nil, cannot send create_pod command")
	}

	return &OrchestrateCreatePodResult{Pod: pod}, nil
}
