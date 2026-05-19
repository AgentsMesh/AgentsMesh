package relay

import "time"

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

	AvgLatencyMs int `json:"avg_latency_ms"` // Average heartbeat latency in milliseconds

	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

func (r *RelayInfo) HasGeoCoords() bool {
	return r.Latitude != 0 || r.Longitude != 0
}
