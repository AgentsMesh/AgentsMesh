package relay

import (
	"testing"
)

func TestSelectFromCandidatesLocked_EmptyCandidates(t *testing.T) {
	m := newTestManager(t)
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := m.selectFromCandidatesLocked("org-test", nil)
	if result != nil {
		t.Error("should return nil for empty candidates")
	}

	result = m.selectFromCandidatesLocked("org-test", []string{})
	if result != nil {
		t.Error("should return nil for zero-length candidates")
	}
}

func TestSelectFromCandidatesLocked_StaleIDs(t *testing.T) {
	m := newTestManager(t)
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := m.selectFromCandidatesLocked("org-test", []string{"nonexistent-1", "nonexistent-2"})
	if result != nil {
		t.Error("should return nil when candidate IDs are not in relay map")
	}
}

func TestHashStringDeterministic(t *testing.T) {
	// Same input should always produce the same hash
	h1 := hashString("org-test" + "relay-1")
	h2 := hashString("org-test" + "relay-1")
	if h1 != h2 {
		t.Errorf("hashString not deterministic: %d vs %d", h1, h2)
	}

	// Different inputs should (almost certainly) produce different hashes
	h3 := hashString("org-test" + "relay-2")
	if h1 == h3 {
		t.Errorf("hash collision: %q and %q both produce %d", "relay-1", "relay-2", h1)
	}
}

func TestHashStringMatchesStdlib(t *testing.T) {
	// Verify manual FNV-1a matches the expected FNV-1a algorithm output
	// FNV-1a 32-bit constants: offset=2166136261, prime=16777619
	input := "hello"
	got := hashString(input)

	// Manually compute expected value
	expected := uint32(2166136261)
	for i := 0; i < len(input); i++ {
		expected ^= uint32(input[i])
		expected *= 16777619
	}
	if got != expected {
		t.Errorf("hashString(%q) = %d, expected %d", input, got, expected)
	}
}

func TestHashStringPairEquivalence(t *testing.T) {
	// hashStringPair(a, b) must produce the same result as hashString(a+b)
	cases := []struct{ a, b string }{
		{"org-test", "relay-1"},
		{"", "relay-1"},
		{"org-test", ""},
		{"", ""},
		{"hello", "world"},
	}
	for _, tc := range cases {
		concat := hashString(tc.a + tc.b)
		pair := hashStringPair(tc.a, tc.b)
		if concat != pair {
			t.Errorf("hashStringPair(%q, %q) = %d, hashString(%q) = %d",
				tc.a, tc.b, pair, tc.a+tc.b, concat)
		}
	}
}

func TestHeartbeatClampsNegativeValues(t *testing.T) {
	m := newTestManager(t)
	if err := m.Register(&RelayInfo{ID: "relay-1", URL: "wss://r1.com"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Pass negative values — should be clamped to 0
	if err := m.HeartbeatWithLatency("relay-1", -5, -10.5, -20.3, 0); err != nil {
		t.Fatalf("HeartbeatWithLatency failed: %v", err)
	}

	r := m.GetRelayByID("relay-1")
	if r.CurrentConnections != 0 {
		t.Errorf("connections: got %d, want 0 (clamped)", r.CurrentConnections)
	}
	if r.CPUUsage != 0 {
		t.Errorf("cpu: got %f, want 0 (clamped)", r.CPUUsage)
	}
	if r.MemoryUsage != 0 {
		t.Errorf("memory: got %f, want 0 (clamped)", r.MemoryUsage)
	}
}

func TestHasGeoCoords(t *testing.T) {
	tests := []struct {
		name     string
		lat, lng float64
		want     bool
	}{
		{"both zero", 0, 0, false},
		{"lat non-zero", 31.23, 0, true},
		{"lng non-zero", 0, 121.47, true},
		{"both non-zero", 31.23, 121.47, true},
		{"negative coords", -33.87, 151.21, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RelayInfo{Latitude: tt.lat, Longitude: tt.lng}
			if got := r.HasGeoCoords(); got != tt.want {
				t.Errorf("HasGeoCoords() = %v, want %v", got, tt.want)
			}
		})
	}
}
