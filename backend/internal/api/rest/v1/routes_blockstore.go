package v1

import (
	"fmt"
	"net/http"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

// maxBlockstoreWriteBody caps write-path request bodies. Block data is
// expected to be small JSON (chart series, a paragraph, a task); 10 MB
// leaves generous headroom for large indicator batches while preventing a
// misbehaving client from pushing multi-GB JSONB blobs into the op log.
const maxBlockstoreWriteBody = 10 << 20 // 10 MB

func registerBlockstoreRoutes(rg *gin.RouterGroup, svc *Services) {
	if svc.Blockstore == nil {
		return
	}
	h := NewBlockstoreHandler(svc.Blockstore)
	blocks := rg.Group("/blocks")
	{
		writeLimiters := []gin.HandlerFunc{
			bodySizeLimit(maxBlockstoreWriteBody),
			middleware.IPRateLimiter(svc.Redis, "blockstore.write", 300, time.Minute),
		}
		blocks.POST("/ops", append(writeLimiters, h.ApplyOps)...)

		blocks.GET("/workspaces", h.ListWorkspaces)
		blocks.POST("/workspaces/default", h.EnsureDefaultWorkspace)
		blocks.POST("/workspaces",
			middleware.IPRateLimiter(svc.Redis, "blockstore.write", 300, time.Minute),
			h.CreateWorkspace)
		blocks.DELETE("/workspaces/:ws_id",
			middleware.IPRateLimiter(svc.Redis, "blockstore.write", 300, time.Minute),
			h.DeleteWorkspace)
		blocks.GET("/workspaces/:ws_id/subtree", h.GetSubtree)
		blocks.GET("/workspaces/:ws_id/ops", h.StreamOps)
		blocks.GET("/workspaces/:ws_id/export", h.ExportWorkspace)
		blocks.GET("/workspaces/:ws_id/type-defs", h.ListTypeDefs)
		blocks.POST("/workspaces/:ws_id/search",
			middleware.IPRateLimiter(svc.Redis, "blockstore.search", 120, time.Minute),
			h.SemanticSearch)
		blocks.POST("/workspaces/:ws_id/memory/retrieve",
			middleware.IPRateLimiter(svc.Redis, "blockstore.search", 120, time.Minute),
			h.MemoryRetrieve)

		blocks.GET("/:id", h.GetBlock)
		blocks.GET("/:id/children", h.ListChildren)
		blocks.GET("/:id/backlinks", h.ListBacklinks)
		blocks.GET("/:id/at", h.GetBlockAt)

	}
}

func bodySizeLimit(max int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > max {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge,
				gin.H{"error": fmt.Sprintf("request body exceeds %d bytes", max)})
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, max)
		c.Next()
	}
}
