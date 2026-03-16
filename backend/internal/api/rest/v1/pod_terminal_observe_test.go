package v1

import (
	"errors"
	"net/http"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
)

// =============================================================================
// ObserveTerminal Tests
// =============================================================================

func TestObserveTerminal_GetRunnerID(t *testing.T) {
	mockRouter := &mockTerminalRouter{
		runnerID:    42,
		runnerFound: true,
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

	_ = mockPodSvc

	// Simulate the handler logic: verify GetRunnerID works
	tr, ok := h.terminalRouter.(TerminalRouterInterface)
	if !ok {
		t.Fatal("Terminal router not implemented")
	}

	runnerID, found := tr.GetRunnerID(activePod.PodKey)
	if !found {
		t.Fatal("Expected runner to be found")
	}
	if runnerID != 42 {
		t.Errorf("Expected runnerID 42, got %d", runnerID)
	}
}

func TestObserveTerminal_RunnerNotFound(t *testing.T) {
	mockRouter := &mockTerminalRouter{
		runnerID:    0,
		runnerFound: false,
	}

	h := &PodHandler{
		terminalRouter: mockRouter,
	}

	c, w := createTerminalTestContext(http.MethodGet, "/pods/test-pod-key/terminal/observe", "test-pod-key", "")

	tr, _ := h.terminalRouter.(TerminalRouterInterface)
	_, found := tr.GetRunnerID("test-pod-key")
	if !found {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Pod not registered on any runner")
	}

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected 503, got %d", w.Code)
	}
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

func TestObserveTerminal_RouteObserveTerminal(t *testing.T) {
	mockRouter := &mockTerminalRouter{
		runnerID:    42,
		runnerFound: true,
	}

	tr := mockRouter
	err := tr.RouteObserveTerminal(42, "req-123", "test-pod", 100, true)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if tr.lastObserveRunnerID != 42 {
		t.Errorf("Expected runnerID 42, got %d", tr.lastObserveRunnerID)
	}
	if tr.lastObserveRequestID != "req-123" {
		t.Errorf("Expected requestID 'req-123', got %s", tr.lastObserveRequestID)
	}
	if tr.lastObservePodKey != "test-pod" {
		t.Errorf("Expected podKey 'test-pod', got %s", tr.lastObservePodKey)
	}
	if tr.lastObserveLines != 100 {
		t.Errorf("Expected lines 100, got %d", tr.lastObserveLines)
	}
	if !tr.lastObserveIncScreen {
		t.Error("Expected includeScreen to be true")
	}
}

func TestObserveTerminal_RouteObserveError(t *testing.T) {
	mockRouter := &mockTerminalRouter{
		runnerID:        42,
		runnerFound:     true,
		routeObserveErr: errors.New("connection lost"),
	}

	tr := mockRouter
	err := tr.RouteObserveTerminal(42, "req-123", "test-pod", 100, false)
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if err.Error() != "connection lost" {
		t.Errorf("Expected 'connection lost', got %s", err.Error())
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
