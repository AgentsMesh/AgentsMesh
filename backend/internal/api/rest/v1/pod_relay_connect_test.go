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
	agentpodSvc "github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/service/geo"
	"github.com/anthropics/agentsmesh/backend/internal/service/relay"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
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

func TestGetPodConnection_SubscribePodError_StillSucceeds(t *testing.T) {
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
	sender := &mockRelayCommandSenderConfigurable{
		sendSubscribePodFn: func(context.Context, int64, string, string, string, bool, int32) error {
			return errors.New("runner disconnected")
		},
	}

	handler := NewPodConnectHandler(podSvc, mgr, tokenGen, sender, nil)

	c, w := newRelayConnectContext(http.MethodGet, "/pods/pod-abc/relay/connect")
	c.Params = gin.Params{{Key: "key", Value: "pod-abc"}}
	setRelayTenantContext(c, 1, 10)

	handler.GetPodConnection(c)

	// Should still succeed — subscribe error is logged but not fatal
	assert.Equal(t, http.StatusOK, w.Code)
	var resp PodConnectResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "pod-abc", resp.PodKey)
	assert.NotEmpty(t, resp.Token)
}

func TestGetPodConnection_NilCommandSender_SkipsSubscribe(t *testing.T) {
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

	// nil commandSender — should skip the subscribe block entirely
	handler := NewPodConnectHandler(podSvc, mgr, tokenGen, nil, nil)

	c, w := newRelayConnectContext(http.MethodGet, "/pods/pod-abc/relay/connect")
	c.Params = gin.Params{{Key: "key", Value: "pod-abc"}}
	setRelayTenantContext(c, 1, 10)

	handler.GetPodConnection(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp PodConnectResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "pod-abc", resp.PodKey)
	assert.NotEmpty(t, resp.Token)
}

func TestGetPodConnection_ZeroRunnerID_SkipsSubscribe(t *testing.T) {
	mgr := newRelayManager(t)
	tokenGen := relay.NewTokenGenerator("test-secret-key-32bytes!!", "test-issuer")

	podSvc := &mockRelayPodService{
		getPodFn: func(_ context.Context, key string) (*agentpod.Pod, error) {
			return &agentpod.Pod{
				PodKey:         key,
				OrganizationID: 1,
				RunnerID:       0, // no runner assigned
				Status:         agentpod.StatusRunning,
			}, nil
		},
	}
	subscribeCalled := false
	sender := &mockRelayCommandSenderConfigurable{
		sendSubscribePodFn: func(context.Context, int64, string, string, string, bool, int32) error {
			subscribeCalled = true
			return nil
		},
	}

	handler := NewPodConnectHandler(podSvc, mgr, tokenGen, sender, nil)

	c, w := newRelayConnectContext(http.MethodGet, "/pods/pod-abc/relay/connect")
	c.Params = gin.Params{{Key: "key", Value: "pod-abc"}}
	setRelayTenantContext(c, 1, 10)

	handler.GetPodConnection(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.False(t, subscribeCalled, "subscribe should not be called when RunnerID is 0")
}

func TestGetPodConnection_WithGeoResolver(t *testing.T) {
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
	sender := &mockRelayCommandSenderConfigurable{}

	resolver := &mockGeoResolver{
		resolveFn: func(ip string) *geo.Location {
			return &geo.Location{
				Latitude:  40.7128,
				Longitude: -74.0060,
				Country:   "US",
			}
		},
	}

	handler := NewPodConnectHandler(podSvc, mgr, tokenGen, sender, resolver)

	c, w := newRelayConnectContext(http.MethodGet, "/pods/pod-abc/relay/connect")
	c.Params = gin.Params{{Key: "key", Value: "pod-abc"}}
	setRelayTenantContext(c, 1, 10)

	handler.GetPodConnection(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp PodConnectResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.RelayURL)
	assert.NotEmpty(t, resp.Token)
}

func TestGetPodConnection_GeoResolverReturnsNil(t *testing.T) {
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
	sender := &mockRelayCommandSenderConfigurable{}

	// Resolver returns nil — no geo info available
	resolver := &mockGeoResolver{
		resolveFn: func(ip string) *geo.Location {
			return nil
		},
	}

	handler := NewPodConnectHandler(podSvc, mgr, tokenGen, sender, resolver)

	c, w := newRelayConnectContext(http.MethodGet, "/pods/pod-abc/relay/connect")
	c.Params = gin.Params{{Key: "key", Value: "pod-abc"}}
	setRelayTenantContext(c, 1, 10)

	handler.GetPodConnection(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

// --- mock types used only in relay connect tests ---
// These are separate from pod_acp_test.go mocks to avoid name collision.

// mockRelayPodService implements PodServiceForHandler for relay connect tests.
type mockRelayPodService struct {
	getPodFn func(ctx context.Context, podKey string) (*agentpod.Pod, error)
}

func (m *mockRelayPodService) ListPods(context.Context, int64, []string, int64, int, int) ([]*agentpod.Pod, int64, error) {
	return nil, 0, nil
}
func (m *mockRelayPodService) CreatePod(context.Context, *agentpodSvc.CreatePodRequest) (*agentpod.Pod, error) {
	return nil, nil
}
func (m *mockRelayPodService) GetPod(ctx context.Context, podKey string) (*agentpod.Pod, error) {
	if m.getPodFn != nil {
		return m.getPodFn(ctx, podKey)
	}
	return nil, errors.New("not found")
}
func (m *mockRelayPodService) TerminatePod(context.Context, string) error { return nil }
func (m *mockRelayPodService) GetPodsByTicket(context.Context, int64) ([]*agentpod.Pod, error) {
	return nil, nil
}
func (m *mockRelayPodService) UpdateAlias(context.Context, string, *string) error { return nil }
func (m *mockRelayPodService) GetActivePodBySourcePodKey(context.Context, string) (*agentpod.Pod, error) {
	return nil, nil
}

// mockRelayCommandSender implements runner.RunnerCommandSender for relay connect tests (no-op).
type mockRelayCommandSender struct{}

func (m *mockRelayCommandSender) SendCreatePod(context.Context, int64, *runnerv1.CreatePodCommand) error {
	return nil
}
func (m *mockRelayCommandSender) SendTerminatePod(context.Context, int64, string) error { return nil }
func (m *mockRelayCommandSender) SendPodInput(context.Context, int64, string, []byte) error {
	return nil
}
func (m *mockRelayCommandSender) SendPrompt(context.Context, int64, string, string) error {
	return nil
}
func (m *mockRelayCommandSender) SendSubscribePod(context.Context, int64, string, string, string, bool, int32) error {
	return nil
}
func (m *mockRelayCommandSender) SendUnsubscribePod(context.Context, int64, string) error {
	return nil
}
func (m *mockRelayCommandSender) SendObservePod(context.Context, int64, string, string, int32, bool) error {
	return nil
}
func (m *mockRelayCommandSender) SendCreateAutopilot(int64, *runnerv1.CreateAutopilotCommand) error {
	return nil
}
func (m *mockRelayCommandSender) SendAutopilotControl(int64, *runnerv1.AutopilotControlCommand) error {
	return nil
}

// mockRelayCommandSenderConfigurable allows configuring individual method behaviors.
type mockRelayCommandSenderConfigurable struct {
	sendSubscribePodFn func(context.Context, int64, string, string, string, bool, int32) error
}

func (m *mockRelayCommandSenderConfigurable) SendCreatePod(context.Context, int64, *runnerv1.CreatePodCommand) error {
	return nil
}
func (m *mockRelayCommandSenderConfigurable) SendTerminatePod(context.Context, int64, string) error {
	return nil
}
func (m *mockRelayCommandSenderConfigurable) SendPodInput(context.Context, int64, string, []byte) error {
	return nil
}
func (m *mockRelayCommandSenderConfigurable) SendPrompt(context.Context, int64, string, string) error {
	return nil
}
func (m *mockRelayCommandSenderConfigurable) SendSubscribePod(ctx context.Context, runnerID int64, podKey, relayURL, token string, snapshot bool, lines int32) error {
	if m.sendSubscribePodFn != nil {
		return m.sendSubscribePodFn(ctx, runnerID, podKey, relayURL, token, snapshot, lines)
	}
	return nil
}
func (m *mockRelayCommandSenderConfigurable) SendUnsubscribePod(context.Context, int64, string) error {
	return nil
}
func (m *mockRelayCommandSenderConfigurable) SendObservePod(context.Context, int64, string, string, int32, bool) error {
	return nil
}
func (m *mockRelayCommandSenderConfigurable) SendCreateAutopilot(int64, *runnerv1.CreateAutopilotCommand) error {
	return nil
}
func (m *mockRelayCommandSenderConfigurable) SendAutopilotControl(int64, *runnerv1.AutopilotControlCommand) error {
	return nil
}

// mockGeoResolver implements geo.Resolver for testing.
type mockGeoResolver struct {
	resolveFn func(ip string) *geo.Location
}

func (m *mockGeoResolver) Resolve(ip string) *geo.Location {
	if m.resolveFn != nil {
		return m.resolveFn(ip)
	}
	return nil
}

func (m *mockGeoResolver) Close() error {
	return nil
}
