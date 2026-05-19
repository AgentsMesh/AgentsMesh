package v1

import (
	"net/http"
	"strconv"

	blockstoreservice "github.com/anthropics/agentsmesh/backend/internal/service/blockstore"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *BlockstoreHandler) SemanticSearch(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}
	wsID, err := uuid.Parse(c.Param("ws_id"))
	if err != nil {
		apierr.ValidationError(c, "invalid workspace id")
		return
	}
	var req struct {
		Query    string  `json:"query"`
		TopK     int     `json:"top_k,omitempty"`
		MinScore float32 `json:"min_score,omitempty"`
		Type     string  `json:"type,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, "invalid search body")
		return
	}
	hits, err := h.service.SemanticSearch(c.Request.Context(), actor, blockstoreservice.SearchInput{
		WorkspaceID: wsID,
		Query:       req.Query,
		TopK:        req.TopK,
		MinScore:    req.MinScore,
		TypeFilter:  req.Type,
	})
	if translateErr(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"hits": hits})
}

func (h *BlockstoreHandler) MemoryRetrieve(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}
	wsID, err := uuid.Parse(c.Param("ws_id"))
	if err != nil {
		apierr.ValidationError(c, "invalid workspace id")
		return
	}
	var req struct {
		Query string `json:"query"`
		K     int    `json:"k,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, "invalid memory body")
		return
	}
	k := req.K
	if k <= 0 {
		k = 5
	}
	if s := c.Query("k"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			k = n
		}
	}
	hits, err := h.service.SemanticSearch(c.Request.Context(), actor, blockstoreservice.SearchInput{
		WorkspaceID: wsID,
		Query:       req.Query,
		TopK:        k,
	})
	if translateErr(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"memories": hits})
}
