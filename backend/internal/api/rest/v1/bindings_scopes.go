package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

func (h *BindingHandler) RequestScopes(c *gin.Context) {
	podKey := getPodKeyFromHeader(c)
	if podKey == "" {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "X-Pod-Key header required")
		return
	}

	bindingID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "invalid binding ID")
		return
	}

	var req ScopeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	binding, err := h.bindingSvc.RequestScopes(c.Request.Context(), bindingID, podKey, req.Scopes)
	if err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"binding": binding})
}

func (h *BindingHandler) ApproveScopes(c *gin.Context) {
	podKey := getPodKeyFromHeader(c)
	if podKey == "" {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "X-Pod-Key header required")
		return
	}

	bindingID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "invalid binding ID")
		return
	}

	var req ScopeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	binding, err := h.bindingSvc.ApproveScopes(c.Request.Context(), bindingID, podKey, req.Scopes)
	if err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"binding": binding})
}

func (h *BindingHandler) Unbind(c *gin.Context) {
	podKey := getPodKeyFromHeader(c)
	if podKey == "" {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "X-Pod-Key header required")
		return
	}

	var req UnbindRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	removed, err := h.bindingSvc.Unbind(c.Request.Context(), podKey, req.TargetPod)
	if err != nil {
		apierr.InternalError(c, err.Error())
		return
	}

	if !removed {
		apierr.ResourceNotFound(c, "no active binding found")
		return
	}

	c.Status(http.StatusNoContent)
}
