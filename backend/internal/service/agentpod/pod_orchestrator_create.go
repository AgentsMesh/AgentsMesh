package agentpod

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"

	agentDomain "github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	podDomain "github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

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
			selectedRunner, err := o.runnerSelector.SelectAvailableRunnerForAgent(ctx, req.OrganizationID, req.UserID, req.AgentSlug)
			if err != nil {
				slog.Warn("runner auto-selection failed", "org_id", req.OrganizationID, "agent_slug", req.AgentSlug, "error", err)
				return nil, ErrNoAvailableRunner
			}
			req.RunnerID = selectedRunner.ID
			slog.Info("runner auto-selected", "runner_id", selectedRunner.ID, "org_id", req.OrganizationID, "agent_slug", req.AgentSlug)
		}
		sessionID = uuid.New().String()
	}

	if req.ConfigOverrides == nil {
		req.ConfigOverrides = make(map[string]interface{})
	}

	// Resolve agent definition once — reused for PodFile merge and mode validation.
	var agentDef *agentDomain.Agent
	if req.AgentSlug != "" && o.agentResolver != nil {
		var err error
		agentDef, err = o.agentResolver.GetAgent(ctx, req.AgentSlug)
		if err != nil {
			return nil, ErrMissingAgentSlug
		}
	}

	// Resolve permission mode: may come from req.PermissionMode (old API path)
	// or from PodFile Layer CONFIG declaration (SSOT path).
	permissionMode := "plan"
	if req.PermissionMode != nil && *req.PermissionMode != "" {
		permissionMode = *req.PermissionMode
	}

	// Build systemOverrides: truly system-internal values injected into PodFile.
	// permission_mode is NOT a system override — it's a user-configurable CONFIG value
	// that flows through PodFile Layer or req.PermissionMode.
	systemOverrides := make(map[string]interface{})
	if !isResumeMode {
		systemOverrides["session_id"] = sessionID
	} else {
		resumeAgentSession := req.ResumeAgentSession == nil || *req.ResumeAgentSession
		if resumeAgentSession {
			systemOverrides["resume_enabled"] = true
			systemOverrides["resume_session"] = sessionID
		}
	}

	// PodFile SSOT: resolve CONFIG values from base PodFile + optional user Layer.
	// Always runs when agentDef has a PodFile (even without a Layer, to inject userPrefs).
	var mergedPodfileSource string
	var podfileCredentialProfile string
	if agentDef != nil && agentDef.PodfileSource != nil {
		// Get user's personal config preferences for CONFIG value resolution.
		var userPrefs map[string]interface{}
		if o.userConfigQuery != nil {
			userPrefs = o.userConfigQuery.GetUserConfigPrefs(ctx, req.UserID, req.AgentSlug)
		}

		layerSrc := ""
		if req.PodfileLayer != nil {
			layerSrc = *req.PodfileLayer
		}

		result, err := extractFromPodfileLayer(
			*agentDef.PodfileSource, layerSrc,
			userPrefs, systemOverrides,
		)
		if err != nil {
			return nil, err
		}
		mergedPodfileSource = result.MergedPodfileSource
		podfileCredentialProfile = result.CredentialProfile
		if result.Mode != "" {
			req.InteractionMode = &result.Mode
		}
		if result.Branch != "" {
			req.BranchName = &result.Branch
		}
		if result.PermissionMode != "" {
			permissionMode = result.PermissionMode
		}
		// REPO slug → resolve RepositoryID for DB record + sandbox config
		if result.RepoSlug != "" && o.repoService != nil {
			repo, repoErr := o.repoService.FindByOrgSlug(ctx, req.OrganizationID, result.RepoSlug)
			if repoErr == nil && repo != nil {
				req.RepositoryID = &repo.ID
			}
		}
		// PROMPT content → override InitialPrompt
		if result.Prompt != "" {
			req.InitialPrompt = result.Prompt
		}
	}

	// Validate interaction mode against agent capabilities
	interactionMode := podDomain.InteractionModePTY
	if req.InteractionMode != nil && *req.InteractionMode != "" {
		interactionMode = *req.InteractionMode
	}
	if agentDef != nil && !agentDef.SupportsMode(interactionMode) {
		return nil, ErrUnsupportedInteractionMode
	}

	// Quota check
	if o.billingService != nil {
		if err := o.billingService.CheckQuota(ctx, req.OrganizationID, "concurrent_pods", 1); err != nil {
			slog.Warn("pod quota check failed", "org_id", req.OrganizationID, "error", err)
			return nil, err
		}
	}

	// Resolve TicketSlug -> TicketID
	if req.TicketID == nil && req.TicketSlug != nil && *req.TicketSlug != "" && o.ticketService != nil {
		t, err := o.ticketService.GetTicketBySlug(ctx, req.OrganizationID, *req.TicketSlug)
		if err == nil && t != nil {
			req.TicketID = &t.ID
		} else if err != nil {
			slog.Warn("ticket slug resolution failed", "org_id", req.OrganizationID, "ticket_slug", *req.TicketSlug, "error", err)
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
		PermissionMode:      permissionMode,
		SessionID:           sessionID,
		SourcePodKey:        req.SourcePodKey,
		CredentialProfileID: dbCredProfileID,
		InteractionMode:     interactionMode,
	})
	if err != nil {
		return nil, err
	}

	podCmd, err := o.buildPodCommand(ctx, req, pod, sourcePod, isResumeMode, mergedPodfileSource, podfileCredentialProfile)
	if err != nil {
		slog.Error("failed to build pod command", "pod_key", pod.PodKey, "error", err)
		return nil, errors.Join(ErrConfigBuildFailed, err)
	}

	if o.podCoordinator != nil {
		slog.Info("dispatching create_pod to runner", "runner_id", req.RunnerID, "pod_key", pod.PodKey, "session_id", sessionID, "resume", isResumeMode)
		if err := o.podCoordinator.CreatePod(ctx, req.RunnerID, podCmd); err != nil {
			slog.Error("failed to dispatch create_pod", "pod_key", pod.PodKey, "error", err)
			if markErr := o.podService.MarkInitFailed(ctx, pod.PodKey, errCodeRunnerUnreachable,
				"Failed to dispatch pod to runner: "+err.Error()); markErr != nil {
				slog.Error("failed to mark pod as init failed", "pod_key", pod.PodKey, "error", markErr)
			}
			return nil, ErrRunnerDispatchFailed
		}
		slog.Info("create_pod dispatched", "pod_key", pod.PodKey)
	} else {
		slog.Warn("PodCoordinator is nil, cannot dispatch create_pod", "pod_key", pod.PodKey)
	}

	return &OrchestrateCreatePodResult{Pod: pod}, nil
}
