package v1

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// RequestScopes handles POST /bindings/:id/scopes
// @Summary Request additional scopes on an existing binding
// @Tags bindings
// @Accept json
// @Produce json
// @Param X-Pod-Key header string true "Pod Key"
// @Param id path int true "Binding ID"
// @Param request body ScopeRequest true "Scope request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Router /bindings/{id}/scopes [post]
func (h *BindingHandler) RequestScopes(c *gin.Context) {
	podKey := getPodKeyFromHeader(c)
	if podKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-Pod-Key header required"})
		return
	}

	bindingID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid binding ID"})
		return
	}

	var req ScopeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	binding, err := h.bindingSvc.RequestScopes(c.Request.Context(), bindingID, podKey, req.Scopes)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"binding": binding})
}

// ApproveScopes handles POST /bindings/:id/scopes/approve
// @Summary Approve pending scope requests
// @Tags bindings
// @Accept json
// @Produce json
// @Param X-Pod-Key header string true "Pod Key"
// @Param id path int true "Binding ID"
// @Param request body ScopeRequest true "Scopes to approve"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Router /bindings/{id}/scopes/approve [post]
func (h *BindingHandler) ApproveScopes(c *gin.Context) {
	podKey := getPodKeyFromHeader(c)
	if podKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-Pod-Key header required"})
		return
	}

	bindingID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid binding ID"})
		return
	}

	var req ScopeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	binding, err := h.bindingSvc.ApproveScopes(c.Request.Context(), bindingID, podKey, req.Scopes)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"binding": binding})
}

// Unbind handles POST /bindings/unbind
// @Summary Remove binding between pods
// @Tags bindings
// @Accept json
// @Produce json
// @Param X-Pod-Key header string true "Pod Key"
// @Param request body UnbindRequest true "Unbind request"
// @Success 204 "No Content"
// @Failure 404 {object} ErrorResponse
// @Router /bindings/unbind [post]
func (h *BindingHandler) Unbind(c *gin.Context) {
	podKey := getPodKeyFromHeader(c)
	if podKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-Pod-Key header required"})
		return
	}

	var req UnbindRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	removed, err := h.bindingSvc.Unbind(c.Request.Context(), podKey, req.TargetPod)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !removed {
		c.JSON(http.StatusNotFound, gin.H{"error": "no active binding found"})
		return
	}

	c.Status(http.StatusNoContent)
}
