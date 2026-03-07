package webhooks

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/service/repository"
	"github.com/gin-gonic/gin"
)

// ===========================================
// handleGitLabWebhookWithRepo Tests
// ===========================================

func TestHandleGitLabWebhookWithRepo_InvalidRepoID(t *testing.T) {
	cfg := &config.Config{}
	router, _, _ := createTestRouterForWithRepo(t, cfg)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = gin.Params{
		{Key: "org_slug", Value: "test-org"},
		{Key: "repo_id", Value: "not-a-number"},
	}
	c.Request = httptest.NewRequest("POST", "/webhooks/test-org/gitlab/not-a-number", bytes.NewReader([]byte(`{}`)))
	c.Request.Header.Set("Content-Type", "application/json")

	router.handleGitLabWebhookWithRepo(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleGitLabWebhookWithRepo_ValidGlobalSecret(t *testing.T) {
	cfg := &config.Config{
		Webhook: config.WebhookConfig{
			GitLabSecret: "test-global-secret",
		},
	}
	router, db, _ := createTestRouterForWithRepo(t, cfg)

	// Create test repository
	db.Exec(`INSERT INTO repositories (id, organization_id, provider_type, provider_base_url, external_id, name, full_path)
		VALUES (123, 1, 'gitlab', 'https://gitlab.com', '456', 'test-repo', 'org/test-repo')`)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = gin.Params{
		{Key: "org_slug", Value: "test-org"},
		{Key: "repo_id", Value: "123"},
	}

	payload := `{"object_kind": "push", "project": {"id": 456}, "ref": "refs/heads/main"}`
	c.Request = httptest.NewRequest("POST", "/webhooks/test-org/gitlab/123", bytes.NewReader([]byte(payload)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-Gitlab-Token", "test-global-secret")

	router.handleGitLabWebhookWithRepo(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestHandleGitLabWebhookWithRepo_Unauthorized(t *testing.T) {
	cfg := &config.Config{
		Webhook: config.WebhookConfig{
			GitLabSecret: "correct-secret",
		},
	}
	router, db, _ := createTestRouterForWithRepo(t, cfg)

	// Create test repository
	db.Exec(`INSERT INTO repositories (id, organization_id, provider_type, provider_base_url, external_id, name, full_path)
		VALUES (123, 1, 'gitlab', 'https://gitlab.com', '456', 'test-repo', 'org/test-repo')`)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = gin.Params{
		{Key: "org_slug", Value: "test-org"},
		{Key: "repo_id", Value: "123"},
	}

	c.Request = httptest.NewRequest("POST", "/webhooks/test-org/gitlab/123", bytes.NewReader([]byte(`{}`)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-Gitlab-Token", "wrong-secret")

	router.handleGitLabWebhookWithRepo(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestHandleGitLabWebhookWithRepo_NoSecretConfigured(t *testing.T) {
	cfg := &config.Config{
		Webhook: config.WebhookConfig{
			GitLabSecret: "", // No secret configured
		},
	}
	router, db, _ := createTestRouterForWithRepo(t, cfg)

	// Create test repository
	db.Exec(`INSERT INTO repositories (id, organization_id, provider_type, provider_base_url, external_id, name, full_path)
		VALUES (123, 1, 'gitlab', 'https://gitlab.com', '456', 'test-repo', 'org/test-repo')`)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = gin.Params{
		{Key: "org_slug", Value: "test-org"},
		{Key: "repo_id", Value: "123"},
	}

	payload := `{"object_kind": "push", "project": {"id": 456}}`
	c.Request = httptest.NewRequest("POST", "/webhooks/test-org/gitlab/123", bytes.NewReader([]byte(payload)))
	c.Request.Header.Set("Content-Type", "application/json")

	router.handleGitLabWebhookWithRepo(c)

	// Without any secret configured, should reject with 401
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d: %s", http.StatusUnauthorized, w.Code, w.Body.String())
	}
}

// ===========================================
// Additional handleGitLabWebhookWithRepo Tests
// ===========================================

func TestHandleGitLabWebhookWithRepo_WithWebhookService(t *testing.T) {
	cfg := &config.Config{
		Webhook: config.WebhookConfig{
			GitLabSecret: "global-secret",
		},
	}
	db := setupTestDBForWithRepo(t)
	logger := testLoggerForGit()
	registry := NewHandlerRegistry(logger)
	SetupDefaultHandlers(registry, logger)

	repoSvc := repository.NewService(db)
	webhookSvc := repository.NewWebhookService(db, cfg, nil, nil)

	router := &WebhookRouter{
		db:             db,
		cfg:            cfg,
		logger:         logger,
		registry:       registry,
		repoService:    repoSvc,
		webhookService: webhookSvc,
	}

	// Create test repository without webhook_config (SQLite can't handle JSONB properly)
	db.Exec(`INSERT INTO repositories (id, organization_id, provider_type, provider_base_url, external_id, name, full_path)
		VALUES (123, 1, 'gitlab', 'https://gitlab.com', '456', 'test-repo', 'org/test-repo')`)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = gin.Params{
		{Key: "org_slug", Value: "test-org"},
		{Key: "repo_id", Value: "123"},
	}

	payload := `{"object_kind": "push", "project": {"id": 456}, "ref": "refs/heads/main"}`
	c.Request = httptest.NewRequest("POST", "/webhooks/test-org/gitlab/123", bytes.NewReader([]byte(payload)))
	c.Request.Header.Set("Content-Type", "application/json")
	// Use global secret - repo secret verification will fail (no webhook_config)
	// and fallback to global secret
	c.Request.Header.Set("X-Gitlab-Token", "global-secret")

	router.handleGitLabWebhookWithRepo(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestHandleGitLabWebhookWithRepo_WebhookServiceReturnsError(t *testing.T) {
	cfg := &config.Config{
		Webhook: config.WebhookConfig{
			GitLabSecret: "", // No global secret, must use repo-specific
		},
	}
	db := setupTestDBForWithRepo(t)
	logger := testLoggerForGit()
	registry := NewHandlerRegistry(logger)
	SetupDefaultHandlers(registry, logger)

	repoSvc := repository.NewService(db)
	webhookSvc := repository.NewWebhookService(db, cfg, nil, nil)

	router := &WebhookRouter{
		db:             db,
		cfg:            cfg,
		logger:         logger,
		registry:       registry,
		repoService:    repoSvc,
		webhookService: webhookSvc,
	}

	// Create test repository without webhook config
	db.Exec(`INSERT INTO repositories (id, organization_id, provider_type, provider_base_url, external_id, name, full_path)
		VALUES (123, 1, 'gitlab', 'https://gitlab.com', '456', 'test-repo', 'org/test-repo')`)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = gin.Params{
		{Key: "org_slug", Value: "test-org"},
		{Key: "repo_id", Value: "123"},
	}

	payload := `{"object_kind": "push", "project": {"id": 456}}`
	c.Request = httptest.NewRequest("POST", "/webhooks/test-org/gitlab/123", bytes.NewReader([]byte(payload)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-Gitlab-Token", "wrong-token")

	router.handleGitLabWebhookWithRepo(c)

	// Without any valid secret, should fail
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d: %s", http.StatusUnauthorized, w.Code, w.Body.String())
	}
}
