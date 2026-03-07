package geo

// NoOpResolver is a Resolver that always returns nil.
// Used when no MMDB file is available (graceful degradation).
type NoOpResolver struct{}

// NewNoOpResolver creates a NoOpResolver.
func NewNoOpResolver() *NoOpResolver {
	return &NoOpResolver{}
}

// Resolve always returns nil — no GeoIP data is available.
func (r *NoOpResolver) Resolve(_ string) *Location { return nil }

// Close is a no-op since NoOpResolver holds no resources.
func (r *NoOpResolver) Close() error { return nil }
