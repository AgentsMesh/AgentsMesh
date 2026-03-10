package client

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWithGRPCHeartbeatInterval(t *testing.T) {
	conn := NewGRPCConnection("localhost:9443", "n", "o", "", "", "",
		WithGRPCHeartbeatInterval(10*time.Second))
	assert.Equal(t, 10*time.Second, conn.heartbeatInterval)
}

func TestWithGRPCInitTimeout(t *testing.T) {
	conn := NewGRPCConnection("localhost:9443", "n", "o", "", "", "",
		WithGRPCInitTimeout(60*time.Second))
	assert.Equal(t, 60*time.Second, conn.initTimeout)
}

func TestWithGRPCRunnerVersion(t *testing.T) {
	conn := NewGRPCConnection("localhost:9443", "n", "o", "", "", "",
		WithGRPCRunnerVersion("1.2.3"))
	assert.Equal(t, "1.2.3", conn.runnerVersion)
}

func TestWithGRPCMCPPort(t *testing.T) {
	conn := NewGRPCConnection("localhost:9443", "n", "o", "", "", "",
		WithGRPCMCPPort(19001))
	assert.Equal(t, 19001, conn.mcpPort)
}

func TestWithGRPCTerminalRateLimit(t *testing.T) {
	conn := NewGRPCConnection("localhost:9443", "n", "o", "", "", "",
		WithGRPCTerminalRateLimit(200*1024))
	assert.Equal(t, 200*1024, conn.terminalRateLimit)
}

func TestWithGRPCServerURL(t *testing.T) {
	conn := NewGRPCConnection("localhost:9443", "n", "o", "", "", "",
		WithGRPCServerURL("https://api.example.com"))
	assert.Equal(t, "https://api.example.com", conn.serverURL)
}

func TestWithGRPCCertRenewalDays(t *testing.T) {
	conn := NewGRPCConnection("localhost:9443", "n", "o", "", "", "",
		WithGRPCCertRenewalDays(60))
	assert.Equal(t, 60, conn.certRenewalDays)
}

func TestWithGRPCCertUrgentDays(t *testing.T) {
	conn := NewGRPCConnection("localhost:9443", "n", "o", "", "", "",
		WithGRPCCertUrgentDays(3))
	assert.Equal(t, 3, conn.certUrgentDays)
}

func TestWithGRPCEndpointChanged(t *testing.T) {
	called := false
	conn := NewGRPCConnection("localhost:9443", "n", "o", "", "", "",
		WithGRPCEndpointChanged(func(newEndpoint string) error {
			called = true
			return nil
		}))
	assert.NotNil(t, conn.onEndpointChanged)
	_ = conn.onEndpointChanged("new:9443")
	assert.True(t, called)
}

func TestMultipleOptions(t *testing.T) {
	conn := NewGRPCConnection("localhost:9443", "n", "o", "", "", "",
		WithGRPCHeartbeatInterval(15*time.Second),
		WithGRPCRunnerVersion("2.0.0"),
		WithGRPCMCPPort(19002),
	)
	assert.Equal(t, 15*time.Second, conn.heartbeatInterval)
	assert.Equal(t, "2.0.0", conn.runnerVersion)
	assert.Equal(t, 19002, conn.mcpPort)
}
