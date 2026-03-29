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
// handleGiteeWebhookWithRepo Tests
// ===========================================

func TestHandleGiteeWebhookWithRepo_InvalidRepoID(t *testing.T) {
	cfg := &config.Config{}
	router, _, _ := createTestRouterForWithRepo(t, cfg)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = gin.Params{
		{Key: "org_slug", Value: "test-org"},
		{Key: "repo_id", Value: "abc"},
	}
	c.Request = httptest.NewRequest("POST", "/webhooks/test-org/gitee/abc", bytes.NewReader([]byte(`{}`)))
	c.Request.Header.Set("Content-Type", "application/json")

	router.handleGiteeWebhookWithRepo(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleGiteeWebhookWithRepo_NoSecretConfigured(t *testing.T) {
	cfg := &config.Config{
		Webhook: config.WebhookConfig{
			GiteeSecret: "",
		},
	}
	router, db, _ := createTestRouterForWithRepo(t, cfg)

	// Create test repository
	db.Exec(`INSERT INTO repositories (id, organization_id, provider_type, provider_base_url, external_id, name, slug)
		VALUES (300, 1, 'gitee', 'https://gitee.com', '999', 'test-repo', 'org/test-repo')`)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = gin.Params{
		{Key: "org_slug", Value: "test-org"},
		{Key: "repo_id", Value: "300"},
	}

	payload := `{"hook_name": "push_hooks", "project": {"id": 999}}`
	c.Request = httptest.NewRequest("POST", "/webhooks/test-org/gitee/300", bytes.NewReader([]byte(payload)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-Gitee-Event", "Push Hook")

	router.handleGiteeWebhookWithRepo(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d: %s", http.StatusUnauthorized, w.Code, w.Body.String())
	}
}

// ===========================================
// Additional handleGiteeWebhookWithRepo Tests
// ===========================================

func TestHandleGiteeWebhookWithRepo_WithWebhookService(t *testing.T) {
	cfg := &config.Config{
		Webhook: config.WebhookConfig{
			GiteeSecret: "",
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
		VALUES (300, 1, 'gitee', 'https://gitee.com', '999', 'test-repo', 'org/test-repo')`)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = gin.Params{
		{Key: "org_slug", Value: "test-org"},
		{Key: "repo_id", Value: "300"},
	}

	payload := `{"hook_name": "push_hooks", "project": {"id": 999}}`
	c.Request = httptest.NewRequest("POST", "/webhooks/test-org/gitee/300", bytes.NewReader([]byte(payload)))
	c.Request.Header.Set("Content-Type", "application/json")

	router.handleGiteeWebhookWithRepo(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d: %s", http.StatusUnauthorized, w.Code, w.Body.String())
	}
}

func TestHandleGiteeWebhookWithRepo_InvalidSignature(t *testing.T) {
	cfg := &config.Config{
		Webhook: config.WebhookConfig{
			GiteeSecret: "gitee-secret",
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
		VALUES (300, 1, 'gitee', 'https://gitee.com', '999', 'test-repo', 'org/test-repo')`)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = gin.Params{
		{Key: "org_slug", Value: "test-org"},
		{Key: "repo_id", Value: "300"},
	}

	payload := `{"hook_name": "push_hooks"}`
	c.Request = httptest.NewRequest("POST", "/webhooks/test-org/gitee/300", bytes.NewReader([]byte(payload)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-Gitee-Token", "wrong-token")

	router.handleGiteeWebhookWithRepo(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d: %s", http.StatusUnauthorized, w.Code, w.Body.String())
	}
}

// ===========================================
// verifyGiteeSignature Additional Tests (WithRepo context)
// ===========================================

func TestVerifyGiteeSignatureWithRepo_TokenMethod(t *testing.T) {
	cfg := &config.Config{}
	db := setupTestDBForWithRepo(t)
	logger := testLoggerForGit()
	registry := NewHandlerRegistry(logger)

	router := &WebhookRouter{
		db:       db,
		cfg:      cfg,
		logger:   logger,
		registry: registry,
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	payload := `{"hook_name": "push_hooks"}`
	c.Request = httptest.NewRequest("POST", "/", bytes.NewReader([]byte(payload)))
	c.Request.Header.Set("X-Gitee-Token", "my-secret")

	result := router.verifyGiteeSignature(c, "my-secret")

	if !result {
		t.Error("expected token verification to succeed")
	}
}

func TestVerifyGiteeSignatureWithRepo_TokenMismatch(t *testing.T) {
	cfg := &config.Config{}
	db := setupTestDBForWithRepo(t)
	logger := testLoggerForGit()
	registry := NewHandlerRegistry(logger)

	router := &WebhookRouter{
		db:       db,
		cfg:      cfg,
		logger:   logger,
		registry: registry,
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	payload := `{"hook_name": "push_hooks"}`
	c.Request = httptest.NewRequest("POST", "/", bytes.NewReader([]byte(payload)))
	c.Request.Header.Set("X-Gitee-Token", "wrong-token")

	result := router.verifyGiteeSignature(c, "my-secret")

	if result {
		t.Error("expected token verification to fail with wrong token")
	}
}

func TestVerifyGiteeSignatureWithRepo_EmptyHeaders(t *testing.T) {
	cfg := &config.Config{}
	db := setupTestDBForWithRepo(t)
	logger := testLoggerForGit()
	registry := NewHandlerRegistry(logger)

	router := &WebhookRouter{
		db:       db,
		cfg:      cfg,
		logger:   logger,
		registry: registry,
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	payload := `{"hook_name": "push_hooks"}`
	c.Request = httptest.NewRequest("POST", "/", bytes.NewReader([]byte(payload)))
	// No X-Gitee-Token or X-Gitee-Timestamp headers

	result := router.verifyGiteeSignature(c, "my-secret")

	if result {
		t.Error("expected verification to fail without proper headers")
	}
}
