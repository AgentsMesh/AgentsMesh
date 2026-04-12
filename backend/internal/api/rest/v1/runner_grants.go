package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/domain/grant"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/anthropics/agentsmesh/backend/pkg/policy"
	"github.com/gin-gonic/gin"
)

func (h *RunnerHandler) ListRunnerGrants(c *gin.Context) {
	if h.grantService == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Grant service not configured")
		return
	}
	runnerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid runner ID")
		return
	}

	if _, err := h.runnerService.GetRunner(c.Request.Context(), runnerID); err != nil {
		apierr.ResourceNotFound(c, "Runner not found")
		return
	}

	tenant := middleware.GetTenant(c)
	sub := policy.NewSubject(tenant.OrganizationID, tenant.UserID, tenant.UserRole)
	if !policy.AllowAdmin(sub, tenant.OrganizationID) {
		apierr.ForbiddenAdmin(c)
		return
	}

	grants, err := h.grantService.ListGrants(c.Request.Context(), grant.TypeRunner, grant.IntResourceID(runnerID))
	if err != nil {
		apierr.InternalError(c, "Failed to list grants")
		return
	}
	c.JSON(http.StatusOK, gin.H{"grants": grants})
}

func (h *RunnerHandler) GrantRunnerAccess(c *gin.Context) {
	if h.grantService == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Grant service not configured")
		return
	}
	runnerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid runner ID")
		return
	}

	var req grantAccessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	r, err := h.runnerService.GetRunner(c.Request.Context(), runnerID)
	if err != nil {
		apierr.ResourceNotFound(c, "Runner not found")
		return
	}

	tenant := middleware.GetTenant(c)
	sub := policy.NewSubject(tenant.OrganizationID, tenant.UserID, tenant.UserRole)
	if !policy.AllowAdmin(sub, tenant.OrganizationID) {
		apierr.ForbiddenAdmin(c)
		return
	}
	if !policy.RunnerPolicy.AllowRead(sub, policy.VisibleResource(
		r.OrganizationID, r.RegisteredByUserID, r.Visibility,
	)) {
		apierr.ForbiddenAccess(c)
		return
	}

	g, err := h.grantService.GrantAccess(c.Request.Context(),
		tenant.OrganizationID, grant.TypeRunner, grant.IntResourceID(runnerID), req.UserID, tenant.UserID)
	if err != nil {
		handleGrantError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"grant": g})
}

func (h *RunnerHandler) RevokeRunnerGrant(c *gin.Context) {
	if h.grantService == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Grant service not configured")
		return
	}
	runnerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid runner ID")
		return
	}

	r, err := h.runnerService.GetRunner(c.Request.Context(), runnerID)
	if err != nil {
		apierr.ResourceNotFound(c, "Runner not found")
		return
	}

	tenant := middleware.GetTenant(c)
	sub := policy.NewSubject(tenant.OrganizationID, tenant.UserID, tenant.UserRole)
	if !policy.AllowAdmin(sub, tenant.OrganizationID) {
		apierr.ForbiddenAdmin(c)
		return
	}
	if !policy.RunnerPolicy.AllowWrite(sub, policy.VisibleResource(
		r.OrganizationID, r.RegisteredByUserID, r.Visibility,
	)) {
		apierr.ForbiddenAccess(c)
		return
	}

	grantID, err := strconv.ParseInt(c.Param("grant_id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid grant ID")
		return
	}

	if err := h.grantService.RevokeAccess(c.Request.Context(), grant.TypeRunner, grant.IntResourceID(runnerID), grantID); err != nil {
		apierr.ResourceNotFound(c, "Grant not found")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Grant revoked"})
}
