package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── extractAPIKey ─────────────────────────────────────────────────

func TestExtractAPIKey(t *testing.T) {
	t.Run("X-API-Key header", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("X-API-Key", "amk_abc123")

		result := extractAPIKey(c)
		assert.Equal(t, "amk_abc123", result)
	})

	t.Run("Bearer amk_ header", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("Authorization", "Bearer amk_xyz789")

		result := extractAPIKey(c)
		assert.Equal(t, "amk_xyz789", result)
	})

	t.Run("Bearer non-amk_ header returns empty", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("Authorization", "Bearer jwt_token_here")

		result := extractAPIKey(c)
		assert.Equal(t, "", result)
	})

	t.Run("no header returns empty", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)

		result := extractAPIKey(c)
		assert.Equal(t, "", result)
	})

	t.Run("X-API-Key takes priority over Authorization", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("X-API-Key", "amk_from_xapikey")
		c.Request.Header.Set("Authorization", "Bearer amk_from_bearer")

		result := extractAPIKey(c)
		assert.Equal(t, "amk_from_xapikey", result)
	})

	t.Run("Authorization without Bearer prefix returns empty", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("Authorization", "amk_nobearer")

		result := extractAPIKey(c)
		assert.Equal(t, "", result)
	})

	t.Run("X-API-Key without amk_ prefix returns empty", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("X-API-Key", "invalid_prefix_key")

		result := extractAPIKey(c)
		assert.Equal(t, "", result)
	})
}

// ─── GetAPIKeyContext ──────────────────────────────────────────────

func TestGetAPIKeyContext(t *testing.T) {
	t.Run("present returns context", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		expected := &APIKeyContext{
			APIKeyID: 42,
			KeyName:  "my-key",
			Scopes:   []string{"pods:read"},
		}
		c.Set("apikey_context", expected)

		result := GetAPIKeyContext(c)
		require.NotNil(t, result)
		assert.Equal(t, int64(42), result.APIKeyID)
		assert.Equal(t, "my-key", result.KeyName)
		assert.Equal(t, []string{"pods:read"}, result.Scopes)
	})

	t.Run("not present returns nil", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		result := GetAPIKeyContext(c)
		assert.Nil(t, result)
	})

	t.Run("wrong type returns nil", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("apikey_context", "not the right type")

		result := GetAPIKeyContext(c)
		assert.Nil(t, result)
	})
}

// ─── handleAPIKeyError ─────────────────────────────────────────────

func TestHandleAPIKeyError(t *testing.T) {
	t.Run("ErrAPIKeyNotFound returns 401", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)

		handleAPIKeyError(c, ErrAPIKeyNotFound)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("ErrAPIKeyDisabled returns 403", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)

		handleAPIKeyError(c, ErrAPIKeyDisabled)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("ErrAPIKeyExpired returns 401", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)

		handleAPIKeyError(c, ErrAPIKeyExpired)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("unknown error returns 500", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)

		handleAPIKeyError(c, errors.New("something unexpected"))
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
