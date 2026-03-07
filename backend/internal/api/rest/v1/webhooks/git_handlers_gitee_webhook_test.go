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
// Gitee Webhook Tests
// ===========================================

func TestHandleGiteeWebhook_NoSecret(t *testing.T) {
	cfg := &config.Config{
		Webhook: config.WebhookConfig{
			GiteeSecret: "",
		},
	}
	router, _ := createTestRouterForGit(cfg)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	payload := `{"hook_name": "push_hooks"}`
	c.Request = httptest.NewRequest("POST", "/webhooks/gitee", bytes.NewReader([]byte(payload)))
	c.Request.Header.Set("Content-Type", "application/json")

	router.handleGiteeWebhook(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestHandleGiteeWebhook_InvalidSignature(t *testing.T) {
	cfg := &config.Config{
		Webhook: config.WebhookConfig{
			GiteeSecret: "test-secret",
		},
	}
	router, _ := createTestRouterForGit(cfg)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("POST", "/webhooks/gitee", bytes.NewReader([]byte(`{}`)))
	c.Request.Header.Set("Content-Type", "application/json")
	// No token header

	router.handleGiteeWebhook(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestHandleGiteeWebhook_ValidToken(t *testing.T) {
	cfg := &config.Config{
		Webhook: config.WebhookConfig{
			GiteeSecret: "test-secret",
		},
	}
	router, _ := createTestRouterForGit(cfg)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	payload := `{"hook_name": "push_hooks"}`
	c.Request = httptest.NewRequest("POST", "/webhooks/gitee", bytes.NewReader([]byte(payload)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-Gitee-Token", "test-secret")

	router.handleGiteeWebhook(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}
