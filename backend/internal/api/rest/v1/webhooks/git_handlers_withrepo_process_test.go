package webhooks

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/service/repository"
	"github.com/gin-gonic/gin"
)

// ===========================================
// processWebhookWithRepo Tests
// ===========================================

func TestProcessWebhookWithRepo_InvalidJSON(t *testing.T) {
	cfg := &config.Config{}
	router, _, _ := createTestRouterForWithRepo(t, cfg)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("POST", "/", bytes.NewReader([]byte("invalid-json")))
	c.Request.Header.Set("Content-Type", "application/json")

	router.processWebhookWithRepo(c, "gitlab", "test-org", 123)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["error"] != "invalid JSON payload" {
		t.Errorf("unexpected error message: %v", response["error"])
	}
	if _, ok := response["code"]; !ok {
		t.Error("expected 'code' field in error response")
	}
}

func TestProcessWebhookWithRepo_RepoNotFound(t *testing.T) {
	cfg := &config.Config{}
	router, _, _ := createTestRouterForWithRepo(t, cfg)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	payload := `{"object_kind": "push"}`
	c.Request = httptest.NewRequest("POST", "/", bytes.NewReader([]byte(payload)))
	c.Request.Header.Set("Content-Type", "application/json")

	router.processWebhookWithRepo(c, "gitlab", "test-org", 9999)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["code"] != "RESOURCE_NOT_FOUND" {
		t.Errorf("expected code 'RESOURCE_NOT_FOUND', got: %v", response["code"])
	}
}

func TestProcessWebhookWithRepo_Success(t *testing.T) {
	cfg := &config.Config{}
	router, db, _ := createTestRouterForWithRepo(t, cfg)

	// Create test repository
	db.Exec(`INSERT INTO repositories (id, organization_id, provider_type, provider_base_url, external_id, name, slug)
		VALUES (100, 1, 'gitlab', 'https://gitlab.com', '500', 'test-repo', 'org/test-repo')`)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	payload := `{"object_kind": "push", "project": {"id": 500}, "ref": "refs/heads/main"}`
	c.Request = httptest.NewRequest("POST", "/", bytes.NewReader([]byte(payload)))
	c.Request.Header.Set("Content-Type", "application/json")

	router.processWebhookWithRepo(c, "gitlab", "test-org", 100)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestProcessWebhookWithRepo_PipelineEvent(t *testing.T) {
	cfg := &config.Config{}
	router, db, _ := createTestRouterForWithRepo(t, cfg)

	// Create test repository
	db.Exec(`INSERT INTO repositories (id, organization_id, provider_type, provider_base_url, external_id, name, slug)
		VALUES (100, 1, 'gitlab', 'https://gitlab.com', '500', 'test-repo', 'org/test-repo')`)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	payload := `{
		"object_kind": "pipeline",
		"project": {"id": 500},
		"object_attributes": {"id": 12345, "status": "success"}
	}`
	c.Request = httptest.NewRequest("POST", "/", bytes.NewReader([]byte(payload)))
	c.Request.Header.Set("Content-Type", "application/json")

	router.processWebhookWithRepo(c, "gitlab", "test-org", 100)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestProcessWebhookWithRepo_MergeRequestEvent(t *testing.T) {
	cfg := &config.Config{}
	router, db, _ := createTestRouterForWithRepo(t, cfg)

	// Create test repository
	db.Exec(`INSERT INTO repositories (id, organization_id, provider_type, provider_base_url, external_id, name, slug)
		VALUES (100, 1, 'gitlab', 'https://gitlab.com', '500', 'test-repo', 'org/test-repo')`)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	payload := `{
		"object_kind": "merge_request",
		"project": {"id": 500},
		"object_attributes": {
			"iid": 42,
			"source_branch": "feature/test",
			"target_branch": "main",
			"state": "opened",
			"url": "https://gitlab.com/org/repo/-/merge_requests/42"
		}
	}`
	c.Request = httptest.NewRequest("POST", "/", bytes.NewReader([]byte(payload)))
	c.Request.Header.Set("Content-Type", "application/json")

	router.processWebhookWithRepo(c, "gitlab", "test-org", 100)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestProcessWebhookWithRepo_ProjectIDMismatchLogsWarning(t *testing.T) {
	cfg := &config.Config{}
	router, db, _ := createTestRouterForWithRepo(t, cfg)

	// Create test repository with external_id = 500
	db.Exec(`INSERT INTO repositories (id, organization_id, provider_type, provider_base_url, external_id, name, slug)
		VALUES (100, 1, 'gitlab', 'https://gitlab.com', '500', 'test-repo', 'org/test-repo')`)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Payload has different project ID (999) than repository external_id (500)
	payload := `{"object_kind": "push", "project": {"id": 999}, "ref": "refs/heads/main"}`
	c.Request = httptest.NewRequest("POST", "/", bytes.NewReader([]byte(payload)))
	c.Request.Header.Set("Content-Type", "application/json")

	router.processWebhookWithRepo(c, "gitlab", "test-org", 100)

	// Should still succeed (mismatch only logs warning)
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestProcessWebhookWithRepo_BuildToJobConversion(t *testing.T) {
	cfg := &config.Config{}
	router, db, _ := createTestRouterForWithRepo(t, cfg)

	// Create test repository
	db.Exec(`INSERT INTO repositories (id, organization_id, provider_type, provider_base_url, external_id, name, slug)
		VALUES (100, 1, 'gitlab', 'https://gitlab.com', '500', 'test-repo', 'org/test-repo')`)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// GitLab legacy: build event should be converted to job
	payload := `{
		"object_kind": "build",
		"project": {"id": 500},
		"build_id": 12345,
		"build_status": "success"
	}`
	c.Request = httptest.NewRequest("POST", "/", bytes.NewReader([]byte(payload)))
	c.Request.Header.Set("Content-Type", "application/json")

	router.processWebhookWithRepo(c, "gitlab", "test-org", 100)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

// ===========================================
// processWebhookWithRepo - Additional Edge Cases
// ===========================================

func TestProcessWebhookWithRepo_NoRepoService(t *testing.T) {
	cfg := &config.Config{}
	db := setupTestDBForWithRepo(t)
	logger := testLoggerForGit()
	registry := NewHandlerRegistry(logger)
	SetupDefaultHandlers(registry, logger)

	// Router without repoService
	router := &WebhookRouter{
		db:       db,
		cfg:      cfg,
		logger:   logger,
		registry: registry,
		// repoService is nil
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	payload := `{"object_kind": "push", "project": {"id": 500}}`
	c.Request = httptest.NewRequest("POST", "/", bytes.NewReader([]byte(payload)))
	c.Request.Header.Set("Content-Type", "application/json")

	router.processWebhookWithRepo(c, "gitlab", "test-org", 100)

	// Without repoService, should still succeed (just won't set OrganizationID)
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestProcessWebhookWithRepo_RegistryNoHandler(t *testing.T) {
	cfg := &config.Config{}
	db := setupTestDBForWithRepo(t)
	logger := testLoggerForGit()

	// Create registry without handlers - registry returns "skipped" not error
	registry := NewHandlerRegistry(logger)
	// Don't set up default handlers

	repoSvc := repository.NewService(infra.NewGitProviderRepository(db))

	router := &WebhookRouter{
		db:          db,
		cfg:         cfg,
		logger:      logger,
		registry:    registry,
		repoService: repoSvc,
	}

	// Create test repository
	db.Exec(`INSERT INTO repositories (id, organization_id, provider_type, provider_base_url, external_id, name, slug)
		VALUES (100, 1, 'gitlab', 'https://gitlab.com', '500', 'test-repo', 'org/test-repo')`)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Use an unknown object_kind that has no handler
	payload := `{"object_kind": "unknown_event_type", "project": {"id": 500}}`
	c.Request = httptest.NewRequest("POST", "/", bytes.NewReader([]byte(payload)))
	c.Request.Header.Set("Content-Type", "application/json")

	router.processWebhookWithRepo(c, "gitlab", "test-org", 100)

	// Registry returns success with "skipped" status when no handler found
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Verify response contains "skipped" status
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["status"] != "skipped" {
		t.Errorf("expected status 'skipped', got: %v", response["status"])
	}
}

func TestProcessWebhookWithRepo_GitHubEventHeader(t *testing.T) {
	cfg := &config.Config{}
	router, db, _ := createTestRouterForWithRepo(t, cfg)

	// Create test repository
	db.Exec(`INSERT INTO repositories (id, organization_id, provider_type, provider_base_url, external_id, name, slug)
		VALUES (100, 1, 'github', 'https://github.com', '500', 'test-repo', 'org/test-repo')`)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// GitHub-style payload (uses X-GitHub-Event header)
	payload := `{"action": "opened", "repository": {"id": 500}}`
	c.Request = httptest.NewRequest("POST", "/", bytes.NewReader([]byte(payload)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-GitHub-Event", "pull_request")

	router.processWebhookWithRepo(c, "github", "test-org", 100)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestProcessWebhookWithRepo_GiteeEventHeader(t *testing.T) {
	cfg := &config.Config{}
	router, db, _ := createTestRouterForWithRepo(t, cfg)

	// Create test repository
	db.Exec(`INSERT INTO repositories (id, organization_id, provider_type, provider_base_url, external_id, name, slug)
		VALUES (100, 1, 'gitee', 'https://gitee.com', '500', 'test-repo', 'org/test-repo')`)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Gitee-style payload (uses X-Gitee-Event header)
	payload := `{"project": {"id": 500}}`
	c.Request = httptest.NewRequest("POST", "/", bytes.NewReader([]byte(payload)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-Gitee-Event", "Merge Request Hook")

	router.processWebhookWithRepo(c, "gitee", "test-org", 100)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}
