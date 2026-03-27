package backend

import (
	"io"
	"log/slog"
	"net/http"
	"sync"
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

// UnregisterRequest represents relay unregistration request
type UnregisterRequest struct {
	RelayID string `json:"relay_id"`
	Reason  string `json:"reason,omitempty"`
}

// drainBody discards up to 4KB of response body so the underlying TCP connection
// can be reused by net/http's connection pool, then closes the body.
func drainBody(body io.ReadCloser) {
	_, _ = io.Copy(io.Discard, io.LimitReader(body, 4096))
	_ = body.Close()
}

// GetRelayURL returns the current relay URL
func (c *Client) GetRelayURL() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.relayURL
}

// IsRegistered returns whether the relay is registered
func (c *Client) IsRegistered() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.registered
}

