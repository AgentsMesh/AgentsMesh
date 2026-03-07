package v1

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/infra/pki"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
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

// ==================== Certificate Renewal ====================

// RenewCertificate renews a runner's certificate.
// POST /api/v1/runners/grpc/renew-certificate
// Authenticated via mTLS - Nginx verifies client certificate and passes CN.
func (h *GRPCRunnerHandler) RenewCertificate(c *gin.Context) {
	// Get identity from Nginx-passed headers
	nodeID := c.GetHeader("X-Client-Cert-CN")
	oldSerial := c.GetHeader("X-Client-Cert-Serial")

	if nodeID == "" {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "Missing client certificate")
		return
	}

	resp, err := h.runnerService.RenewCertificate(c.Request.Context(), nodeID, oldSerial, h.pkiService)
	if err != nil {
		switch {
		case errors.Is(err, runner.ErrRunnerNotFound):
			apierr.ResourceNotFound(c, "Runner not found")
		case errors.Is(err, runner.ErrCertificateMismatch):
			apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "Certificate mismatch")
		default:
			apierr.InternalError(c, "Certificate renewal failed")
		}
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
		apierr.InvalidInput(c, "Invalid runner ID")
		return
	}

	tenant := middleware.GetTenant(c)
	if tenant == nil {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "Unauthorized")
		return
	}

	// Check admin permission
	if tenant.UserRole != "owner" && tenant.UserRole != "admin" {
		apierr.ForbiddenAdmin(c)
		return
	}

	// Verify runner belongs to organization
	r, err := h.runnerService.GetRunner(c.Request.Context(), runnerID)
	if err != nil {
		apierr.ResourceNotFound(c, "Runner not found")
		return
	}

	if r.OrganizationID != tenant.OrganizationID {
		apierr.ForbiddenAccess(c)
		return
	}

	resp, err := h.runnerService.GenerateReactivationToken(c.Request.Context(), runnerID, tenant.UserID)
	if err != nil {
		apierr.InternalError(c, "Failed to generate reactivation token")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"reactivation_token": resp.Token,
		"expires_in":         resp.ExpiresIn,
		"command":            resp.Command,
	})
}

// Reactivate reactivates a runner using a one-time token.
// POST /api/v1/runners/grpc/reactivate
// No authentication required - token serves as authentication.
func (h *GRPCRunnerHandler) Reactivate(c *gin.Context) {
	var req ReactivateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	resp, err := h.runnerService.Reactivate(
		c.Request.Context(),
		&runner.ReactivateRequest{Token: req.Token},
		h.pkiService,
	)
	if err != nil {
		switch {
		case errors.Is(err, runner.ErrInvalidToken),
			errors.Is(err, runner.ErrTokenExpired):
			apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "Invalid or expired token")
		default:
			apierr.InternalError(c, "Failed to reactivate runner")
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"certificate":    resp.Certificate,
		"private_key":    resp.PrivateKey,
		"ca_certificate": resp.CACertificate,
	})
}

// ==================== Route Registration ====================

// GetDiscovery returns the current gRPC endpoint for runner auto-discovery.
// GET /api/v1/runners/grpc/discovery
// Authenticated via mTLS - requires X-Client-Cert-CN header (same as RenewCertificate).
func (h *GRPCRunnerHandler) GetDiscovery(c *gin.Context) {
	nodeID := c.GetHeader("X-Client-Cert-CN")
	if nodeID == "" {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "Missing client certificate")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"grpc_endpoint": h.config.GRPC.Endpoint,
	})
}

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

		// Discovery - returns current gRPC endpoint (authenticated via mTLS, X-Client-Cert-* headers)
		grpcPublic.GET("/discovery", handler.GetDiscovery)
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
