package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/infra/pki"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	"github.com/gin-gonic/gin"
)

// GRPCRunnerHandler handles gRPC/mTLS Runner registration and management.
type GRPCRunnerHandler struct {
	runnerService *runner.Service
	pkiService    *pki.Service
	config        *config.Config
}

// NewGRPCRunnerHandler creates a new gRPC runner handler.
func NewGRPCRunnerHandler(runnerService *runner.Service, pkiService *pki.Service, cfg *config.Config) *GRPCRunnerHandler {
	return &GRPCRunnerHandler{
		runnerService: runnerService,
		pkiService:    pkiService,
		config:        cfg,
	}
}

// ==================== Interactive Registration (Tailscale-style) ====================

// RequestAuthURLRequest represents request for authorization URL.
type RequestAuthURLRequest struct {
	MachineKey string            `json:"machine_key" binding:"required"`
	NodeID     string            `json:"node_id"`
	Labels     map[string]string `json:"labels"`
}

// RequestAuthURL creates a pending auth request and returns an authorization URL.
// POST /api/v1/runners/grpc/auth-url
// No authentication required - Runner initiates registration.
func (h *GRPCRunnerHandler) RequestAuthURL(c *gin.Context) {
	var req RequestAuthURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	frontendURL := h.config.FrontendURL
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}

	resp, err := h.runnerService.RequestAuthURL(c.Request.Context(), &runner.RequestAuthURLRequest{
		MachineKey: req.MachineKey,
		NodeID:     req.NodeID,
		Labels:     req.Labels,
	}, frontendURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create auth request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"auth_url":   resp.AuthURL,
		"auth_key":   resp.AuthKey,
		"expires_in": resp.ExpiresIn,
	})
}

// GetAuthStatus returns the current status of a pending authorization.
// GET /api/v1/runners/grpc/auth-status?key=xxx
// No authentication required - Runner polls for completion.
func (h *GRPCRunnerHandler) GetAuthStatus(c *gin.Context) {
	authKey := c.Query("key")
	if authKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing auth key"})
		return
	}

	resp, err := h.runnerService.GetAuthStatus(c.Request.Context(), authKey, h.pkiService)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// AuthorizeRunnerRequest represents authorization request from Web UI.
type AuthorizeRunnerRequest struct {
	AuthKey string `json:"auth_key" binding:"required"`
	NodeID  string `json:"node_id"`
}

// AuthorizeRunner authorizes a pending auth request.
// POST /api/v1/organizations/:slug/runners/grpc/authorize
// Requires JWT authentication.
func (h *GRPCRunnerHandler) AuthorizeRunner(c *gin.Context) {
	var req AuthorizeRunnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenant := middleware.GetTenant(c)
	if tenant == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Check admin permission
	if tenant.UserRole != "owner" && tenant.UserRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin permission required"})
		return
	}

	r, err := h.runnerService.AuthorizeRunner(c.Request.Context(), req.AuthKey, tenant.OrganizationID, req.NodeID)
	if err != nil {
		switch err {
		case runner.ErrRunnerAlreadyExists:
			c.JSON(http.StatusConflict, gin.H{"error": "Runner with this node_id already exists"})
		case runner.ErrRunnerQuotaExceeded:
			c.JSON(http.StatusPaymentRequired, gin.H{
				"error": "Runner quota exceeded",
				"code":  "RUNNER_QUOTA_EXCEEDED",
			})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"runner_id": r.ID,
		"node_id":   r.NodeID,
		"message":   "Runner authorized successfully",
	})
}

// ==================== Pre-generated Token Registration ====================

// GenerateGRPCTokenRequest represents request to generate registration token.
type GenerateGRPCTokenRequest struct {
	Name      string            `json:"name"`
	Labels    map[string]string `json:"labels"`
	SingleUse bool              `json:"single_use"`
	MaxUses   int               `json:"max_uses"`
	ExpiresIn int               `json:"expires_in"` // seconds
}

// GenerateGRPCToken creates a new pre-generated registration token.
// POST /api/v1/organizations/:slug/runners/grpc/tokens
// Requires JWT authentication (admin).
func (h *GRPCRunnerHandler) GenerateGRPCToken(c *gin.Context) {
	var req GenerateGRPCTokenRequest
	// Allow empty body - all fields are optional
	_ = c.ShouldBindJSON(&req)

	tenant := middleware.GetTenant(c)
	if tenant == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Check admin permission
	if tenant.UserRole != "owner" && tenant.UserRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin permission required"})
		return
	}

	serverURL := h.config.ServerURL
	if serverURL == "" {
		serverURL = "http://localhost"
	}

	resp, err := h.runnerService.GenerateGRPCRegistrationToken(
		c.Request.Context(),
		tenant.OrganizationID,
		tenant.UserID,
		&runner.GenerateGRPCRegistrationTokenRequest{
			Name:      req.Name,
			Labels:    req.Labels,
			SingleUse: req.SingleUse,
			MaxUses:   req.MaxUses,
			ExpiresIn: req.ExpiresIn,
		},
		serverURL,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"token":      resp.Token,
		"expires_at": resp.ExpiresAt,
		"command":    resp.Command,
		"message":    "Save this token securely. It will only be shown once.",
	})
}

// ListGRPCTokens lists all gRPC registration tokens for an organization.
// GET /api/v1/organizations/:slug/runners/grpc/tokens
// Requires JWT authentication (admin).
func (h *GRPCRunnerHandler) ListGRPCTokens(c *gin.Context) {
	tenant := middleware.GetTenant(c)
	if tenant == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Check admin permission
	if tenant.UserRole != "owner" && tenant.UserRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin permission required"})
		return
	}

	tokens, err := h.runnerService.ListGRPCRegistrationTokens(c.Request.Context(), tenant.OrganizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list tokens"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tokens": tokens})
}

// DeleteGRPCToken deletes a gRPC registration token.
// DELETE /api/v1/organizations/:slug/runners/grpc/tokens/:id
// Requires JWT authentication (admin).
func (h *GRPCRunnerHandler) DeleteGRPCToken(c *gin.Context) {
	tokenID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token ID"})
		return
	}

	tenant := middleware.GetTenant(c)
	if tenant == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Check admin permission
	if tenant.UserRole != "owner" && tenant.UserRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin permission required"})
		return
	}

	if err := h.runnerService.DeleteGRPCRegistrationToken(c.Request.Context(), tokenID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Token deleted"})
}

// RegisterWithTokenRequest represents request to register with pre-generated token.
type RegisterWithTokenRequest struct {
	Token  string `json:"token" binding:"required"`
	NodeID string `json:"node_id"`
}

// RegisterWithToken registers a new runner using a pre-generated token.
// POST /api/v1/runners/grpc/register
// No authentication required - token serves as authentication.
func (h *GRPCRunnerHandler) RegisterWithToken(c *gin.Context) {
	var req RegisterWithTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.runnerService.RegisterWithToken(
		c.Request.Context(),
		&runner.RegisterWithTokenRequest{
			Token:  req.Token,
			NodeID: req.NodeID,
		},
		h.pkiService,
	)
	if err != nil {
		switch err {
		case runner.ErrInvalidToken:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		case runner.ErrTokenExpired:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token expired"})
		case runner.ErrTokenExhausted:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token usage exhausted"})
		case runner.ErrRunnerAlreadyExists:
			c.JSON(http.StatusConflict, gin.H{"error": "Runner with this node_id already exists"})
		case runner.ErrRunnerQuotaExceeded:
			c.JSON(http.StatusPaymentRequired, gin.H{
				"error": "Runner quota exceeded",
				"code":  "RUNNER_QUOTA_EXCEEDED",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register runner"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"runner_id":      resp.RunnerID,
		"certificate":    resp.Certificate,
		"private_key":    resp.PrivateKey,
		"ca_certificate": resp.CACertificate,
		"org_slug":       resp.OrgSlug,
	})
}

// ==================== Certificate Renewal ====================

// RenewCertificate renews a runner's certificate.
// POST /api/v1/runners/grpc/renew-certificate
// Authenticated via mTLS - Nginx verifies client certificate and passes CN.
func (h *GRPCRunnerHandler) RenewCertificate(c *gin.Context) {
	// Get identity from Nginx-passed headers
	nodeID := c.GetHeader("X-Client-Cert-CN")
	oldSerial := c.GetHeader("X-Client-Cert-Serial")

	if nodeID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing client certificate"})
		return
	}

	resp, err := h.runnerService.RenewCertificate(c.Request.Context(), nodeID, oldSerial, h.pkiService)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"certificate": resp.Certificate,
		"private_key": resp.PrivateKey,
		"expires_at":  resp.ExpiresAt,
	})
}

// ==================== Reactivation (Expired Certificate Recovery) ====================

// GenerateReactivationToken generates a one-time token for reactivating a runner.
// POST /api/v1/organizations/:slug/runners/:id/reactivate
// Requires JWT authentication (admin).
func (h *GRPCRunnerHandler) GenerateReactivationToken(c *gin.Context) {
	runnerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid runner ID"})
		return
	}

	tenant := middleware.GetTenant(c)
	if tenant == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

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

	resp, err := h.runnerService.GenerateReactivationToken(c.Request.Context(), runnerID, tenant.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate reactivation token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"reactivation_token": resp.Token,
		"expires_in":         resp.ExpiresIn,
		"command":            resp.Command,
	})
}

// ReactivateRequest represents request to reactivate a runner.
type ReactivateRequest struct {
	Token string `json:"token" binding:"required"`
}

// Reactivate reactivates a runner using a one-time token.
// POST /api/v1/runners/grpc/reactivate
// No authentication required - token serves as authentication.
func (h *GRPCRunnerHandler) Reactivate(c *gin.Context) {
	var req ReactivateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.runnerService.Reactivate(
		c.Request.Context(),
		&runner.ReactivateRequest{Token: req.Token},
		h.pkiService,
	)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"certificate":    resp.Certificate,
		"private_key":    resp.PrivateKey,
		"ca_certificate": resp.CACertificate,
	})
}

// ==================== Route Registration ====================

// RegisterGRPCRunnerRoutes registers gRPC runner routes.
func RegisterGRPCRunnerRoutes(r *gin.RouterGroup, handler *GRPCRunnerHandler) {
	// Public endpoints (no auth required)
	// These are used by Runner CLI for registration
	grpcPublic := r.Group("/runners/grpc")
	{
		// Tailscale-style interactive registration
		grpcPublic.POST("/auth-url", handler.RequestAuthURL)
		grpcPublic.GET("/auth-status", handler.GetAuthStatus)

		// Pre-generated token registration
		grpcPublic.POST("/register", handler.RegisterWithToken)

		// Reactivation (for expired certificates)
		grpcPublic.POST("/reactivate", handler.Reactivate)

		// Certificate renewal (authenticated via mTLS, X-Client-Cert-* headers)
		grpcPublic.POST("/renew-certificate", handler.RenewCertificate)
	}
}

// RegisterOrgGRPCRunnerRoutes registers organization-scoped gRPC runner routes.
// These require JWT authentication.
func RegisterOrgGRPCRunnerRoutes(rg *gin.RouterGroup, handler *GRPCRunnerHandler) {
	// Organization-scoped endpoints (require JWT auth + tenant context)
	grpc := rg.Group("/grpc")
	{
		// Authorize pending auth (Web UI action)
		grpc.POST("/authorize", handler.AuthorizeRunner)

		// Token management
		grpc.GET("/tokens", handler.ListGRPCTokens)
		grpc.POST("/tokens", handler.GenerateGRPCToken)
		grpc.DELETE("/tokens/:id", handler.DeleteGRPCToken)
	}

	// Reactivation token generation (per-runner)
	rg.POST("/:id/reactivate", handler.GenerateReactivationToken)
}
