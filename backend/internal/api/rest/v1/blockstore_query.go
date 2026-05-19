package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *BlockstoreHandler) GetBlock(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierr.ValidationError(c, "invalid block id")
		return
	}
	b, err := h.service.GetBlock(c.Request.Context(), actor, id)
	if translateErr(c, err) {
		return
	}
	c.JSON(http.StatusOK, b)
}

func (h *BlockstoreHandler) ListChildren(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierr.ValidationError(c, "invalid block id")
		return
	}
	rel := c.DefaultQuery("rel", "nest")
	res, err := h.service.ListChildren(c.Request.Context(), actor, id, rel)
	if translateErr(c, err) {
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *BlockstoreHandler) ListBacklinks(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierr.ValidationError(c, "invalid block id")
		return
	}
	refs, err := h.service.ListBacklinks(c.Request.Context(), actor, id)
	if translateErr(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"refs": refs})
}

func (h *BlockstoreHandler) GetSubtree(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}
	wsID, err := uuid.Parse(c.Param("ws_id"))
	if err != nil {
		apierr.ValidationError(c, "invalid workspace id")
		return
	}
	rootID, err := uuid.Parse(c.Query("root"))
	if err != nil {
		apierr.ValidationError(c, "invalid root id")
		return
	}
	maxDepth, _ := strconv.Atoi(c.DefaultQuery("max_depth", "64"))
	res, err := h.service.ListSubtree(c.Request.Context(), actor, wsID, rootID, maxDepth)
	if translateErr(c, err) {
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *BlockstoreHandler) StreamOps(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}
	wsID, err := uuid.Parse(c.Param("ws_id"))
	if err != nil {
		apierr.ValidationError(c, "invalid workspace id")
		return
	}
	after, _ := strconv.ParseInt(c.DefaultQuery("after", "0"), 10, 64)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "200"))
	ops, err := h.service.StreamOps(c.Request.Context(), actor, wsID, after, limit)
	if translateErr(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"ops": ops})
}

func (h *BlockstoreHandler) ExportWorkspace(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}
	wsID, err := uuid.Parse(c.Param("ws_id"))
	if err != nil {
		apierr.ValidationError(c, "invalid workspace id")
		return
	}
	out, err := h.service.ExportWorkspace(c.Request.Context(), actor, wsID)
	if translateErr(c, err) {
		return
	}
	c.Header("Content-Disposition",
		`attachment; filename="blockstore-`+wsID.String()+`.json"`)
	c.JSON(http.StatusOK, out)
}

func (h *BlockstoreHandler) GetBlockAt(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierr.ValidationError(c, "invalid block id")
		return
	}
	opID, _ := strconv.ParseInt(c.DefaultQuery("op_id", "0"), 10, 64)
	snap, err := h.service.GetBlockAt(c.Request.Context(), actor, id, opID)
	if translateErr(c, err) {
		return
	}
	c.JSON(http.StatusOK, snap)
}

// type_defs live outside nest hierarchy — frontend MUST scan via this route to build the indicator registry.
func (h *BlockstoreHandler) ListTypeDefs(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}
	wsID, err := uuid.Parse(c.Param("ws_id"))
	if err != nil {
		apierr.ValidationError(c, "invalid workspace id")
		return
	}
	blocks, err := h.service.ListTypeDefBlocks(c.Request.Context(), actor, wsID)
	if translateErr(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"blocks": blocks})
}
