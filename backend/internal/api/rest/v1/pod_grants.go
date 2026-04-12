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

func (h *PodHandler) ListPodGrants(c *gin.Context) {
	if h.grantService == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Grant service not configured")
		return
	}
	podKey := c.Param("key")
	pod, err := h.podService.GetPod(c.Request.Context(), podKey)
	if err != nil {
		apierr.ResourceNotFound(c, "Pod not found")
		return
	}

	tenant := middleware.GetTenant(c)
	sub := policy.NewSubject(tenant.OrganizationID, tenant.UserID, tenant.UserRole)
	if !policy.PodPolicy.AllowWrite(sub, policy.PodResource(pod.OrganizationID, pod.CreatedByID)) {
		apierr.ForbiddenAccess(c)
		return
	}

	grants, err := h.grantService.ListGrants(c.Request.Context(), grant.TypePod, podKey)
	if err != nil {
		apierr.InternalError(c, "Failed to list grants")
		return
	}
	c.JSON(http.StatusOK, gin.H{"grants": grants})
}

func (h *PodHandler) GrantPodAccess(c *gin.Context) {
	if h.grantService == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Grant service not configured")
		return
	}
	podKey := c.Param("key")
	var req grantAccessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	pod, err := h.podService.GetPod(c.Request.Context(), podKey)
	if err != nil {
		apierr.ResourceNotFound(c, "Pod not found")
		return
	}

	tenant := middleware.GetTenant(c)
	sub := policy.NewSubject(tenant.OrganizationID, tenant.UserID, tenant.UserRole)
	if !policy.PodPolicy.AllowWrite(sub, policy.PodResource(pod.OrganizationID, pod.CreatedByID)) {
		apierr.ForbiddenAccess(c)
		return
	}

	g, err := h.grantService.GrantAccess(c.Request.Context(),
		tenant.OrganizationID, grant.TypePod, podKey, req.UserID, tenant.UserID)
	if err != nil {
		handleGrantError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"grant": g})
}

func (h *PodHandler) RevokePodGrant(c *gin.Context) {
	if h.grantService == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Grant service not configured")
		return
	}
	podKey := c.Param("key")
	grantID, err := strconv.ParseInt(c.Param("grant_id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid grant ID")
		return
	}

	pod, err := h.podService.GetPod(c.Request.Context(), podKey)
	if err != nil {
		apierr.ResourceNotFound(c, "Pod not found")
		return
	}

	tenant := middleware.GetTenant(c)
	sub := policy.NewSubject(tenant.OrganizationID, tenant.UserID, tenant.UserRole)
	if !policy.PodPolicy.AllowWrite(sub, policy.PodResource(pod.OrganizationID, pod.CreatedByID)) {
		apierr.ForbiddenAccess(c)
		return
	}

	if err := h.grantService.RevokeAccess(c.Request.Context(), grant.TypePod, podKey, grantID); err != nil {
		apierr.ResourceNotFound(c, "Grant not found")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Grant revoked"})
}
