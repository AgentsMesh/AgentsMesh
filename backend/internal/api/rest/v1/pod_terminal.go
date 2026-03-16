package v1

import (
	"net/http"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// TerminalRouterInterface defines the interface for terminal router operations
type TerminalRouterInterface interface {
	GetRunnerID(podKey string) (int64, bool)
	RouteInput(podKey string, data []byte) error
	RouteResize(podKey string, cols, rows int) error
	RouteObserveTerminal(runnerID int64, requestID, podKey string, lines int32, includeScreen bool) error
}

// TerminalOutputResponse matches Runner's tools.TerminalOutput structure
type TerminalOutputResponse struct {
	PodKey     string `json:"pod_key"`
	Output     string `json:"output"`
	Screen     string `json:"screen,omitempty"`
	CursorX    int    `json:"cursor_x"`
	CursorY    int    `json:"cursor_y"`
	TotalLines int    `json:"total_lines"`
	HasMore    bool   `json:"has_more"`
}

// ObserveTerminalRequest represents terminal observation request
type ObserveTerminalRequest struct {
	Lines         int  `form:"lines"`
	IncludeScreen bool `form:"include_screen"` // If true, include current screen snapshot
}

// TerminalInputRequest represents terminal input request
type TerminalInputRequest struct {
	Input string `json:"input" binding:"required"`
}

// TerminalResizeRequest represents terminal resize request
type TerminalResizeRequest struct {
	Cols int `json:"cols" binding:"required,min=1"`
	Rows int `json:"rows" binding:"required,min=1"`
}

// ObserveTerminal returns recent terminal output for observation
// GET /api/v1/organizations/:slug/pods/:key/terminal/observe
func (h *PodHandler) ObserveTerminal(c *gin.Context) {
	podKey := c.Param("key")

	var req ObserveTerminalRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	pod, err := h.podService.GetPod(c.Request.Context(), podKey)
	if err != nil {
		apierr.ResourceNotFound(c, "Pod not found")
		return
	}

	tenant := middleware.GetTenant(c)
	if pod.OrganizationID != tenant.OrganizationID {
		apierr.ForbiddenAccess(c)
		return
	}

	// Proxy terminal observation to runner via gRPC
	if h.terminalQueryService == nil || h.terminalRouter == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Terminal observation not available")
		return
	}

	tr, ok := h.terminalRouter.(TerminalRouterInterface)
	if !ok {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Terminal router interface not implemented")
		return
	}

	runnerID, found := tr.GetRunnerID(podKey)
	if !found {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Pod not registered on any runner")
		return
	}

	lines := req.Lines
	if lines == -1 {
		lines = 10000
	}
	if lines <= 0 {
		lines = 100
	}

	result, err := h.terminalQueryService.ObserveTerminal(
		c.Request.Context(),
		runnerID,
		podKey,
		int32(lines),
		req.IncludeScreen,
		func(runnerID int64, requestID, podKey string, lines int32, includeScreen bool) error {
			return tr.RouteObserveTerminal(runnerID, requestID, podKey, lines, includeScreen)
		},
	)
	if err != nil {
		apierr.InternalError(c, "Failed to observe terminal: "+err.Error())
		return
	}

	if result.Error != "" {
		apierr.InternalError(c, result.Error)
		return
	}

	response := TerminalOutputResponse{
		PodKey:     podKey,
		Output:     result.Output,
		CursorX:    result.CursorX,
		CursorY:    result.CursorY,
		TotalLines: result.TotalLines,
		HasMore:    result.HasMore,
	}

	if req.IncludeScreen {
		response.Screen = result.Screen
	}

	c.JSON(http.StatusOK, response)
}

// SendTerminalInput sends input to the terminal
// POST /api/v1/organizations/:slug/pods/:key/terminal/input
func (h *PodHandler) SendTerminalInput(c *gin.Context) {
	podKey := c.Param("key")

	var req TerminalInputRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	pod, err := h.podService.GetPod(c.Request.Context(), podKey)
	if err != nil {
		apierr.ResourceNotFound(c, "Pod not found")
		return
	}

	tenant := middleware.GetTenant(c)
	if pod.OrganizationID != tenant.OrganizationID {
		apierr.ForbiddenAccess(c)
		return
	}

	if !pod.IsActive() {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Pod is not active")
		return
	}

	if h.terminalRouter == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Terminal router not available")
		return
	}

	tr, ok := h.terminalRouter.(TerminalRouterInterface)
	if !ok {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Terminal router interface not implemented")
		return
	}

	if err := tr.RouteInput(podKey, []byte(req.Input)); err != nil {
		apierr.InternalError(c, "Failed to send input: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Input sent"})
}

// ResizeTerminal resizes the terminal
// POST /api/v1/organizations/:slug/pods/:key/terminal/resize
func (h *PodHandler) ResizeTerminal(c *gin.Context) {
	podKey := c.Param("key")

	var req TerminalResizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	pod, err := h.podService.GetPod(c.Request.Context(), podKey)
	if err != nil {
		apierr.ResourceNotFound(c, "Pod not found")
		return
	}

	tenant := middleware.GetTenant(c)
	if pod.OrganizationID != tenant.OrganizationID {
		apierr.ForbiddenAccess(c)
		return
	}

	if !pod.IsActive() {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Pod is not active")
		return
	}

	if h.terminalRouter == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Terminal router not available")
		return
	}

	tr, ok := h.terminalRouter.(TerminalRouterInterface)
	if !ok {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Terminal router interface not implemented")
		return
	}

	if err := tr.RouteResize(podKey, req.Cols, req.Rows); err != nil {
		apierr.InternalError(c, "Failed to resize terminal: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Terminal resized"})
}
