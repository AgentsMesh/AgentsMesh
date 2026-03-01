package v1

import (
	"net/http"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/middleware"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// RequestBinding handles POST /bindings
// @Summary Request binding to another pod
// @Tags bindings
// @Accept json
// @Produce json
// @Param X-Pod-Key header string true "Pod Key"
// @Param request body BindingRequest true "Binding request"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Router /bindings [post]
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

	// Get org ID from tenant context (set by TenantMiddleware)
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

// AcceptBinding handles POST /bindings/accept
// @Summary Accept a pending binding request
// @Tags bindings
// @Accept json
// @Produce json
// @Param X-Pod-Key header string true "Pod Key"
// @Param request body AcceptRequest true "Accept request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Router /bindings/accept [post]
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

// RejectBinding handles POST /bindings/reject
// @Summary Reject a pending binding request
// @Tags bindings
// @Accept json
// @Produce json
// @Param X-Pod-Key header string true "Pod Key"
// @Param request body RejectRequest true "Reject request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Router /bindings/reject [post]
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
