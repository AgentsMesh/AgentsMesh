package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alicebob/miniredis/v2"
)

func setupRedisForTest(t *testing.T) (*redis.Client, func()) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return client, func() { client.Close(); mr.Close() }
}

func newTestRouter(mw gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/test", mw, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	return r
}

func TestRateLimiter_AllowsWithinLimit(t *testing.T) {
	client, cleanup := setupRedisForTest(t)
	defer cleanup()

	mw := middleware.IPRateLimiter(client, "login", 5, time.Minute)
	router := newTestRouter(mw)

	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/test", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "request %d should succeed", i+1)
	}
}

func TestRateLimiter_BlocksOverLimit(t *testing.T) {
	client, cleanup := setupRedisForTest(t)
	defer cleanup()

	mw := middleware.IPRateLimiter(client, "login", 3, time.Minute)
	router := newTestRouter(mw)

	// First 3 requests succeed.
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/test", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// 4th request is blocked.
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/test", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	var resp apierr.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, apierr.RATE_LIMITED, resp.Code)
}

func TestRateLimiter_DifferentScopesAreIndependent(t *testing.T) {
	client, cleanup := setupRedisForTest(t)
	defer cleanup()

	gin.SetMode(gin.TestMode)
	r := gin.New()

	loginMw := middleware.IPRateLimiter(client, "login", 2, time.Minute)
	registerMw := middleware.IPRateLimiter(client, "register", 2, time.Minute)

	r.POST("/login", loginMw, func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
	r.POST("/register", registerMw, func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	// Exhaust login limit.
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/login", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// Login is now blocked.
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/login", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	// Register should still work (different scope).
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/register", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRateLimiter_NilRedisIsNoOp(t *testing.T) {
	mw := middleware.IPRateLimiter(nil, "login", 1, time.Minute)
	router := newTestRouter(mw)

	// Should allow unlimited requests when Redis is nil.
	for i := 0; i < 10; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/test", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}
}

func TestRateLimiter_EmptyKeySkips(t *testing.T) {
	client, cleanup := setupRedisForTest(t)
	defer cleanup()

	mw := middleware.RateLimiter(client, middleware.RateLimitConfig{
		MaxAttempts: 1,
		Window:      time.Minute,
		KeyFunc:     func(c *gin.Context) string { return "" },
	})
	router := newTestRouter(mw)

	for i := 0; i < 10; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/test", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}
}
