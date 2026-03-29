package webhooks

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/service/repository"
	"github.com/gin-gonic/gin"
)

// ===========================================
// handleGitHubWebhookWithRepo Tests
// ===========================================

func TestHandleGitHubWebhookWithRepo_InvalidRepoID(t *testing.T) {
	cfg := &config.Config{}
	router, _, _ := createTestRouterForWithRepo(t, cfg)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = gin.Params{
		{Key: "org_slug", Value: "test-org"},
		{Key: "repo_id", Value: "invalid"},
	}
	c.Request = httptest.NewRequest("POST", "/webhooks/test-org/github/invalid", bytes.NewReader([]byte(`{}`)))
	c.Request.Header.Set("Content-Type", "application/json")

	router.handleGitHubWebhookWithRepo(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleGitHubWebhookWithRepo_NoSecretConfigured(t *testing.T) {
	cfg := &config.Config{
		Webhook: config.WebhookConfig{
			GitHubSecret: "",
		},
	}
	router, db, _ := createTestRouterForWithRepo(t, cfg)

	// Create test repository
	db.Exec(`INSERT INTO repositories (id, organization_id, provider_type, provider_base_url, external_id, name, slug)
		VALUES (200, 1, 'github', 'https://github.com', '789', 'test-repo', 'org/test-repo')`)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = gin.Params{
		{Key: "org_slug", Value: "test-org"},
		{Key: "repo_id", Value: "200"},
	}

	payload := `{"action": "opened", "repository": {"id": 789}}`
	c.Request = httptest.NewRequest("POST", "/webhooks/test-org/github/200", bytes.NewReader([]byte(payload)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-GitHub-Event", "push")

	router.handleGitHubWebhookWithRepo(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d: %s", http.StatusUnauthorized, w.Code, w.Body.String())
	}
}

// ===========================================
// Additional handleGitHubWebhookWithRepo Tests
// ===========================================

func TestHandleGitHubWebhookWithRepo_WithWebhookService(t *testing.T) {
	cfg := &config.Config{
		Webhook: config.WebhookConfig{
			GitHubSecret: "",
		},
	}
	db := setupTestDBForWithRepo(t)
	logger := testLoggerForGit()
	registry := NewHandlerRegistry(logger)
	SetupDefaultHandlers(registry, logger)

	gitRepo := infra.NewGitProviderRepository(db)
	repoSvc := repository.NewService(gitRepo)
	webhookSvc := repository.NewWebhookService(gitRepo, cfg, nil, nil)

	router := &WebhookRouter{
		db:             db,
		cfg:            cfg,
		logger:         logger,
		registry:       registry,
		repoService:    repoSvc,
		webhookService: webhookSvc,
	}

	// Create test repository
	db.Exec(`INSERT INTO repositories (id, organization_id, provider_type, provider_base_url, external_id, name, slug)
		VALUES (200, 1, 'github', 'https://github.com', '789', 'test-repo', 'org/test-repo')`)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = gin.Params{
		{Key: "org_slug", Value: "test-org"},
		{Key: "repo_id", Value: "200"},
	}

	payload := `{"action": "opened", "repository": {"id": 789}}`
	c.Request = httptest.NewRequest("POST", "/webhooks/test-org/github/200", bytes.NewReader([]byte(payload)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-GitHub-Event", "push")

	router.handleGitHubWebhookWithRepo(c)

	// No secret configured, should reject with 401
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d: %s", http.StatusUnauthorized, w.Code, w.Body.String())
	}
}

func TestHandleGitHubWebhookWithRepo_InvalidSignature(t *testing.T) {
	cfg := &config.Config{
		Webhook: config.WebhookConfig{
			GitHubSecret: "github-secret",
		},
	}
	db := setupTestDBForWithRepo(t)
	logger := testLoggerForGit()
	registry := NewHandlerRegistry(logger)
	SetupDefaultHandlers(registry, logger)

	gitRepo2 := infra.NewGitProviderRepository(db)
	repoSvc := repository.NewService(gitRepo2)

	router := &WebhookRouter{
		db:          db,
		cfg:         cfg,
		logger:      logger,
		registry:    registry,
		repoService: repoSvc,
	}

	// Create test repository
	db.Exec(`INSERT INTO repositories (id, organization_id, provider_type, provider_base_url, external_id, name, slug)
		VALUES (200, 1, 'github', 'https://github.com', '789', 'test-repo', 'org/test-repo')`)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = gin.Params{
		{Key: "org_slug", Value: "test-org"},
		{Key: "repo_id", Value: "200"},
	}

	payload := `{"action": "opened"}`
	c.Request = httptest.NewRequest("POST", "/webhooks/test-org/github/200", bytes.NewReader([]byte(payload)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-Hub-Signature-256", "sha256=invalid-signature")

	router.handleGitHubWebhookWithRepo(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d: %s", http.StatusUnauthorized, w.Code, w.Body.String())
	}
}
