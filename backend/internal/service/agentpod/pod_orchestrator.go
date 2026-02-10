package agentpod

import (
	"context"
	"errors"
	"log"

	"github.com/google/uuid"

	podDomain "github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/gitprovider"
	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"github.com/anthropics/agentsmesh/backend/internal/service/agent"
	userService "github.com/anthropics/agentsmesh/backend/internal/service/user"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// Typed errors returned by PodOrchestrator.
// Callers (REST / MCP) map these to protocol-specific responses.
var (
	ErrMissingRunnerID         = errors.New("runner_id is required")
	ErrMissingAgentTypeID      = errors.New("agent_type_id is required")
	ErrSourcePodNotFound       = errors.New("source pod not found")
	ErrSourcePodAccessDenied   = errors.New("source pod belongs to different organization")
	ErrSourcePodNotTerminated  = errors.New("source pod is not terminated")
	ErrSourcePodAlreadyResumed = errors.New("source pod already resumed")
	ErrResumeRunnerMismatch    = errors.New("resume requires same runner")
	ErrConfigBuildFailed       = errors.New("failed to build pod configuration")
)

// OrchestrateCreatePodRequest is the unified Pod creation request (protocol-agnostic).
type OrchestrateCreatePodRequest struct {
	// Identity (extracted from auth context by the caller)
	OrganizationID int64
	UserID         int64

	// Basic parameters
	RunnerID          int64
	AgentTypeID       *int64
	CustomAgentTypeID *int64
	RepositoryID      *int64
	RepositoryURL     *string
	TicketID          *int64
	TicketIdentifier  *string
	InitialPrompt     string
	BranchName        *string
	PermissionMode    *string
	CredentialProfileID *int64
	ConfigOverrides   map[string]interface{}
	Cols              int32
	Rows              int32

	// Resume related
	SourcePodKey       string
	ResumeAgentSession *bool
}

// OrchestrateCreatePodResult is the result of a successful Pod creation.
type OrchestrateCreatePodResult struct {
	Pod     *podDomain.Pod
	Warning string // Non-fatal error (e.g. Runner communication failed but DB record created)
}

// --- Narrow interfaces for PodOrchestrator dependencies ---

// PodCoordinatorForOrchestrator sends commands to Runner.
type PodCoordinatorForOrchestrator interface {
	CreatePod(ctx context.Context, runnerID int64, cmd *runnerv1.CreatePodCommand) error
}

// BillingServiceForOrchestrator checks quota.
type BillingServiceForOrchestrator interface {
	CheckQuota(ctx context.Context, orgID int64, quotaType string, amount int) error
}

// UserServiceForOrchestrator retrieves Git credentials.
type UserServiceForOrchestrator interface {
	GetDefaultGitCredential(ctx context.Context, userID int64) (*user.GitCredential, error)
	GetDecryptedCredentialToken(ctx context.Context, userID, credentialID int64) (*userService.DecryptedCredential, error)
}

// RepositoryServiceForOrchestrator resolves repository info.
type RepositoryServiceForOrchestrator interface {
	GetByID(ctx context.Context, id int64) (*gitprovider.Repository, error)
}

// TicketServiceForOrchestrator resolves ticket info.
type TicketServiceForOrchestrator interface {
	GetTicket(ctx context.Context, ticketID int64) (*ticket.Ticket, error)
}

// PodOrchestratorDeps holds all dependencies for PodOrchestrator.
type PodOrchestratorDeps struct {
	PodService     *PodService
	ConfigBuilder  *agent.ConfigBuilder
	PodCoordinator PodCoordinatorForOrchestrator  // optional
	BillingService BillingServiceForOrchestrator   // optional
	UserService    UserServiceForOrchestrator      // optional
	RepoService    RepositoryServiceForOrchestrator // optional
	TicketService  TicketServiceForOrchestrator    // optional
}

// PodOrchestrator encapsulates the complete Pod creation workflow.
// Both REST and MCP paths delegate to this single implementation.
type PodOrchestrator struct {
	podService     *PodService
	configBuilder  *agent.ConfigBuilder
	podCoordinator PodCoordinatorForOrchestrator
	billingService BillingServiceForOrchestrator
	userService    UserServiceForOrchestrator
	repoService    RepositoryServiceForOrchestrator
	ticketService  TicketServiceForOrchestrator
}

// NewPodOrchestrator creates a new PodOrchestrator.
func NewPodOrchestrator(deps *PodOrchestratorDeps) *PodOrchestrator {
	return &PodOrchestrator{
		podService:     deps.PodService,
		configBuilder:  deps.ConfigBuilder,
		podCoordinator: deps.PodCoordinator,
		billingService: deps.BillingService,
		userService:    deps.UserService,
		repoService:    deps.RepoService,
		ticketService:  deps.TicketService,
	}
}

// CreatePod orchestrates the full Pod creation flow:
//  1. Resume mode handling (validate source pod, inherit config)
//  2. Normal mode validation
//  3. Quota check
//  4. DB record creation
//  5. Config building (repository, ticket, credentials, ConfigBuilder)
//  6. Send command to Runner via PodCoordinator
func (o *PodOrchestrator) CreatePod(ctx context.Context, req *OrchestrateCreatePodRequest) (*OrchestrateCreatePodResult, error) {
	// === 1. Resume mode handling ===
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
		// === 2. Normal mode validation ===
		if req.RunnerID == 0 {
			return nil, ErrMissingRunnerID
		}
		if req.AgentTypeID == nil {
			return nil, ErrMissingAgentTypeID
		}
		sessionID = uuid.New().String()
	}

	// Add session_id to config overrides for NEW sessions only
	if req.ConfigOverrides == nil {
		req.ConfigOverrides = make(map[string]interface{})
	}
	if !isResumeMode {
		req.ConfigOverrides["session_id"] = sessionID
	}

	// === 3. Quota check ===
	if o.billingService != nil {
		if err := o.billingService.CheckQuota(ctx, req.OrganizationID, "concurrent_pods", 1); err != nil {
			return nil, err // Caller maps billing.ErrQuotaExceeded / billing.ErrSubscriptionFrozen
		}
	}

	// === 4. DB record creation ===
	pod, err := o.podService.CreatePod(ctx, &CreatePodRequest{
		OrganizationID:    req.OrganizationID,
		RunnerID:          req.RunnerID,
		AgentTypeID:       req.AgentTypeID,
		CustomAgentTypeID: req.CustomAgentTypeID,
		RepositoryID:      req.RepositoryID,
		TicketID:          req.TicketID,
		CreatedByID:       req.UserID,
		InitialPrompt:     req.InitialPrompt,
		BranchName:        req.BranchName,
		SessionID:         sessionID,
		SourcePodKey:      req.SourcePodKey,
	})
	if err != nil {
		return nil, err // Includes ErrSandboxAlreadyResumed
	}

	// === 5. Build Pod command ===
	podCmd, err := o.buildPodCommand(ctx, req, pod, sourcePod, isResumeMode)
	if err != nil {
		log.Printf("[pod-orchestrator] Failed to build pod command: %v", err)
		return nil, errors.Join(ErrConfigBuildFailed, err)
	}

	// === 6. Send command to Runner ===
	if o.podCoordinator != nil {
		log.Printf("[pod-orchestrator] Sending create_pod to runner %d for pod %s (resume=%v)", req.RunnerID, pod.PodKey, isResumeMode)
		if err := o.podCoordinator.CreatePod(ctx, req.RunnerID, podCmd); err != nil {
			log.Printf("[pod-orchestrator] Failed to send create_pod: %v", err)
			return &OrchestrateCreatePodResult{
				Pod:     pod,
				Warning: "Pod created but runner communication failed: " + err.Error(),
			}, nil
		}
		log.Printf("[pod-orchestrator] create_pod sent successfully for pod %s", pod.PodKey)
	} else {
		log.Printf("[pod-orchestrator] PodCoordinator is nil, cannot send create_pod command")
	}

	return &OrchestrateCreatePodResult{Pod: pod}, nil
}

// handleResumeMode validates the source pod and inherits configuration.
func (o *PodOrchestrator) handleResumeMode(ctx context.Context, req *OrchestrateCreatePodRequest) (*podDomain.Pod, string, error) {
	sourcePod, err := o.podService.GetPod(ctx, req.SourcePodKey)
	if err != nil {
		return nil, "", ErrSourcePodNotFound
	}

	// Verify source pod belongs to same organization
	if sourcePod.OrganizationID != req.OrganizationID {
		return nil, "", ErrSourcePodAccessDenied
	}

	// Verify source pod is terminated
	if sourcePod.Status != podDomain.StatusTerminated &&
		sourcePod.Status != podDomain.StatusCompleted &&
		sourcePod.Status != podDomain.StatusOrphaned {
		return nil, "", ErrSourcePodNotTerminated
	}

	// Check if source pod has already been resumed
	existingResumePod, err := o.podService.GetActivePodBySourcePodKey(ctx, req.SourcePodKey)
	if err == nil && existingResumePod != nil {
		return nil, "", ErrSourcePodAlreadyResumed
	}

	// Inherit runner_id from source pod if not provided
	if req.RunnerID == 0 {
		req.RunnerID = sourcePod.RunnerID
	} else if sourcePod.RunnerID != req.RunnerID {
		return nil, "", ErrResumeRunnerMismatch
	}

	// Inherit configuration from source pod
	if req.AgentTypeID == nil {
		req.AgentTypeID = sourcePod.AgentTypeID
	}
	if req.CustomAgentTypeID == nil {
		req.CustomAgentTypeID = sourcePod.CustomAgentTypeID
	}
	if req.RepositoryID == nil {
		req.RepositoryID = sourcePod.RepositoryID
	}
	if req.TicketID == nil {
		req.TicketID = sourcePod.TicketID
	}
	if req.BranchName == nil {
		req.BranchName = sourcePod.BranchName
	}

	// Reuse session ID from source pod
	var sessionID string
	if sourcePod.SessionID != nil && *sourcePod.SessionID != "" {
		sessionID = *sourcePod.SessionID
	} else {
		sessionID = uuid.New().String()
	}

	// Set resume configuration
	resumeAgentSession := req.ResumeAgentSession == nil || *req.ResumeAgentSession
	if resumeAgentSession {
		if req.ConfigOverrides == nil {
			req.ConfigOverrides = make(map[string]interface{})
		}
		req.ConfigOverrides["resume_enabled"] = true
		req.ConfigOverrides["resume_session"] = sessionID
	}

	return sourcePod, sessionID, nil
}

// buildPodCommand constructs the CreatePodCommand using ConfigBuilder.
func (o *PodOrchestrator) buildPodCommand(
	ctx context.Context,
	req *OrchestrateCreatePodRequest,
	pod *podDomain.Pod,
	sourcePod *podDomain.Pod,
	isResumeMode bool,
) (*runnerv1.CreatePodCommand, error) {
	// Get permission mode
	permissionMode := "plan"
	if req.PermissionMode != nil {
		permissionMode = *req.PermissionMode
	} else if pod.PermissionMode != nil {
		permissionMode = *pod.PermissionMode
	}

	// For resume mode, set local_path to source pod's sandbox path
	if isResumeMode && sourcePod != nil && sourcePod.SandboxPath != nil {
		if req.ConfigOverrides == nil {
			req.ConfigOverrides = make(map[string]interface{})
		}
		req.ConfigOverrides["sandbox_local_path"] = *sourcePod.SandboxPath
	}

	// Resolve repository info
	repositoryURL := ""
	sourceBranch := ""
	preparationScript := ""
	preparationTimeout := 300
	if req.RepositoryURL != nil && *req.RepositoryURL != "" {
		repositoryURL = *req.RepositoryURL
	} else if req.RepositoryID != nil && o.repoService != nil {
		repo, err := o.repoService.GetByID(ctx, *req.RepositoryID)
		if err == nil && repo != nil {
			repositoryURL = repo.CloneURL
			if repo.DefaultBranch != "" {
				sourceBranch = repo.DefaultBranch
			}
			if repo.PreparationScript != nil {
				preparationScript = *repo.PreparationScript
			}
			if repo.PreparationTimeout != nil {
				preparationTimeout = *repo.PreparationTimeout
			}
		}
	}
	// Override branch if specified in request
	if req.BranchName != nil && *req.BranchName != "" {
		sourceBranch = *req.BranchName
	}

	// Resolve ticket ID
	ticketID := ""
	if req.TicketIdentifier != nil && *req.TicketIdentifier != "" {
		ticketID = *req.TicketIdentifier
	} else if req.TicketID != nil && o.ticketService != nil {
		t, err := o.ticketService.GetTicket(ctx, *req.TicketID)
		if err == nil && t != nil {
			ticketID = t.Identifier
		}
	}

	// Get Git credentials
	credentialType := ""
	gitToken := ""
	sshPrivateKey := ""
	if o.userService != nil {
		gitCred := o.getUserGitCredential(ctx, req.UserID)
		if gitCred != nil {
			credentialType = gitCred.Type
			switch gitCred.Type {
			case "oauth", "pat":
				gitToken = gitCred.Token
			case "ssh_key":
				sshPrivateKey = gitCred.SSHPrivateKey
			case "runner_local":
				// No credentials needed
			}
		}
	}

	// Build config overrides
	configOverrides := make(map[string]interface{})
	if req.ConfigOverrides != nil {
		for k, v := range req.ConfigOverrides {
			configOverrides[k] = v
		}
	}
	configOverrides["permission_mode"] = permissionMode

	// Handle sandbox_local_path for Resume mode
	localPath := ""
	if path, ok := configOverrides["sandbox_local_path"].(string); ok && path != "" {
		localPath = path
		repositoryURL = "" // Resume mode: don't clone repository
		delete(configOverrides, "sandbox_local_path")
	}

	// Build the request for ConfigBuilder
	buildReq := &agent.ConfigBuildRequest{
		AgentTypeID:         *req.AgentTypeID,
		OrganizationID:      req.OrganizationID,
		UserID:              req.UserID,
		CredentialProfileID: req.CredentialProfileID,
		RepositoryURL:       repositoryURL,
		SourceBranch:        sourceBranch,
		CredentialType:      credentialType,
		GitToken:            gitToken,
		SSHPrivateKey:       sshPrivateKey,
		TicketID:            ticketID,
		PreparationScript:   preparationScript,
		PreparationTimeout:  preparationTimeout,
		LocalPath:           localPath,
		ConfigOverrides:     configOverrides,
		InitialPrompt:       req.InitialPrompt,
		PodKey:              pod.PodKey,
		MCPPort:             19000,
		Cols:                req.Cols,
		Rows:                req.Rows,
	}

	return o.configBuilder.BuildPodCommand(ctx, buildReq)
}

// getUserGitCredential retrieves the default Git credential for a user.
// Returns nil if using runner_local (Runner will use local Git config).
func (o *PodOrchestrator) getUserGitCredential(ctx context.Context, userID int64) *userService.DecryptedCredential {
	if o.userService == nil {
		return nil
	}

	defaultCred, err := o.userService.GetDefaultGitCredential(ctx, userID)
	if err != nil || defaultCred == nil {
		return nil
	}

	// runner_local → let Runner use local config
	if defaultCred.CredentialType == "runner_local" {
		return nil
	}

	decrypted, err := o.userService.GetDecryptedCredentialToken(ctx, userID, defaultCred.ID)
	if err != nil {
		log.Printf("[pod-orchestrator] Failed to decrypt Git credential: %v", err)
		return nil
	}

	return decrypted
}
