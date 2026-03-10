// Package client provides gRPC connection management for Runner.
package client

import "time"

// IsInitialized returns whether the connection has completed initialization.
func (c *GRPCConnection) IsInitialized() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.initialized
}

// GetAvailableAgents returns the list of available agents.
func (c *GRPCConnection) GetAvailableAgents() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.availableAgents
}

// LastActivityTime returns the most recent send or recv timestamp.
// Used by the Watchdog health checker to detect stuck connections.
func (c *GRPCConnection) LastActivityTime() time.Time {
	sendNs := c.lastSendTime.Load()
	recvNs := c.lastRecvTime.Load()
	latest := max(sendNs, recvNs)
	if latest == 0 {
		return time.Time{}
	}
	return time.Unix(0, latest)
}

// Ensure GRPCConnection implements Connection interface.
var _ Connection = (*GRPCConnection)(nil)
