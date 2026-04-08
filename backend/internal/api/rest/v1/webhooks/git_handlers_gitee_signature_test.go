package webhooks

import (
	"net/http/httptest"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/gin-gonic/gin"
)

// ===========================================
// verifyGiteeSignature Tests
// ===========================================

func TestVerifyGiteeSignature_ValidToken(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestRouterForGit(t, cfg)

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
	router, _ := createTestRouterForGit(t, cfg)

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
	router, _ := createTestRouterForGit(t, cfg)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("POST", "/", nil)

	result := router.verifyGiteeSignature(c, "test-secret")
	if result {
		t.Error("expected signature verification to fail with no headers")
	}
}
