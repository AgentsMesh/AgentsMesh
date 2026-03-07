package geo

import (
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
)

// createTestMMDB generates a minimal MMDB file for testing.
// Contains entries for a few well-known IPs.
func createTestMMDB(t *testing.T) string {
	t.Helper()

	writer, err := mmdbwriter.New(mmdbwriter.Options{
		DatabaseType:            "Test-City",
		RecordSize:              28,
		IncludeReservedNetworks: true,
	})
	if err != nil {
		t.Fatalf("failed to create mmdb writer: %v", err)
	}

	// Insert 8.8.8.8 → Mountain View, CA (37.386, -122.084), US
	_, network8, _ := net.ParseCIDR("8.8.8.0/24")
	err = writer.Insert(network8, mmdbtype.Map{
		"location": mmdbtype.Map{
			"latitude":  mmdbtype.Float64(37.386),
			"longitude": mmdbtype.Float64(-122.084),
		},
		"country": mmdbtype.Map{
			"iso_code": mmdbtype.String("US"),
		},
	})
	if err != nil {
		t.Fatalf("failed to insert 8.8.8.0/24: %v", err)
	}

	// Insert 1.1.1.0/24 → Sydney, AU (-33.8688, 151.2093)
	_, network1, _ := net.ParseCIDR("1.1.1.0/24")
	err = writer.Insert(network1, mmdbtype.Map{
		"location": mmdbtype.Map{
			"latitude":  mmdbtype.Float64(-33.8688),
			"longitude": mmdbtype.Float64(151.2093),
		},
		"country": mmdbtype.Map{
			"iso_code": mmdbtype.String("AU"),
		},
	})
	if err != nil {
		t.Fatalf("failed to insert 1.1.1.0/24: %v", err)
	}

	// Insert 10.0.0.0/8 → no location data (simulate unknown)
	_, network10, _ := net.ParseCIDR("10.0.0.0/8")
	err = writer.Insert(network10, mmdbtype.Map{
		"location": mmdbtype.Map{
			"latitude":  mmdbtype.Float64(0),
			"longitude": mmdbtype.Float64(0),
		},
	})
	if err != nil {
		t.Fatalf("failed to insert 10.0.0.0/8: %v", err)
	}

	// Write to temp file
	path := filepath.Join(t.TempDir(), "test.mmdb")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer f.Close()

	_, err = writer.WriteTo(f)
	if err != nil {
		t.Fatalf("failed to write mmdb: %v", err)
	}

	return path
}

func TestNoOpResolver(t *testing.T) {
	r := NewNoOpResolver()

	loc := r.Resolve("8.8.8.8")
	if loc != nil {
		t.Error("NoOpResolver should always return nil")
	}

	if err := r.Close(); err != nil {
		t.Errorf("NoOpResolver.Close() error: %v", err)
	}
}

func TestMMDBResolver_InvalidPath(t *testing.T) {
	_, err := NewMMDBResolver("/nonexistent/path.mmdb")
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestMMDBResolver_ResolveKnownIP(t *testing.T) {
	path := createTestMMDB(t)
	r, err := NewMMDBResolver(path)
	if err != nil {
		t.Fatalf("NewMMDBResolver failed: %v", err)
	}
	defer r.Close()

	// 8.8.8.8 → Mountain View
	loc := r.Resolve("8.8.8.8")
	if loc == nil {
		t.Fatal("expected non-nil location for 8.8.8.8")
	}
	if loc.Country != "US" {
		t.Errorf("country: got %q, want %q", loc.Country, "US")
	}
	if loc.Latitude < 37 || loc.Latitude > 38 {
		t.Errorf("latitude out of range: %v", loc.Latitude)
	}
	if loc.Longitude > -121 || loc.Longitude < -123 {
		t.Errorf("longitude out of range: %v", loc.Longitude)
	}
}

func TestMMDBResolver_ResolveSydney(t *testing.T) {
	path := createTestMMDB(t)
	r, err := NewMMDBResolver(path)
	if err != nil {
		t.Fatalf("NewMMDBResolver failed: %v", err)
	}
	defer r.Close()

	// 1.1.1.1 → Sydney
	loc := r.Resolve("1.1.1.1")
	if loc == nil {
		t.Fatal("expected non-nil location for 1.1.1.1")
	}
	if loc.Country != "AU" {
		t.Errorf("country: got %q, want %q", loc.Country, "AU")
	}
	if loc.Latitude > -33 || loc.Latitude < -34 {
		t.Errorf("latitude out of range: %v", loc.Latitude)
	}
}

func TestMMDBResolver_ResolveUnknownIP(t *testing.T) {
	path := createTestMMDB(t)
	r, err := NewMMDBResolver(path)
	if err != nil {
		t.Fatalf("NewMMDBResolver failed: %v", err)
	}
	defer r.Close()

	// 192.168.1.1 — not in test DB
	loc := r.Resolve("192.168.1.1")
	if loc != nil {
		t.Error("expected nil for IP not in database")
	}
}

func TestMMDBResolver_ResolveZeroCoords(t *testing.T) {
	path := createTestMMDB(t)
	r, err := NewMMDBResolver(path)
	if err != nil {
		t.Fatalf("NewMMDBResolver failed: %v", err)
	}
	defer r.Close()

	// 10.0.0.1 → lat=0, lng=0, no country → should return nil
	loc := r.Resolve("10.0.0.1")
	if loc != nil {
		t.Error("expected nil for (0,0) with no country")
	}
}

func TestMMDBResolver_ResolveInvalidIP(t *testing.T) {
	path := createTestMMDB(t)
	r, err := NewMMDBResolver(path)
	if err != nil {
		t.Fatalf("NewMMDBResolver failed: %v", err)
	}
	defer r.Close()

	if loc := r.Resolve("not-an-ip"); loc != nil {
		t.Error("expected nil for invalid IP string")
	}
	if loc := r.Resolve(""); loc != nil {
		t.Error("expected nil for empty string")
	}
}

func TestMMDBResolver_ResolveIPv6(t *testing.T) {
	path := createTestMMDB(t)
	r, err := NewMMDBResolver(path)
	if err != nil {
		t.Fatalf("NewMMDBResolver failed: %v", err)
	}
	defer r.Close()

	// IPv6 loopback — not in test DB, should return nil without panic
	if loc := r.Resolve("::1"); loc != nil {
		t.Error("expected nil for IPv6 loopback not in DB")
	}

	// IPv6 format of known IPv4 (may not be in test DB depending on MMDB config)
	// Ensure no panic
	_ = r.Resolve("2001:4860:4860::8888")
}

func TestMMDBResolver_Close(t *testing.T) {
	path := createTestMMDB(t)
	r, err := NewMMDBResolver(path)
	if err != nil {
		t.Fatalf("NewMMDBResolver failed: %v", err)
	}

	if err := r.Close(); err != nil {
		t.Errorf("Close() error: %v", err)
	}
}

func TestMMDBResolver_ResolveAfterClose(t *testing.T) {
	path := createTestMMDB(t)
	r, err := NewMMDBResolver(path)
	if err != nil {
		t.Fatalf("NewMMDBResolver failed: %v", err)
	}
	r.Close()

	// Lookup on closed reader should return nil (triggers Lookup error path)
	loc := r.Resolve("8.8.8.8")
	if loc != nil {
		t.Error("expected nil after Close()")
	}
}

// TestResolverInterface verifies both implementations satisfy the interface.
func TestResolverInterface(t *testing.T) {
	var _ Resolver = (*NoOpResolver)(nil)
	var _ Resolver = (*MMDBResolver)(nil)
}
