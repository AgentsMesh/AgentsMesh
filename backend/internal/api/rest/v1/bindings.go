package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/binding"
	"github.com/gin-gonic/gin"
)

// BindingHandler handles binding API endpoints
type BindingHandler struct {
	bindingSvc *binding.Service
}

// NewBindingHandler creates a new binding handler
func NewBindingHandler(bindingSvc *binding.Service) *BindingHandler {
	return &BindingHandler{
		bindingSvc: bindingSvc,
	}
}

// BindingRequest represents a request to create a binding
type BindingRequest struct {
	TargetPod string   `json:"target_pod" binding:"required"`
	Scopes    []string `json:"scopes" binding:"required"`
	Policy    string   `json:"policy,omitempty"`
}

// AcceptRequest represents a request to accept a binding
type AcceptRequest struct {
	BindingID int64 `json:"binding_id" binding:"required"`
}

// RejectRequest represents a request to reject a binding
type RejectRequest struct {
	BindingID int64  `json:"binding_id" binding:"required"`
	Reason    string `json:"reason,omitempty"`
}

// ScopeRequest represents a request for additional scopes
type ScopeRequest struct {
	Scopes []string `json:"scopes" binding:"required"`
}

// UnbindRequest represents a request to unbind
type UnbindRequest struct {
	TargetPod string `json:"target_pod" binding:"required"`
}

// getPodKeyFromHeader extracts pod key from X-Pod-Key header
func getPodKeyFromHeader(c *gin.Context) string {
	return c.GetHeader("X-Pod-Key")
}

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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-Pod-Key header required"})
		return
	}

	var req BindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get org ID from tenant context (set by PodAuthMiddleware or TenantMiddleware)
	tenant := middleware.GetTenant(c)
	if tenant == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid organization context"})
		return
	}
	orgIDInt64 := tenant.OrganizationID

	binding, err := h.bindingSvc.RequestBinding(c.Request.Context(), orgIDInt64, podKey, req.TargetPod, req.Scopes, req.Policy)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-Pod-Key header required"})
		return
	}

	var req AcceptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	binding, err := h.bindingSvc.AcceptBinding(c.Request.Context(), req.BindingID, podKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-Pod-Key header required"})
		return
	}

	var req RejectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	binding, err := h.bindingSvc.RejectBinding(c.Request.Context(), req.BindingID, podKey, req.Reason)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"binding": binding})
}

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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-Pod-Key header required"})
		return
	}

	var statusFilter *string
	if status := c.Query("status"); status != "" {
		statusFilter = &status
	}

	bindings, err := h.bindingSvc.GetBindingsForPod(c.Request.Context(), podKey, statusFilter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-Pod-Key header required"})
		return
	}

	pending, err := h.bindingSvc.GetPendingRequests(c.Request.Context(), podKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-Pod-Key header required"})
		return
	}

	pods, err := h.bindingSvc.GetBoundPods(c.Request.Context(), podKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-Pod-Key header required"})
		return
	}

	targetPod := c.Param("target_pod")
	if targetPod == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target_pod is required"})
		return
	}

	isBound, err := h.bindingSvc.IsBound(c.Request.Context(), podKey, targetPod)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
