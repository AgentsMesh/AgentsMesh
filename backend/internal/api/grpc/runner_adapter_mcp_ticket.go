package grpc

import (
	"context"
	"errors"
	"strconv"
	"strings"

	ticketDomain "github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/service/ticket"
	"github.com/anthropics/agentsmesh/backend/pkg/blocknote"
)

// mcpTicketResponse wraps ticket.Ticket to add resolved slug fields and content pagination
// metadata for MCP responses. The runner expects parent_ticket_slug (string) instead of
// parent_ticket_id (int64).
type mcpTicketResponse struct {
	*ticketDomain.Ticket
	ParentTicketSlug  string `json:"parent_ticket_slug,omitempty"`
	ContentTotalLines int    `json:"content_total_lines,omitempty"` // Total lines after conversion
	ContentOffset     int    `json:"content_offset,omitempty"`      // Start line of this response (0-based)
	ContentLimit      int    `json:"content_limit,omitempty"`       // Number of lines returned
}

// contentPaginationParams holds parsed content pagination parameters.
type contentPaginationParams struct {
	Offset int
	Limit  int
}

// enrichTicketForMCP resolves the parent ticket's numeric ID to its slug and
// converts BlockNote JSON content to readable plain text with line-range pagination.
func (a *GRPCRunnerAdapter) enrichTicketForMCP(ctx context.Context, orgID int64, t *ticketDomain.Ticket, pagination *contentPaginationParams) *mcpTicketResponse {
	// Shallow copy ticket to avoid mutating the domain object.
	// NOTE: ticketCopy.Content still shares the same *string pointer with t.Content.
	// Always reassign the pointer (ticketCopy.Content = &newVal) instead of
	// dereference-modifying (*ticketCopy.Content = "...") to keep the original safe.
	ticketCopy := *t
	resp := &mcpTicketResponse{Ticket: &ticketCopy}

	if t.ParentTicketID != nil {
		parent, err := a.ticketService.GetTicketByIDOrSlug(ctx, orgID, strconv.FormatInt(*t.ParentTicketID, 10))
		if err == nil {
			resp.ParentTicketSlug = parent.Slug
		}
	}

	// Convert BlockNote JSON content to plain text and apply line-range pagination
	if ticketCopy.Content != nil && *ticketCopy.Content != "" {
		plainText := blocknote.ToPlainText(*ticketCopy.Content)
		lines := strings.Split(plainText, "\n")
		totalLines := len(lines)

		offset := 0
		limit := 200
		if pagination != nil {
			offset = pagination.Offset
			limit = pagination.Limit
		}

		// Clamp offset
		if offset < 0 {
			offset = 0
		}
		if offset > totalLines {
			offset = totalLines
		}

		// Clamp end
		end := offset + limit
		if end > totalLines {
			end = totalLines
		}

		selected := lines[offset:end]
		content := strings.Join(selected, "\n")
		ticketCopy.Content = &content

		resp.ContentTotalLines = totalLines
		resp.ContentOffset = offset
		resp.ContentLimit = len(selected)
	}

	return resp
}

// ==================== Ticket MCP Methods ====================

// mcpSearchTickets handles the "search_tickets" MCP method.
func (a *GRPCRunnerAdapter) mcpSearchTickets(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	var params struct {
		RepositoryID      *int64  `json:"repository_id"`
		Status            string  `json:"status"`
		Priority          string  `json:"priority"`
		AssigneeID        *int64  `json:"assignee_id"`
		ParentTicketSlug  *string `json:"parent_ticket_slug"`
		Query             string  `json:"query"`
		Limit             int     `json:"limit"`
		Page              int     `json:"page"`
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

	// Resolve parent ticket slug to ID
	var parentTicketID *int64
	if params.ParentTicketSlug != nil && *params.ParentTicketSlug != "" {
		parentTicket, err := a.ticketService.GetTicketByIDOrSlug(ctx, tc.OrganizationID, *params.ParentTicketSlug)
		if err != nil {
			return nil, newMcpError(404, "parent ticket not found")
		}
		parentTicketID = &parentTicket.ID
	}

	tickets, _, err := a.ticketService.ListTickets(ctx, &ticket.ListTicketsFilter{
		OrganizationID: tc.OrganizationID,
		RepositoryID:   params.RepositoryID,
		Status:         params.Status,
		Priority:       params.Priority,
		AssigneeID:     params.AssigneeID,
		ParentTicketID: parentTicketID,
		Query:          params.Query,
		UserRole:       tc.UserRole,
		Limit:          limit,
		Offset:         offset,
	})
	if err != nil {
		return nil, newMcpError(500, "failed to search tickets")
	}

	enriched := make([]*mcpTicketResponse, len(tickets))
	for i, t := range tickets {
		enriched[i] = a.enrichTicketForMCP(ctx, tc.OrganizationID, t, nil)
	}
	return map[string]interface{}{"tickets": enriched}, nil
}

// mcpGetTicket handles the "get_ticket" MCP method.
// Supports content_offset and content_limit for paginated reading of ticket content.
func (a *GRPCRunnerAdapter) mcpGetTicket(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	var params struct {
		TicketSlug    string `json:"ticket_slug"`
		ContentOffset *int   `json:"content_offset"` // Start line (0-based), default 0
		ContentLimit  *int   `json:"content_limit"`  // Number of lines, default 200
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}

	if params.TicketSlug == "" {
		return nil, newMcpError(400, "ticket_slug is required")
	}

	t, err := a.ticketService.GetTicketByIDOrSlug(ctx, tc.OrganizationID, params.TicketSlug)
	if err != nil {
		return nil, newMcpError(404, "ticket not found")
	}

	// Build pagination params
	pagination := &contentPaginationParams{Offset: 0, Limit: 200}
	if params.ContentOffset != nil {
		pagination.Offset = *params.ContentOffset
	}
	if params.ContentLimit != nil {
		pagination.Limit = *params.ContentLimit
	}

	return map[string]interface{}{"ticket": a.enrichTicketForMCP(ctx, tc.OrganizationID, t, pagination)}, nil
}

// mcpCreateTicket handles the "create_ticket" MCP method.
func (a *GRPCRunnerAdapter) mcpCreateTicket(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	var params struct {
		RepositoryID     *int64  `json:"repository_id"`
		Title            string  `json:"title"`
		Content          string  `json:"content"`
		Priority         string  `json:"priority"`
		ParentTicketSlug *string `json:"parent_ticket_slug"`
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}

	if params.Title == "" {
		return nil, newMcpError(400, "title is required")
	}
	if params.Priority == "" {
		params.Priority = "medium"
	}

	var content *string
	if params.Content != "" {
		content = &params.Content
	}

	// Resolve parent ticket slug to ID
	var parentTicketID *int64
	if params.ParentTicketSlug != nil && *params.ParentTicketSlug != "" {
		parentTicket, err := a.ticketService.GetTicketByIDOrSlug(ctx, tc.OrganizationID, *params.ParentTicketSlug)
		if err != nil {
			return nil, newMcpError(404, "parent ticket not found")
		}
		parentTicketID = &parentTicket.ID
	}

	t, err := a.ticketService.CreateTicket(ctx, &ticket.CreateTicketRequest{
		OrganizationID: tc.OrganizationID,
		RepositoryID:   params.RepositoryID,
		ReporterID:     tc.UserID,
		Title:          params.Title,
		Content:        content,
		Priority:       params.Priority,
		ParentTicketID: parentTicketID,
	})
	if err != nil {
		return nil, newMcpError(500, "failed to create ticket")
	}

	return map[string]interface{}{"ticket": a.enrichTicketForMCP(ctx, tc.OrganizationID, t, nil)}, nil
}

// mcpUpdateTicket handles the "update_ticket" MCP method.
func (a *GRPCRunnerAdapter) mcpUpdateTicket(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	var params struct {
		TicketSlug string  `json:"ticket_slug"`
		Title      *string `json:"title"`
		Content    *string `json:"content"`
		Status     *string `json:"status"`
		Priority   *string `json:"priority"`
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}

	if params.TicketSlug == "" {
		return nil, newMcpError(400, "ticket_slug is required")
	}

	t, err := a.ticketService.GetTicketByIDOrSlug(ctx, tc.OrganizationID, params.TicketSlug)
	if err != nil {
		return nil, newMcpError(404, "ticket not found")
	}

	updates := make(map[string]interface{})
	if params.Title != nil {
		updates["title"] = *params.Title
	}
	if params.Content != nil {
		updates["content"] = *params.Content
	}
	if params.Status != nil {
		updates["status"] = *params.Status
	}
	if params.Priority != nil {
		updates["priority"] = *params.Priority
	}

	t, err = a.ticketService.UpdateTicket(ctx, t.ID, updates)
	if err != nil {
		return nil, newMcpError(500, "failed to update ticket")
	}

	return map[string]interface{}{"ticket": a.enrichTicketForMCP(ctx, tc.OrganizationID, t, nil)}, nil
}

// mcpPostComment handles the "post_comment" MCP method.
func (a *GRPCRunnerAdapter) mcpPostComment(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	var params struct {
		TicketSlug string  `json:"ticket_slug"`
		Content    string  `json:"content"`
		ParentID   *int64  `json:"parent_id"`
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}

	if params.TicketSlug == "" {
		return nil, newMcpError(400, "ticket_slug is required")
	}
	if params.Content == "" {
		return nil, newMcpError(400, "content is required")
	}

	t, err := a.ticketService.GetTicketByIDOrSlug(ctx, tc.OrganizationID, params.TicketSlug)
	if err != nil {
		return nil, newMcpError(404, "ticket not found")
	}

	comment, err := a.ticketService.CreateComment(ctx, t.ID, tc.UserID, params.Content, params.ParentID, nil)
	if err != nil {
		return nil, newMcpError(500, "failed to post comment")
	}

	return map[string]interface{}{"comment": comment}, nil
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
		AgentTypeID:         params.AgentTypeID,
		CustomAgentTypeID:   params.CustomAgentTypeID,
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
	case errors.Is(err, agentpod.ErrRunnerDispatchFailed):
		return newMcpError(502, "failed to dispatch pod to runner")
	default:
		return newMcpErrorf(500, "failed to create pod: %v", err)
	}
}
