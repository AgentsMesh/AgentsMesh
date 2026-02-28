// Package client provides gRPC connection management for Runner.
package client

import "time"

// GRPCConnectionOption is a functional option for GRPCConnection.
type GRPCConnectionOption func(*GRPCConnection)

// WithGRPCHeartbeatInterval sets the heartbeat interval.
func WithGRPCHeartbeatInterval(d time.Duration) GRPCConnectionOption {
	return func(c *GRPCConnection) {
		c.heartbeatInterval = d
	}
}

// WithGRPCInitTimeout sets the initialization timeout.
func WithGRPCInitTimeout(d time.Duration) GRPCConnectionOption {
	return func(c *GRPCConnection) {
		c.initTimeout = d
	}
}

// WithGRPCRunnerVersion sets the runner version.
func WithGRPCRunnerVersion(version string) GRPCConnectionOption {
	return func(c *GRPCConnection) {
		c.runnerVersion = version
	}
}

// WithGRPCMCPPort sets the MCP port.
func WithGRPCMCPPort(port int) GRPCConnectionOption {
	return func(c *GRPCConnection) {
		c.mcpPort = port
	}
}

// WithGRPCTerminalRateLimit sets the terminal output rate limit in bytes per second.
// Default is 100KB/s. Set to 0 to disable rate limiting.
// Recommended: Set to ~80% of server upload bandwidth to leave room for control messages.
func WithGRPCTerminalRateLimit(bytesPerSecond int) GRPCConnectionOption {
	return func(c *GRPCConnection) {
		c.terminalRateLimit = bytesPerSecond
	}
}

// WithGRPCServerURL sets the HTTP server URL for REST API calls.
func WithGRPCServerURL(serverURL string) GRPCConnectionOption {
	return func(c *GRPCConnection) {
		c.serverURL = serverURL
	}
}

// WithGRPCCertRenewalDays sets the days before expiry to trigger renewal.
func WithGRPCCertRenewalDays(days int) GRPCConnectionOption {
	return func(c *GRPCConnection) {
		c.certRenewalDays = days
	}
}

// WithGRPCCertUrgentDays sets the days before expiry for urgent reconnection.
func WithGRPCCertUrgentDays(days int) GRPCConnectionOption {
	return func(c *GRPCConnection) {
		c.certUrgentDays = days
	}
}

// WithGRPCTLSServerName sets the TLS server name for SNI and certificate verification.
// This is needed when the server certificate's SANs don't include the public hostname
// (e.g., server cert is for "agentmesh-backend" but endpoint is "agentsmesh.ai:9443").
func WithGRPCTLSServerName(name string) GRPCConnectionOption {
	return func(c *GRPCConnection) {
		c.tlsServerName = name
	}
}
