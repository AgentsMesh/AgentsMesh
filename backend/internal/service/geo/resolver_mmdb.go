package geo

import (
	"net"

	"github.com/oschwald/maxminddb-golang"
)

type mmdbRecord struct {
	Location struct {
		Latitude  float64 `maxminddb:"latitude"`
		Longitude float64 `maxminddb:"longitude"`
	} `maxminddb:"location"`
	Country struct {
		ISOCode string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
}

type MMDBResolver struct {
	db *maxminddb.Reader
}

func NewMMDBResolver(path string) (*MMDBResolver, error) {
	db, err := maxminddb.Open(path)
	if err != nil {
		return nil, err
	}
	return &MMDBResolver{db: db}, nil
}

// Resolve returns the geographic location for the given IP string.
// Returns nil if the IP cannot be parsed or looked up.
func (r *MMDBResolver) Resolve(ip string) *Location {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return nil
	}

	var rec mmdbRecord
	if err := r.db.Lookup(parsed, &rec); err != nil {
		return nil
	}

	if rec.Location.Latitude == 0 && rec.Location.Longitude == 0 && rec.Country.ISOCode == "" {
		return nil
	}

	return &Location{
		Latitude:  rec.Location.Latitude,
		Longitude: rec.Location.Longitude,
		Country:   rec.Country.ISOCode,
	}
}

func (r *MMDBResolver) Close() error {
	return r.db.Close()
}
