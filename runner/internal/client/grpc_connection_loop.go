// Package client provides gRPC connection management for Runner.
package client

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// connectionLoop manages the connection lifecycle with auto-reconnection.
func (c *GRPCConnection) connectionLoop() {
	logger.GRPC().Info("Connection loop starting", "endpoint", c.endpoint)
	for {
		select {
		case <-c.stopCh:
			logger.GRPC().Info("Connection loop stopped")
			return
		default:
		}

		// Try to connect
		if err := c.Connect(); err != nil {
			attempt := c.reconnectStrategy.AttemptCount()
			delay := c.reconnectStrategy.NextDelay()
			logger.GRPC().Warn("Failed to connect, will retry",
				"attempt", attempt+1,
				"endpoint", c.endpoint,
				"error", err,
				"retry_in", delay)

			// After every 3 failed attempts, try auto-discovering a new endpoint.
			// This self-heals runners with stale grpc_endpoint configs (e.g. after
			// server port changes in dev or server migrations in prod).
			if (attempt+1)%3 == 0 && c.serverURL != "" {
				c.tryEndpointDiscovery()
			}

			select {
			case <-c.stopCh:
				return
			case <-time.After(delay):
			}
			continue
		}

		// Reset reconnect strategy on successful connection
		c.reconnectStrategy.Reset()

		// Run the connection (blocks until disconnected)
		c.runConnection()

		// Check if we should stop
		select {
		case <-c.stopCh:
			return
		default:
		}

		// Check for fatal errors that should not be retried
		if fatalErr := c.getFatalError(); fatalErr != nil {
			logger.GRPC().Error("Connection terminated due to fatal error (will not reconnect)",
				"error", fatalErr)
			return
		}

		// Wait before reconnecting
		logger.GRPC().Info("Connection closed, will attempt to reconnect")
		select {
		case <-c.stopCh:
			return
		case <-time.After(c.reconnectStrategy.CurrentInterval()):
		}
	}
}

// tryEndpointDiscovery queries the backend discovery endpoint and updates the gRPC
// endpoint if it has changed. This allows runners to self-heal when the server's
// gRPC port or hostname changes without requiring full re-registration.
// Uses mTLS authentication — if the TLS config cannot be built (e.g. certificates
// not yet provisioned), discovery is silently skipped.
func (c *GRPCConnection) tryEndpointDiscovery() {
	log := logger.GRPC()
	log.Info("Trying endpoint auto-discovery", "server_url", c.serverURL, "current_endpoint", c.endpoint)

	// Build mTLS config using existing cert/key/ca files
	tlsConfig, err := buildMTLSConfig(c.certFile, c.keyFile, c.caFile)
	if err != nil {
		log.Warn("Cannot perform endpoint discovery (mTLS config failed, certificates may not exist yet)", "error", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	newEndpoint, err := DiscoverGRPCEndpoint(ctx, c.serverURL, tlsConfig)
	if err != nil {
		log.Warn("Endpoint discovery failed", "error", err)
		return
	}

	c.mu.Lock()
	if newEndpoint == c.endpoint {
		currentEndpoint := c.endpoint
		c.mu.Unlock()
		log.Debug("Endpoint unchanged after discovery", "endpoint", currentEndpoint)
		return
	}
	oldEndpoint := c.endpoint
	c.endpoint = newEndpoint
	c.mu.Unlock()

	log.Info("Auto-discovered new gRPC endpoint",
		"old_endpoint", oldEndpoint,
		"new_endpoint", newEndpoint)

	// Persist the new endpoint to config file so restarts use the updated value
	if c.onEndpointChanged != nil {
		if err := c.onEndpointChanged(newEndpoint); err != nil {
			log.Warn("Failed to persist updated endpoint to config", "error", err)
		} else {
			log.Info("Updated grpc_endpoint in config file")
		}
	}

	// Reset backoff so we reconnect quickly with the new endpoint
	c.reconnectStrategy.Reset()
}

// Note: runConnection, recvWatchdog, buildMTLSConfig are in grpc_connection_run.go
