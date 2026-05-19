package v1

import (
	"github.com/anthropics/agentsmesh/backend/internal/service/binding"
	"github.com/gin-gonic/gin"
)

type BindingHandler struct {
	bindingSvc *binding.Service
}

func NewBindingHandler(bindingSvc *binding.Service) *BindingHandler {
	return &BindingHandler{
		bindingSvc: bindingSvc,
	}
}

type BindingRequest struct {
	TargetPod string   `json:"target_pod" binding:"required"`
	Scopes    []string `json:"scopes" binding:"required"`
	Policy    string   `json:"policy,omitempty"`
}

type AcceptRequest struct {
	BindingID int64 `json:"binding_id" binding:"required"`
}

type RejectRequest struct {
	BindingID int64  `json:"binding_id" binding:"required"`
	Reason    string `json:"reason,omitempty"`
}

type ScopeRequest struct {
	Scopes []string `json:"scopes" binding:"required"`
}

type UnbindRequest struct {
	TargetPod string `json:"target_pod" binding:"required"`
}

func getPodKeyFromHeader(c *gin.Context) string {
	return c.GetHeader("X-Pod-Key")
}
