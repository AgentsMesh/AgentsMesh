package admin

import (
	"net/http"
	"strconv"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/admin"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/organization"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/runner"
	adminservice "github.com/AgentsMesh/AgentsMesh/backend/internal/service/admin"

	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// RunnerHandler handles runner management requests
type RunnerHandler struct {
	adminService *adminservice.Service
}

// NewRunnerHandler creates a new runner handler
func NewRunnerHandler(adminSvc *adminservice.Service) *RunnerHandler {
	return &RunnerHandler{
		adminService: adminSvc,
	}
}

// RegisterRoutes registers runner management routes
func (h *RunnerHandler) RegisterRoutes(rg *gin.RouterGroup) {
	runnersGroup := rg.Group("/runners")
	{
		runnersGroup.GET("", h.ListRunners)
		runnersGroup.GET("/:id", h.GetRunner)
		runnersGroup.POST("/:id/disable", h.DisableRunner)
		runnersGroup.POST("/:id/enable", h.EnableRunner)
		runnersGroup.DELETE("/:id", h.DeleteRunner)
	}
}

// logAction is a helper method that delegates to the shared LogAdminAction function
func (h *RunnerHandler) logAction(c *gin.Context, action admin.AuditAction, targetType admin.TargetType, targetID int64, oldData, newData interface{}) {
	LogAdminAction(c, h.adminService, action, targetType, targetID, oldData, newData)
}

// ListRunners returns a list of runners with pagination
func (h *RunnerHandler) ListRunners(c *gin.Context) {
	query := &adminservice.RunnerListQuery{
		Search:   c.Query("search"),
		Status:   c.Query("status"),
		Page:     1,
		PageSize: 20,
	}

	if page, err := strconv.Atoi(c.Query("page")); err == nil {
		query.Page = page
	}
	if pageSize, err := strconv.Atoi(c.Query("page_size")); err == nil {
		query.PageSize = pageSize
	}
	if orgIDStr := c.Query("org_id"); orgIDStr != "" {
		if orgID, err := strconv.ParseInt(orgIDStr, 10, 64); err == nil {
			query.OrgID = &orgID
		}
	}

	result, err := h.adminService.ListRunners(c.Request.Context(), query)
	if err != nil {
		apierr.InternalError(c, "Failed to list runners")
		return
	}

	// Convert to response format
	runnerList := make([]gin.H, len(result.Data))
	for i, rwo := range result.Data {
		runnerList[i] = runnerResponseWithOrg(&rwo.Runner, rwo.Organization)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        runnerList,
		"total":       result.Total,
		"page":        result.Page,
		"page_size":   result.PageSize,
		"total_pages": result.TotalPages,
	})
}

// GetRunner returns a single runner
func (h *RunnerHandler) GetRunner(c *gin.Context) {
	runnerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid runner ID")
		return
	}

	rwo, err := h.adminService.GetRunnerWithOrg(c.Request.Context(), runnerID)
	if err != nil {
		if err == adminservice.ErrRunnerNotFound {
			apierr.ResourceNotFound(c, "Runner not found")
			return
		}
		apierr.InternalError(c, "Failed to get runner")
		return
	}

	// Log view action
	h.logAction(c, admin.AuditActionRunnerView, admin.TargetTypeRunner, runnerID, nil, nil)

	c.JSON(http.StatusOK, runnerResponseWithOrg(&rwo.Runner, rwo.Organization))
}

// runnerResponse creates a sanitized runner response
func runnerResponse(r *runner.Runner) gin.H {
	return gin.H{
		"id":                  r.ID,
		"organization_id":     r.OrganizationID,
		"node_id":             r.NodeID,
		"description":         r.Description,
		"status":              r.Status,
		"is_enabled":          r.IsEnabled,
		"runner_version":      r.RunnerVersion,
		"current_pods":        r.CurrentPods,
		"max_concurrent_pods": r.MaxConcurrentPods,
		"available_agents":    r.AvailableAgents,
		"host_info":           r.HostInfo,
		"last_heartbeat":      r.LastHeartbeat,
		"created_at":          r.CreatedAt,
		"updated_at":          r.UpdatedAt,
	}
}

// runnerResponseWithOrg creates a runner response with organization info
func runnerResponseWithOrg(r *runner.Runner, org *organization.Organization) gin.H {
	response := runnerResponse(r)

	if org != nil {
		response["organization"] = gin.H{
			"id":   org.ID,
			"name": org.Name,
			"slug": org.Slug,
		}
	}

	return response
}
