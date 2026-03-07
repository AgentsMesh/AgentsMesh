package relay

import (
	"testing"
)

func TestSelectRelayForPodGeo_NoGeoFallback(t *testing.T) {
	m := newTestManager(t)
	_ = m.Register(&RelayInfo{ID: "relay-1", URL: "wss://r1.com", Healthy: true, Capacity: 100})
	_ = m.Register(&RelayInfo{ID: "relay-2", URL: "wss://r2.com", Healthy: true, Capacity: 100})

	// Without geo, should behave like SelectRelayWithAffinity
	geoResult := m.SelectRelayForPodGeo(GeoSelectOptions{OrgSlug: "org-test"})
	affinityResult := m.SelectRelayWithAffinity("org-test")

	if geoResult == nil || affinityResult == nil {
		t.Fatal("both methods should return a relay")
	}
	if geoResult.ID != affinityResult.ID {
		t.Errorf("no-geo should match affinity: geo=%q affinity=%q", geoResult.ID, affinityResult.ID)
	}
}

func TestSelectRelayForPodGeo_PrefersNearby(t *testing.T) {
	m := newTestManager(t)

	_ = m.Register(&RelayInfo{
		ID: "relay-tokyo", URL: "wss://tokyo.relay.com",
		Healthy: true, Capacity: 100,
		Latitude: 35.6762, Longitude: 139.6503,
	})
	_ = m.Register(&RelayInfo{
		ID: "relay-frankfurt", URL: "wss://frankfurt.relay.com",
		Healthy: true, Capacity: 100,
		Latitude: 50.1109, Longitude: 8.6821,
	})
	_ = m.Register(&RelayInfo{
		ID: "relay-virginia", URL: "wss://virginia.relay.com",
		Healthy: true, Capacity: 100,
		Latitude: 38.9072, Longitude: -77.0369,
	})

	// User in Shanghai — closest to Tokyo
	result := m.SelectRelayForPodGeo(GeoSelectOptions{
		OrgSlug: "org-test", Latitude: 31.2304, Longitude: 121.4737, HasUserLocation: true,
	})
	if result == nil {
		t.Fatal("SelectRelayForPodGeo returned nil")
	}
	if result.ID != "relay-tokyo" {
		t.Errorf("Shanghai user should get Tokyo relay, got %q", result.ID)
	}

	// User in Berlin — closest to Frankfurt
	result = m.SelectRelayForPodGeo(GeoSelectOptions{
		OrgSlug: "org-test", Latitude: 52.5200, Longitude: 13.4050, HasUserLocation: true,
	})
	if result == nil {
		t.Fatal("SelectRelayForPodGeo returned nil")
	}
	if result.ID != "relay-frankfurt" {
		t.Errorf("Berlin user should get Frankfurt relay, got %q", result.ID)
	}

	// User in New York — closest to Virginia
	result = m.SelectRelayForPodGeo(GeoSelectOptions{
		OrgSlug: "org-test", Latitude: 40.7128, Longitude: -74.0060, HasUserLocation: true,
	})
	if result == nil {
		t.Fatal("SelectRelayForPodGeo returned nil")
	}
	if result.ID != "relay-virginia" {
		t.Errorf("New York user should get Virginia relay, got %q", result.ID)
	}
}

func TestSelectRelayForPodGeo_AffinityWithinNearby(t *testing.T) {
	m := newTestManager(t)

	// Two nearby relays in the same region (~400km apart, within 500km tolerance)
	_ = m.Register(&RelayInfo{
		ID: "relay-sh", URL: "wss://sh.relay.com",
		Healthy: true, Capacity: 100,
		Latitude: 31.2304, Longitude: 121.4737,
	})
	_ = m.Register(&RelayInfo{
		ID: "relay-nj", URL: "wss://nj.relay.com",
		Healthy: true, Capacity: 100,
		Latitude: 32.0603, Longitude: 118.7969,
	})

	// Same org should consistently get the same relay
	r1 := m.SelectRelayForPodGeo(GeoSelectOptions{
		OrgSlug: "stable-org", Latitude: 30.5, Longitude: 120.0, HasUserLocation: true,
	})
	r2 := m.SelectRelayForPodGeo(GeoSelectOptions{
		OrgSlug: "stable-org", Latitude: 30.5, Longitude: 120.0, HasUserLocation: true,
	})
	if r1 == nil || r2 == nil {
		t.Fatal("expected non-nil results")
	}
	if r1.ID != r2.ID {
		t.Errorf("same org should get stable relay: %q vs %q", r1.ID, r2.ID)
	}
}

func TestSelectRelayForPodGeo_FallbackWhenNearbyUnhealthy(t *testing.T) {
	m := newTestManager(t)

	_ = m.Register(&RelayInfo{
		ID: "relay-near", URL: "wss://near.relay.com",
		Healthy: true, Capacity: 100,
		Latitude: 31.2304, Longitude: 121.4737,
	})
	_ = m.Register(&RelayInfo{
		ID: "relay-far", URL: "wss://far.relay.com",
		Healthy: true, Capacity: 100,
		Latitude: 51.5074, Longitude: -0.1278,
	})

	m.mu.Lock()
	m.relays["relay-near"].Healthy = false
	m.mu.Unlock()

	result := m.SelectRelayForPodGeo(GeoSelectOptions{
		OrgSlug: "org-test", Latitude: 31.0, Longitude: 121.0, HasUserLocation: true,
	})
	if result == nil {
		t.Fatal("should fallback to far relay")
	}
	if result.ID != "relay-far" {
		t.Errorf("should select far relay as fallback, got %q", result.ID)
	}
}

func TestSelectRelayForPodGeo_RelayWithoutGeo(t *testing.T) {
	m := newTestManager(t)

	_ = m.Register(&RelayInfo{
		ID: "relay-nogeo", URL: "wss://nogeo.relay.com",
		Healthy: true, Capacity: 100,
	})
	_ = m.Register(&RelayInfo{
		ID: "relay-geo", URL: "wss://geo.relay.com",
		Healthy: true, Capacity: 100,
		Latitude: 31.2304, Longitude: 121.4737,
	})

	result := m.SelectRelayForPodGeo(GeoSelectOptions{
		OrgSlug: "org-test", Latitude: 31.0, Longitude: 121.0, HasUserLocation: true,
	})
	if result == nil {
		t.Fatal("expected a relay")
	}
	if result.ID != "relay-geo" {
		t.Errorf("should prefer relay with geo data, got %q", result.ID)
	}
}

func TestSelectRelayForPodGeo_NoRelays(t *testing.T) {
	m := newTestManager(t)
	result := m.SelectRelayForPodGeo(GeoSelectOptions{
		OrgSlug: "org-test", Latitude: 31.0, Longitude: 121.0, HasUserLocation: true,
	})
	if result != nil {
		t.Error("should return nil with no relays")
	}
}

func TestSelectRelayForPodGeo_AllOverloaded(t *testing.T) {
	m := newTestManager(t)
	_ = m.Register(&RelayInfo{
		ID: "relay-1", URL: "wss://r1.com",
		Healthy: true, Capacity: 100, CPUUsage: 90,
		Latitude: 31.2304, Longitude: 121.4737,
	})

	result := m.SelectRelayForPodGeo(GeoSelectOptions{
		OrgSlug: "org-test", Latitude: 31.0, Longitude: 121.0, HasUserLocation: true,
	})
	if result == nil {
		t.Fatal("overloaded relay should be selected via fallback")
	}
	if result.ID != "relay-1" {
		t.Errorf("expected relay-1 via fallback, got %q", result.ID)
	}
}

func TestSelectRelayForPodGeo_AllTrulyUnavailable(t *testing.T) {
	m := newTestManager(t)
	_ = m.Register(&RelayInfo{
		ID: "relay-1", URL: "wss://r1.com",
		Healthy: true, Capacity: 100,
		Latitude: 31.2304, Longitude: 121.4737,
	})

	m.mu.Lock()
	m.relays["relay-1"].Healthy = false
	m.mu.Unlock()

	result := m.SelectRelayForPodGeo(GeoSelectOptions{
		OrgSlug: "org-test", Latitude: 31.0, Longitude: 121.0, HasUserLocation: true,
	})
	if result != nil {
		t.Error("unhealthy relay should not be selected even via fallback")
	}
}

func TestSelectRelayForPodGeo_SingleRelay(t *testing.T) {
	m := newTestManager(t)
	_ = m.Register(&RelayInfo{
		ID: "relay-only", URL: "wss://only.relay.com",
		Healthy: true, Capacity: 100,
		Latitude: 31.2304, Longitude: 121.4737,
	})

	result := m.SelectRelayForPodGeo(GeoSelectOptions{
		OrgSlug: "org-test", Latitude: 51.5074, Longitude: -0.1278, HasUserLocation: true,
	})
	if result == nil {
		t.Fatal("should return the only relay even if far")
	}
	if result.ID != "relay-only" {
		t.Errorf("expected relay-only, got %q", result.ID)
	}
}

func TestSelectRelayForPodGeo_ThresholdCap(t *testing.T) {
	m := newTestManager(t)

	_ = m.Register(&RelayInfo{
		ID: "relay-tokyo", URL: "wss://tokyo.relay.com",
		Healthy: true, Capacity: 100,
		Latitude: 35.6762, Longitude: 139.6503,
	})
	_ = m.Register(&RelayInfo{
		ID: "relay-virginia", URL: "wss://virginia.relay.com",
		Healthy: true, Capacity: 100,
		Latitude: 38.9072, Longitude: -77.0369,
	})

	// Shanghai user — Tokyo ~1800km, Virginia ~12000km
	// threshold = min(1800*1.5, 1800+2000) = 2700, Virginia far beyond
	result := m.SelectRelayForPodGeo(GeoSelectOptions{
		OrgSlug: "org-test", Latitude: 31.2304, Longitude: 121.4737, HasUserLocation: true,
	})
	if result == nil {
		t.Fatal("should return a relay")
	}
	if result.ID != "relay-tokyo" {
		t.Errorf("threshold cap should prevent Virginia from being nearby, got %q", result.ID)
	}
}
