package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync/atomic"
	"time"
)

// SendHeartbeat sends a heartbeat to the backend
func (c *Client) SendHeartbeat(ctx context.Context, connections int) error {
	c.mu.RLock()
	if !c.registered {
		c.mu.RUnlock()
		return fmt.Errorf("relay not registered")
	}
	lastLatency := c.lastLatencyMs
	needCert := c.tlsCert == "" || c.tlsKey == "" // Request cert if we don't have one
	c.mu.RUnlock()

	// Get CPU and memory usage (simplified)
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	memoryUsage := float64(memStats.Alloc) / float64(memStats.Sys) * 100

	req := HeartbeatRequest{
		RelayID:     c.relayID,
		Connections: connections,
		CPUUsage:    0, // CPU usage would need more sophisticated measurement
		MemoryUsage: memoryUsage,
		LatencyMs:   lastLatency, // Send last measured latency
		NeedCert:    needCert,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal heartbeat request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/internal/relays/heartbeat", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create heartbeat request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Internal-Secret", c.internalAPISecret)

	// Measure round-trip latency
	start := time.Now()
	resp, err := c.httpClient.Do(httpReq)
	latency := time.Since(start)

	if err != nil {
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}
	defer drainBody(resp.Body)

	// Store latency for next heartbeat
	c.mu.Lock()
	c.lastLatencyMs = int(latency.Milliseconds())
	c.mu.Unlock()

	if resp.StatusCode != http.StatusOK {
		// Try to re-register if heartbeat fails
		if resp.StatusCode == http.StatusNotFound {
			c.mu.Lock()
			c.registered = false
			c.mu.Unlock()
			return fmt.Errorf("relay not found, need to re-register")
		}
		return fmt.Errorf("heartbeat failed with status: %d", resp.StatusCode)
	}

	// Parse response to check for certificate
	var heartbeatResp HeartbeatResponse
	if err := json.NewDecoder(resp.Body).Decode(&heartbeatResp); err == nil {
		if heartbeatResp.TLSCert != "" && heartbeatResp.TLSKey != "" {
			c.mu.Lock()
			c.tlsCert = heartbeatResp.TLSCert
			c.tlsKey = heartbeatResp.TLSKey
			c.tlsExpiry = heartbeatResp.TLSExpiry
			c.mu.Unlock()

			// Save certificate to files for persistence across restarts
			if err := c.saveCertificateFiles(heartbeatResp.TLSCert, heartbeatResp.TLSKey); err != nil {
				c.logger.Warn("Failed to save certificate files", "error", err)
			}

			c.logger.Info("TLS certificate received from heartbeat", "expiry", heartbeatResp.TLSExpiry)
		}
	}

	return nil
}

// StartHeartbeat starts the heartbeat loop.
// It logs lifecycle events and recovers from panics to provide diagnostics.
// The getConnections callback has a 5-second timeout to prevent deadlock propagation.
func (c *Client) StartHeartbeat(ctx context.Context, interval time.Duration, getConnections func() int) {
	c.logger.Info("Heartbeat goroutine started", "interval", interval)
	defer func() {
		if r := recover(); r != nil {
			c.logger.Error("Heartbeat goroutine panicked", "panic", r)
		}
		c.logger.Info("Heartbeat goroutine stopped")
	}()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Tracks whether a getConnections goroutine is still running.
	// Prevents goroutine accumulation when getConnections is blocked.
	var getConnBusy atomic.Bool

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Heartbeat stopping: context cancelled")
			return
		case <-ticker.C:
			var connections int
			if getConnBusy.Load() {
				// Previous getConnections still running — skip to avoid goroutine pileup
				c.logger.Debug("Previous getConnections still running, using 0")
				connections = 0
			} else {
				// Get connections with timeout to prevent deadlock propagation
				// (e.g., if Stats() is blocked by a lock held by a slow WebSocket write)
				connCh := make(chan int, 1)
				getConnBusy.Store(true)
				go func() {
					defer getConnBusy.Store(false)
					connCh <- getConnections()
				}()

				select {
				case connections = <-connCh:
					// Success
				case <-time.After(5 * time.Second):
					c.logger.Warn("getConnections callback timed out, using 0")
					connections = 0
				case <-ctx.Done():
					c.logger.Info("Heartbeat stopping: context cancelled during getConnections")
					return
				}
			}

			if err := c.SendHeartbeat(ctx, connections); err != nil {
				c.logger.Warn("Heartbeat failed", "error", err)

				// Try to re-register
				if err := c.Register(ctx); err != nil {
					c.logger.Error("Re-registration failed", "error", err)
				}
			}
		}
	}
}
