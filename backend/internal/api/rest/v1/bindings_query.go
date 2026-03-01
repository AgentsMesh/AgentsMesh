package v1

import (
	"net/http"

	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// ListBindings handles GET /bindings
// @Summary List all bindings for this pod
// @Tags bindings
// @Accept json
// @Produce json
// @Param X-Pod-Key header string true "Pod Key"
// @Param status query string false "Filter by status"
// @Success 200 {object} map[string]interface{}
// @Router /bindings [get]
func (h *BindingHandler) ListBindings(c *gin.Context) {
	podKey := getPodKeyFromHeader(c)
	if podKey == "" {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "X-Pod-Key header required")
		return
	}

	var statusFilter *string
	if status := c.Query("status"); status != "" {
		statusFilter = &status
	}

	bindings, err := h.bindingSvc.GetBindingsForPod(c.Request.Context(), podKey, statusFilter)
	if err != nil {
		apierr.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"bindings": bindings,
		"total":    len(bindings),
	})
}

// GetPendingBindings handles GET /bindings/pending
// @Summary Get pending binding requests for this pod
// @Tags bindings
// @Accept json
// @Produce json
// @Param X-Pod-Key header string true "Pod Key"
// @Success 200 {object} map[string]interface{}
// @Router /bindings/pending [get]
func (h *BindingHandler) GetPendingBindings(c *gin.Context) {
	podKey := getPodKeyFromHeader(c)
	if podKey == "" {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "X-Pod-Key header required")
		return
	}

	pending, err := h.bindingSvc.GetPendingRequests(c.Request.Context(), podKey)
	if err != nil {
		apierr.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pending": pending,
		"count":   len(pending),
	})
}

// GetBoundPods handles GET /bindings/pods
// @Summary Get list of pods this pod is bound to
// @Tags bindings
// @Accept json
// @Produce json
// @Param X-Pod-Key header string true "Pod Key"
// @Success 200 {object} map[string]interface{}
// @Router /bindings/pods [get]
func (h *BindingHandler) GetBoundPods(c *gin.Context) {
	podKey := getPodKeyFromHeader(c)
	if podKey == "" {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "X-Pod-Key header required")
		return
	}

	pods, err := h.bindingSvc.GetBoundPods(c.Request.Context(), podKey)
	if err != nil {
		apierr.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pods":  pods,
		"count": len(pods),
	})
}

// CheckBinding handles GET /bindings/check/:target_pod
// @Summary Check if bound to a specific pod
// @Tags bindings
// @Accept json
// @Produce json
// @Param X-Pod-Key header string true "Pod Key"
// @Param target_pod path string true "Target Pod Key"
// @Success 200 {object} map[string]interface{}
// @Router /bindings/check/{target_pod} [get]
func (h *BindingHandler) CheckBinding(c *gin.Context) {
	podKey := getPodKeyFromHeader(c)
	if podKey == "" {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "X-Pod-Key header required")
		return
	}

	targetPod := c.Param("target_pod")
	if targetPod == "" {
		apierr.BadRequest(c, apierr.MISSING_REQUIRED, "target_pod is required")
		return
	}

	isBound, err := h.bindingSvc.IsBound(c.Request.Context(), podKey, targetPod)
	if err != nil {
		apierr.InternalError(c, err.Error())
		return
	}

	response := gin.H{
		"is_bound": isBound,
		"binding":  nil,
	}

	if isBound {
		// Try to get the binding details
		binding, err := h.bindingSvc.GetActiveBinding(c.Request.Context(), podKey, targetPod)
		if err != nil {
			binding, _ = h.bindingSvc.GetActiveBinding(c.Request.Context(), targetPod, podKey)
		}
		if binding != nil {
			response["binding"] = binding
		}
	}

	c.JSON(http.StatusOK, response)
}
