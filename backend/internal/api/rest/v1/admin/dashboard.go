package admin

import (
	"net/http"

	adminservice "github.com/anthropics/agentsmesh/backend/internal/service/admin"

	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

type DashboardHandler struct {
	adminService *adminservice.Service
}

func NewDashboardHandler(adminSvc *adminservice.Service) *DashboardHandler {
	return &DashboardHandler{
		adminService: adminSvc,
	}
}

func (h *DashboardHandler) RegisterRoutes(rg *gin.RouterGroup) {
	dashboardGroup := rg.Group("/dashboard")
	{
		dashboardGroup.GET("/stats", h.GetStats)
	}
}

func (h *DashboardHandler) GetStats(c *gin.Context) {
	stats, err := h.adminService.GetDashboardStats(c.Request.Context())
	if err != nil {
		apierr.InternalError(c, "Failed to get dashboard stats")
		return
	}

	c.JSON(http.StatusOK, stats)
}
