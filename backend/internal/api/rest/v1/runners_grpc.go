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

type GRPCRunnerHandler struct {
	runnerService *runner.Service
	pkiService    *pki.Service
	config        *config.Config
}

func NewGRPCRunnerHandler(runnerService *runner.Service, pkiService *pki.Service, cfg *config.Config) *GRPCRunnerHandler {
	return &GRPCRunnerHandler{
		runnerService: runnerService,
		pkiService:    pkiService,
		config:        cfg,
	}
}

// POST /api/v1/runners/grpc/renew-certificate — authenticated via mTLS (Nginx passes CN).
func (h *GRPCRunnerHandler) RenewCertificate(c *gin.Context) {
	if h.pkiService == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "PKI service not configured")
		return
	}

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

	if tenant.UserRole != "owner" && tenant.UserRole != "admin" {
		apierr.ForbiddenAdmin(c)
		return
	}

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

func (h *GRPCRunnerHandler) Reactivate(c *gin.Context) {
	if h.pkiService == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "PKI service not configured")
		return
	}

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

// GET /api/v1/runners/grpc/discovery — mTLS-authenticated via X-Client-Cert-CN.
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

func RegisterGRPCRunnerRoutes(r *gin.RouterGroup, handler *GRPCRunnerHandler) {
	grpcPublic := r.Group("/runners/grpc")
	{
		grpcPublic.POST("/auth-url", handler.RequestAuthURL)
		grpcPublic.GET("/auth-status", handler.GetAuthStatus)

		grpcPublic.POST("/register", handler.RegisterWithToken)

		grpcPublic.POST("/reactivate", handler.Reactivate)

		grpcPublic.POST("/renew-certificate", handler.RenewCertificate)

		grpcPublic.GET("/discovery", handler.GetDiscovery)
	}
}

func RegisterOrgGRPCRunnerRoutes(rg *gin.RouterGroup, handler *GRPCRunnerHandler) {
	grpc := rg.Group("/grpc")
	{
		grpc.POST("/authorize", handler.AuthorizeRunner)

		grpc.GET("/tokens", handler.ListGRPCTokens)
		grpc.POST("/tokens", handler.GenerateGRPCToken)
		grpc.DELETE("/tokens/:id", handler.DeleteGRPCToken)
	}

	rg.POST("/:id/reactivate", handler.GenerateReactivationToken)
}
