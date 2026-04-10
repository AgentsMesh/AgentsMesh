package v1

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/relay"
)

// --- helpers for relay connect tests ---

// newRelayManager creates a relay.Manager and registers a healthy relay for testing.
func newRelayManager(t *testing.T) *relay.Manager {
	t.Helper()
	mgr := relay.NewManagerWithOptions()
	t.Cleanup(mgr.Stop)
	err := mgr.Register(&relay.RelayInfo{
		ID:            "relay-1",
		URL:           "wss://relay.example.com",
		Region:        "us-east",
		Capacity:      100,
		Healthy:       true,
		LastHeartbeat: time.Now(),
	})
	require.NoError(t, err)
	return mgr
}

func newRelayConnectContext(method, path string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, nil)
	return c, w
}

func setRelayTenantContext(c *gin.Context, orgID, userID int64) {
	tc := &middleware.TenantContext{
		OrganizationID:   orgID,
		OrganizationSlug: "test-org",
		UserID:           userID,
		UserRole:         "member",
	}
	c.Set("tenant", tc)
	c.Set("user_id", userID)
}

// --- GetPodConnection tests ---

func TestGetPodConnection_Success(t *testing.T) {
	mgr := newRelayManager(t)
	tokenGen := relay.NewTokenGenerator("test-secret-key-32bytes!!", "test-issuer")

	podSvc := &mockRelayPodService{
		getPodFn: func(_ context.Context, key string) (*agentpod.Pod, error) {
			return &agentpod.Pod{
				PodKey:         key,
				OrganizationID: 1,
				CreatedByID:    10,
				RunnerID:       42,
				Status:         agentpod.StatusRunning,
			}, nil
		},
	}
	sender := &mockRelayCommandSender{}

	handler := NewPodConnectHandler(podSvc, mgr, tokenGen, sender, nil)

	c, w := newRelayConnectContext(http.MethodGet, "/pods/pod-abc/relay/connect")
	c.Params = gin.Params{{Key: "key", Value: "pod-abc"}}
	setRelayTenantContext(c, 1, 10)

	handler.GetPodConnection(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp PodConnectResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "wss://relay.example.com", resp.RelayURL)
	assert.Equal(t, "pod-abc", resp.PodKey)
	assert.NotEmpty(t, resp.Token)
}

func TestGetPodConnection_RelayNotConfigured(t *testing.T) {
	tokenGen := relay.NewTokenGenerator("test-secret-key-32bytes!!", "test-issuer")

	podSvc := &mockRelayPodService{}

	// nil relayManager means relay is not configured
	handler := NewPodConnectHandler(podSvc, nil, tokenGen, nil, nil)

	c, w := newRelayConnectContext(http.MethodGet, "/pods/pod-abc/relay/connect")
	c.Params = gin.Params{{Key: "key", Value: "pod-abc"}}
	setRelayTenantContext(c, 1, 10)

	handler.GetPodConnection(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "SERVICE_UNAVAILABLE", resp["code"])
}

func TestGetPodConnection_NoHealthyRelays(t *testing.T) {
	// Create a manager with no relays registered
	mgr := relay.NewManagerWithOptions()
	t.Cleanup(mgr.Stop)

	tokenGen := relay.NewTokenGenerator("test-secret-key-32bytes!!", "test-issuer")
	podSvc := &mockRelayPodService{}

	handler := NewPodConnectHandler(podSvc, mgr, tokenGen, nil, nil)

	c, w := newRelayConnectContext(http.MethodGet, "/pods/pod-abc/relay/connect")
	c.Params = gin.Params{{Key: "key", Value: "pod-abc"}}
	setRelayTenantContext(c, 1, 10)

	handler.GetPodConnection(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestGetPodConnection_PodNotFound(t *testing.T) {
	mgr := newRelayManager(t)
	tokenGen := relay.NewTokenGenerator("test-secret-key-32bytes!!", "test-issuer")

	podSvc := &mockRelayPodService{
		getPodFn: func(context.Context, string) (*agentpod.Pod, error) {
			return nil, errors.New("not found")
		},
	}
	handler := NewPodConnectHandler(podSvc, mgr, tokenGen, nil, nil)

	c, w := newRelayConnectContext(http.MethodGet, "/pods/nonexistent/relay/connect")
	c.Params = gin.Params{{Key: "key", Value: "nonexistent"}}
	setRelayTenantContext(c, 1, 10)

	handler.GetPodConnection(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "RESOURCE_NOT_FOUND", resp["code"])
}

func TestGetPodConnection_PodNotActive(t *testing.T) {
	mgr := newRelayManager(t)
	tokenGen := relay.NewTokenGenerator("test-secret-key-32bytes!!", "test-issuer")

	podSvc := &mockRelayPodService{
		getPodFn: func(_ context.Context, key string) (*agentpod.Pod, error) {
			return &agentpod.Pod{
				PodKey:         key,
				OrganizationID: 1,
				CreatedByID:    10,
				RunnerID:       42,
				Status:         agentpod.StatusTerminated,
			}, nil
		},
	}
	handler := NewPodConnectHandler(podSvc, mgr, tokenGen, nil, nil)

	c, w := newRelayConnectContext(http.MethodGet, "/pods/pod-abc/relay/connect")
	c.Params = gin.Params{{Key: "key", Value: "pod-abc"}}
	setRelayTenantContext(c, 1, 10)

	handler.GetPodConnection(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "VALIDATION_FAILED", resp["code"])
}

func TestGetPodConnection_Unauthorized(t *testing.T) {
	mgr := newRelayManager(t)
	tokenGen := relay.NewTokenGenerator("test-secret-key-32bytes!!", "test-issuer")

	podSvc := &mockRelayPodService{
		getPodFn: func(_ context.Context, key string) (*agentpod.Pod, error) {
			return &agentpod.Pod{
				PodKey:         key,
				OrganizationID: 999, // different org
				RunnerID:       42,
				Status:         agentpod.StatusRunning,
			}, nil
		},
	}
	handler := NewPodConnectHandler(podSvc, mgr, tokenGen, nil, nil)

	c, w := newRelayConnectContext(http.MethodGet, "/pods/pod-abc/relay/connect")
	c.Params = gin.Params{{Key: "key", Value: "pod-abc"}}
	setRelayTenantContext(c, 1, 10) // org 1, pod in org 999

	handler.GetPodConnection(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestGetPodConnection_NoTenant(t *testing.T) {
	mgr := newRelayManager(t)
	tokenGen := relay.NewTokenGenerator("test-secret-key-32bytes!!", "test-issuer")

	podSvc := &mockRelayPodService{
		getPodFn: func(_ context.Context, key string) (*agentpod.Pod, error) {
			return &agentpod.Pod{
				PodKey:         key,
				OrganizationID: 1,
				RunnerID:       42,
				Status:         agentpod.StatusRunning,
			}, nil
		},
	}
	handler := NewPodConnectHandler(podSvc, mgr, tokenGen, nil, nil)

	c, w := newRelayConnectContext(http.MethodGet, "/pods/pod-abc/relay/connect")
	c.Params = gin.Params{{Key: "key", Value: "pod-abc"}}
	// No tenant context set

	handler.GetPodConnection(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetPodConnection_NoUserID(t *testing.T) {
	mgr := newRelayManager(t)
	tokenGen := relay.NewTokenGenerator("test-secret-key-32bytes!!", "test-issuer")

	podSvc := &mockRelayPodService{
		getPodFn: func(_ context.Context, key string) (*agentpod.Pod, error) {
			return &agentpod.Pod{
				PodKey:         key,
				OrganizationID: 1,
				CreatedByID:    10,
				RunnerID:       42,
				Status:         agentpod.StatusRunning,
			}, nil
		},
	}
	handler := NewPodConnectHandler(podSvc, mgr, tokenGen, nil, nil)

	c, w := newRelayConnectContext(http.MethodGet, "/pods/pod-abc/relay/connect")
	c.Params = gin.Params{{Key: "key", Value: "pod-abc"}}
	// Set tenant but without user_id in gin context
	tc := &middleware.TenantContext{
		OrganizationID:   1,
		OrganizationSlug: "test-org",
		UserID:           10,
		UserRole:         "member",
	}
	c.Set("tenant", tc)
	// user_id not set — GetUserID returns 0

	handler.GetPodConnection(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetPodConnection_MemberForbiddenOthersPod(t *testing.T) {
	mgr := newRelayManager(t)
	tokenGen := relay.NewTokenGenerator("test-secret-key-32bytes!!", "test-issuer")

	podSvc := &mockRelayPodService{
		getPodFn: func(_ context.Context, key string) (*agentpod.Pod, error) {
			return &agentpod.Pod{
				PodKey:         key,
				OrganizationID: 1,
				CreatedByID:    99, // owned by different user
				RunnerID:       42,
				Status:         agentpod.StatusRunning,
			}, nil
		},
	}
	handler := NewPodConnectHandler(podSvc, mgr, tokenGen, nil, nil)

	c, w := newRelayConnectContext(http.MethodGet, "/pods/pod-abc/relay/connect")
	c.Params = gin.Params{{Key: "key", Value: "pod-abc"}}
	setRelayTenantContext(c, 1, 10) // userID=10, role=member

	handler.GetPodConnection(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}
