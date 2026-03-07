package v1

import (
	"net/http"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// TerminalRouterInterface defines the interface for terminal router operations
type TerminalRouterInterface interface {
	GetRecentOutput(podKey string, lines int) []byte
	GetScreenSnapshot(podKey string) string
	GetCursorPosition(podKey string) (row, col int)
	RouteInput(podKey string, data []byte) error
	RouteResize(podKey string, cols, rows int) error
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

	// Get terminal output from router if available
	if h.terminalRouter == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Terminal router not available")
		return
	}

	tr, ok := h.terminalRouter.(TerminalRouterInterface)
	if !ok {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Terminal router interface not implemented")
		return
	}

	lines := req.Lines
	if lines == -1 {
		lines = 10000 // Get all available output
	}
	if lines <= 0 {
		lines = 100 // Default to last 100 lines
	}

	// Get recent output (processed, without ANSI escape sequences)
	output := tr.GetRecentOutput(podKey, lines)

	// Get cursor position from virtual terminal
	cursorRow, cursorCol := tr.GetCursorPosition(podKey)

	// Calculate total lines (rough estimate from output)
	totalLines := 0
	for _, b := range output {
		if b == '\n' {
			totalLines++
		}
	}
	if len(output) > 0 && output[len(output)-1] != '\n' {
		totalLines++ // Count last line if not ending with newline
	}

	// Build response matching Runner's TerminalOutput structure
	response := TerminalOutputResponse{
		PodKey:     podKey,
		Output:     string(output),
		CursorX:    cursorCol,
		CursorY:    cursorRow,
		TotalLines: totalLines,
		HasMore:    totalLines >= lines,
	}

	// Include screen snapshot if requested
	if req.IncludeScreen {
		response.Screen = tr.GetScreenSnapshot(podKey)
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
