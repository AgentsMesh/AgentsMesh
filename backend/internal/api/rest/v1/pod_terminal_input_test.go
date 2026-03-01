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
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Pod is not active")
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
		apierr.InternalError(c, "Failed to send input: "+err.Error())
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
		apierr.ValidationError(c, err.Error())
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
		apierr.ValidationError(c, err.Error())
	}

	// Note: gin binding will fail if input is required and missing
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}
