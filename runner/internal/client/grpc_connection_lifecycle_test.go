package client

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNewGRPCConnection_Defaults(t *testing.T) {
	conn := NewGRPCConnection("localhost:9443", "node-1", "test-org", "/cert", "/key", "/ca")

	assert.Equal(t, "localhost:9443", conn.endpoint)
	assert.Equal(t, "node-1", conn.nodeID)
	assert.Equal(t, "test-org", conn.orgSlug)
	assert.Equal(t, "/cert", conn.certFile)
	assert.Equal(t, "/key", conn.keyFile)
	assert.Equal(t, "/ca", conn.caFile)
	assert.Equal(t, 30*time.Second, conn.heartbeatInterval)
	assert.Equal(t, 30*time.Second, conn.initTimeout)
	assert.NotNil(t, conn.reconnectStrategy)
	assert.NotNil(t, conn.controlCh)
	assert.NotNil(t, conn.terminalCh)
	assert.NotNil(t, conn.stopCh)
	assert.NotNil(t, conn.reconnectCh)
	assert.NotNil(t, conn.initResultCh)
	assert.Equal(t, 100, cap(conn.controlCh))
	assert.Equal(t, 1000, cap(conn.terminalCh))
	assert.False(t, conn.initialized)
}

func TestSetHandler(t *testing.T) {
	conn := NewGRPCConnection("localhost:9443", "node-1", "org", "", "", "")
	handler := &mockHandler{}
	conn.SetHandler(handler)
	assert.Equal(t, handler, conn.handler)
}

func TestOrgSlug_SetGet(t *testing.T) {
	conn := NewGRPCConnection("localhost:9443", "node-1", "org-1", "", "", "")
	assert.Equal(t, "org-1", conn.GetOrgSlug())

	conn.SetOrgSlug("org-2")
	assert.Equal(t, "org-2", conn.GetOrgSlug())
}

func TestIsInitialized(t *testing.T) {
	conn := newTestConnection()
	assert.False(t, conn.IsInitialized())

	conn.mu.Lock()
	conn.initialized = true
	conn.mu.Unlock()
	assert.True(t, conn.IsInitialized())
}

func TestGetAvailableAgents(t *testing.T) {
	conn := newTestConnection()
	assert.Empty(t, conn.GetAvailableAgents())

	conn.mu.Lock()
	conn.availableAgents = []string{"claude", "aider"}
	conn.mu.Unlock()
	assert.Equal(t, []string{"claude", "aider"}, conn.GetAvailableAgents())
}

func TestSetFatalError(t *testing.T) {
	conn := newTestConnection()
	assert.Nil(t, conn.getFatalError())

	conn.setFatalError(status.Error(codes.Unauthenticated, "bad cert"))
	assert.NotNil(t, conn.getFatalError())
}

func TestIsFatalStreamError(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		fatal   bool
		msgPart string
	}{
		{
			name:  "non-grpc error",
			err:   assert.AnError,
			fatal: false,
		},
		{
			name:    "unauthenticated runner not found",
			err:     status.Error(codes.Unauthenticated, "runner not found"),
			fatal:   true,
			msgPart: "re-register",
		},
		{
			name:    "permission denied runner disabled",
			err:     status.Error(codes.PermissionDenied, "runner is disabled"),
			fatal:   true,
			msgPart: "disabled",
		},
		{
			name:  "transient unavailable",
			err:   status.Error(codes.Unavailable, "server busy"),
			fatal: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fatal, msg := isFatalStreamError(tt.err)
			assert.Equal(t, tt.fatal, fatal)
			if tt.fatal {
				assert.Contains(t, msg, tt.msgPart)
			}
		})
	}
}

func TestParseGRPCEndpoint_Extended(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		{"localhost:9443", "localhost:9443", false},
		{"grpcs://api.example.com:9443", "api.example.com:9443", false},
		{"grpc://localhost:9443", "localhost:9443", false},
		{"https://example.com:9443", "", true}, // unsupported scheme
		{"10.0.0.1:443", "10.0.0.1:443", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseGRPCEndpoint(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestHeartbeatMonitor_OnSentOnAck(t *testing.T) {
	unhealthyCalled := false
	m := NewHeartbeatMonitor(3, func() { unhealthyCalled = true })

	assert.Equal(t, int32(0), m.MissedCount())

	m.OnSent()
	assert.Equal(t, int32(1), m.MissedCount())

	m.OnSent()
	assert.Equal(t, int32(2), m.MissedCount())

	m.OnAck()
	assert.Equal(t, int32(0), m.MissedCount())
	assert.False(t, unhealthyCalled)
}

func TestHeartbeatMonitor_TriggersUnhealthy(t *testing.T) {
	unhealthyCalled := false
	m := NewHeartbeatMonitor(3, func() { unhealthyCalled = true })

	m.OnSent() // 1
	m.OnSent() // 2
	assert.False(t, unhealthyCalled)

	m.OnSent() // 3 -> threshold
	assert.True(t, unhealthyCalled)
}

func TestTriggerReconnect(t *testing.T) {
	conn := newTestConnection()
	conn.triggerReconnect()

	select {
	case <-conn.reconnectCh:
		// success
	default:
		t.Fatal("expected reconnect signal")
	}

	// Second call should not block (channel already has capacity 1)
	conn.triggerReconnect()
}
