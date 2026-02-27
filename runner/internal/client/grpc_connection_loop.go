// Package client provides gRPC connection management for Runner.
package client

import (
	"context"
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/safego"
	"google.golang.org/grpc/metadata"
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
			delay := c.reconnectStrategy.NextDelay()
			logger.GRPC().Warn("Failed to connect, will retry",
				"attempt", c.reconnectStrategy.AttemptCount(),
				"error", err,
				"retry_in", delay)

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

	logger.GRPC().Info("Bidirectional stream established")

	done := make(chan struct{})
	readLoopDone := make(chan struct{}) // Signal when readLoop exits

	// WaitGroup tracks all child goroutines spawned in this connection lifecycle.
	// We must wait for them to exit before returning, otherwise reconnection
	// spawns new goroutines while old ones are still running → goroutine leak.
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
