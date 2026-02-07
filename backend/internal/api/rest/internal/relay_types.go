package internal

// RegisterRequest is the relay registration request
type RegisterRequest struct {
	RelayID     string `json:"relay_id" binding:"required"`
	RelayName   string `json:"relay_name"`   // Name for DNS auto-registration (e.g., "us-east-1")
	IP          string `json:"ip"`           // Public IP for DNS auto-registration
	URL         string `json:"url"`          // Public URL for browsers (optional if DNS auto)
	InternalURL string `json:"internal_url"` // Internal URL for runners (Docker network)
	Region      string `json:"region"`
	Capacity    int    `json:"capacity"`
}

// RegisterResponse is the relay registration response
type RegisterResponse struct {
	Status      string `json:"status"`
	URL         string `json:"url,omitempty"`          // Generated URL (if DNS auto-registration)
	InternalURL string `json:"internal_url,omitempty"` // Echoed back for confirmation
	DNSCreated  bool   `json:"dns_created,omitempty"`  // Whether DNS record was created

	// TLS certificate (if ACME is enabled)
	TLSCert   string `json:"tls_cert,omitempty"`   // PEM encoded certificate chain
	TLSKey    string `json:"tls_key,omitempty"`    // PEM encoded private key
	TLSExpiry string `json:"tls_expiry,omitempty"` // Certificate expiry time (RFC3339)
}

// HeartbeatRequest is the relay heartbeat request
type HeartbeatRequest struct {
	RelayID     string  `json:"relay_id" binding:"required"`
	Connections int     `json:"connections"`
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	LatencyMs   int     `json:"latency_ms,omitempty"` // Heartbeat round-trip latency in milliseconds
	NeedCert    bool    `json:"need_cert,omitempty"`  // Whether relay needs TLS certificate
}

// HeartbeatResponse is the relay heartbeat response
type HeartbeatResponse struct {
	Status string `json:"status"`

	// TLS certificate (if ACME is enabled and relay doesn't have current cert)
	TLSCert   string `json:"tls_cert,omitempty"`
	TLSKey    string `json:"tls_key,omitempty"`
	TLSExpiry string `json:"tls_expiry,omitempty"`
}

// UnregisterRequest is the relay unregistration request (graceful shutdown)
type UnregisterRequest struct {
	RelayID string `json:"relay_id" binding:"required"`
	Reason  string `json:"reason,omitempty"` // shutdown, maintenance, etc.
}
