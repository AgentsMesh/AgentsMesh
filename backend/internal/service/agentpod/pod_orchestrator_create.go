package agentpod

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	agentDomain "github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	podDomain "github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	otelinit "github.com/anthropics/agentsmesh/backend/internal/infra/otel"
)

// CreatePod orchestrates the full Pod creation flow:
// resume handling -> validation -> quota -> DB record -> config build -> dispatch to Runner.
func (o *PodOrchestrator) CreatePod(ctx context.Context, req *OrchestrateCreatePodRequest) (*OrchestrateCreatePodResult, error) {
	createStart := time.Now()
	defer func() {
		otelinit.PodCreateDuration.Record(ctx, float64(time.Since(createStart).Milliseconds()))
	}()

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

			hints := o.buildAffinityHints(ctx, req)
			repoHistory := o.fetchRepoHistory(ctx, req.OrganizationID, hints)

			selectedRunner, err := o.runnerSelector.SelectRunnerWithAffinity(
				ctx, req.OrganizationID, req.UserID, req.AgentSlug, hints, repoHistory,
			)
			if err != nil {
				slog.WarnContext(ctx, "runner auto-selection failed", "org_id", req.OrganizationID, "agent_slug", req.AgentSlug, "error", err)
				return nil, ErrNoAvailableRunner
			}
			req.RunnerID = selectedRunner.ID
			slog.InfoContext(ctx, "runner auto-selected", "runner_id", selectedRunner.ID, "org_id", req.OrganizationID, "agent_slug", req.AgentSlug)
		}
		sessionID = uuid.New().String()
	}

	// Resolve agent definition once — reused for AgentFile merge and mode validation.
	var agentDef *agentDomain.Agent
	if req.AgentSlug != "" && o.agentResolver != nil {
		var err error
		agentDef, err = o.agentResolver.GetAgent(ctx, req.AgentSlug)
		if err != nil {
			return nil, ErrMissingAgentSlug
		}
	}

	// --- AgentFile Layer resolution ---
	resolved := &agentfileResolved{}

	// Build systemOverrides: truly system-internal values injected into AgentFile.
	systemOverrides := make(map[string]interface{})
	if !isResumeMode {
		systemOverrides["session_id"] = sessionID
	} else {
		systemOverrides["session_id"] = ""
		resumeAgentSession := req.ResumeAgentSession == nil || *req.ResumeAgentSession
		if resumeAgentSession {
			systemOverrides["resume_enabled"] = true
			systemOverrides["resume_session"] = sessionID
		}
	}

	// AgentFile SSOT: resolve CONFIG values from base AgentFile + optional user Layer.
	if agentDef != nil && agentDef.AgentfileSource != nil {
		var userPrefs map[string]interface{}
		if o.userConfigQuery != nil {
			userPrefs = o.userConfigQuery.GetUserConfigPrefs(ctx, req.UserID, req.AgentSlug)
		}

		layerSrc := ""
		if req.AgentfileLayer != nil {
			layerSrc = *req.AgentfileLayer
		}

		result, err := extractFromAgentfileLayer(
			*agentDef.AgentfileSource, layerSrc,
			userPrefs, systemOverrides,
		)
		if err != nil {
			return nil, err
		}
		resolved.MergedAgentfileSource = result.MergedAgentfileSource
		resolved.CredentialProfile = result.CredentialProfile
		if result.Mode != "" {
			resolved.InteractionMode = result.Mode
		}
		if result.Branch != "" {
			resolved.BranchName = result.Branch
		}
		if result.PermissionMode != "" {
			resolved.PermissionMode = result.PermissionMode
		}
		if result.RepoSlug != "" && o.repoService != nil {
			repo, repoErr := o.repoService.FindByOrgSlug(ctx, req.OrganizationID, result.RepoSlug)
			if repoErr == nil && repo != nil {
				resolved.RepositoryID = &repo.ID
			}
		}
		if result.Prompt != "" {
			resolved.Prompt = result.Prompt
		}
	}

	// --- Compute effective values: resolved (AgentFile) > source pod (resume) > defaults ---
	effectiveInteractionMode := firstNonEmpty(resolved.InteractionMode, podDomain.InteractionModePTY)
	// Permission mode: AgentFile > source pod (resume inheritance) > default.
	sourcePermMode := ""
	if sourcePod != nil && sourcePod.PermissionMode != nil {
		sourcePermMode = *sourcePod.PermissionMode
	}
	effectivePermissionMode := firstNonEmpty(resolved.PermissionMode, sourcePermMode, podDomain.PermissionModeBypass)
	effectiveBranch := firstNonEmptyPtr(resolved.BranchName, req.BranchName) // req.BranchName only from resume
	effectiveRepoID := firstNonNilInt64(resolved.RepositoryID, req.RepositoryID)

	// Validate interaction mode against agent capabilities
	if agentDef != nil && !agentDef.SupportsMode(effectiveInteractionMode) {
		return nil, ErrUnsupportedInteractionMode
	}

	// Quota check
	if o.billingService != nil {
		if err := o.billingService.CheckQuota(ctx, req.OrganizationID, "concurrent_pods", 1); err != nil {
			slog.WarnContext(ctx, "pod quota check failed", "org_id", req.OrganizationID, "error", err)
			return nil, err
		}
	}

	// Resolve TicketSlug -> TicketID
	if req.TicketID == nil && req.TicketSlug != nil && *req.TicketSlug != "" && o.ticketService != nil {
		t, err := o.ticketService.GetTicketBySlug(ctx, req.OrganizationID, *req.TicketSlug)
		if err == nil && t != nil {
			req.TicketID = &t.ID
		} else if err != nil {
			slog.WarnContext(ctx, "ticket slug resolution failed", "org_id", req.OrganizationID, "ticket_slug", *req.TicketSlug, "error", err)
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
		RepositoryID:        effectiveRepoID,
		TicketID:            req.TicketID,
		CreatedByID:         req.UserID,
		Prompt:              resolved.Prompt,
		Alias:               req.Alias,
		BranchName:          effectiveBranch,
		PermissionMode:      effectivePermissionMode,
		SessionID:           sessionID,
		SourcePodKey:        req.SourcePodKey,
		CredentialProfileID: dbCredProfileID,
		InteractionMode:     effectiveInteractionMode,
		Perpetual:           req.Perpetual,
	})
	if err != nil {
		return nil, err
	}

	podCmd, err := o.buildPodCommand(ctx, req, pod, sourcePod, isResumeMode, resolved)
	if err != nil {
		slog.ErrorContext(ctx, "failed to build pod command", "pod_key", pod.PodKey, "error", err)
		return nil, errors.Join(ErrConfigBuildFailed, err)
	}

	if o.podCoordinator != nil {
		slog.InfoContext(ctx, "dispatching create_pod to runner", "runner_id", req.RunnerID, "pod_key", pod.PodKey, "session_id", sessionID, "resume", isResumeMode)
		if err := o.podCoordinator.CreatePod(ctx, req.RunnerID, podCmd); err != nil {
			slog.ErrorContext(ctx, "failed to dispatch create_pod", "pod_key", pod.PodKey, "error", err)
			if markErr := o.podService.MarkInitFailed(ctx, pod.PodKey, errCodeRunnerUnreachable,
				"Failed to dispatch pod to runner: "+err.Error()); markErr != nil {
				slog.ErrorContext(ctx, "failed to mark pod as init failed", "pod_key", pod.PodKey, "error", markErr)
			}
			return nil, ErrRunnerDispatchFailed
		}
		slog.InfoContext(ctx, "create_pod dispatched", "pod_key", pod.PodKey)
	} else {
		slog.WarnContext(ctx, "PodCoordinator is nil, cannot dispatch create_pod", "pod_key", pod.PodKey)
	}

	return &OrchestrateCreatePodResult{Pod: pod}, nil
}
