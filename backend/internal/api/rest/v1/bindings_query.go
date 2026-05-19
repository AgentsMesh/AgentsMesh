package v1

import (
	"net/http"

	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

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
