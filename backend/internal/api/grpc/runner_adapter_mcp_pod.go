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
// When agentfile_layer is provided, it is the SSOT for pod configuration (MODE, CONFIG, REPO, etc.).
// repository_id and credential_profile_id remain as separate fields (platform-level ID references).
func (a *GRPCRunnerAdapter) mcpCreatePod(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	var params struct {
		AgentSlug           string  `json:"agent_slug"`
		RunnerID            int64   `json:"runner_id"`
		TicketSlug          *string `json:"ticket_slug"`
		Alias               *string `json:"alias"`
		AgentfileLayer        *string `json:"agentfile_layer"`
		Cols                int32   `json:"cols"`
		Rows                int32   `json:"rows"`
		SourcePodKey        string  `json:"source_pod_key"`
		ResumeAgentSession  *bool   `json:"resume_agent_session"`
		// Platform-level ID references (cannot be expressed as AgentFile declarations on Runner side)
		RepositoryID        *int64  `json:"repository_id"`
		CredentialProfileID *int64  `json:"credential_profile_id"`
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
		TicketSlug:          params.TicketSlug,
		Alias:               params.Alias,
		CredentialProfileID: params.CredentialProfileID,
		AgentfileLayer:      params.AgentfileLayer,
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
	case errors.Is(err, agentpod.ErrInvalidAgentfileLayer):
		return newMcpError(400, err.Error())
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
