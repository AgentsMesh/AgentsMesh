package webhooks

import (
	"net/http/httptest"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/gin-gonic/gin"
)

// ===========================================
// extractObjectKind Tests for Gitee
// ===========================================

func TestExtractObjectKind_Gitee(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestRouterForGit(t, cfg)

	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		header   string
		hookName string
		expected string
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
// mapGiteeEventToKind Tests
// ===========================================

func TestMapGiteeEventToKind(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestRouterForGit(t, cfg)

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
