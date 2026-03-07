package webhooks

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/service/repository"
	"github.com/gin-gonic/gin"
)

// ===========================================
// Route Registration Tests
// ===========================================

func TestRegisterRoutes_WithRepoEndpoints(t *testing.T) {
	cfg := &config.Config{}
	router, _, _ := createTestRouterForWithRepo(t, cfg)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	rg := engine.Group("/webhooks")
	router.RegisterRoutes(rg)

	// Test that routes are registered
	routes := engine.Routes()

	expectedRoutes := map[string]bool{
		"POST /webhooks/:org_slug/gitlab/:repo_id": false,
		"POST /webhooks/:org_slug/github/:repo_id": false,
		"POST /webhooks/:org_slug/gitee/:repo_id":  false,
	}

	for _, route := range routes {
		key := route.Method + " " + route.Path
		if _, ok := expectedRoutes[key]; ok {
			expectedRoutes[key] = true
		}
	}

	for route, found := range expectedRoutes {
		if !found {
			t.Errorf("expected route not found: %s", route)
		}
	}
}

// ===========================================
// WithRepositoryService Option Tests
// ===========================================

func TestWithRepositoryService(t *testing.T) {
	db := setupTestDBForWithRepo(t)
	repoSvc := repository.NewService(infra.NewGitProviderRepository(db))

	opt := WithRepositoryService(repoSvc)

	router := &WebhookRouter{}
	opt(router)

	if router.repoService != repoSvc {
		t.Error("WithRepositoryService did not set repoService correctly")
	}
}
