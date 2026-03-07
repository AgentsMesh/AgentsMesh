package internal

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/service/relay"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// newTestHandler creates a RelayHandler backed by an in-memory Manager.
// dnsService, acmeManager, and geoResolver are nil (optional dependencies).
func newTestHandler(t *testing.T) (*RelayHandler, *relay.Manager) {
	t.Helper()
	m := relay.NewManagerWithOptions()
	t.Cleanup(func() { m.Stop() })
	handler := NewRelayHandler(m, nil, nil, nil)
	return handler, m
}

// jsonRequest creates an *http.Request with JSON body for gin test contexts.
func jsonRequest(t *testing.T, method, path string, body interface{}) *http.Request {
	t.Helper()
	data, err := json.Marshal(body)
	require.NoError(t, err)
	req := httptest.NewRequest(method, path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// === Register Handler Tests ===

func TestRegisterHandler_Success(t *testing.T) {
	handler, m := newTestHandler(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = jsonRequest(t, "POST", "/register", RegisterRequest{
		RelayID: "relay-1",
		URL:     "wss://relay.example.com:8443",
		Region:  "us-east",
	})

	handler.Register(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp RegisterResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "registered", resp.Status)
	assert.Equal(t, "wss://relay.example.com:8443", resp.URL)

	// Verify relay is in manager
	r := m.GetRelayByID("relay-1")
	require.NotNil(t, r)
	assert.Equal(t, "us-east", r.Region)
	assert.Equal(t, 1000, r.Capacity) // default capacity
}

func TestRegisterHandler_DefaultValues(t *testing.T) {
	handler, m := newTestHandler(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = jsonRequest(t, "POST", "/register", RegisterRequest{
		RelayID: "relay-1",
		URL:     "ws://192.168.1.1:8080",
	})

	handler.Register(c)

	assert.Equal(t, http.StatusOK, w.Code)

	r := m.GetRelayByID("relay-1")
	require.NotNil(t, r)
	assert.Equal(t, "default", r.Region)
	assert.Equal(t, 1000, r.Capacity)
}

func TestRegisterHandler_ValidationError_MissingRelayID(t *testing.T) {
	handler, _ := newTestHandler(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = jsonRequest(t, "POST", "/register", RegisterRequest{
		URL: "wss://relay.example.com",
	})

	handler.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegisterHandler_InvalidURLScheme(t *testing.T) {
	handler, _ := newTestHandler(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = jsonRequest(t, "POST", "/register", RegisterRequest{
		RelayID: "relay-1",
		URL:     "http://relay.example.com",
	})

	handler.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegisterHandler_MissingURL(t *testing.T) {
	handler, _ := newTestHandler(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = jsonRequest(t, "POST", "/register", RegisterRequest{
		RelayID: "relay-1",
	})

	handler.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// === Heartbeat Handler Tests ===

func TestHeartbeatHandler_Success(t *testing.T) {
	handler, m := newTestHandler(t)

	// First register a relay
	if err := m.Register(&relay.RelayInfo{ID: "relay-1", URL: "wss://relay.com"}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = jsonRequest(t, "POST", "/heartbeat", HeartbeatRequest{
		RelayID:     "relay-1",
		Connections: 50,
		CPUUsage:    25.5,
		MemoryUsage: 60.0,
		LatencyMs:   100,
	})

	handler.Heartbeat(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp HeartbeatResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "ok", resp.Status)

	// Verify metrics updated
	r := m.GetRelayByID("relay-1")
	require.NotNil(t, r)
	assert.Equal(t, 50, r.CurrentConnections)
	assert.InDelta(t, 25.5, r.CPUUsage, 0.01)
	assert.Equal(t, 100, r.AvgLatencyMs)
}

func TestHeartbeatHandler_NotFound(t *testing.T) {
	handler, _ := newTestHandler(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = jsonRequest(t, "POST", "/heartbeat", HeartbeatRequest{
		RelayID: "unknown-relay",
	})

	handler.Heartbeat(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHeartbeatHandler_ValidationError(t *testing.T) {
	handler, _ := newTestHandler(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = jsonRequest(t, "POST", "/heartbeat", map[string]interface{}{
		// missing required relay_id
		"connections": 10,
	})

	handler.Heartbeat(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// === Unregister Handler Tests ===

func TestUnregisterHandler_Success(t *testing.T) {
	handler, m := newTestHandler(t)

	if err := m.Register(&relay.RelayInfo{ID: "relay-1", URL: "wss://relay.com"}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = jsonRequest(t, "POST", "/unregister", UnregisterRequest{
		RelayID: "relay-1",
		Reason:  "shutdown",
	})

	handler.Unregister(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp UnregisterResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "unregistered", resp.Status)
	assert.Equal(t, "shutdown", resp.Reason)

	// Verify relay removed
	assert.Nil(t, m.GetRelayByID("relay-1"))
}

func TestUnregisterHandler_NotFound_Idempotent(t *testing.T) {
	handler, _ := newTestHandler(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = jsonRequest(t, "POST", "/unregister", UnregisterRequest{
		RelayID: "unknown-relay",
		Reason:  "shutdown",
	})

	handler.Unregister(c)

	// Should be 200 (idempotent), not 404
	assert.Equal(t, http.StatusOK, w.Code)

	var resp UnregisterResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "not_found", resp.Status)
}

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
