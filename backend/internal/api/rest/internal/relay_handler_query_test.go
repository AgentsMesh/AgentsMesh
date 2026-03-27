package internal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/service/relay"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// === ForceUnregister Handler Tests ===

func TestForceUnregisterHandler_Success(t *testing.T) {
	handler, m := newTestHandler(t)

	if err := m.Register(&relay.RelayInfo{ID: "relay-1", URL: "wss://relay.com"}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/relay-1", nil)
	c.Params = gin.Params{{Key: "relay_id", Value: "relay-1"}}

	handler.ForceUnregister(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp UnregisterResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "unregistered", resp.Status)
	assert.Equal(t, "relay-1", resp.RelayID)

	assert.Nil(t, m.GetRelayByID("relay-1"))
}

func TestForceUnregisterHandler_NotFound_Idempotent(t *testing.T) {
	handler, _ := newTestHandler(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/unknown", nil)
	c.Params = gin.Params{{Key: "relay_id", Value: "unknown"}}

	handler.ForceUnregister(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp UnregisterResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "not_found", resp.Status)
}

func TestForceUnregisterHandler_EmptyRelayID(t *testing.T) {
	handler, _ := newTestHandler(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/", nil)
	// No relay_id param set

	handler.ForceUnregister(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// === Stats Handler Tests ===

func TestStatsHandler(t *testing.T) {
	handler, m := newTestHandler(t)

	// Register two relays
	m.Register(&relay.RelayInfo{ID: "relay-1", URL: "wss://r1.com", CurrentConnections: 10})
	m.Register(&relay.RelayInfo{ID: "relay-2", URL: "wss://r2.com", CurrentConnections: 20})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/stats", nil)

	handler.Stats(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var stats relay.Stats
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &stats))
	assert.Equal(t, 2, stats.TotalRelays)
	assert.Equal(t, 2, stats.HealthyRelays)
	// Note: connections are reset during Register (copy semantics), so they'll be 0
}

func TestStatsHandler_Empty(t *testing.T) {
	handler, _ := newTestHandler(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/stats", nil)

	handler.Stats(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var stats relay.Stats
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &stats))
	assert.Equal(t, 0, stats.TotalRelays)
}

// === List Handler Tests ===

func TestListHandler(t *testing.T) {
	handler, m := newTestHandler(t)

	m.Register(&relay.RelayInfo{ID: "relay-1", URL: "wss://r1.com"})
	m.Register(&relay.RelayInfo{ID: "relay-2", URL: "wss://r2.com"})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	handler.List(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Relays []relay.RelayInfo `json:"relays"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 2, len(resp.Relays))
}

func TestListHandler_Empty(t *testing.T) {
	handler, _ := newTestHandler(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	handler.List(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

// === Get Handler Tests ===

func TestGetHandler_Success(t *testing.T) {
	handler, m := newTestHandler(t)

	m.Register(&relay.RelayInfo{ID: "relay-1", URL: "wss://relay.com", Region: "ap-east"})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/relay-1", nil)
	c.Params = gin.Params{{Key: "relay_id", Value: "relay-1"}}

	handler.Get(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var got relay.RelayInfo
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "relay-1", got.ID)
	assert.Equal(t, "ap-east", got.Region)
	assert.True(t, got.Healthy)
}

func TestGetHandler_NotFound(t *testing.T) {
	handler, _ := newTestHandler(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/unknown", nil)
	c.Params = gin.Params{{Key: "relay_id", Value: "unknown"}}

	handler.Get(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetHandler_EmptyRelayID(t *testing.T) {
	handler, _ := newTestHandler(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	// No relay_id param

	handler.Get(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// === InternalAPIAuth Middleware Tests ===

func TestInternalAPIAuth_ValidSecret(t *testing.T) {
	router := gin.New()
	router.Use(InternalAPIAuth("test-secret"))
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/ping", nil)
	req.Header.Set("X-Internal-Secret", "test-secret")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestInternalAPIAuth_MissingHeader(t *testing.T) {
	router := gin.New()
	router.Use(InternalAPIAuth("test-secret"))
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/ping", nil)
	// No X-Internal-Secret header
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestInternalAPIAuth_WrongSecret(t *testing.T) {
	router := gin.New()
	router.Use(InternalAPIAuth("test-secret"))
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/ping", nil)
	req.Header.Set("X-Internal-Secret", "wrong-secret")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestInternalAPIAuth_EmptySecretPanics(t *testing.T) {
	assert.Panics(t, func() {
		InternalAPIAuth("")
	})
}
