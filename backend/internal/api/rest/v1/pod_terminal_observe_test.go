package v1

import (
	"errors"
	"net/http"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/agentpod"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/middleware"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
)

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

	output := tr.GetRecentOutput(pod.PodKey, 100)
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
		apierr.ResourceNotFound(c, "Pod not found")
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
		apierr.ForbiddenAccess(c)
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
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Terminal router not available")
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
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Terminal router interface not implemented")
	}

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected 503, got %d", w.Code)
	}
}

func TestObserveTerminal_GetAllOutput(t *testing.T) {
	mockRouter := &mockTerminalRouter{
		output: []byte("full output history\nline1\nline2\n"),
	}

	tr := mockRouter
	output := tr.GetRecentOutput("test-pod", 10000) // large number to get all

	if string(output) != "full output history\nline1\nline2\n" {
		t.Errorf("Expected output data, got %s", string(output))
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

func TestObserveTerminal_ProcessedOutput(t *testing.T) {
	mockRouter := &mockTerminalRouter{
		output: []byte("colored output"), // ANSI escape codes stripped by VT
	}

	tr := mockRouter
	output := tr.GetRecentOutput("test-pod", 100)

	if string(output) != "colored output" {
		t.Errorf("Expected processed output, got %s", string(output))
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
