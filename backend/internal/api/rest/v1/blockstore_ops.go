package v1

import (
	"net/http"

	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	blockstoreservice "github.com/anthropics/agentsmesh/backend/internal/service/blockstore"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *BlockstoreHandler) ApplyOps(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}
	var req blockstoreservice.ApplyOpsInput
	if err := c.ShouldBindJSON(&req); err != nil {
		translateErr(c, err)
		return
	}
	res, err := h.service.ApplyOps(c.Request.Context(), actor, req)
	if translateErr(c, err) {
		return
	}
	status := http.StatusOK
	if !res.WasReplay {
		status = http.StatusCreated
	}
	c.JSON(status, res)
}

func (h *BlockstoreHandler) ListWorkspaces(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}
	list, err := h.service.ListWorkspaces(c.Request.Context(), actor)
	if translateErr(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"workspaces": list})
}

func (h *BlockstoreHandler) EnsureDefaultWorkspace(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}
	ws, err := h.service.EnsureDefaultWorkspace(c.Request.Context(), actor)
	if translateErr(c, err) {
		return
	}
	c.JSON(http.StatusOK, ws)
}

func (h *BlockstoreHandler) CreateWorkspace(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}
	var req struct {
		Slug string `json:"slug"`
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		translateErr(c, err)
		return
	}
	ws, err := h.service.CreateWorkspace(c.Request.Context(), actor, req.Slug, req.Name)
	if translateErr(c, err) {
		return
	}
	c.JSON(http.StatusCreated, ws)
}

func (h *BlockstoreHandler) DeleteWorkspace(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}
	wsID, err := uuid.Parse(c.Param("ws_id"))
	if err != nil {
		apierr.ValidationError(c, "invalid workspace id")
		return
	}
	if err := h.service.DeleteWorkspace(c.Request.Context(), actor, wsID); err != nil {
		translateErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
