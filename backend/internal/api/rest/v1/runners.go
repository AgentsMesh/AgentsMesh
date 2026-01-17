package v1

import (
	"net/http"
	"strconv"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/organization"
	runner "github.com/anthropics/agentsmesh/backend/internal/service/runner"
	"github.com/gin-gonic/gin"
)

// RunnerHandler handles runner-related requests
type RunnerHandler struct {
	runnerService *runner.Service
	orgService    organization.Interface
}

// NewRunnerHandler creates a new runner handler
func NewRunnerHandler(runnerService *runner.Service) *RunnerHandler {
	return &RunnerHandler{
		runnerService: runnerService,
	}
}

// SetOrgService sets the organization service for looking up org slug during registration
func (h *RunnerHandler) SetOrgService(orgService organization.Interface) {
	h.orgService = orgService
}

// ListRunners lists runners in organization
// GET /api/v1/organizations/:slug/runners
func (h *RunnerHandler) ListRunners(c *gin.Context) {
	tenant := middleware.GetTenant(c)

	runners, err := h.runnerService.ListRunners(c.Request.Context(), tenant.OrganizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list runners"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"runners": runners})
}

// GetRunner returns runner by ID
// GET /api/v1/organizations/:slug/runners/:id
func (h *RunnerHandler) GetRunner(c *gin.Context) {
	runnerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid runner ID"})
		return
	}

	r, err := h.runnerService.GetRunner(c.Request.Context(), runnerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Runner not found"})
		return
	}

	tenant := middleware.GetTenant(c)
	if r.OrganizationID != tenant.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"runner": r})
}

// DeleteRunner deletes a runner
// DELETE /api/v1/organizations/:slug/runners/:id
func (h *RunnerHandler) DeleteRunner(c *gin.Context) {
	runnerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid runner ID"})
		return
	}

	tenant := middleware.GetTenant(c)

	// Check admin permission
	if tenant.UserRole != "owner" && tenant.UserRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin permission required"})
		return
	}

	r, err := h.runnerService.GetRunner(c.Request.Context(), runnerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Runner not found"})
		return
	}

	if r.OrganizationID != tenant.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if err := h.runnerService.DeleteRunner(c.Request.Context(), runnerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete runner"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Runner deleted"})
}

// CreateTokenRequest represents registration token creation request
type CreateTokenRequest struct {
	Description string     `json:"description"`
	MaxUses     *int       `json:"max_uses"`
	ExpiresAt   *time.Time `json:"expires_at"`
}

// CreateRegistrationToken creates a new registration token
// POST /api/v1/organizations/:slug/runners/tokens
func (h *RunnerHandler) CreateRegistrationToken(c *gin.Context) {
	var req CreateTokenRequest
	// Allow empty body - all fields are optional
	_ = c.ShouldBindJSON(&req)

	tenant := middleware.GetTenant(c)

	// Check admin permission
	if tenant.UserRole != "owner" && tenant.UserRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin permission required"})
		return
	}

	token, err := h.runnerService.CreateRegistrationToken(
		c.Request.Context(),
		tenant.OrganizationID,
		tenant.UserID,
		req.Description,
		req.MaxUses,
		req.ExpiresAt,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create token"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"token":   token,
		"message": "Save this token securely. It will only be shown once.",
	})
}

// RegisterRunnerRequest represents runner registration request
type RegisterRunnerRequest struct {
	RegistrationToken string `json:"registration_token" binding:"required"`
	NodeID            string `json:"node_id" binding:"required"`
	Description       string `json:"description"`
	MaxConcurrentPods int    `json:"max_concurrent_pods"`
}

// RegisterRunner registers a new runner
// POST /api/v1/runners/register
// Response includes org_slug for the runner to use in subsequent API calls
func (h *RunnerHandler) RegisterRunner(c *gin.Context) {
	var req RegisterRunnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	maxPods := req.MaxConcurrentPods
	if maxPods == 0 {
		maxPods = 5
	}

	r, err := h.runnerService.RegisterRunner(
		c.Request.Context(),
		req.RegistrationToken,
		req.NodeID,
		req.Description,
		maxPods,
	)
	if err != nil {
		switch err {
		case runner.ErrInvalidToken:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid registration token"})
		case runner.ErrTokenExpired:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Registration token expired"})
		case runner.ErrTokenExhausted:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Registration token usage exhausted"})
		case runner.ErrRunnerAlreadyExists:
			c.JSON(http.StatusConflict, gin.H{"error": "Runner with this node_id already exists"})
		case runner.ErrRunnerQuotaExceeded:
			c.JSON(http.StatusPaymentRequired, gin.H{
				"error": "Runner quota exceeded. Please upgrade your plan to register more runners.",
				"code":  "RUNNER_QUOTA_EXCEEDED",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register runner"})
		}
		return
	}

	// Lookup org slug for the runner to use in org-scoped API paths
	var orgSlug string
	if h.orgService != nil {
		org, err := h.orgService.GetByID(c.Request.Context(), r.OrganizationID)
		if err == nil {
			orgSlug = org.Slug
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"runner_id": r.ID,
		"org_slug":  orgSlug,
		"message":   "Runner registered. Use gRPC/mTLS registration to obtain certificates for secure communication.",
	})
}

// UpdateRunnerRequest represents runner update request
type UpdateRunnerRequest struct {
	Description       *string `json:"description"`
	MaxConcurrentPods *int    `json:"max_concurrent_pods"`
	IsEnabled         *bool   `json:"is_enabled"`
}

// UpdateRunner updates a runner
// PUT /api/v1/organizations/:slug/runners/:id
func (h *RunnerHandler) UpdateRunner(c *gin.Context) {
	runnerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid runner ID"})
		return
	}

	var req UpdateRunnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenant := middleware.GetTenant(c)

	// Check admin permission
	if tenant.UserRole != "owner" && tenant.UserRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin permission required"})
		return
	}

	// Verify runner belongs to organization
	r, err := h.runnerService.GetRunner(c.Request.Context(), runnerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Runner not found"})
		return
	}

	if r.OrganizationID != tenant.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Update runner
	updated, err := h.runnerService.UpdateRunner(c.Request.Context(), runnerID, runner.RunnerUpdateInput{
		Description:       req.Description,
		MaxConcurrentPods: req.MaxConcurrentPods,
		IsEnabled:         req.IsEnabled,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update runner"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"runner": updated})
}


// ListAvailableRunners lists available runners for pods
// GET /api/v1/organizations/:slug/runners/available
func (h *RunnerHandler) ListAvailableRunners(c *gin.Context) {
	tenant := middleware.GetTenant(c)

	runners, err := h.runnerService.ListAvailableRunners(c.Request.Context(), tenant.OrganizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list runners"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"runners": runners})
}

// ListRegistrationTokens lists registration tokens
// GET /api/v1/organizations/:slug/runners/tokens
func (h *RunnerHandler) ListRegistrationTokens(c *gin.Context) {
	tenant := middleware.GetTenant(c)

	// Check admin permission
	if tenant.UserRole != "owner" && tenant.UserRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin permission required"})
		return
	}

	tokens, err := h.runnerService.ListRegistrationTokens(c.Request.Context(), tenant.OrganizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list tokens"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tokens": tokens})
}

// RevokeRegistrationToken revokes a registration token
// DELETE /api/v1/organizations/:slug/runners/tokens/:id
func (h *RunnerHandler) RevokeRegistrationToken(c *gin.Context) {
	tokenID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token ID"})
		return
	}

	tenant := middleware.GetTenant(c)

	// Check admin permission
	if tenant.UserRole != "owner" && tenant.UserRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin permission required"})
		return
	}

	if err := h.runnerService.RevokeRegistrationToken(c.Request.Context(), tokenID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Token revoked"})
}

