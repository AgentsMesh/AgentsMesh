package internal

// RegisterRequest is the relay registration request
type RegisterRequest struct {
	RelayID   string `json:"relay_id" binding:"required,max=128"`
	RelayName string `json:"relay_name" binding:"max=128"`              // Name for DNS auto-registration (e.g., "us-east-1")
	IP        string `json:"ip" binding:"omitempty,ip"`                 // Public IP for DNS and geo-aware selection. Required for geo; if omitted and URL uses a hostname, relay won't have geo coords.
	URL       string `json:"url" binding:"omitempty"`                   // Public URL via reverse proxy (e.g. wss://example.com/relay). Scheme validated by parseRelayURL (ws/wss only).
	Region    string `json:"region" binding:"max=64"`
	Capacity  int    `json:"capacity" binding:"min=0,max=100000"`
}

// RegisterResponse is the relay registration response
type RegisterResponse struct {
	Status     string `json:"status"`
	URL        string `json:"url,omitempty"`         // Generated URL (if DNS auto-registration)
	DNSCreated bool   `json:"dns_created,omitempty"` // Whether DNS record was created

	// TLS certificate (if ACME is enabled)
	TLSCert   string `json:"tls_cert,omitempty"`   // PEM encoded certificate chain
	TLSKey    string `json:"tls_key,omitempty"`    // PEM encoded private key
	TLSExpiry string `json:"tls_expiry,omitempty"` // Certificate expiry time (RFC3339)
}

// HeartbeatRequest is the relay heartbeat request
type HeartbeatRequest struct {
	RelayID     string  `json:"relay_id" binding:"required,max=128"`
	Connections int     `json:"connections" binding:"min=0,max=1000000"`
	CPUUsage    float64 `json:"cpu_usage" binding:"min=0,max=100"`
	MemoryUsage float64 `json:"memory_usage" binding:"min=0,max=100"`
	LatencyMs   int     `json:"latency_ms,omitempty" binding:"min=0,max=60000"`
	NeedCert    bool    `json:"need_cert,omitempty"`
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
	RelayID string `json:"relay_id" binding:"required,max=128"`
	Reason  string `json:"reason,omitempty" binding:"max=256"` // shutdown, maintenance, etc.
}

// UnregisterResponse is the relay unregistration response
type UnregisterResponse struct {
	Status  string `json:"status"`
	RelayID string `json:"relay_id,omitempty"`
	Reason  string `json:"reason,omitempty"`
}
