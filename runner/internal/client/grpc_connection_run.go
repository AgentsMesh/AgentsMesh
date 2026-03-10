// Package client provides gRPC connection management for Runner.
package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/safego"
	"google.golang.org/grpc/metadata"
)

// runConnection establishes the bidirectional stream and handles messages.
// All child goroutines are tracked via WaitGroup to prevent goroutine leaks on reconnection.
func (c *GRPCConnection) runConnection() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Clear terminal queue before establishing new connection
	// Old terminal output is stale after reconnection and would:
	// 1. Delay initialization by flooding the new stream
	// 2. Potentially cause immediate timeout if backend is slow
	// TUI frames are expendable - users will see fresh output after reconnection
	c.drainTerminalQueue()

	// Add org_slug to metadata for organization routing
	ctx = metadata.AppendToOutgoingContext(ctx, "x-org-slug", c.orgSlug)

	logger.GRPC().Debug("Establishing bidirectional stream", "org", c.orgSlug)

	// Create bidirectional stream
	stream, err := c.client.Connect(ctx)
	if err != nil {
		logger.GRPC().Error("Failed to establish stream", "error", err)
		return
	}

	c.mu.Lock()
	c.stream = stream
	c.mu.Unlock()

	// Initialize recv liveness timestamp so the watchdog doesn't fire prematurely.
	c.lastRecvTime.Store(time.Now().UnixNano())

	// Create heartbeat monitor for this connection.
	// Triggers reconnect if 3 consecutive heartbeats go unacknowledged,
	// detecting upstream (Runner->Backend) path failure.
	c.heartbeatMonitor = NewHeartbeatMonitor(3, func() {
		cancel() // Cancel stream context -> triggers reconnection
	})

	logger.GRPC().Info("Bidirectional stream established")

	done := make(chan struct{})
	readLoopDone := make(chan struct{}) // Signal when readLoop exits

	// WaitGroup tracks all child goroutines spawned in this connection lifecycle.
	// We must wait for them to exit before returning, otherwise reconnection
	// spawns new goroutines while old ones are still running -> goroutine leak.
	var wg sync.WaitGroup

	logger.GRPC().Debug("Starting read/write loops")

	// Start write loop
	wg.Add(1)
	safego.Go("grpc-write-loop", func() {
		defer wg.Done()
		c.writeLoop(ctx, done)
	})

	// IMPORTANT: Start read loop BEFORE initialization
	// The read loop must be running to receive the initialize_result response
	wg.Add(1)
	safego.Go("grpc-read-loop", func() {
		defer wg.Done()
		c.readLoop(ctx, readLoopDone)
	})

	// Perform initialization (blocks until handshake completes or times out)
	if err := c.performInitialization(ctx); err != nil {
		logger.GRPC().Error("Initialization failed", "error", err)
		close(done)
		wg.Wait()
		return
	}

	// Start heartbeat loop (only after successful initialization)
	wg.Add(1)
	safego.Go("grpc-heartbeat", func() {
		defer wg.Done()
		c.heartbeatLoop(ctx, done)
	})

	// Start certificate renewal checker
	wg.Add(1)
	safego.Go("grpc-cert-renewal", func() {
		defer wg.Done()
		c.certRenewalChecker(ctx, done)
	})

	// Start recv watchdog — detects half-dead connections where readLoop
	// is stuck on Recv() after the server closed the downstream.
	// Backend sends downstream pings every 30s; if nothing arrives for
	// 3x heartbeatInterval the connection is dead.
	wg.Add(1)
	safego.Go("grpc-recv-watchdog", func() {
		defer wg.Done()
		c.recvWatchdog(done, cancel)
	})

	// Monitor for reconnection signal (certificate renewal)
	wg.Add(1)
	safego.Go("grpc-reconnect-monitor", func() {
		defer wg.Done()
		select {
		case <-c.reconnectCh:
			logger.GRPC().Info("Reconnection requested for certificate renewal")
			cancel() // Cancel context to trigger reconnection
		case <-done:
			return
		case <-c.stopCh:
			return
		}
	})

	// Wait for context cancellation, stop signal, or readLoop exit
	select {
	case <-ctx.Done():
		logger.GRPC().Debug("Context cancelled, closing connection")
	case <-c.stopCh:
		logger.GRPC().Debug("Stop signal received, closing connection")
	case <-readLoopDone:
		logger.GRPC().Debug("Read loop exited, closing connection")
	}

	// Clear stream to prevent sending to disconnected stream
	// This ensures sendTerminal/sendControl will reject new messages during reconnect
	c.mu.Lock()
	c.stream = nil
	c.mu.Unlock()

	// Signal other goroutines to stop
	close(done)

	// Wait for all child goroutines to exit before returning.
	// This prevents goroutine accumulation across reconnections.
	wg.Wait()
	logger.GRPC().Debug("All child goroutines exited, runConnection returning")
}

// recvWatchdog monitors for recv liveness. If no message is received from the
// server for 3x heartbeatInterval (90s by default, matching the backend's
// downstream pong timeout), the connection is considered half-dead and
// reconnection is triggered by cancelling the stream context.
//
// This handles the case where the backend has closed the downstream send loop
// (e.g. pong timeout) but the runner's stream.Recv() keeps blocking because
// the gRPC transport hasn't detected the closure.
func (c *GRPCConnection) recvWatchdog(done <-chan struct{}, cancel context.CancelFunc) {
	recvTimeout := 3 * c.heartbeatInterval
	ticker := time.NewTicker(c.heartbeatInterval)
	defer ticker.Stop()

	log := logger.GRPC()

	for {
		select {
		case <-c.stopCh:
			return
		case <-done:
			return
		case <-ticker.C:
			lastRecvNs := c.lastRecvTime.Load()
			if lastRecvNs == 0 {
				continue // Not yet initialized
			}
			lastRecv := time.Unix(0, lastRecvNs)
			since := time.Since(lastRecv)
			if since > recvTimeout {
				log.Error("Recv watchdog: no message from server, triggering reconnect",
					"timeout", recvTimeout, "last_recv_ago", since)
				cancel() // Cancel stream context -> unblocks Recv() -> readLoop exits
				return
			}
		}
	}
}

// buildMTLSConfig builds a TLS config for mTLS HTTP requests using the runner's
// certificate, key, and CA files. Returns an error if any file cannot be loaded.
// This follows the same pattern as RenewCertificate in grpc_registration_renewal.go.
func buildMTLSConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}

	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caPool,
		MinVersion:   tls.VersionTLS13,
	}, nil
}
