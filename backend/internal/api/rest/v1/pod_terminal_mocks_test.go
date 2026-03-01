package v1

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/agentpod"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

// Mock terminal router for testing
type mockTerminalRouter struct {
	output         []byte
	screen         string
	cursorRow      int
	cursorCol      int
	routeInputErr  error
	routeResizeErr error
}

func (m *mockTerminalRouter) GetRecentOutput(podKey string, lines int) []byte {
	return m.output
}

func (m *mockTerminalRouter) GetScreenSnapshot(podKey string) string {
	return m.screen
}

func (m *mockTerminalRouter) GetCursorPosition(podKey string) (row, col int) {
	return m.cursorRow, m.cursorCol
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
