package v1

import (
	"net/http"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

func (h *BindingHandler) RequestBinding(c *gin.Context) {
	podKey := getPodKeyFromHeader(c)
	if podKey == "" {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "X-Pod-Key header required")
		return
	}

	var req BindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	tenant := middleware.GetTenant(c)
	if tenant == nil {
		apierr.InternalError(c, "invalid organization context")
		return
	}
	orgIDInt64 := tenant.OrganizationID

	binding, err := h.bindingSvc.RequestBinding(c.Request.Context(), orgIDInt64, podKey, req.TargetPod, req.Scopes, req.Policy)
	if err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{"binding": binding})
}

func (h *BindingHandler) AcceptBinding(c *gin.Context) {
	podKey := getPodKeyFromHeader(c)
	if podKey == "" {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "X-Pod-Key header required")
		return
	}

	var req AcceptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	binding, err := h.bindingSvc.AcceptBinding(c.Request.Context(), req.BindingID, podKey)
	if err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"binding": binding})
}

func (h *BindingHandler) RejectBinding(c *gin.Context) {
	podKey := getPodKeyFromHeader(c)
	if podKey == "" {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "X-Pod-Key header required")
		return
	}

	var req RejectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	binding, err := h.bindingSvc.RejectBinding(c.Request.Context(), req.BindingID, podKey, req.Reason)
	if err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"binding": binding})
}
