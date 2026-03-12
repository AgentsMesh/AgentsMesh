package v1

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/tokenusage"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	tokenusagesvc "github.com/anthropics/agentsmesh/backend/internal/service/tokenusage"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
)

// validGranularities defines allowed granularity values.
var validGranularities = map[string]bool{
	"day":   true,
	"week":  true,
	"month": true,
}

// TokenUsageHandler handles token usage HTTP requests.
type TokenUsageHandler struct {
	svc *tokenusagesvc.Service
}

// NewTokenUsageHandler creates a new token usage handler.
func NewTokenUsageHandler(svc *tokenusagesvc.Service) *TokenUsageHandler {
	return &TokenUsageHandler{svc: svc}
}

// GetDashboard returns all token usage data in a single response.
// The 5 queries run concurrently via errgroup for lower latency.
// GET /token-usage/dashboard?start_time=&end_time=&agent_type=&user_id=&model=&granularity=
func (h *TokenUsageHandler) GetDashboard(c *gin.Context) {
	tenant := c.MustGet("tenant").(*middleware.TenantContext)
	if !isAdminOrOwner(tenant) {
		apierr.ForbiddenAdmin(c)
		return
	}

	filter, err := parseFilter(c)
	if err != nil {
		apierr.AbortBadRequest(c, "VALIDATION_FAILED", err.Error())
		return
	}

	orgID := tenant.OrganizationID
	ctx := c.Request.Context()

	var (
		summary    *tokenusage.UsageSummary
		timeSeries []tokenusage.TimeSeriesPoint
		byAgent    []tokenusage.AgentUsage
		byUser     []tokenusage.UserUsage
		byModel    []tokenusage.ModelUsage
	)

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		summary, err = h.svc.GetSummary(gCtx, orgID, filter)
		return err
	})
	g.Go(func() error {
		var err error
		timeSeries, err = h.svc.GetTimeSeries(gCtx, orgID, filter)
		return err
	})
	g.Go(func() error {
		var err error
		byAgent, err = h.svc.GetByAgent(gCtx, orgID, filter)
		return err
	})
	g.Go(func() error {
		var err error
		byUser, err = h.svc.GetByUser(gCtx, orgID, filter)
		return err
	})
	g.Go(func() error {
		var err error
		byModel, err = h.svc.GetByModel(gCtx, orgID, filter)
		return err
	})

	if err := g.Wait(); err != nil {
		slog.Error("failed to get token usage dashboard", "org_id", orgID, "error", err)
		apierr.InternalError(c, "failed to retrieve token usage data")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"summary":     summary,
		"time_series": timeSeries,
		"by_agent":    byAgent,
		"by_user":     byUser,
		"by_model":    byModel,
	})
}

// RegisterTokenUsageRoutes registers token usage routes on the given group.
func RegisterTokenUsageRoutes(rg *gin.RouterGroup, svc *tokenusagesvc.Service) {
	handler := NewTokenUsageHandler(svc)
	group := rg.Group("/token-usage")
	{
		group.GET("/dashboard", handler.GetDashboard)
	}
}

// isAdminOrOwner checks if the tenant has owner or admin role.
func isAdminOrOwner(tenant *middleware.TenantContext) bool {
	return tenant.UserRole == "owner" || tenant.UserRole == "admin"
}

// parseFilter extracts AggregationFilter from query parameters.
func parseFilter(c *gin.Context) (tokenusage.AggregationFilter, error) {
	var filter tokenusage.AggregationFilter

	// Default to last 30 days
	filter.EndTime = time.Now()
	filter.StartTime = filter.EndTime.AddDate(0, 0, -30)

	if s := c.Query("start_time"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return filter, err
		}
		filter.StartTime = t
	}
	if s := c.Query("end_time"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return filter, err
		}
		filter.EndTime = t
	}

	if filter.StartTime.After(filter.EndTime) {
		return filter, fmt.Errorf("start_time must be before end_time")
	}

	// Cap date range to prevent expensive aggregation queries (max 1 year).
	const maxDateRange = 366 * 24 * time.Hour
	if filter.EndTime.Sub(filter.StartTime) > maxDateRange {
		return filter, fmt.Errorf("date range cannot exceed 366 days")
	}

	if s := c.Query("granularity"); s != "" {
		if !validGranularities[s] {
			return filter, fmt.Errorf("invalid granularity %q, must be one of: day, week, month", s)
		}
		filter.Granularity = s
	} else {
		filter.Granularity = "day"
	}

	if s := c.Query("agent_type"); s != "" {
		filter.AgentTypeSlug = &s
	}
	if s := c.Query("user_id"); s != "" {
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return filter, err
		}
		filter.UserID = &id
	}
	if s := c.Query("model"); s != "" {
		filter.Model = &s
	}

	return filter, nil
}
