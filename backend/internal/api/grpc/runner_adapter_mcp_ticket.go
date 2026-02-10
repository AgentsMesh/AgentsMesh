package grpc

import (
	"context"
	"errors"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/service/ticket"
)

// ==================== Ticket MCP Methods ====================

// mcpSearchTickets handles the "search_tickets" MCP method.
func (a *GRPCRunnerAdapter) mcpSearchTickets(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	var params struct {
		RepositoryID *int64  `json:"repository_id"`
		Status       string  `json:"status"`
		Type         string  `json:"type"`
		Priority     string  `json:"priority"`
		AssigneeID   *int64  `json:"assignee_id"`
		ParentID     *int64  `json:"parent_id"`
		Query        string  `json:"query"`
		Limit        int     `json:"limit"`
		Page         int     `json:"page"`
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}

	offset := 0
	if params.Page > 0 {
		offset = (params.Page - 1) * limit
	}

	tickets, _, err := a.ticketService.ListTickets(ctx, &ticket.ListTicketsFilter{
		OrganizationID: tc.OrganizationID,
		RepositoryID:   params.RepositoryID,
		Status:         params.Status,
		Type:           params.Type,
		AssigneeID:     params.AssigneeID,
		UserRole:       tc.UserRole,
		Limit:          limit,
		Offset:         offset,
	})
	if err != nil {
		return nil, newMcpError(500, "failed to search tickets")
	}

	return map[string]interface{}{"tickets": tickets}, nil
}

// mcpGetTicket handles the "get_ticket" MCP method.
func (a *GRPCRunnerAdapter) mcpGetTicket(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	var params struct {
		TicketID string `json:"ticket_id"`
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}

	if params.TicketID == "" {
		return nil, newMcpError(400, "ticket_id is required")
	}

	t, err := a.ticketService.GetTicketByIdentifier(ctx, params.TicketID)
	if err != nil {
		return nil, newMcpError(404, "ticket not found")
	}

	if t.OrganizationID != tc.OrganizationID {
		return nil, newMcpError(403, "access denied")
	}

	return map[string]interface{}{"ticket": t}, nil
}

// mcpCreateTicket handles the "create_ticket" MCP method.
func (a *GRPCRunnerAdapter) mcpCreateTicket(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	var params struct {
		RepositoryID   *int64 `json:"repository_id"`
		Title          string `json:"title"`
		Description    string `json:"description"`
		Type           string `json:"type"`
		Priority       string `json:"priority"`
		ParentTicketID *int64 `json:"parent_ticket_id"`
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}

	if params.Title == "" {
		return nil, newMcpError(400, "title is required")
	}
	if params.Type == "" {
		params.Type = "task"
	}
	if params.Priority == "" {
		params.Priority = "medium"
	}

	var description *string
	if params.Description != "" {
		description = &params.Description
	}

	t, err := a.ticketService.CreateTicket(ctx, &ticket.CreateTicketRequest{
		OrganizationID: tc.OrganizationID,
		RepositoryID:   params.RepositoryID,
		ReporterID:     tc.UserID,
		Type:           params.Type,
		Title:          params.Title,
		Description:    description,
		Priority:       params.Priority,
		ParentTicketID: params.ParentTicketID,
	})
	if err != nil {
		return nil, newMcpError(500, "failed to create ticket")
	}

	return map[string]interface{}{"ticket": t}, nil
}

// mcpUpdateTicket handles the "update_ticket" MCP method.
func (a *GRPCRunnerAdapter) mcpUpdateTicket(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	var params struct {
		TicketID    string  `json:"ticket_id"`
		Title       *string `json:"title"`
		Description *string `json:"description"`
		Status      *string `json:"status"`
		Priority    *string `json:"priority"`
		Type        *string `json:"type"`
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}

	if params.TicketID == "" {
		return nil, newMcpError(400, "ticket_id is required")
	}

	t, err := a.ticketService.GetTicketByIdentifier(ctx, params.TicketID)
	if err != nil {
		return nil, newMcpError(404, "ticket not found")
	}
	if t.OrganizationID != tc.OrganizationID {
		return nil, newMcpError(403, "access denied")
	}

	updates := make(map[string]interface{})
	if params.Title != nil {
		updates["title"] = *params.Title
	}
	if params.Description != nil {
		updates["description"] = *params.Description
	}
	if params.Status != nil {
		updates["status"] = *params.Status
	}
	if params.Priority != nil {
		updates["priority"] = *params.Priority
	}
	if params.Type != nil {
		updates["type"] = *params.Type
	}

	t, err = a.ticketService.UpdateTicket(ctx, t.ID, updates)
	if err != nil {
		return nil, newMcpError(500, "failed to update ticket")
	}

	return map[string]interface{}{"ticket": t}, nil
}

// ==================== Pod MCP Methods ====================

// mcpCreatePod handles the "create_pod" MCP method.
// Delegates to PodOrchestrator for the full creation flow (DB + config + Runner command).
func (a *GRPCRunnerAdapter) mcpCreatePod(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	var params struct {
		RunnerID          int64                  `json:"runner_id"`
		AgentTypeID       *int64                 `json:"agent_type_id"`
		CustomAgentTypeID *int64                 `json:"custom_agent_type_id"`
		RepositoryID      *int64                 `json:"repository_id"`
		RepositoryURL     *string                `json:"repository_url"`
		TicketID          *int64                 `json:"ticket_id"`
		TicketIdentifier  *string                `json:"ticket_identifier"`
		InitialPrompt     string                 `json:"initial_prompt"`
		BranchName        *string                `json:"branch_name"`
		PermissionMode    *string                `json:"permission_mode"`
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
		AgentTypeID:         params.AgentTypeID,
		CustomAgentTypeID:   params.CustomAgentTypeID,
		RepositoryID:        params.RepositoryID,
		RepositoryURL:       params.RepositoryURL,
		TicketID:            params.TicketID,
		TicketIdentifier:    params.TicketIdentifier,
		InitialPrompt:       params.InitialPrompt,
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
	case errors.Is(err, agentpod.ErrMissingAgentTypeID):
		return newMcpError(400, "agent_type_id is required")
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
	default:
		return newMcpErrorf(500, "failed to create pod: %v", err)
	}
}
