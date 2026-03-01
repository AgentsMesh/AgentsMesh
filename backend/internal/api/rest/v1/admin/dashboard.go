package admin

import (
	"net/http"

	adminservice "github.com/AgentsMesh/AgentsMesh/backend/internal/service/admin"

	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// DashboardHandler handles dashboard-related requests
type DashboardHandler struct {
	adminService *adminservice.Service
}

// NewDashboardHandler creates a new dashboard handler
func NewDashboardHandler(adminSvc *adminservice.Service) *DashboardHandler {
	return &DashboardHandler{
		adminService: adminSvc,
	}
}

// RegisterRoutes registers dashboard routes
func (h *DashboardHandler) RegisterRoutes(rg *gin.RouterGroup) {
	dashboardGroup := rg.Group("/dashboard")
	{
		dashboardGroup.GET("/stats", h.GetStats)
	}
}

// GetStats returns dashboard statistics
func (h *DashboardHandler) GetStats(c *gin.Context) {
	stats, err := h.adminService.GetDashboardStats(c.Request.Context())
	if err != nil {
		apierr.InternalError(c, "Failed to get dashboard stats")
		return
	}

	c.JSON(http.StatusOK, stats)
}
