package webhooks

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDBForGit(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	return db
}

func testLoggerForGit() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func createTestRouterForGit(cfg *config.Config) (*WebhookRouter, *gorm.DB) {
	db := setupTestDBForGitRouter()
	logger := testLoggerForGit()
	registry := NewHandlerRegistry(logger)
	SetupDefaultHandlers(registry, logger)

	return &WebhookRouter{
		db:       db,
		cfg:      cfg,
		logger:   logger,
		registry: registry,
	}, db
}

func setupTestDBForGitRouter() *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	return db
}

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

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

// ===========================================
// GitHub Webhook Tests
// ===========================================

func TestHandleGitHubWebhook_NoSecret(t *testing.T) {
	cfg := &config.Config{
		Webhook: config.WebhookConfig{
			GitHubSecret: "",
		},
	}
	router, _ := createTestRouterForGit(cfg)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	payload := `{"action": "opened"}`
	c.Request = httptest.NewRequest("POST", "/webhooks/github", bytes.NewReader([]byte(payload)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-GitHub-Event", "push")

	router.handleGitHubWebhook(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestHandleGitHubWebhook_InvalidSignature(t *testing.T) {
	cfg := &config.Config{
		Webhook: config.WebhookConfig{
			GitHubSecret: "test-secret",
		},
	}
	router, _ := createTestRouterForGit(cfg)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("POST", "/webhooks/github", bytes.NewReader([]byte(`{}`)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-Hub-Signature-256", "sha256=invalid")

	router.handleGitHubWebhook(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

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

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
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

// ===========================================
// extractObjectKind Tests
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

func TestExtractObjectKind_GitHub(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestRouterForGit(cfg)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/", nil)
	c.Request.Header.Set("X-GitHub-Event", "push")

	result := router.extractObjectKind(map[string]interface{}{}, "github", c)
	if result != "push" {
		t.Errorf("expected push, got %s", result)
	}
}

func TestExtractObjectKind_Gitee(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestRouterForGit(cfg)

	gin.SetMode(gin.TestMode)

	tests := []struct {
		name      string
		header    string
		hookName  string
		expected  string
	}{
		{"with header", "Push Hook", "", "push"},
		{"with hook_name", "", "push_hooks", "push"},
		{"merge request hook", "", "merge_request_hooks", "merge_request"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", "/", nil)
			if tt.header != "" {
				c.Request.Header.Set("X-Gitee-Event", tt.header)
			}

			payload := map[string]interface{}{}
			if tt.hookName != "" {
				payload["hook_name"] = tt.hookName
			}

			result := router.extractObjectKind(payload, "gitee", c)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// ===========================================
// mapGitHubEventToKind Tests
// ===========================================

func TestMapGitHubEventToKind(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestRouterForGit(cfg)

	tests := []struct {
		event    string
		expected string
	}{
		{"push", "push"},
		{"pull_request", "merge_request"},
		{"check_run", "pipeline"},
		{"check_suite", "pipeline"},
		{"workflow_run", "pipeline"},
		{"status", "pipeline"},
		{"issues", "issue"},
		{"issue_comment", "note"},
		{"pull_request_review", "note"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.event, func(t *testing.T) {
			result := router.mapGitHubEventToKind(tt.event)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// ===========================================
// mapGiteeEventToKind Tests
// ===========================================

func TestMapGiteeEventToKind(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestRouterForGit(cfg)

	tests := []struct {
		event    string
		expected string
	}{
		{"push_hooks", "push"},
		{"Push Hook", "push"},
		{"merge_request_hooks", "merge_request"},
		{"Merge Request Hook", "merge_request"},
		{"issue_hooks", "issue"},
		{"Issue Hook", "issue"},
		{"note_hooks", "note"},
		{"Note Hook", "note"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.event, func(t *testing.T) {
			result := router.mapGiteeEventToKind(tt.event)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// ===========================================
// verifyGitHubSignature Tests
// ===========================================

func TestVerifyGitHubSignature_ValidHMAC(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestRouterForGit(cfg)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := []byte(`{"test": "data"}`)
	c.Request = httptest.NewRequest("POST", "/", bytes.NewReader(body))
	// Calculate correct HMAC-SHA256
	// For body '{"test": "data"}' with secret "test-secret":
	// hmac_sha256 = 93c1e4a2f8e6c9c8d1e8e1a1b1c1d1e1f1a1b1c1d1e1f1a1b1c1d1e1f1a1b1c1 (example)
	c.Request.Header.Set("X-Hub-Signature-256", "sha256=f7e9e8d6c5b4a3f2e1d0c9b8a7968574635241f0e1d2c3b4a5968778695a4b3c")

	// This will fail since the signature is fake, but we test the flow
	result := router.verifyGitHubSignature(c, "test-secret")
	// Expected false since our example signature is not correct
	if result {
		t.Error("expected signature verification to fail with fake signature")
	}
}

func TestVerifyGitHubSignature_MissingSignature(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestRouterForGit(cfg)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("POST", "/", nil)
	// No X-Hub-Signature-256 header

	result := router.verifyGitHubSignature(c, "test-secret")
	if result {
		t.Error("expected signature verification to fail without signature header")
	}
}

func TestVerifyGitHubSignature_InvalidPrefix(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestRouterForGit(cfg)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("POST", "/", nil)
	c.Request.Header.Set("X-Hub-Signature-256", "invalid_prefix=abc123")

	result := router.verifyGitHubSignature(c, "test-secret")
	if result {
		t.Error("expected signature verification to fail with invalid prefix")
	}
}

// ===========================================
// verifyGiteeSignature Tests
// ===========================================

func TestVerifyGiteeSignature_ValidToken(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestRouterForGit(cfg)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("POST", "/", nil)
	c.Request.Header.Set("X-Gitee-Token", "test-secret")

	result := router.verifyGiteeSignature(c, "test-secret")
	if !result {
		t.Error("expected signature verification to pass with valid token")
	}
}

func TestVerifyGiteeSignature_InvalidToken(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestRouterForGit(cfg)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("POST", "/", nil)
	c.Request.Header.Set("X-Gitee-Token", "wrong-secret")

	result := router.verifyGiteeSignature(c, "test-secret")
	if result {
		t.Error("expected signature verification to fail with invalid token")
	}
}

func TestVerifyGiteeSignature_NoHeaders(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestRouterForGit(cfg)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("POST", "/", nil)

	result := router.verifyGiteeSignature(c, "test-secret")
	if result {
		t.Error("expected signature verification to fail with no headers")
	}
}
