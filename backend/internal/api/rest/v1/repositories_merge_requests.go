package v1

import (
	"net/http"
	"strconv"

	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// ListRepositoryMergeRequests lists merge requests for a repository
// GET /api/v1/organizations/:slug/repositories/:id/merge-requests
// Query params:
//   - branch: filter by source branch (optional)
//   - state: filter by state (opened, merged, closed, all) (optional, default: all)
func (h *RepositoryHandler) ListRepositoryMergeRequests(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid repository ID")
		return
	}

	// Get optional query parameters
	branch := c.Query("branch")
	state := c.DefaultQuery("state", "all")

	// Fetch merge requests from service
	mrs, err := h.repositoryService.ListMergeRequests(c.Request.Context(), id, branch, state)
	if err != nil {
		apierr.InternalError(c, err.Error())
		return
	}

	// Transform to response format
	type MRResponse struct {
		ID             int64   `json:"id"`
		MRIID          int     `json:"mr_iid"`
		Title          string  `json:"title"`
		State          string  `json:"state"`
		MRURL          string  `json:"mr_url"`
		SourceBranch   string  `json:"source_branch"`
		TargetBranch   string  `json:"target_branch"`
		PipelineStatus *string `json:"pipeline_status,omitempty"`
		PipelineID     *int64  `json:"pipeline_id,omitempty"`
		PipelineURL    *string `json:"pipeline_url,omitempty"`
		TicketID       *int64  `json:"ticket_id,omitempty"`
		PodID          *int64  `json:"pod_id,omitempty"`
	}

	result := make([]MRResponse, 0, len(mrs))
	for _, mr := range mrs {
		result = append(result, MRResponse{
			ID:             mr.ID,
			MRIID:          mr.MRIID,
			Title:          mr.Title,
			State:          mr.State,
			MRURL:          mr.MRURL,
			SourceBranch:   mr.SourceBranch,
			TargetBranch:   mr.TargetBranch,
			PipelineStatus: mr.PipelineStatus,
			PipelineID:     mr.PipelineID,
			PipelineURL:    mr.PipelineURL,
			TicketID:       mr.TicketID,
			PodID:          mr.PodID,
		})
	}

	c.JSON(http.StatusOK, gin.H{"merge_requests": result})
}
