package admin

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/domain/admin"
	"github.com/anthropics/agentsmesh/backend/internal/domain/organization"
	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	adminservice "github.com/anthropics/agentsmesh/backend/internal/service/admin"

	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

type RunnerHandler struct {
	adminService *adminservice.Service
}

func NewRunnerHandler(adminSvc *adminservice.Service) *RunnerHandler {
	return &RunnerHandler{
		adminService: adminSvc,
	}
}

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

func (h *RunnerHandler) logAction(c *gin.Context, action admin.AuditAction, targetType admin.TargetType, targetID int64, oldData, newData interface{}) {
	LogAdminAction(c, h.adminService, action, targetType, targetID, oldData, newData)
}

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

func (h *RunnerHandler) GetRunner(c *gin.Context) {
	runnerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid runner ID")
		return
	}

	rwo, err := h.adminService.GetRunnerWithOrg(c.Request.Context(), runnerID)
	if err != nil {
		if errors.Is(err, adminservice.ErrRunnerNotFound) {
			apierr.ResourceNotFound(c, "Runner not found")
			return
		}
		apierr.InternalError(c, "Failed to get runner")
		return
	}

	h.logAction(c, admin.AuditActionRunnerView, admin.TargetTypeRunner, runnerID, nil, nil)

	c.JSON(http.StatusOK, runnerResponseWithOrg(&rwo.Runner, rwo.Organization))
}

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
