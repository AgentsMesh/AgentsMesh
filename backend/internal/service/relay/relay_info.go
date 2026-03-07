package relay

import "time"

// RelayInfo holds information about a relay server.
//
// All fields are value types (string, int, float64, time.Time, bool).
// This struct is safe to shallow-copy via `copy := *info`.
// If a reference type (slice, map, pointer) is ever added, update all copy sites
// (Register, GetRelays, GetRelayByID, selectFromCandidatesLocked) to deep-copy.
type RelayInfo struct {
	ID                 string    `json:"id"`
	URL                string    `json:"url"`          // Public WebSocket URL via reverse proxy (e.g. wss://example.com/relay)
	Region             string    `json:"region"`       // Geographic region
	Capacity           int       `json:"capacity"`     // Maximum connections
	CurrentConnections int       `json:"connections"`  // Current active connections
	CPUUsage           float64   `json:"cpu_usage"`    // CPU usage percentage
	MemoryUsage        float64   `json:"memory_usage"` // Memory usage percentage
	LastHeartbeat      time.Time `json:"last_heartbeat"`
	Healthy            bool      `json:"healthy"`

	// Metrics for enhanced load balancing
	AvgLatencyMs int `json:"avg_latency_ms"` // Average heartbeat latency in milliseconds

	// GeoIP-resolved coordinates (populated at registration from relay's public IP)
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// HasGeoCoords returns true if the relay has resolved geographic coordinates.
// Uses non-zero check: (0,0) is in the Gulf of Guinea — no real relay would be there.
func (r *RelayInfo) HasGeoCoords() bool {
	return r.Latitude != 0 || r.Longitude != 0
}
