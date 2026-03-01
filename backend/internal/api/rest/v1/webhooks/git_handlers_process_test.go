package webhooks

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/config"
	"github.com/gin-gonic/gin"
)

// ===========================================
// processWebhook Tests
// ===========================================

func TestProcessWebhook_InvalidJSON(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestRouterForGit(cfg)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("POST", "/webhooks/test", bytes.NewReader([]byte(`invalid json`)))
	c.Request.Header.Set("Content-Type", "application/json")

	router.processWebhook(c, "test")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestProcessWebhook_BuildToJob(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestRouterForGit(cfg)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// GitLab legacy: "build" should be converted to "job"
	payload := `{"object_kind": "build"}`
	c.Request = httptest.NewRequest("POST", "/webhooks/gitlab", bytes.NewReader([]byte(payload)))
	c.Request.Header.Set("Content-Type", "application/json")

	router.processWebhook(c, "gitlab")

	// Should not fail - job is a valid event type
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}
