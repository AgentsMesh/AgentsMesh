// Package geo provides IP-to-location resolution using MMDB databases.
// It supports MaxMind and DB-IP MMDB formats, and includes a Haversine distance
// calculator for geo-aware relay selection.
package geo

// Location represents a geographic coordinate with country info.
type Location struct {
	Latitude  float64
	Longitude float64
	Country   string // ISO 3166-1 alpha-2 (e.g. "US", "CN")
}

// Resolver resolves an IP address to a geographic location.
type Resolver interface {
	// Resolve returns the location for the given IP, or nil if unknown.
	Resolve(ip string) *Location
	// Close releases resources held by the resolver.
	Close() error
}
