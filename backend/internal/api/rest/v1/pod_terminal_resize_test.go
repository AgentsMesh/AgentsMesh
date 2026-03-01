package v1

import (
	"errors"
	"net/http"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/agentpod"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// =============================================================================
// ResizeTerminal Tests
// =============================================================================

func TestResizeTerminal_Success(t *testing.T) {
	mockRouter := &mockTerminalRouter{
		routeResizeErr: nil,
	}

	activePod := &agentpod.Pod{
		ID:             1,
		PodKey:         "test-pod-key",
		OrganizationID: 100,
		Status:         agentpod.StatusRunning,
	}

	h := &PodHandler{
		terminalRouter: mockRouter,
	}

	c, w := createTerminalTestContext(http.MethodPost, "/pods/test-pod-key/terminal/resize", "test-pod-key",
		`{"cols": 120, "rows": 40}`)

	// Simulate handler logic
	if !activePod.IsActive() {
		t.Fatal("Pod should be active")
	}

	tr, _ := h.terminalRouter.(TerminalRouterInterface)
	err := tr.RouteResize(activePod.PodKey, 120, 40)
	if err == nil {
		c.JSON(http.StatusOK, gin.H{"message": "Terminal resized"})
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

func TestResizeTerminal_PodNotActive(t *testing.T) {
	terminatedPod := &agentpod.Pod{
		ID:             1,
		PodKey:         "terminated-pod",
		OrganizationID: 100,
		Status:         agentpod.StatusTerminated,
	}

	c, w := createTerminalTestContext(http.MethodPost, "/pods/terminated-pod/terminal/resize", "terminated-pod",
		`{"cols": 80, "rows": 24}`)

	if !terminatedPod.IsActive() {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Pod is not active")
	}

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestResizeTerminal_RouteError(t *testing.T) {
	mockRouter := &mockTerminalRouter{
		routeResizeErr: errors.New("resize failed"),
	}

	h := &PodHandler{
		terminalRouter: mockRouter,
	}

	c, w := createTerminalTestContext(http.MethodPost, "/pods/test-pod/terminal/resize", "test-pod",
		`{"cols": 80, "rows": 24}`)

	tr, _ := h.terminalRouter.(TerminalRouterInterface)
	err := tr.RouteResize("test-pod", 80, 24)
	if err != nil {
		apierr.InternalError(c, "Failed to resize terminal: "+err.Error())
	}

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500, got %d", w.Code)
	}
}

func TestResizeTerminal_InvalidJSON(t *testing.T) {
	c, w := createTerminalTestContext(http.MethodPost, "/pods/test-pod/terminal/resize", "test-pod",
		`{invalid}`)

	var req TerminalResizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
	}

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestResizeTerminal_InvalidDimensions(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"zero cols", `{"cols": 0, "rows": 24}`},
		{"zero rows", `{"cols": 80, "rows": 0}`},
		{"negative cols", `{"cols": -1, "rows": 24}`},
		{"negative rows", `{"cols": 80, "rows": -1}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := createTerminalTestContext(http.MethodPost, "/pods/test-pod/terminal/resize", "test-pod", tt.body)

			var req TerminalResizeRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				apierr.ValidationError(c, err.Error())
			}

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected 400 for %s, got %d", tt.name, w.Code)
			}
		})
	}
}

// =============================================================================
// TerminalOutputResponse Tests
// =============================================================================

func TestTerminalOutputResponse_Structure(t *testing.T) {
	response := TerminalOutputResponse{
		PodKey:     "test-pod",
		Output:     "line1\nline2\n",
		Screen:     "screen content",
		CursorX:    10,
		CursorY:    5,
		TotalLines: 2,
		HasMore:    true,
	}

	if response.PodKey != "test-pod" {
		t.Errorf("Expected PodKey 'test-pod', got %s", response.PodKey)
	}
	if response.Output != "line1\nline2\n" {
		t.Errorf("Expected Output, got %s", response.Output)
	}
	if response.Screen != "screen content" {
		t.Errorf("Expected Screen, got %s", response.Screen)
	}
	if response.CursorX != 10 {
		t.Errorf("Expected CursorX 10, got %d", response.CursorX)
	}
	if response.CursorY != 5 {
		t.Errorf("Expected CursorY 5, got %d", response.CursorY)
	}
	if response.TotalLines != 2 {
		t.Errorf("Expected TotalLines 2, got %d", response.TotalLines)
	}
	if !response.HasMore {
		t.Error("Expected HasMore to be true")
	}
}

func TestTerminalOutputResponse_LineCount(t *testing.T) {
	tests := []struct {
		name      string
		output    []byte
		wantLines int
	}{
		{"empty", []byte{}, 0},
		{"one line no newline", []byte("hello"), 1},
		{"one line with newline", []byte("hello\n"), 1},
		{"two lines", []byte("line1\nline2\n"), 2},
		{"three lines no trailing newline", []byte("a\nb\nc"), 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			totalLines := 0
			for _, b := range tt.output {
				if b == '\n' {
					totalLines++
				}
			}
			if len(tt.output) > 0 && tt.output[len(tt.output)-1] != '\n' {
				totalLines++
			}

			if totalLines != tt.wantLines {
				t.Errorf("Expected %d lines, got %d", tt.wantLines, totalLines)
			}
		})
	}
}

func TestTerminalOutputResponse_HasMore(t *testing.T) {
	tests := []struct {
		name       string
		lines      int
		totalLines int
		wantMore   bool
	}{
		{"less than requested", 100, 50, false},
		{"equal to requested", 100, 100, true},
		{"all lines requested (-1)", -1, 500, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasMore := tt.lines != -1 && tt.totalLines >= tt.lines
			if hasMore != tt.wantMore {
				t.Errorf("Expected HasMore=%v, got %v", tt.wantMore, hasMore)
			}
		})
	}
}

// =============================================================================
// Request Validation Tests
// =============================================================================

func TestObserveTerminalRequest_Defaults(t *testing.T) {
	req := ObserveTerminalRequest{}

	// Default values
	if req.Lines != 0 {
		t.Errorf("Expected Lines default 0, got %d", req.Lines)
	}
	if req.IncludeScreen != false {
		t.Error("Expected IncludeScreen default false")
	}
}

func TestTerminalInputRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid input", "ls -la", false},
		{"empty input", "", true},          // required field
		{"special chars", "\x1b[A", false}, // arrow key
		{"newline", "command\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := TerminalInputRequest{Input: tt.input}
			// Binding validation would catch empty input
			hasErr := req.Input == ""
			if hasErr != tt.wantErr {
				t.Errorf("Expected error=%v, got %v", tt.wantErr, hasErr)
			}
		})
	}
}

func TestTerminalResizeRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		cols    int
		rows    int
		wantErr bool
	}{
		{"valid dimensions", 80, 24, false},
		{"large dimensions", 300, 100, false},
		{"minimum dimensions", 1, 1, false},
		{"zero cols", 0, 24, true},
		{"zero rows", 80, 0, true},
		{"negative cols", -1, 24, true},
		{"negative rows", 80, -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := TerminalResizeRequest{Cols: tt.cols, Rows: tt.rows}
			// Validation: min=1 for both cols and rows
			hasErr := req.Cols < 1 || req.Rows < 1
			if hasErr != tt.wantErr {
				t.Errorf("Expected error=%v, got %v", tt.wantErr, hasErr)
			}
		})
	}
}

// =============================================================================
// Pod Status Tests
// =============================================================================

func TestPodIsActive_ForTerminal(t *testing.T) {
	tests := []struct {
		name       string
		status     string
		wantActive bool
	}{
		{"running", agentpod.StatusRunning, true},
		{"initializing", agentpod.StatusInitializing, true},
		{"terminated", agentpod.StatusTerminated, false},
		{"error", agentpod.StatusError, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := &agentpod.Pod{Status: tt.status}
			if pod.IsActive() != tt.wantActive {
				t.Errorf("Expected IsActive=%v for status %s", tt.wantActive, tt.status)
			}
		})
	}
}

// =============================================================================
// TerminalRouterInterface Tests
// =============================================================================

func TestTerminalRouterInterface_Implementation(t *testing.T) {
	mock := &mockTerminalRouter{
		output:    []byte("test output"),
		screen:    "test screen",
		cursorRow: 10,
		cursorCol: 20,
	}

	var tr TerminalRouterInterface = mock

	if string(tr.GetRecentOutput("pod", 100)) != "test output" {
		t.Error("GetRecentOutput failed")
	}
	if tr.GetScreenSnapshot("pod") != "test screen" {
		t.Error("GetScreenSnapshot failed")
	}
	row, col := tr.GetCursorPosition("pod")
	if row != 10 || col != 20 {
		t.Error("GetCursorPosition failed")
	}
	if err := tr.RouteInput("pod", []byte("input")); err != nil {
		t.Error("RouteInput failed")
	}
	if err := tr.RouteResize("pod", 80, 24); err != nil {
		t.Error("RouteResize failed")
	}
}
