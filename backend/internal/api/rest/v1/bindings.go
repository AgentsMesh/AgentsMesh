package v1

import (
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
