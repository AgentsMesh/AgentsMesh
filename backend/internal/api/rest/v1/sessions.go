package v1

import (
	"log"
	"net/http"
	"strconv"

	"github.com/anthropics/agentmesh/backend/internal/middleware"
	"github.com/anthropics/agentmesh/backend/internal/service/agent"
	"github.com/anthropics/agentmesh/backend/internal/service/gitprovider"
	"github.com/anthropics/agentmesh/backend/internal/service/repository"
	"github.com/anthropics/agentmesh/backend/internal/service/runner"
	"github.com/anthropics/agentmesh/backend/internal/service/session"
	"github.com/anthropics/agentmesh/backend/internal/service/sshkey"
	"github.com/anthropics/agentmesh/backend/internal/service/ticket"
	"github.com/anthropics/agentmesh/backend/internal/service/user"
	"github.com/gin-gonic/gin"
)

// SessionHandler handles session-related requests
type SessionHandler struct {
	sessionService     *session.Service
	runnerService      *runner.Service
	agentService       *agent.Service
	repositoryService  *repository.Service
	ticketService      *ticket.Service
	gitProviderService *gitprovider.Service
	sshKeyService      *sshkey.Service
	userService        *user.Service // For user credential retrieval (权限跟人走)
	runnerConnMgr      *runner.ConnectionManager
	sessionCoordinator *runner.SessionCoordinator
	terminalRouter     interface{} // *runner.TerminalRouter, optional
}

// NewSessionHandler creates a new session handler
func NewSessionHandler(sessionService *session.Service, runnerService *runner.Service, agentService *agent.Service) *SessionHandler {
	return &SessionHandler{
		sessionService: sessionService,
		runnerService:  runnerService,
		agentService:   agentService,
	}
}

// SetRunnerConnectionManager sets the runner connection manager
func (h *SessionHandler) SetRunnerConnectionManager(cm *runner.ConnectionManager) {
	h.runnerConnMgr = cm
}

// SetSessionCoordinator sets the session coordinator for session lifecycle management
func (h *SessionHandler) SetSessionCoordinator(sc *runner.SessionCoordinator) {
	h.sessionCoordinator = sc
}

// SetTerminalRouter sets the terminal router for terminal operations
func (h *SessionHandler) SetTerminalRouter(tr interface{}) {
	h.terminalRouter = tr
}

// SetRepositoryService sets the repository service for repository lookups
func (h *SessionHandler) SetRepositoryService(rs *repository.Service) {
	h.repositoryService = rs
}

// SetTicketService sets the ticket service for ticket lookups
func (h *SessionHandler) SetTicketService(ts *ticket.Service) {
	h.ticketService = ts
}

// SetGitProviderService sets the git provider service for git token lookups
func (h *SessionHandler) SetGitProviderService(gps *gitprovider.Service) {
	h.gitProviderService = gps
}

// SetSSHKeyService sets the SSH key service for SSH private key lookups
func (h *SessionHandler) SetSSHKeyService(sks *sshkey.Service) {
	h.sshKeyService = sks
}

// SetUserService sets the user service for user credential retrieval (权限跟人走)
func (h *SessionHandler) SetUserService(us *user.Service) {
	h.userService = us
}

// ListSessionsRequest represents session list request
type ListSessionsRequest struct {
	Status string `form:"status"`
	Limit  int    `form:"limit"`
	Offset int    `form:"offset"`
}

// ListSessions lists sessions
// GET /api/v1/organizations/:slug/sessions
func (h *SessionHandler) ListSessions(c *gin.Context) {
	var req ListSessionsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenant := middleware.GetTenant(c)

	limit := req.Limit
	if limit == 0 {
		limit = 20
	}

	sessions, total, err := h.sessionService.ListSessions(
		c.Request.Context(),
		tenant.OrganizationID,
		nil, // TeamID is deprecated
		req.Status,
		limit,
		req.Offset,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list sessions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sessions": sessions,
		"total":    total,
		"limit":    limit,
		"offset":   req.Offset,
	})
}

// CreateSessionRequest represents session creation request
type CreateSessionRequest struct {
	RunnerID          int64   `json:"runner_id" binding:"required"`
	AgentTypeID       *int64  `json:"agent_type_id"`
	CustomAgentTypeID *int64  `json:"custom_agent_type_id"`
	RepositoryID      *int64  `json:"repository_id"`
	RepositoryURL     *string `json:"repository_url"`      // Direct repository URL (takes precedence over repository_id)
	TicketID          *int64  `json:"ticket_id"`
	TicketIdentifier  *string `json:"ticket_identifier"`   // Direct ticket identifier (takes precedence over ticket_id)
	InitialPrompt     string  `json:"initial_prompt"`
	BranchName        *string `json:"branch_name"`
	PermissionMode    *string `json:"permission_mode"`     // "plan", "default", or "bypassPermissions"

	// PluginConfig allows advanced users to pass additional configuration to sandbox plugins
	// Fields: init_script, init_timeout, env_vars
	PluginConfig map[string]interface{} `json:"plugin_config"`
}

// CreateSession creates a new session
// POST /api/v1/organizations/:slug/sessions
func (h *SessionHandler) CreateSession(c *gin.Context) {
	var req CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenant := middleware.GetTenant(c)

	// Create session record in database
	sess, err := h.sessionService.CreateSession(c.Request.Context(), &session.CreateSessionRequest{
		OrganizationID:    tenant.OrganizationID,
		RunnerID:          req.RunnerID,
		AgentTypeID:       req.AgentTypeID,
		CustomAgentTypeID: req.CustomAgentTypeID,
		RepositoryID:      req.RepositoryID,
		TicketID:          req.TicketID,
		CreatedByID:       tenant.UserID,
		InitialPrompt:     req.InitialPrompt,
		BranchName:        req.BranchName,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}

	// Send create_session command to runner via SessionCoordinator
	if h.sessionCoordinator != nil {
		// Get permission mode from request or session settings (default to "plan")
		permissionMode := "plan"
		if req.PermissionMode != nil {
			permissionMode = *req.PermissionMode
		} else if sess.PermissionMode != nil {
			permissionMode = *sess.PermissionMode
		}

		// Build PluginConfig for Runner's Sandbox plugins
		pluginConfig := h.buildPluginConfig(c, &req)

		createReq := &runner.CreateSessionRequest{
			SessionID:      sess.SessionKey,
			InitialCommand: "claude", // Default command to run Claude Code CLI
			InitialPrompt:  req.InitialPrompt,
			PermissionMode: permissionMode,
			PluginConfig:   pluginConfig,
		}

		// Log the request
		log.Printf("[sessions] Sending create_session to runner %d for session %s with plugin_config: %v",
			req.RunnerID, sess.SessionKey, pluginConfig)

		if err := h.sessionCoordinator.CreateSession(c.Request.Context(), req.RunnerID, createReq); err != nil {
			// Log the error but don't fail - session is created, runner might be offline
			log.Printf("[sessions] Failed to send create_session: %v", err)
			c.JSON(http.StatusCreated, gin.H{
				"session": sess,
				"warning": "Session created but runner communication failed: " + err.Error(),
			})
			return
		}
		log.Printf("[sessions] create_session sent successfully for session %s", sess.SessionKey)
	} else {
		log.Printf("[sessions] SessionCoordinator is nil, cannot send create_session command")
	}

	c.JSON(http.StatusCreated, gin.H{"session": sess})
}

// GetSession returns session by key
// GET /api/v1/organizations/:slug/sessions/:key
func (h *SessionHandler) GetSession(c *gin.Context) {
	sessionKey := c.Param("key")

	sess, err := h.sessionService.GetSession(c.Request.Context(), sessionKey)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	tenant := middleware.GetTenant(c)
	if sess.OrganizationID != tenant.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// All organization members can access sessions (Team-based access control removed)
	c.JSON(http.StatusOK, gin.H{"session": sess})
}

// TerminateSession terminates a session
// POST /api/v1/organizations/:slug/sessions/:key/terminate
func (h *SessionHandler) TerminateSession(c *gin.Context) {
	sessionKey := c.Param("key")

	sess, err := h.sessionService.GetSession(c.Request.Context(), sessionKey)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	tenant := middleware.GetTenant(c)
	if sess.OrganizationID != tenant.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Only creator or admin can terminate
	if sess.CreatedByID != tenant.UserID && tenant.UserRole == "member" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only session creator or admin can terminate"})
		return
	}

	if err := h.sessionService.TerminateSession(c.Request.Context(), sessionKey); err != nil {
		if err == session.ErrSessionTerminated {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Session already terminated"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to terminate session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Session terminated"})
}

// GetSessionConnection returns connection info for session
// GET /api/v1/organizations/:slug/sessions/:key/connect
func (h *SessionHandler) GetSessionConnection(c *gin.Context) {
	sessionKey := c.Param("key")

	sess, err := h.sessionService.GetSession(c.Request.Context(), sessionKey)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	tenant := middleware.GetTenant(c)
	if sess.OrganizationID != tenant.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if !sess.IsActive() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session is not active"})
		return
	}

	// Return WebSocket connection URL
	c.JSON(http.StatusOK, gin.H{
		"session_key": sessionKey,
		"ws_url":      "/api/v1/ws/terminal/" + sessionKey,
		"status":      sess.Status,
	})
}

// ListSessionsByTicket lists sessions for a ticket
// GET /api/v1/organizations/:slug/tickets/:id/sessions
func (h *SessionHandler) ListSessionsByTicket(c *gin.Context) {
	ticketID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ticket ID"})
		return
	}

	sessions, err := h.sessionService.GetSessionsByTicket(c.Request.Context(), ticketID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list sessions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sessions": sessions})
}

// GetConnectionInfo returns connection info for session (alias for GetSessionConnection)
// GET /api/v1/organizations/:slug/sessions/:key/connect
func (h *SessionHandler) GetConnectionInfo(c *gin.Context) {
	h.GetSessionConnection(c)
}

// SendPromptRequest represents prompt sending request
type SendPromptRequest struct {
	Prompt string `json:"prompt" binding:"required"`
}

// SendPrompt sends a prompt to the session
// POST /api/v1/organizations/:slug/sessions/:key/send-prompt
func (h *SessionHandler) SendPrompt(c *gin.Context) {
	sessionKey := c.Param("key")

	var req SendPromptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sess, err := h.sessionService.GetSession(c.Request.Context(), sessionKey)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	tenant := middleware.GetTenant(c)
	if sess.OrganizationID != tenant.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if !sess.IsActive() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session is not active"})
		return
	}

	// TODO: Implement WebSocket-based prompt sending to runner
	// For now, return not implemented
	_ = req.Prompt
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Prompt sending via REST not implemented. Use WebSocket terminal."})
}

// TerminalRouterInterface defines the interface for terminal router operations
type TerminalRouterInterface interface {
	GetRecentOutput(sessionID string, lines int, raw bool) []byte
	GetScreenSnapshot(sessionID string) string
	GetCursorPosition(sessionID string) (row, col int)
	GetAllScrollbackData(sessionID string) []byte
	RouteInput(sessionID string, data []byte) error
	RouteResize(sessionID string, cols, rows int) error
}

// TerminalOutputResponse matches Runner's tools.TerminalOutput structure
type TerminalOutputResponse struct {
	SessionKey string `json:"session_key"`
	Output     string `json:"output"`
	Screen     string `json:"screen,omitempty"`
	CursorX    int    `json:"cursor_x"`
	CursorY    int    `json:"cursor_y"`
	TotalLines int    `json:"total_lines"`
	HasMore    bool   `json:"has_more"`
}

// ObserveTerminalRequest represents terminal observation request
type ObserveTerminalRequest struct {
	Lines         int  `form:"lines"`
	Raw           bool `form:"raw"`            // If true, return raw output; otherwise return processed output
	IncludeScreen bool `form:"include_screen"` // If true, include current screen snapshot
}

// ObserveTerminal returns recent terminal output for observation
// GET /api/v1/organizations/:slug/sessions/:key/terminal/observe
func (h *SessionHandler) ObserveTerminal(c *gin.Context) {
	sessionKey := c.Param("key")

	var req ObserveTerminalRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sess, err := h.sessionService.GetSession(c.Request.Context(), sessionKey)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	tenant := middleware.GetTenant(c)
	if sess.OrganizationID != tenant.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// All organization members can access sessions (Team-based access control removed)

	// Get terminal output from router if available
	if h.terminalRouter == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Terminal router not available"})
		return
	}

	tr, ok := h.terminalRouter.(TerminalRouterInterface)
	if !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Terminal router interface not implemented"})
		return
	}

	lines := req.Lines
	if lines <= 0 {
		lines = 100 // Default to last 100 lines
	}

	var output []byte
	if lines == -1 {
		// Get all raw scrollback data
		output = tr.GetAllScrollbackData(sessionKey)
	} else {
		// Get recent output (processed by default, raw if requested)
		output = tr.GetRecentOutput(sessionKey, lines, req.Raw)
	}

	// Get cursor position from virtual terminal
	cursorRow, cursorCol := tr.GetCursorPosition(sessionKey)

	// Calculate total lines (rough estimate from output)
	totalLines := 0
	for _, b := range output {
		if b == '\n' {
			totalLines++
		}
	}
	if len(output) > 0 && output[len(output)-1] != '\n' {
		totalLines++ // Count last line if not ending with newline
	}

	// Build response matching Runner's TerminalOutput structure
	response := TerminalOutputResponse{
		SessionKey: sessionKey,
		Output:     string(output),
		CursorX:    cursorCol,
		CursorY:    cursorRow,
		TotalLines: totalLines,
		HasMore:    lines != -1 && totalLines >= lines,
	}

	// Include screen snapshot if requested
	if req.IncludeScreen {
		response.Screen = tr.GetScreenSnapshot(sessionKey)
	}

	c.JSON(http.StatusOK, response)
}

// TerminalInputRequest represents terminal input request
type TerminalInputRequest struct {
	Input string `json:"input" binding:"required"`
}

// SendTerminalInput sends input to the terminal
// POST /api/v1/organizations/:slug/sessions/:key/terminal/input
func (h *SessionHandler) SendTerminalInput(c *gin.Context) {
	sessionKey := c.Param("key")

	var req TerminalInputRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sess, err := h.sessionService.GetSession(c.Request.Context(), sessionKey)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	tenant := middleware.GetTenant(c)
	if sess.OrganizationID != tenant.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if !sess.IsActive() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session is not active"})
		return
	}

	// All organization members can access sessions (Team-based access control removed)

	if h.terminalRouter == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Terminal router not available"})
		return
	}

	tr, ok := h.terminalRouter.(TerminalRouterInterface)
	if !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Terminal router interface not implemented"})
		return
	}

	if err := tr.RouteInput(sessionKey, []byte(req.Input)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send input: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Input sent"})
}

// TerminalResizeRequest represents terminal resize request
type TerminalResizeRequest struct {
	Cols int `json:"cols" binding:"required,min=1"`
	Rows int `json:"rows" binding:"required,min=1"`
}

// ResizeTerminal resizes the terminal
// POST /api/v1/organizations/:slug/sessions/:key/terminal/resize
func (h *SessionHandler) ResizeTerminal(c *gin.Context) {
	sessionKey := c.Param("key")

	var req TerminalResizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sess, err := h.sessionService.GetSession(c.Request.Context(), sessionKey)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	tenant := middleware.GetTenant(c)
	if sess.OrganizationID != tenant.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if !sess.IsActive() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session is not active"})
		return
	}

	if h.terminalRouter == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Terminal router not available"})
		return
	}

	tr, ok := h.terminalRouter.(TerminalRouterInterface)
	if !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Terminal router interface not implemented"})
		return
	}

	if err := tr.RouteResize(sessionKey, req.Cols, req.Rows); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resize terminal: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Terminal resized"})
}

// buildPluginConfig builds the PluginConfig map for Runner's Sandbox plugins
// It resolves repository_id -> repository_url, ticket_id -> ticket_identifier
//
// Credential Strategy (权限跟人走):
// - Repository is self-contained with provider_type and provider_base_url
// - Credentials are obtained from the current user's OAuth identity or Git connections
// - If no user credentials available, Runner uses its local Git configuration
func (h *SessionHandler) buildPluginConfig(c *gin.Context, req *CreateSessionRequest) map[string]interface{} {
	config := make(map[string]interface{})
	userID := middleware.GetUserID(c)

	// 1. Resolve Repository URL
	// Priority: repository_url > repository_id
	if req.RepositoryURL != nil && *req.RepositoryURL != "" {
		config["repository_url"] = *req.RepositoryURL
	} else if req.RepositoryID != nil && h.repositoryService != nil {
		repo, err := h.repositoryService.GetByID(c.Request.Context(), *req.RepositoryID)
		if err == nil && repo != nil {
			// Use repository's self-contained clone URL
			config["repository_url"] = repo.CloneURL

			// Store provider info for potential use by Runner
			config["provider_type"] = repo.ProviderType
			config["provider_base_url"] = repo.ProviderBaseURL

			// Get credentials from current user (权限跟人走)
			// Try OAuth identity first, then PAT connections
			if h.userService != nil {
				token := h.getUserGitToken(c, userID, repo.ProviderType, repo.ProviderBaseURL)
				if token != "" {
					config["git_token"] = token
				}
				// If no token found, Runner will use its local Git configuration
			}
		}
	}

	// 2. Resolve Branch Name
	if req.BranchName != nil && *req.BranchName != "" {
		config["branch"] = *req.BranchName
	} else if req.RepositoryID != nil && h.repositoryService != nil {
		// Use repository's default branch if not specified
		repo, err := h.repositoryService.GetByID(c.Request.Context(), *req.RepositoryID)
		if err == nil && repo != nil && repo.DefaultBranch != "" {
			config["branch"] = repo.DefaultBranch
		}
	}

	// 3. Resolve Ticket Identifier
	// Priority: ticket_identifier > ticket_id
	if req.TicketIdentifier != nil && *req.TicketIdentifier != "" {
		config["ticket_identifier"] = *req.TicketIdentifier
	} else if req.TicketID != nil && h.ticketService != nil {
		t, err := h.ticketService.GetTicket(c.Request.Context(), *req.TicketID)
		if err == nil && t != nil {
			config["ticket_identifier"] = t.Identifier
		}
	}

	// 4. Merge user-provided PluginConfig (can override above values)
	if req.PluginConfig != nil {
		for k, v := range req.PluginConfig {
			config[k] = v
		}
	}

	return config
}

// getUserGitToken retrieves the Git access token for the current user
// Implements "权限跟人走" - credentials follow the person, not the organization
//
// Priority:
// 1. OAuth identity matching provider type (for public providers like github.com, gitlab.com)
// 2. PAT connection matching provider type + base URL (for private GitLab instances)
//
// Returns empty string if no credentials found (Runner will use local Git config)
func (h *SessionHandler) getUserGitToken(c *gin.Context, userID int64, providerType, providerBaseURL string) string {
	ctx := c.Request.Context()

	// 1. Try OAuth identity first (for github.com, gitlab.com, gitee.com)
	// OAuth identities only exist for public providers
	if isPublicProvider(providerType, providerBaseURL) {
		tokens, err := h.userService.GetDecryptedTokens(ctx, userID, providerType)
		if err == nil && tokens.AccessToken != "" {
			return tokens.AccessToken
		}
	}

	// 2. Try PAT connections (for private GitLab or additional accounts)
	conn, err := h.userService.GetGitConnectionByProviderAndURL(ctx, userID, providerType, providerBaseURL)
	if err == nil && conn != nil {
		decryptedTokens, err := h.userService.GetDecryptedConnectionToken(ctx, userID, conn.ID)
		if err == nil && decryptedTokens.AccessToken != "" {
			return decryptedTokens.AccessToken
		}
	}

	// No credentials found - Runner will use its local Git configuration
	return ""
}

// isPublicProvider checks if the provider is a public provider (github.com, gitlab.com, gitee.com)
func isPublicProvider(providerType, providerBaseURL string) bool {
	switch providerType {
	case "github":
		return providerBaseURL == "https://github.com" || providerBaseURL == "https://api.github.com"
	case "gitlab":
		return providerBaseURL == "https://gitlab.com"
	case "gitee":
		return providerBaseURL == "https://gitee.com"
	default:
		return false
	}
}
