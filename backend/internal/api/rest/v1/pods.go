package v1

import (
	"context"
	"log"
	"net/http"
	"strconv"

	agentDomain "github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/agent"
	"github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	"github.com/gin-gonic/gin"
)

// PodHandler handles pod-related requests
// Uses interfaces for service dependencies to enable dependency inversion and easier testing
type PodHandler struct {
	podService        PodServiceForHandler        // Pod CRUD operations
	runnerService     *runner.Service             // Runner management (not abstracted - rarely mocked)
	agentService      AgentServiceForHandler      // Agent type and credentials
	billingService    BillingServiceForHandler    // Quota checking
	repositoryService RepositoryServiceForHandler // Repository lookup
	ticketService     TicketServiceForHandler     // Ticket lookup
	userService       UserServiceForPod           // User credential retrieval (权限跟人走)
	runnerConnMgr     *runner.RunnerConnectionManager   // Runner gRPC connections (not abstracted)
	podCoordinator    *runner.PodCoordinator      // Pod coordination (not abstracted)
	terminalRouter    interface{}                 // *runner.TerminalRouter, optional
	configBuilder     *agent.ConfigBuilder        // New protocol: builds pod config from agent type templates
}

// PodHandlerOption is a functional option for configuring PodHandler
type PodHandlerOption func(*PodHandler)

// WithRunnerConnectionManager sets the runner connection manager
func WithRunnerConnectionManager(cm *runner.RunnerConnectionManager) PodHandlerOption {
	return func(h *PodHandler) {
		h.runnerConnMgr = cm
	}
}

// WithPodCoordinator sets the pod coordinator
func WithPodCoordinator(pc *runner.PodCoordinator) PodHandlerOption {
	return func(h *PodHandler) {
		h.podCoordinator = pc
	}
}

// WithTerminalRouter sets the terminal router
func WithTerminalRouter(tr interface{}) PodHandlerOption {
	return func(h *PodHandler) {
		h.terminalRouter = tr
	}
}

// WithRepositoryService sets the repository service
func WithRepositoryService(rs RepositoryServiceForHandler) PodHandlerOption {
	return func(h *PodHandler) {
		h.repositoryService = rs
	}
}

// WithTicketService sets the ticket service
func WithTicketService(ts TicketServiceForHandler) PodHandlerOption {
	return func(h *PodHandler) {
		h.ticketService = ts
	}
}

// WithUserService sets the user service for credential retrieval (权限跟人走)
func WithUserService(us UserServiceForPod) PodHandlerOption {
	return func(h *PodHandler) {
		h.userService = us
	}
}

// WithBillingService sets the billing service for quota checking
func WithBillingService(bs BillingServiceForHandler) PodHandlerOption {
	return func(h *PodHandler) {
		h.billingService = bs
	}
}

// compositeAgentProvider implements agent.AgentConfigProvider by combining three sub-services
// This allows PodHandler to work with the split service architecture
type compositeAgentProvider struct {
	agentTypeSvc  *agent.AgentTypeService
	credentialSvc *agent.CredentialProfileService
	userConfigSvc *agent.UserConfigService
}

func (p *compositeAgentProvider) GetAgentType(ctx context.Context, id int64) (*agentDomain.AgentType, error) {
	return p.agentTypeSvc.GetAgentType(ctx, id)
}

func (p *compositeAgentProvider) GetUserEffectiveConfig(ctx context.Context, userID, agentTypeID int64, overrides agentDomain.ConfigValues) agentDomain.ConfigValues {
	return p.userConfigSvc.GetUserEffectiveConfig(ctx, userID, agentTypeID, overrides)
}

func (p *compositeAgentProvider) GetEffectiveCredentialsForPod(ctx context.Context, userID, agentTypeID int64, profileID *int64) (agentDomain.EncryptedCredentials, bool, error) {
	return p.credentialSvc.GetEffectiveCredentialsForPod(ctx, userID, agentTypeID, profileID)
}

// NewPodHandler creates a new pod handler with required dependencies and optional configurations
func NewPodHandler(
	podService *agentpod.PodService,
	runnerService *runner.Service,
	agentTypeSvc *agent.AgentTypeService,
	credentialSvc *agent.CredentialProfileService,
	userConfigSvc *agent.UserConfigService,
	opts ...PodHandlerOption,
) *PodHandler {
	// Create composite provider for ConfigBuilder
	provider := &compositeAgentProvider{
		agentTypeSvc:  agentTypeSvc,
		credentialSvc: credentialSvc,
		userConfigSvc: userConfigSvc,
	}

	h := &PodHandler{
		podService:    podService,
		runnerService: runnerService,
		agentService:  provider,
		configBuilder: agent.NewConfigBuilder(provider),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// WithPodService sets the pod service (for testing with mock implementations)
func WithPodService(ps PodServiceForHandler) PodHandlerOption {
	return func(h *PodHandler) {
		h.podService = ps
	}
}

// WithAgentService sets the agent service (for testing with mock implementations)
func WithAgentService(as AgentServiceForHandler) PodHandlerOption {
	return func(h *PodHandler) {
		h.agentService = as
	}
}


// ListPodsRequest represents pod list request
type ListPodsRequest struct {
	Status string `form:"status"`
	Limit  int    `form:"limit"`
	Offset int    `form:"offset"`
}

// ListPods lists pods
// GET /api/v1/organizations/:slug/pods
func (h *PodHandler) ListPods(c *gin.Context) {
	var req ListPodsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenant := middleware.GetTenant(c)

	limit := req.Limit
	if limit == 0 {
		limit = 20
	}

	pods, total, err := h.podService.ListPods(
		c.Request.Context(),
		tenant.OrganizationID,
		nil, // TeamID is deprecated
		req.Status,
		limit,
		req.Offset,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list pods"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pods":   pods,
		"total":  total,
		"limit":  limit,
		"offset": req.Offset,
	})
}

// CreatePodRequest represents pod creation request
type CreatePodRequest struct {
	RunnerID          int64   `json:"runner_id" binding:"required"`
	AgentTypeID       *int64  `json:"agent_type_id" binding:"required"` // Required for new protocol
	CustomAgentTypeID *int64  `json:"custom_agent_type_id"`
	RepositoryID      *int64  `json:"repository_id"`
	RepositoryURL     *string `json:"repository_url"`      // Direct repository URL (takes precedence over repository_id)
	TicketID          *int64  `json:"ticket_id"`
	TicketIdentifier  *string `json:"ticket_identifier"`   // Direct ticket identifier (takes precedence over ticket_id)
	InitialPrompt     string  `json:"initial_prompt"`
	BranchName        *string `json:"branch_name"`
	PermissionMode    *string `json:"permission_mode"`     // "plan", "default", or "bypassPermissions"

	// CredentialProfileID specifies which credential profile to use
	// - nil or 0: RunnerHost mode (use Runner's local environment, no credentials injected)
	// - >0: Use specified credential profile ID
	CredentialProfileID *int64 `json:"credential_profile_id"`

	// ConfigOverrides allows users to override agent type default configuration
	ConfigOverrides map[string]interface{} `json:"config_overrides"`
}

// CreatePod creates a new pod
// POST /api/v1/organizations/:slug/pods
func (h *PodHandler) CreatePod(c *gin.Context) {
	var req CreatePodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenant := middleware.GetTenant(c)

	// Check concurrent pod quota before creation
	if h.billingService != nil {
		if err := h.billingService.CheckQuota(c.Request.Context(), tenant.OrganizationID, "concurrent_pods", 1); err != nil {
			if err == ErrQuotaExceeded {
				c.JSON(http.StatusPaymentRequired, gin.H{
					"error": "Concurrent pod quota exceeded. Please upgrade your plan or terminate existing pods.",
					"code":  "CONCURRENT_POD_QUOTA_EXCEEDED",
				})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check quota"})
			return
		}
	}

	// Create pod record in database
	pod, err := h.podService.CreatePod(c.Request.Context(), &agentpod.CreatePodRequest{
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create pod"})
		return
	}

	// Send create_pod command to runner via PodCoordinator
	if h.podCoordinator != nil {
		// Get permission mode from request or pod settings (default to "plan")
		permissionMode := "plan"
		if req.PermissionMode != nil {
			permissionMode = *req.PermissionMode
		} else if pod.PermissionMode != nil {
			permissionMode = *pod.PermissionMode
		}

		// Build pod config using ConfigBuilder (new protocol only)
		podConfig, err := h.buildPodConfigWithNewProtocol(c, &req, pod.PodKey, permissionMode)
		if err != nil {
			log.Printf("[pods] Failed to build pod config: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to build pod configuration: " + err.Error(),
				"code":  "POD_CONFIG_BUILD_FAILED",
			})
			return
		}

		createReq := h.convertPodConfigToRequest(podConfig, pod.PodKey)
		log.Printf("[pods] Sending create_pod to runner %d for pod %s", req.RunnerID, pod.PodKey)

		if err := h.podCoordinator.CreatePod(c.Request.Context(), req.RunnerID, createReq); err != nil {
			// Log the error but don't fail - pod is created, runner might be offline
			log.Printf("[pods] Failed to send create_pod: %v", err)
			c.JSON(http.StatusCreated, gin.H{
				"pod":     pod,
				"warning": "Pod created but runner communication failed: " + err.Error(),
			})
			return
		}
		log.Printf("[pods] create_pod sent successfully for pod %s", pod.PodKey)
	} else {
		log.Printf("[pods] PodCoordinator is nil, cannot send create_pod command")
	}

	c.JSON(http.StatusCreated, gin.H{"pod": pod})
}

// GetPod returns pod by key
// GET /api/v1/organizations/:slug/pods/:key
func (h *PodHandler) GetPod(c *gin.Context) {
	podKey := c.Param("key")

	pod, err := h.podService.GetPod(c.Request.Context(), podKey)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pod not found"})
		return
	}

	tenant := middleware.GetTenant(c)
	if pod.OrganizationID != tenant.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// All organization members can access pods (Team-based access control removed)
	c.JSON(http.StatusOK, gin.H{"pod": pod})
}

// TerminatePod terminates a pod
// POST /api/v1/organizations/:slug/pods/:key/terminate
func (h *PodHandler) TerminatePod(c *gin.Context) {
	podKey := c.Param("key")

	pod, err := h.podService.GetPod(c.Request.Context(), podKey)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pod not found"})
		return
	}

	tenant := middleware.GetTenant(c)
	if pod.OrganizationID != tenant.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Only creator or admin can terminate
	if pod.CreatedByID != tenant.UserID && tenant.UserRole == "member" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only pod creator or admin can terminate"})
		return
	}

	if err := h.podService.TerminatePod(c.Request.Context(), podKey); err != nil {
		if err == ErrPodTerminated {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Pod already terminated"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to terminate pod"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Pod terminated"})
}

// GetPodConnection returns connection info for pod
// GET /api/v1/organizations/:slug/pods/:key/connect
func (h *PodHandler) GetPodConnection(c *gin.Context) {
	podKey := c.Param("key")

	pod, err := h.podService.GetPod(c.Request.Context(), podKey)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pod not found"})
		return
	}

	tenant := middleware.GetTenant(c)
	if pod.OrganizationID != tenant.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if !pod.IsActive() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pod is not active"})
		return
	}

	// Return WebSocket connection URL
	c.JSON(http.StatusOK, gin.H{
		"pod_key": podKey,
		"ws_url":  "/api/v1/ws/terminal/" + podKey,
		"status":  pod.Status,
	})
}

// ListPodsByTicket lists pods for a ticket
// GET /api/v1/organizations/:slug/tickets/:id/pods
func (h *PodHandler) ListPodsByTicket(c *gin.Context) {
	ticketID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ticket ID"})
		return
	}

	pods, err := h.podService.GetPodsByTicket(c.Request.Context(), ticketID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list pods"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"pods": pods})
}

// GetConnectionInfo returns connection info for pod (alias for GetPodConnection)
// GET /api/v1/organizations/:slug/pods/:key/connect
func (h *PodHandler) GetConnectionInfo(c *gin.Context) {
	h.GetPodConnection(c)
}

// SendPromptRequest represents prompt sending request
type SendPromptRequest struct {
	Prompt string `json:"prompt" binding:"required"`
}

// SendPrompt sends a prompt to the pod
// POST /api/v1/organizations/:slug/pods/:key/send-prompt
func (h *PodHandler) SendPrompt(c *gin.Context) {
	podKey := c.Param("key")

	var req SendPromptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pod, err := h.podService.GetPod(c.Request.Context(), podKey)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pod not found"})
		return
	}

	tenant := middleware.GetTenant(c)
	if pod.OrganizationID != tenant.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if !pod.IsActive() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pod is not active"})
		return
	}

	// TODO: Implement prompt sending via gRPC to runner
	// For now, return not implemented
	_ = req.Prompt
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Prompt sending via REST not implemented. Use terminal WebSocket."})
}

// NOTE: Terminal-related handlers (ObserveTerminal, SendTerminalInput, ResizeTerminal)
// have been moved to pod_terminal.go for better code organization

// NOTE: buildPodConfigWithNewProtocol and related config resolution functions
// have been moved to pod_config.go for better code organization

// NOTE: mapCredentialsToEnvVars, getUserGitCredential, getUserGitToken, isPublicProvider
// have been moved to pod_credential.go for better code organization
