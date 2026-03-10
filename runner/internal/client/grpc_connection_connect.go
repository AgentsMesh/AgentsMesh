// Package client provides gRPC connection management for Runner.
package client

import (
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/safego"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// Connect establishes a gRPC connection with mTLS using advancedtls for certificate hot-reloading.
func (c *GRPCConnection) Connect() error {
	// Close existing connection and certificate providers (important for reconnection)
	// This prevents resource leaks, TLS session conflicts, and goroutine leaks from pemfile providers
	c.mu.Lock()
	if c.conn != nil {
		logger.GRPC().Debug("Closing existing gRPC connection before reconnect")
		c.conn.Close()
		c.conn = nil
	}
	// Close old certificate providers to prevent goroutine leaks
	// Each pemfile.Provider runs background goroutines for file watching
	if c.identityProvider != nil {
		c.identityProvider.Close()
		c.identityProvider = nil
	}
	if c.rootProvider != nil {
		c.rootProvider.Close()
		c.rootProvider = nil
	}
	c.mu.Unlock()

	// Parse endpoint to extract host:port (remove scheme like grpcs://)
	dialTarget, err := parseGRPCEndpoint(c.endpoint)
	if err != nil {
		return fmt.Errorf("failed to parse gRPC endpoint: %w", err)
	}

	// Create advancedtls credentials with file-based certificate reloading
	creds, err := c.createAdvancedTLSCredentials()
	if err != nil {
		return fmt.Errorf("failed to create TLS credentials: %w", err)
	}

	// Fix advancedtls SNI bug: advancedtls.ClientHandshake sets ServerName to the
	// full authority string (e.g. "agentsmesh.ai:9443" including port), unlike
	// standard gRPC credentials which strips the port via net.SplitHostPort.
	// An SNI with port violates RFC 6066 and is rejected by some network middleboxes
	// (corporate firewalls, DPI). Set the correct hostname-only ServerName here so
	// advancedtls won't overwrite it with the authority.
	host, _, err := net.SplitHostPort(dialTarget)
	if err != nil {
		host = dialTarget // No port in dialTarget, use as-is
	}
	if err := creds.OverrideServerName(host); err != nil { //nolint:staticcheck // advancedtls has no other public API to set ServerName; the unexported serverNameOverride field is inaccessible
		logger.GRPC().Warn("Failed to override TLS server name", "error", err)
	}

	// gRPC dial options
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	}

	// Connect to server
	conn, err := grpc.Dial(dialTarget, dialOpts...)
	if err != nil {
		return fmt.Errorf("failed to dial gRPC server: %w", err)
	}

	// Create gRPC service client
	client := runnerv1.NewRunnerServiceClient(conn)

	c.mu.Lock()
	c.conn = conn
	c.client = client
	c.creds = creds
	c.initialized = false
	c.mu.Unlock()

	logger.GRPC().Info("Connected to server with advancedtls", "endpoint", c.endpoint, "org", c.orgSlug)
	return nil
}

// Start starts the connection management loop.
func (c *GRPCConnection) Start() {
	logger.GRPC().Info("gRPC connection manager starting", "endpoint", c.endpoint)
	c.loopWg.Add(1)
	safego.Go("grpc-connection-loop", func() {
		defer c.loopWg.Done()
		c.connectionLoop()
	})
}

// Stop stops the connection and releases resources.
func (c *GRPCConnection) Stop() {
	c.stopOnce.Do(func() {
		logger.GRPC().Info("gRPC connection stopping")
		close(c.stopCh)

		// Wait for connectionLoop to exit before cleaning up resources
		c.loopWg.Wait()

		// Wait for in-flight async handlers (handleCreatePod, etc.)
		c.handlerWg.Wait()

		c.mu.Lock()
		if c.conn != nil {
			c.conn.Close()
			c.conn = nil
		}
		// Close certificate providers to prevent goroutine leaks
		if c.identityProvider != nil {
			c.identityProvider.Close()
			c.identityProvider = nil
		}
		if c.rootProvider != nil {
			c.rootProvider.Close()
			c.rootProvider = nil
		}
		c.mu.Unlock()
		logger.GRPC().Info("gRPC connection stopped")
	})
}

// parseGRPCEndpoint parses a gRPC endpoint URL and returns the host:port for dialing.
// Supports formats:
//   - grpcs://host:port -> host:port (TLS)
//   - grpc://host:port  -> host:port (plain)
//   - host:port         -> host:port (as-is)
func parseGRPCEndpoint(endpoint string) (string, error) {
	log := logger.GRPC()

	// If it doesn't contain a scheme, assume it's already host:port
	if !strings.Contains(endpoint, "://") {
		return endpoint, nil
	}

	// Parse as URL
	u, err := url.Parse(endpoint)
	if err != nil {
		log.Error("Invalid endpoint URL", "endpoint", endpoint, "error", err)
		return "", err
	}

	// Validate scheme
	switch u.Scheme {
	case "grpc", "grpcs":
		// Valid gRPC schemes
	default:
		log.Error("Unsupported gRPC scheme", "scheme", u.Scheme, "endpoint", endpoint)
		return "", fmt.Errorf("unsupported scheme %q", u.Scheme)
	}

	// Return host:port
	if u.Host == "" {
		log.Error("Missing host in endpoint URL", "endpoint", endpoint)
		return "", fmt.Errorf("missing host in endpoint")
	}

	logger.GRPCTrace().Trace("Parsed gRPC endpoint", "endpoint", endpoint, "dial_target", u.Host)
	return u.Host, nil
}
