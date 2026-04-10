package v1

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/service/geo"
	"github.com/anthropics/agentsmesh/backend/internal/service/relay"
)

func TestGetPodConnection_SubscribePodError_StillSucceeds(t *testing.T) {
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
				CreatedByID:    10,
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
				CreatedByID:    10,
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
				CreatedByID:    10,
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
				CreatedByID:    10,
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
