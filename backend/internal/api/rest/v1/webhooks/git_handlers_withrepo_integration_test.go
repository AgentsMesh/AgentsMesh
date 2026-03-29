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
// Integration Test: Full Flow
// ===========================================

func TestGitLabWebhookWithRepo_FullFlow(t *testing.T) {
	cfg := &config.Config{
		Webhook: config.WebhookConfig{
			GitLabSecret: "test-secret",
		},
	}
	router, db, _ := createTestRouterForWithRepo(t, cfg)

	// Create test repository with specific org_id
	db.Exec(`INSERT INTO repositories (id, organization_id, provider_type, provider_base_url, external_id, name, slug)
		VALUES (42, 100, 'gitlab', 'https://gitlab.com', '12345', 'my-repo', 'my-org/my-repo')`)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	rg := engine.Group("/webhooks")
	router.RegisterRoutes(rg)

	// Create test request
	payload := `{
		"object_kind": "merge_request",
		"project": {"id": 12345},
		"object_attributes": {
			"iid": 99,
			"source_branch": "feature/AM-123",
			"target_branch": "main",
			"state": "merged",
			"url": "https://gitlab.com/my-org/my-repo/-/merge_requests/99"
		}
	}`

	req := httptest.NewRequest("POST", "/webhooks/my-org/gitlab/42", bytes.NewReader([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Gitlab-Token", "test-secret")

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}
