package geo

type NoOpResolver struct{}

func NewNoOpResolver() *NoOpResolver {
	return &NoOpResolver{}
}

func (r *NoOpResolver) Resolve(_ string) *Location { return nil }

func (r *NoOpResolver) Close() error { return nil }
