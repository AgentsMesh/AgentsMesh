package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Client communicates with Backend API
type Client struct {
	baseURL           string
	internalAPISecret string
	relayID           string
	relayName         string // Name for DNS auto-registration
	relayURL          string
	relayRegion       string
	relayCapacity     int
	relayIP           string // Public IP for DNS auto-registration
	autoIP            bool   // Auto-detect public IP

	httpClient *http.Client

	// Registration state
	registered bool

	// TLS certificate (received from backend)
	tlsCert   string // PEM encoded certificate chain
	tlsKey    string // PEM encoded private key
	tlsExpiry string // Certificate expiry time (RFC3339)

	// Certificate storage paths
	certFile string // Path to save certificate PEM
	keyFile  string // Path to save private key PEM

	// Latency tracking for load balancing
	lastLatencyMs int // Last measured heartbeat round-trip latency

	mu sync.RWMutex

	logger *slog.Logger
}

// ClientConfig holds configuration for backend client
type ClientConfig struct {
	BaseURL           string
	InternalAPISecret string
	RelayID           string
	RelayName         string
	RelayURL          string
	RelayRegion       string
	RelayCapacity     int
	AutoIP            bool
	CertFile          string // Path to save/load certificate PEM
	KeyFile           string // Path to save/load private key PEM
}

// NewClient creates a new backend client
func NewClient(baseURL, internalAPISecret, relayID, relayURL, relayRegion string, relayCapacity int) *Client {
	return &Client{
		baseURL:           baseURL,
		internalAPISecret: internalAPISecret,
		relayID:           relayID,
		relayURL:          relayURL,
		relayRegion:       relayRegion,
		relayCapacity:     relayCapacity,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: slog.With("component", "backend_client"),
	}
}

// NewClientWithConfig creates a new backend client with full configuration
func NewClientWithConfig(cfg ClientConfig) *Client {
	c := &Client{
		baseURL:           cfg.BaseURL,
		internalAPISecret: cfg.InternalAPISecret,
		relayID:           cfg.RelayID,
		relayName:         cfg.RelayName,
		relayURL:          cfg.RelayURL,
		relayRegion:       cfg.RelayRegion,
		relayCapacity:     cfg.RelayCapacity,
		autoIP:            cfg.AutoIP,
		certFile:          cfg.CertFile,
		keyFile:           cfg.KeyFile,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: slog.With("component", "backend_client"),
	}

	// Try to load existing certificate from files
	if cfg.CertFile != "" && cfg.KeyFile != "" {
		if err := c.loadCertificateFiles(); err == nil {
			c.logger.Info("Loaded existing TLS certificate from files")
		}
	}

	return c
}

// RegisterRequest represents relay registration request
type RegisterRequest struct {
	RelayID  string `json:"relay_id"`
	RelayName string `json:"relay_name,omitempty"` // Name for DNS auto-registration
	IP       string `json:"ip,omitempty"`          // Public IP for DNS auto-registration
	URL      string `json:"url,omitempty"`         // Public URL (optional if DNS auto)
	Region   string `json:"region"`
	Capacity int    `json:"capacity"`
}

// RegisterResponse represents relay registration response
type RegisterResponse struct {
	Status     string `json:"status"`
	URL        string `json:"url,omitempty"`         // Generated URL (if DNS auto-registration)
	DNSCreated bool   `json:"dns_created,omitempty"` // Whether DNS record was created

	// TLS certificate (if ACME is enabled on backend)
	TLSCert   string `json:"tls_cert,omitempty"`   // PEM encoded certificate chain
	TLSKey    string `json:"tls_key,omitempty"`    // PEM encoded private key
	TLSExpiry string `json:"tls_expiry,omitempty"` // Certificate expiry time (RFC3339)
}

// HeartbeatRequest represents heartbeat request
type HeartbeatRequest struct {
	RelayID     string  `json:"relay_id"`
	Connections int     `json:"connections"`
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	LatencyMs   int     `json:"latency_ms,omitempty"` // Heartbeat round-trip latency
	NeedCert    bool    `json:"need_cert,omitempty"`  // Whether relay needs TLS certificate
}

// HeartbeatResponse represents heartbeat response
type HeartbeatResponse struct {
	Status string `json:"status"`

	// TLS certificate (if ACME is enabled on backend)
	TLSCert   string `json:"tls_cert,omitempty"`
	TLSKey    string `json:"tls_key,omitempty"`
	TLSExpiry string `json:"tls_expiry,omitempty"`
}

// SessionClosedRequest represents session closed notification
type SessionClosedRequest struct {
	PodKey    string `json:"pod_key"`
	SessionID string `json:"session_id"`
}

// drainBody discards up to 4KB of response body so the underlying TCP connection
// can be reused by net/http's connection pool, then closes the body.
func drainBody(body io.ReadCloser) {
	_, _ = io.Copy(io.Discard, io.LimitReader(body, 4096))
	body.Close()
}

// Register registers this relay with the backend
func (c *Client) Register(ctx context.Context) error {
	// Auto-detect public IP if enabled and relay name is set
	if c.autoIP && c.relayName != "" && c.relayIP == "" {
		ip, err := c.detectPublicIP(ctx)
		if err != nil {
			c.logger.Warn("Failed to auto-detect public IP", "error", err)
		} else {
			c.relayIP = ip
			c.logger.Info("Auto-detected public IP", "ip", ip)
		}
	}

	req := RegisterRequest{
		RelayID:  c.relayID,
		RelayName: c.relayName,
		IP:       c.relayIP,
		URL:      c.relayURL,
		Region:   c.relayRegion,
		Capacity: c.relayCapacity,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal register request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/internal/relays/register", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create register request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Internal-Secret", c.internalAPISecret)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send register request: %w", err)
	}
	defer drainBody(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registration failed with status: %d", resp.StatusCode)
	}

	// Parse response
	var regResp RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&regResp); err != nil {
		c.logger.Warn("Failed to decode registration response", "error", err)
	} else {
		c.mu.Lock()
		// Update relay URL if returned from backend (DNS auto-registration)
		if regResp.URL != "" && regResp.DNSCreated {
			c.relayURL = regResp.URL
			c.logger.Info("DNS record created, using generated URL", "url", regResp.URL)
		}
		// Store TLS certificate if returned from backend (ACME)
		if regResp.TLSCert != "" && regResp.TLSKey != "" {
			c.tlsCert = regResp.TLSCert
			c.tlsKey = regResp.TLSKey
			c.tlsExpiry = regResp.TLSExpiry
			c.mu.Unlock()

			// Save certificate to files for persistence across restarts
			if err := c.saveCertificateFiles(regResp.TLSCert, regResp.TLSKey); err != nil {
				c.logger.Warn("Failed to save certificate files", "error", err)
			}

			c.logger.Info("TLS certificate received from backend", "expiry", regResp.TLSExpiry)
		} else {
			c.mu.Unlock()
		}
	}

	c.mu.Lock()
	c.registered = true
	c.mu.Unlock()

	c.logger.Info("Registered with backend", "relay_id", c.relayID, "url", c.relayURL)
	return nil
}

// detectPublicIP detects the public IP address using external services
func (c *Client) detectPublicIP(ctx context.Context) (string, error) {
	// List of public IP detection services
	services := []string{
		"https://api.ipify.org",
		"https://ifconfig.me/ip",
		"https://icanhazip.com",
	}

	for _, url := range services {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			continue
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			continue
		}

		body, err := func() ([]byte, error) {
			defer drainBody(resp.Body)
			return io.ReadAll(io.LimitReader(resp.Body, 256)) // IP address is at most 45 bytes
		}()
		if err != nil {
			continue
		}

		ip := strings.TrimSpace(string(body))
		// Validate IP format with strict parsing
		if ip != "" && net.ParseIP(ip) != nil {
			return ip, nil
		}
	}

	return "", fmt.Errorf("failed to detect public IP from all services")
}

// GetRelayURL returns the current relay URL
func (c *Client) GetRelayURL() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.relayURL
}

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

// NotifySessionClosed notifies backend that a session is closed
func (c *Client) NotifySessionClosed(ctx context.Context, podKey, sessionID string) error {
	req := SessionClosedRequest{
		PodKey:    podKey,
		SessionID: sessionID,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal session closed request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/internal/relays/session-closed", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create session closed request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Internal-Secret", c.internalAPISecret)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send session closed notification: %w", err)
	}
	defer drainBody(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("session closed notification failed with status: %d", resp.StatusCode)
	}

	c.logger.Info("Notified backend of session closed", "pod_key", podKey, "session_id", sessionID)
	return nil
}

// UnregisterRequest represents relay unregistration request
type UnregisterRequest struct {
	RelayID string `json:"relay_id"`
	Reason  string `json:"reason,omitempty"`
}

// Unregister notifies backend that this relay is shutting down
func (c *Client) Unregister(ctx context.Context, reason string) error {
	req := UnregisterRequest{
		RelayID: c.relayID,
		Reason:  reason,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal unregister request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/internal/relays/unregister", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create unregister request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Internal-Secret", c.internalAPISecret)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send unregister request: %w", err)
	}
	defer drainBody(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unregistration failed with status: %d", resp.StatusCode)
	}

	c.mu.Lock()
	c.registered = false
	c.mu.Unlock()

	c.logger.Info("Unregistered from backend", "relay_id", c.relayID, "reason", reason)
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

// IsRegistered returns whether the relay is registered
func (c *Client) IsRegistered() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.registered
}

// HasTLSCertificate returns whether a TLS certificate is available
func (c *Client) HasTLSCertificate() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tlsCert != "" && c.tlsKey != ""
}

// GetTLSCertificate returns the TLS certificate and key (PEM encoded)
func (c *Client) GetTLSCertificate() (cert, key string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tlsCert, c.tlsKey
}

// GetTLSExpiry returns the TLS certificate expiry time string
func (c *Client) GetTLSExpiry() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tlsExpiry
}

// saveCertificateFiles saves certificate and key to files atomically.
// Uses write-to-temp-then-rename to prevent inconsistent state on crash.
func (c *Client) saveCertificateFiles(cert, key string) error {
	if c.certFile == "" || c.keyFile == "" {
		return nil // No paths configured, skip saving
	}

	// Write key first (more sensitive) — atomic via temp file + rename
	if err := atomicWriteFile(c.keyFile, []byte(key), 0600); err != nil {
		return fmt.Errorf("failed to write key file: %w", err)
	}

	if err := atomicWriteFile(c.certFile, []byte(cert), 0644); err != nil {
		return fmt.Errorf("failed to write certificate file: %w", err)
	}

	c.logger.Info("Certificate files saved", "cert_file", c.certFile, "key_file", c.keyFile)
	return nil
}

// atomicWriteFile writes data to a temp file then renames it to the target path.
// This ensures the target file is never in a partially-written state.
func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp.*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}

	return os.Rename(tmpName, path)
}

// loadCertificateFiles loads certificate and key from files
func (c *Client) loadCertificateFiles() error {
	if c.certFile == "" || c.keyFile == "" {
		return fmt.Errorf("certificate paths not configured")
	}

	certData, err := os.ReadFile(c.certFile)
	if err != nil {
		return fmt.Errorf("failed to read certificate file: %w", err)
	}

	keyData, err := os.ReadFile(c.keyFile)
	if err != nil {
		return fmt.Errorf("failed to read key file: %w", err)
	}

	c.mu.Lock()
	c.tlsCert = string(certData)
	c.tlsKey = string(keyData)
	c.mu.Unlock()

	return nil
}
