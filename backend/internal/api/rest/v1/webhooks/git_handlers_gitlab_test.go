package webhooks

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/gin-gonic/gin"
)

// ===========================================
// GitLab Webhook Tests
// ===========================================

func TestHandleGitLabWebhook_ValidToken(t *testing.T) {
	cfg := &config.Config{
		Webhook: config.WebhookConfig{
			GitLabSecret: "test-secret",
		},
	}
	router, _ := createTestRouterForGit(cfg)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	payload := `{"object_kind": "pipeline", "object_attributes": {"id": 123, "status": "success"}}`
	c.Request = httptest.NewRequest("POST", "/webhooks/gitlab", bytes.NewReader([]byte(payload)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-Gitlab-Token", "test-secret")

	router.handleGitLabWebhook(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestHandleGitLabWebhook_InvalidToken(t *testing.T) {
	cfg := &config.Config{
		Webhook: config.WebhookConfig{
			GitLabSecret: "test-secret",
		},
	}
	router, _ := createTestRouterForGit(cfg)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("POST", "/webhooks/gitlab", bytes.NewReader([]byte(`{}`)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-Gitlab-Token", "wrong-secret")

	router.handleGitLabWebhook(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestHandleGitLabWebhook_NoSecret(t *testing.T) {
	cfg := &config.Config{
		Webhook: config.WebhookConfig{
			GitLabSecret: "",
		},
	}
	router, _ := createTestRouterForGit(cfg)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	payload := `{"object_kind": "push", "ref": "refs/heads/main"}`
	c.Request = httptest.NewRequest("POST", "/webhooks/gitlab", bytes.NewReader([]byte(payload)))
	c.Request.Header.Set("Content-Type", "application/json")

	router.handleGitLabWebhook(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

// ===========================================
// extractObjectKind Tests for GitLab
// ===========================================

func TestExtractObjectKind_GitLab(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestRouterForGit(cfg)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/", nil)

	tests := []struct {
		name     string
		payload  map[string]interface{}
		expected string
	}{
		{
			name:     "pipeline event",
			payload:  map[string]interface{}{"object_kind": "pipeline"},
			expected: "pipeline",
		},
		{
			name:     "push event",
			payload:  map[string]interface{}{"object_kind": "push"},
			expected: "push",
		},
		{
			name:     "no object_kind",
			payload:  map[string]interface{}{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := router.extractObjectKind(tt.payload, "gitlab", c)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}
