package relay

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIsHandshakeError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "bad handshake error",
			err:      &testError{msg: "websocket: bad handshake"},
			expected: true,
		},
		{
			name:     "401 unauthorized",
			err:      &testError{msg: "websocket dial: unexpected status 401"},
			expected: true,
		},
		{
			name:     "403 forbidden",
			err:      &testError{msg: "websocket dial: unexpected status 403"},
			expected: true,
		},
		{
			name:     "connection refused",
			err:      &testError{msg: "dial tcp: connection refused"},
			expected: false,
		},
		{
			name:     "timeout error",
			err:      &testError{msg: "dial tcp: i/o timeout"},
			expected: false,
		},
		{
			name:     "500 internal server error",
			err:      &testError{msg: "websocket dial: unexpected status 500"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isHandshakeError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// testError is a simple error type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestClientUpdateToken(t *testing.T) {
	client := NewClient("wss://relay.example.com", "pod-123", "session-456", "old-token", nil)

	// Verify initial token
	client.connMu.RLock()
	initialToken := client.token
	client.connMu.RUnlock()
	assert.Equal(t, "old-token", initialToken)

	// Update token
	client.UpdateToken("new-token")

	// Verify updated token
	client.connMu.RLock()
	updatedToken := client.token
	client.connMu.RUnlock()
	assert.Equal(t, "new-token", updatedToken)
}

func TestClientGetSessionID(t *testing.T) {
	client := NewClient("wss://relay.example.com", "pod-123", "session-456", "token", nil)
	assert.Equal(t, "session-456", client.GetSessionID())
}

func TestClientGetRelayURL(t *testing.T) {
	client := NewClient("wss://relay.example.com", "pod-123", "session-456", "token", nil)
	assert.Equal(t, "wss://relay.example.com", client.GetRelayURL())
}

func TestClientSetTokenExpiredHandler(t *testing.T) {
	client := NewClient("wss://relay.example.com", "pod-123", "session-456", "token", nil)

	// Initially nil
	assert.Nil(t, client.onTokenExpired)

	// Set handler
	handlerCalled := false
	client.SetTokenExpiredHandler(func() string {
		handlerCalled = true
		return "refreshed-token"
	})

	// Verify handler is set
	assert.NotNil(t, client.onTokenExpired)

	// Call handler
	result := client.onTokenExpired()
	assert.True(t, handlerCalled)
	assert.Equal(t, "refreshed-token", result)
}

func TestClientTokenRefreshIntegration(t *testing.T) {
	// This test simulates the token refresh flow without actual network connections

	client := NewClient("wss://relay.example.com", "pod-123", "session-456", "old-token", nil)

	tokenRefreshRequested := make(chan struct{}, 1)
	newTokenDelivered := make(chan string, 1)

	// Simulate token expired handler (what Runner does)
	client.SetTokenExpiredHandler(func() string {
		tokenRefreshRequested <- struct{}{}
		// Simulate waiting for Backend response (with timeout)
		select {
		case token := <-newTokenDelivered:
			return token
		case <-time.After(100 * time.Millisecond):
			return ""
		}
	})

	// Simulate token refresh flow
	go func() {
		// Wait for token refresh request
		<-tokenRefreshRequested
		// Simulate Backend sending new token
		time.Sleep(10 * time.Millisecond)
		newTokenDelivered <- "new-token-from-backend"
	}()

	// Trigger token refresh
	result := client.onTokenExpired()
	assert.Equal(t, "new-token-from-backend", result)
}

func TestClientTokenRefreshTimeout(t *testing.T) {
	client := NewClient("wss://relay.example.com", "pod-123", "session-456", "old-token", nil)

	// Set handler that simulates timeout (no response from Backend)
	client.SetTokenExpiredHandler(func() string {
		// Simulate waiting with short timeout
		time.Sleep(50 * time.Millisecond)
		return "" // Timeout, no token received
	})

	start := time.Now()
	result := client.onTokenExpired()
	elapsed := time.Since(start)

	assert.Empty(t, result)
	assert.True(t, elapsed >= 50*time.Millisecond)
}
