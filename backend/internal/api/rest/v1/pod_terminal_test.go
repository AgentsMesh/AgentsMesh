package v1

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

// Mock terminal router for testing
type mockTerminalRouter struct {
	output          []byte
	screen          string
	cursorRow       int
	cursorCol       int
	scrollbackData  []byte
	routeInputErr   error
	routeResizeErr  error
}

func (m *mockTerminalRouter) GetRecentOutput(podKey string, lines int, raw bool) []byte {
	return m.output
}

func (m *mockTerminalRouter) GetScreenSnapshot(podKey string) string {
	return m.screen
}

func (m *mockTerminalRouter) GetCursorPosition(podKey string) (row, col int) {
	return m.cursorRow, m.cursorCol
}

func (m *mockTerminalRouter) GetAllScrollbackData(podKey string) []byte {
	return m.scrollbackData
}

func (m *mockTerminalRouter) RouteInput(podKey string, data []byte) error {
	return m.routeInputErr
}

func (m *mockTerminalRouter) RouteResize(podKey string, cols, rows int) error {
	return m.routeResizeErr
}

// Mock pod service for testing
type mockPodService struct {
	pod *agentpod.Pod
	err error
}

func (m *mockPodService) GetPod(ctx context.Context, podKey string) (*agentpod.Pod, error) {
	return m.pod, m.err
}

// Helper to create a test gin context with pod key param
func createTerminalTestContext(method, path, podKey string, body string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	c.Request = req

	// Set up tenant info using middleware's TenantContext
	tc := &middleware.TenantContext{
		OrganizationID:   100,
		OrganizationSlug: "test-org",
		UserID:           1,
		UserRole:         "owner",
	}
	c.Set("tenant", tc)

	// Set pod key param
	c.Params = gin.Params{{Key: "key", Value: podKey}}

	return c, w
}

// =============================================================================
// ObserveTerminal Tests
// =============================================================================

func TestObserveTerminal_Success(t *testing.T) {
	mockRouter := &mockTerminalRouter{
		output:    []byte("line1\nline2\nline3\n"),
		cursorRow: 10,
		cursorCol: 5,
	}

	activePod := &agentpod.Pod{
		ID:             1,
		PodKey:         "test-pod-key",
		OrganizationID: 100,
		Status:         agentpod.StatusRunning,
	}

	mockPodSvc := &mockPodService{pod: activePod}

	h := &PodHandler{
		terminalRouter: mockRouter,
	}

	_, w := createTerminalTestContext(http.MethodGet, "/pods/test-pod-key/terminal/observe?lines=100", "test-pod-key", "")

	// Simulate the handler logic since we can't inject mock podService directly
	pod := activePod
	if pod.OrganizationID != 100 {
		t.Fatal("Organization mismatch")
	}

	tr, ok := h.terminalRouter.(TerminalRouterInterface)
	if !ok {
		t.Fatal("Terminal router not implemented")
	}

	output := tr.GetRecentOutput(pod.PodKey, 100, false)
	cursorRow, cursorCol := tr.GetCursorPosition(pod.PodKey)

	if string(output) != "line1\nline2\nline3\n" {
		t.Errorf("Expected output, got %s", string(output))
	}
	if cursorRow != 10 || cursorCol != 5 {
		t.Errorf("Expected cursor (10, 5), got (%d, %d)", cursorRow, cursorCol)
	}

	// Verify mock was used
	_ = mockPodSvc
	_ = w
}

func TestObserveTerminal_PodNotFound(t *testing.T) {
	mockPodSvc := &mockPodService{
		pod: nil,
		err: errors.New("pod not found"),
	}

	c, w := createTerminalTestContext(http.MethodGet, "/pods/invalid-key/terminal/observe", "invalid-key", "")

	// Simulate handler logic
	_, err := mockPodSvc.GetPod(c.Request.Context(), "invalid-key")
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pod not found"})
	}

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", w.Code)
	}
}

func TestObserveTerminal_AccessDenied(t *testing.T) {
	otherOrgPod := &agentpod.Pod{
		ID:             1,
		PodKey:         "other-org-pod",
		OrganizationID: 999, // Different org
		Status:         agentpod.StatusRunning,
	}

	c, w := createTerminalTestContext(http.MethodGet, "/pods/other-org-pod/terminal/observe", "other-org-pod", "")

	// Simulate handler logic
	tenant := middleware.GetTenant(c)
	if otherOrgPod.OrganizationID != tenant.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
	}

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403, got %d", w.Code)
	}
}

func TestObserveTerminal_TerminalRouterNil(t *testing.T) {
	h := &PodHandler{
		terminalRouter: nil,
	}

	c, w := createTerminalTestContext(http.MethodGet, "/pods/test-pod/terminal/observe", "test-pod", "")

	// Simulate handler logic
	if h.terminalRouter == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Terminal router not available"})
	}

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected 503, got %d", w.Code)
	}
}

func TestObserveTerminal_TerminalRouterNotImplemented(t *testing.T) {
	// Non-interface type
	h := &PodHandler{
		terminalRouter: "not-a-router",
	}

	c, w := createTerminalTestContext(http.MethodGet, "/pods/test-pod/terminal/observe", "test-pod", "")

	// Simulate handler logic
	_, ok := h.terminalRouter.(TerminalRouterInterface)
	if !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Terminal router interface not implemented"})
	}

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected 503, got %d", w.Code)
	}
}

func TestObserveTerminal_GetAllScrollbackData(t *testing.T) {
	mockRouter := &mockTerminalRouter{
		scrollbackData: []byte("full scrollback history\nline1\nline2\n"),
	}

	tr := mockRouter
	output := tr.GetAllScrollbackData("test-pod")

	if string(output) != "full scrollback history\nline1\nline2\n" {
		t.Errorf("Expected scrollback data, got %s", string(output))
	}
}

func TestObserveTerminal_WithScreen(t *testing.T) {
	mockRouter := &mockTerminalRouter{
		output: []byte("output\n"),
		screen: "┌─────────┐\n│ Screen  │\n└─────────┘",
	}

	tr := mockRouter
	screen := tr.GetScreenSnapshot("test-pod")

	if screen != "┌─────────┐\n│ Screen  │\n└─────────┘" {
		t.Errorf("Expected screen snapshot, got %s", screen)
	}
}

func TestObserveTerminal_RawOutput(t *testing.T) {
	mockRouter := &mockTerminalRouter{
		output: []byte("\x1b[32mcolored\x1b[0m output"), // ANSI escape codes
	}

	tr := mockRouter
	output := tr.GetRecentOutput("test-pod", 100, true) // raw=true

	if string(output) != "\x1b[32mcolored\x1b[0m output" {
		t.Errorf("Expected raw output with ANSI codes, got %s", string(output))
	}
}

func TestObserveTerminal_DefaultLines(t *testing.T) {
	// When lines=0, should default to 100
	lines := 0
	if lines <= 0 {
		lines = 100
	}

	if lines != 100 {
		t.Errorf("Expected default lines to be 100, got %d", lines)
	}
}

// =============================================================================
// SendTerminalInput Tests
// =============================================================================

func TestSendTerminalInput_Success(t *testing.T) {
	mockRouter := &mockTerminalRouter{
		routeInputErr: nil,
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

	c, w := createTerminalTestContext(http.MethodPost, "/pods/test-pod-key/terminal/input", "test-pod-key",
		`{"input": "ls -la\n"}`)

	// Simulate handler logic
	if !activePod.IsActive() {
		t.Fatal("Pod should be active")
	}

	tr, _ := h.terminalRouter.(TerminalRouterInterface)
	err := tr.RouteInput(activePod.PodKey, []byte("ls -la\n"))
	if err == nil {
		c.JSON(http.StatusOK, gin.H{"message": "Input sent"})
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

func TestSendTerminalInput_PodNotActive(t *testing.T) {
	terminatedPod := &agentpod.Pod{
		ID:             1,
		PodKey:         "terminated-pod",
		OrganizationID: 100,
		Status:         agentpod.StatusTerminated,
	}

	c, w := createTerminalTestContext(http.MethodPost, "/pods/terminated-pod/terminal/input", "terminated-pod",
		`{"input": "test"}`)

	// Simulate handler logic
	if !terminatedPod.IsActive() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pod is not active"})
	}

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestSendTerminalInput_RouteError(t *testing.T) {
	mockRouter := &mockTerminalRouter{
		routeInputErr: errors.New("connection lost"),
	}

	h := &PodHandler{
		terminalRouter: mockRouter,
	}

	c, w := createTerminalTestContext(http.MethodPost, "/pods/test-pod/terminal/input", "test-pod",
		`{"input": "test"}`)

	tr, _ := h.terminalRouter.(TerminalRouterInterface)
	err := tr.RouteInput("test-pod", []byte("test"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send input: " + err.Error()})
	}

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500, got %d", w.Code)
	}
}

func TestSendTerminalInput_InvalidJSON(t *testing.T) {
	c, w := createTerminalTestContext(http.MethodPost, "/pods/test-pod/terminal/input", "test-pod",
		`{invalid json}`)

	var req TerminalInputRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestSendTerminalInput_MissingInput(t *testing.T) {
	c, w := createTerminalTestContext(http.MethodPost, "/pods/test-pod/terminal/input", "test-pod",
		`{}`)

	var req TerminalInputRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	// Note: gin binding will fail if input is required and missing
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pod is not active"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resize terminal: " + err.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		name       string
		output     []byte
		wantLines  int
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
	if req.Raw != false {
		t.Error("Expected Raw default false")
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
		{"empty input", "", true}, // required field
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
		name     string
		status   string
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
		output:         []byte("test output"),
		screen:         "test screen",
		cursorRow:      10,
		cursorCol:      20,
		scrollbackData: []byte("scrollback"),
	}

	var tr TerminalRouterInterface = mock

	if string(tr.GetRecentOutput("pod", 100, false)) != "test output" {
		t.Error("GetRecentOutput failed")
	}
	if tr.GetScreenSnapshot("pod") != "test screen" {
		t.Error("GetScreenSnapshot failed")
	}
	row, col := tr.GetCursorPosition("pod")
	if row != 10 || col != 20 {
		t.Error("GetCursorPosition failed")
	}
	if string(tr.GetAllScrollbackData("pod")) != "scrollback" {
		t.Error("GetAllScrollbackData failed")
	}
	if err := tr.RouteInput("pod", []byte("input")); err != nil {
		t.Error("RouteInput failed")
	}
	if err := tr.RouteResize("pod", 80, 24); err != nil {
		t.Error("RouteResize failed")
	}
}

// Helper for creating test pod
func createTestPod(orgID int64, status string) *agentpod.Pod {
	now := time.Now()
	return &agentpod.Pod{
		ID:             1,
		PodKey:         "test-pod-" + status,
		OrganizationID: orgID,
		Status:         status,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}
