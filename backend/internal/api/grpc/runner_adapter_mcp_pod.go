package grpc

import (
	"context"
	"errors"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
)

// ==================== Pod MCP Methods ====================

// mcpCreatePod handles the "create_pod" MCP method.
// Delegates to PodOrchestrator for the full creation flow (DB + config + Runner command).
func (a *GRPCRunnerAdapter) mcpCreatePod(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	var params struct {
		RunnerID          int64                  `json:"runner_id"`
		AgentSlug       string                 `json:"agent_slug"`
		RepositoryID      *int64                 `json:"repository_id"`
		RepositoryURL     *string                `json:"repository_url"`
		TicketSlug        *string                `json:"ticket_slug"`
		InitialPrompt     string                 `json:"initial_prompt"`
		Alias             *string                `json:"alias"`
		BranchName        *string                `json:"branch_name"`
		PermissionMode    *string                `json:"permission_mode"`
		// CredentialProfileID specifies which credential profile to use
		// - nil (field absent): use user's default profile, fallback to RunnerHost if no default
		// - 0: explicit RunnerHost mode (use Runner's local environment, no credentials injected)
		// - >0: use specified credential profile ID
		CredentialProfileID *int64               `json:"credential_profile_id"`
		ConfigOverrides   map[string]interface{} `json:"config_overrides"`
		Cols              int32                  `json:"cols"`
		Rows              int32                  `json:"rows"`
		SourcePodKey      string                 `json:"source_pod_key"`
		ResumeAgentSession *bool                 `json:"resume_agent_session"`
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}

	// Delegate to PodOrchestrator for the complete creation flow
	result, err := a.podOrchestrator.CreatePod(ctx, &agentpod.OrchestrateCreatePodRequest{
		OrganizationID:      tc.OrganizationID,
		UserID:              tc.UserID,
		RunnerID:            params.RunnerID,
		AgentSlug:           params.AgentSlug,
		RepositoryID:        params.RepositoryID,
		RepositoryURL:       params.RepositoryURL,
		TicketSlug:          params.TicketSlug,
		InitialPrompt:       params.InitialPrompt,
		Alias:               params.Alias,
		BranchName:          params.BranchName,
		PermissionMode:      params.PermissionMode,
		CredentialProfileID: params.CredentialProfileID,
		ConfigOverrides:     params.ConfigOverrides,
		Cols:                params.Cols,
		Rows:                params.Rows,
		SourcePodKey:        params.SourcePodKey,
		ResumeAgentSession:  params.ResumeAgentSession,
	})
	if err != nil {
		return nil, mapOrchestratorErrorToMCP(err)
	}

	resp := map[string]interface{}{
		"pod": map[string]interface{}{
			"pod_key": result.Pod.PodKey,
			"status":  result.Pod.Status,
		},
	}
	if result.Warning != "" {
		resp["warning"] = result.Warning
	}

	return resp, nil
}

// mapOrchestratorErrorToMCP maps PodOrchestrator errors to MCP error responses.
func mapOrchestratorErrorToMCP(err error) *mcpError {
	switch {
	case errors.Is(err, agentpod.ErrMissingRunnerID):
		return newMcpError(400, "runner_id is required")
	case errors.Is(err, agentpod.ErrMissingAgentSlug):
		return newMcpError(400, "agent_slug is required")
	case errors.Is(err, agentpod.ErrSourcePodNotTerminated):
		return newMcpError(400, "source pod is not terminated")
	case errors.Is(err, agentpod.ErrResumeRunnerMismatch):
		return newMcpError(400, "resume requires same runner")
	case errors.Is(err, agentpod.ErrSourcePodAccessDenied):
		return newMcpError(403, "source pod access denied")
	case errors.Is(err, agentpod.ErrSourcePodNotFound):
		return newMcpError(404, "source pod not found")
	case errors.Is(err, agentpod.ErrSourcePodAlreadyResumed):
		return newMcpError(409, "source pod already resumed")
	case errors.Is(err, agentpod.ErrSandboxAlreadyResumed):
		return newMcpError(409, "sandbox already resumed")
	case errors.Is(err, agentpod.ErrConfigBuildFailed):
		return newMcpError(500, "failed to build pod configuration")
	case errors.Is(err, agentpod.ErrRunnerDispatchFailed):
		return newMcpError(502, "failed to dispatch pod to runner")
	default:
		return newMcpErrorf(500, "failed to create pod: %v", err)
	}
}
