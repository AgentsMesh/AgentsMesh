package geo

type Location struct {
	Latitude  float64
	Longitude float64
	Country   string // ISO 3166-1 alpha-2 (e.g. "US", "CN")
}

type Resolver interface {
	Resolve(ip string) *Location
	Close() error
}
